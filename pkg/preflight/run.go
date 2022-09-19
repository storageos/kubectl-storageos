package preflight

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	cursor "github.com/ahmetalpbalkan/go-cursor"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/replicatedhq/troubleshoot/cmd/util"
	analyzer "github.com/replicatedhq/troubleshoot/pkg/analyze"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootclientsetscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"github.com/replicatedhq/troubleshoot/pkg/docrewrite"
	"github.com/replicatedhq/troubleshoot/pkg/k8sutil"
	"github.com/replicatedhq/troubleshoot/pkg/preflight"
	"github.com/replicatedhq/troubleshoot/pkg/specs"
	"github.com/spf13/viper"
	spin "github.com/tj/go-spin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	// httpSpecTimeout defines how long to wait to fetch a remote spec.
	httpSpecTimeout = 10 * time.Second

	// httpUploadTimeout defines how long to wait to upload a bundle.
	httpUploadTimeout = 10 * time.Minute
)

func Run(v *viper.Viper, arg string) error {
	fmt.Print(cursor.Hide())
	defer fmt.Print(cursor.Show())

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		<-signalChan
		fmt.Print(cursor.Show())
		os.Exit(0)
	}()

	var collectResults []preflight.CollectResult
	preflightSpecName := ""
	finishedCh := make(chan bool, 1)
	progressCh := make(chan interface{}) // non-zero buffer will result in missed messages

	defer func() {
		close(finishedCh)
		close(progressCh)
	}()

	if v.GetBool("interactive") {
		s := spin.New()
		go func() {
			lastMsg := ""
			for {
				select {
				case msg, ok := <-progressCh:
					if !ok {
						continue
					}
					switch msg := msg.(type) {
					case error:
						c := color.New(color.FgHiRed)
						c.Println(fmt.Sprintf("%s\r * %v", cursor.ClearEntireLine(), msg))
					case string:
						if lastMsg == msg {
							break
						}
						lastMsg = msg
						c := color.New(color.FgCyan)
						c.Println(fmt.Sprintf("%s\r * %s", cursor.ClearEntireLine(), msg))
					}
				case <-time.After(time.Millisecond * 100):
					fmt.Printf("\r  \033[36mRunning Preflight checks\033[m %s ", s.Next())
				case <-finishedCh:
					fmt.Printf("\r%s\r", cursor.ClearEntireLine())
					return
				}
			}
		}()
	} else {
		// make sure we don't block any senders
		go func() {
			for {
				select {
				case msg, ok := <-progressCh:
					if !ok {
						return
					}
					fmt.Fprintf(os.Stderr, "%v\n", msg)
				case <-finishedCh:
					return
				}
			}
		}()
	}

	specs, err := loadSpecs(arg)
	if err != nil {
		return err
	}

	if err := troubleshootclientsetscheme.AddToScheme(scheme.Scheme); err != nil {
		return errors.Wrap(err, "failed to load scheme")
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode

	uploadResultsTo := ""
	for _, spec := range specs {
		obj, _, err := decode([]byte(spec), nil, nil)
		if err != nil {
			return errors.Wrapf(err, "failed to parse %s", arg)
		}

		if preflightSpec, ok := obj.(*troubleshootv1beta2.Preflight); ok {
			r, err := collectInCluster(preflightSpec, finishedCh, progressCh)
			if err != nil {
				return errors.Wrap(err, "failed to collect in cluster")
			}
			collectResults = append(collectResults, *r)
			preflightSpecName = preflightSpec.Name
			uploadResultsTo = preflightSpec.Spec.UploadResultsTo
		}
		if hostPreflightSpec, ok := obj.(*troubleshootv1beta2.HostPreflight); ok {
			if len(hostPreflightSpec.Spec.Collectors) > 0 {
				r, err := collectHost(hostPreflightSpec, finishedCh, progressCh)
				if err != nil {
					return errors.Wrap(err, "failed to collect from host")
				}
				collectResults = append(collectResults, *r)
			}
			if len(hostPreflightSpec.Spec.RemoteCollectors) > 0 {
				r, err := collectRemote(hostPreflightSpec, finishedCh, progressCh)
				if err != nil {
					return errors.Wrap(err, "failed to collect remotely")
				}
				collectResults = append(collectResults, *r)
			}
			preflightSpecName = hostPreflightSpec.Name
		}
	}

	if collectResults == nil {
		return errors.New("no results")
	}

	analyzeResults := []*analyzer.AnalyzeResult{}
	for _, res := range collectResults {
		analyzeResults = append(analyzeResults, res.Analyze()...)
	}

	if uploadResultsTo != "" {
		err := uploadResults(uploadResultsTo, analyzeResults)
		if err != nil {
			progressCh <- err
		}
	}

	finishedCh <- true

	if v.GetBool("interactive") {
		if len(analyzeResults) == 0 {
			return errors.New("no data has been collected")
		}
		return showInteractiveResults(preflightSpecName, analyzeResults)
	}

	return showStdoutResults(v.GetString("format"), preflightSpecName, analyzeResults)
}

func loadSpecs(arg string) ([]string, error) {
	spec, err := loadSpecContent(arg)
	if err != nil {
		return []string{}, err
	}
	spec, err = docrewrite.ConvertToV1Beta2(spec)
	if err != nil {
		return []string{}, errors.Wrap(err, "failed to convert to v1beta2")
	}
	return strings.Split(string(spec), "\n---\n"), nil
}

func loadSpecContent(arg string) ([]byte, error) {
	if strings.HasPrefix(arg, "secret/") {
		// format secret/namespace-name/secret-name
		pathParts := strings.Split(arg, "/")
		if len(pathParts) != 3 {
			return nil, errors.Errorf("path %s must have 3 components", arg)
		}

		spec, err := specs.LoadFromSecret(pathParts[1], pathParts[2], "preflight-spec")
		if err != nil {
			return nil, errors.Wrap(err, "failed to get spec from secret")
		}
		return spec, nil
	}
	if _, err := os.Stat(arg); err == nil {
		return ioutil.ReadFile(arg)
	}
	if util.IsURL(arg) {
		req, err := http.NewRequest("GET", arg, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Replicated_Preflight/v1beta2")
		client := &http.Client{
			Timeout: httpSpecTimeout,
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		return ioutil.ReadAll(resp.Body)
	}
	return nil, fmt.Errorf("%s is not a URL and was not found", arg)
}

func collectInCluster(preflightSpec *troubleshootv1beta2.Preflight, finishedCh chan bool, progressCh chan interface{}) (*preflight.CollectResult, error) {
	v := viper.GetViper()

	restConfig, err := k8sutil.GetRESTConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert kube flags to rest config")
	}

	collectOpts := preflight.CollectOpts{
		Namespace:              v.GetString("namespace"),
		IgnorePermissionErrors: v.GetBool("collect-without-permissions"),
		ProgressChan:           progressCh,
		KubernetesRestConfig:   restConfig,
	}

	if err := parseTimeFlags(v, preflightSpec.Spec.Collectors); err != nil {
		return nil, err
	}

	collectResults, err := preflight.Collect(collectOpts, preflightSpec)
	if err == nil {
		return &collectResults, nil
	}
	if !collectResults.IsRBACAllowed() && preflightSpec.Spec.UploadResultsTo != "" {
		clusterCollectResults := collectResults.(preflight.ClusterCollectResult)
		err := uploadErrors(preflightSpec.Spec.UploadResultsTo, clusterCollectResults.Collectors)
		if err != nil {
			progressCh <- err
		}
	}
	return nil, err
}

func collectRemote(preflightSpec *troubleshootv1beta2.HostPreflight, finishedCh chan bool, progressCh chan interface{}) (*preflight.CollectResult, error) {
	v := viper.GetViper()

	restConfig, err := k8sutil.GetRESTConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert kube flags to rest config")
	}

	labelSelector, err := labels.Parse(v.GetString("selector"))
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse selector")
	}

	namespace := v.GetString("namespace")
	if namespace == "" {
		namespace = "default"
	}

	timeout := v.GetDuration("request-timeout")
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	collectOpts := preflight.CollectOpts{
		Namespace:              namespace,
		IgnorePermissionErrors: v.GetBool("collect-without-permissions"),
		ProgressChan:           progressCh,
		KubernetesRestConfig:   restConfig,
		Image:                  v.GetString("collector-image"),
		PullPolicy:             v.GetString("collector-pullpolicy"),
		LabelSelector:          labelSelector.String(),
		Timeout:                timeout,
	}

	collectResults, err := preflight.CollectRemote(collectOpts, preflightSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect from remote")
	}

	return &collectResults, nil
}

func collectHost(hostPreflightSpec *troubleshootv1beta2.HostPreflight, finishedCh chan bool, progressCh chan interface{}) (*preflight.CollectResult, error) {
	collectOpts := preflight.CollectOpts{
		ProgressChan: progressCh,
	}

	collectResults, err := preflight.CollectHost(collectOpts, hostPreflightSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect from host")
	}

	return &collectResults, nil
}

func parseTimeFlags(v *viper.Viper, collectors []*troubleshootv1beta2.Collect) error {
	var (
		sinceTime time.Time
		err       error
	)
	switch {
	case v.GetString("since-time") != "" && v.GetString("since") != "":
		return errors.Errorf("at most one of `sinceTime` or `since` may be specified")
	case v.GetString("since-time") != "":
		sinceTime, err = time.Parse(time.RFC3339, v.GetString("since-time"))
		if err != nil {
			return errors.Wrap(err, "unable to parse --since-time flag")
		}
	case v.GetString("since") != "":
		parsedDuration, err := time.ParseDuration(v.GetString("since"))
		if err != nil {
			return errors.Wrap(err, "unable to parse --since flag")
		}
		now := time.Now()
		sinceTime = now.Add(0 - parsedDuration)
	}

	for _, collector := range collectors {
		if collector.Logs != nil {
			if collector.Logs.Limits == nil {
				collector.Logs.Limits = new(troubleshootv1beta2.LogLimits)
			}
			collector.Logs.Limits.SinceTime = metav1.NewTime(sinceTime)
		}
	}
	return nil
}
