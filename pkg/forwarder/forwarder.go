package forwarder

import (
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
)

const (
	errCLIPodUnavailable = "storageos cli pod unavailable for request - please ensure storageos-cli deployment is running"
	cliLabel             = "app.kubernetes.io/component=storageos-cli"
)

func ForwardToCLIPod(log *logger.Logger, command []string) error {
	clientConfig, err := pluginutils.NewClientConfig()
	if err != nil {
		return errors.WithStack(err)
	}
	log.Verbose = true

	cliPodNamespace, err := getCLIPodNamespace(clientConfig)
	if err != nil {
		return err
	}

	cliPodName, err := getCLIPodName(clientConfig, cliPodNamespace)
	if err != nil {
		return err
	}

	stdout, stderr, err := pluginutils.ExecToPod(clientConfig, command, "", cliPodName, cliPodNamespace, nil)
	if stdout != "" {
		log.Info(stdout)
	}
	if stderr != "" {
		log.Info(stderr)
	}

	return err
}

func getCLIPodNamespace(clientConfig *rest.Config) (string, error) {
	stosCluster, err := pluginutils.GetFirstStorageOSCluster(clientConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to discover storageos namespace for storageos-cli deployment")
	}
	return stosCluster.GetNamespace(), nil
}

func getCLIPodName(clientConfig *rest.Config, namespace string) (string, error) {
	cliPodList, err := pluginutils.ListPods(clientConfig, namespace, cliLabel)
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
