package installer

import (
	"context"
	"fmt"
	"strings"

	goversion "github.com/hashicorp/go-version"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	oldOperatorYamlUrlPrefix = "https://github.com/storageos/cluster-operator/releases/download/"
	oldOperatorYamlUrlSuffix = "/storageos-operator.yaml"
	oldClusterYamlUrlPrefix  = "https://raw.githubusercontent.com/storageos/cluster-operator/"
	oldClusterYamlUrlSuffix  = "/deploy/crds/storageos.com_v1_storageoscluster_cr.yaml"
	oldSecretYamlUrlPrefix   = "https://raw.githubusercontent.com/storageos/cluster-operator/"
	oldSecretYamlUrlSuffix   = "/deploy/secret.yaml"
	oldOperatorName          = "storageos-cluster-operator"
	oldOperatorNamespace     = "storageos-operator"
	oldClusterNamespace      = "kube-system"

	newOperatorName = "storageos-controller-manager"
)

func GetExistingOperatorVersion(namespace string) (string, error) {
	oldNS := oldOperatorNamespace
	newNS := defaultStosOperatorNS
	if namespace != "" {
		oldNS = namespace
		newNS = namespace
	}
	config, err := pluginutils.NewClientConfig()
	if err != nil {
		return "", err
	}

	clientset, err := pluginutils.GetClientsetFromConfig(config)
	if err != nil {
		return "", err
	}

	stosDeployment, err := clientset.AppsV1().Deployments(oldNS).Get(context.TODO(), oldOperatorName, metav1.GetOptions{})
	if err != nil {
		stosDeployment, err = clientset.AppsV1().Deployments(newNS).Get(context.TODO(), newOperatorName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
	}
	imageName := stosDeployment.Spec.Template.Spec.Containers[0].Image
	splitImageName := strings.SplitAfter(imageName, ":")
	version := splitImageName[len(splitImageName)-1]

	//TODO: this check exists for testing purposes while new operator has not been released
	// if the operator image tag is 'develop', return empty string to default to placeholder repo manifest.
	// Remove when new operator image is released with version tag.
	if version == "develop" {
		return "", nil
	}
	lessThan, err := versionIsLessThan(version, "v2.2.0")
	if err != nil {
		return "", err
	}
	if lessThan {
		return "", fmt.Errorf("kubectl storageos does not support storageos operator version less than v2.2.0")
	}
	return version, nil
}

func OperatorUrlByVersion(version string) (string, error) {
	lessThan, err := versionIsLessThan(version, "v2.5.0")
	if err != nil {
		return "", err
	}
	if lessThan {
		return fmt.Sprintf("%s%s%s", oldOperatorYamlUrlPrefix, version, oldOperatorYamlUrlSuffix), nil
	}

	// TODO: return new operator yaml url once released
	return "", nil
}

func ClusterUrlByVersion(version string) (string, error) {
	lessThan, err := versionIsLessThan(version, "v2.5.0")
	if err != nil {
		return "", err
	}
	if lessThan {
		return fmt.Sprintf("%s%s%s", oldClusterYamlUrlPrefix, version, oldClusterYamlUrlSuffix), nil
	}

	// TODO: return new cluster yaml url once released
	return "", nil
}

func SecretUrlByVersion(version string) (string, error) {
	lessThan, err := versionIsLessThan(version, "v2.5.0")
	if err != nil {
		return "", err
	}
	if lessThan {
		return fmt.Sprintf("%s%s%s", oldSecretYamlUrlPrefix, version, oldSecretYamlUrlSuffix), nil
	}
	// new operator does not have separate secret and cluster yamls, therefore return empty string
	return "", nil
}

func versionIsLessThan(version, marker string) (bool, error) {
	ver, err := goversion.NewVersion(version)
	if err != nil {
		return false, err
	}
	mar, err := goversion.NewVersion(marker)
	if err != nil {
		return false, err
	}
	if ver.LessThan(mar) {
		return true, nil
	}
	return false, nil
}
