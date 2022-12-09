package forwarder

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

const (
	errCLIPodUnavailable = "storageos cli pod unavailable for request - please ensure storageos-cli deployment is running"
	cliLabel             = "app.kubernetes.io/component=storageos-cli"
)

type forwarder struct {
	clientConfig *rest.Config
	log          *logger.Logger
}

func NewForwarder(log *logger.Logger) (*forwarder, error) {
	clientConfig, err := pluginutils.NewClientConfig()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Verbose = true
	return &forwarder{
		clientConfig: clientConfig,
		log:          log,
	}, nil
}

func (f *forwarder) ForwardToCLIPod(command []string) error {
	cliPodNamespace, err := f.getCLIPodNamespace()
	if err != nil {
		return err
	}
	cliPodName, err := f.getCLIPodName(cliPodNamespace)
	if err != nil {
		return err
	}

	stdout, stderr, err := pluginutils.ExecToPod(f.clientConfig, command, "", cliPodName, cliPodNamespace, nil)
	if stdout != "" {
		f.log.Info(stdout)
	}
	if stderr != "" {
		f.log.Info(stderr)
	}

	return err
}

func (f *forwarder) getCLIPodNamespace() (string, error) {
	stosCluster, err := pluginutils.GetFirstStorageOSCluster(f.clientConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to discover storageos namespace for storageos-cli deployment")
	}
	return stosCluster.GetNamespace(), nil
}

func (f *forwarder) getCLIPodName(namespace string) (string, error) {
	cliPodList, err := pluginutils.ListPods(f.clientConfig, namespace, cliLabel)
	if err != nil {
		return "", errors.WithStack(errors.Wrap(err, errCLIPodUnavailable))
	}
	if len(cliPodList.Items) == 0 {
		return "", errors.WithStack(fmt.Errorf(errCLIPodUnavailable))
	}

	for _, pod := range cliPodList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.GetName(), nil
		}
	}

	return "", errors.WithStack(fmt.Errorf(errCLIPodUnavailable))
}
