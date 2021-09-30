package version

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/storageos/kubectl-storageos/pkg/consts"
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

	// URLs to installation manifests
	stosOperatorManifestsImageUrl = "docker.io/storageos/operator-manifests"

	newClusterYamlUrlPrefix = "https://github.com/storageos/kubectl-storageos/releases/download/"
	newClusterYamlUrlSuffix = "/storageos-cluster.yaml"

	etcdOperatorYamlUrl = "https://github.com/storageos/etcd-cluster-operator/releases/download/v0.3.1/storageos-etcd-cluster-operator.yaml"
	etcdClusterYamlUrl  = "https://github.com/storageos/etcd-cluster-operator/releases/download/v0.3.1/storageos-etcd-cluster.yaml"
)

var (
	// EnableUnofficialRelease allows the installer to install not official of operator.
	// This could be change with build flag:
	// -X github.com/storageos/kubectl-storageos/pkg/version.EnableUnofficialRelease=true
	EnableUnofficialRelease string
	enableUnofficialRelease bool

	versionRegexp *regexp.Regexp
)

func init() {
	var err error

	if EnableUnofficialRelease != "" {
		enableUnofficialRelease, err = strconv.ParseBool(EnableUnofficialRelease)
		if err != nil {
			panic(err)
		}
	}

	versionRegexp, err = regexp.Compile("v([0-9]+.[0-9]+.[0-9]+)")
	if err != nil {
		panic(err)
	}
}

func GetExistingOperatorVersion(namespace string) (string, error) {
	oldNS := consts.OldOperatorNamespace
	newNS := consts.NewOperatorNamespace
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
		return "", errors.Wrap(err, consts.ErrUnableToContructClientFromConfig)
	}

	stosDeployment, errOld := clientset.AppsV1().Deployments(oldNS).Get(context.TODO(), consts.OldOperatorName, metav1.GetOptions{})
	if errOld != nil {
		var errNew error
		stosDeployment, errNew = clientset.AppsV1().Deployments(newNS).Get(context.TODO(), consts.NewOperatorName, metav1.GetOptions{})
		if errNew != nil {
			errNew = errors.Wrap(errNew, errOld.Error())
			return "", errors.Wrap(errNew, "unable to detect StorageOS version")
		}
	}
	imageName := stosDeployment.Spec.Template.Spec.Containers[0].Image
	splitImageName := strings.SplitAfter(imageName, ":")
	version := splitImageName[len(splitImageName)-1]

	lessThan, err := VersionIsLessThan(version, consts.OperatorOldestSupportedVersion)
	if err != nil {
		return "", err
	}
	if lessThan {
		return "", fmt.Errorf("kubectl storageos does not support storageos operator version less than %s", consts.OperatorOldestSupportedVersion)
	}

	return version, nil
}

func cleanupVersion(version string) string {
	if version == "develop" {
		return OperatorLatestSupportedVersion()
	}
	return versionRegexp.FindString(version)
}

func OperatorUrlByVersion(version string) (string, error) {
	lessThanOrEqual, err := VersionIsLessThanOrEqual(version, ClusterOperatorLastVersion())
	if err != nil {
		return "", err
	}
	if lessThanOrEqual {
		return fmt.Sprintf("%s%s%s", oldOperatorYamlUrlPrefix, version, oldOperatorYamlUrlSuffix), nil
	}

	return fmt.Sprintf("%s:%s", stosOperatorManifestsImageUrl, version), nil
}

func ClusterUrlByVersion(version string) (string, error) {
	lessThanOrEqual, err := VersionIsLessThanOrEqual(version, ClusterOperatorLastVersion())
	if err != nil {
		return "", err
	}
	if lessThanOrEqual {
		return fmt.Sprintf("%s%s%s", oldClusterYamlUrlPrefix, version, oldClusterYamlUrlSuffix), nil
	}

	return fmt.Sprintf("%s%s%s", newClusterYamlUrlPrefix, version, newClusterYamlUrlSuffix), nil
}

func SecretUrlByVersion(version string) (string, error) {
	lessThanOrEqual, err := VersionIsLessThanOrEqual(version, ClusterOperatorLastVersion())
	if err != nil {
		return "", err
	}
	if lessThanOrEqual {
		return fmt.Sprintf("%s%s%s", oldSecretYamlUrlPrefix, version, oldSecretYamlUrlSuffix), nil
	}
	// new operator does not have separate secret and cluster yamls, therefore return empty string

	return "", nil
}

func VersionIsLessThanOrEqual(version, marker string) (bool, error) {
	version = cleanupVersion(version)
	marker = cleanupVersion(marker)

	ver, err := goversion.NewVersion(version)
	if err != nil {
		return false, errors.WithStack(err)
	}
	mar, err := goversion.NewVersion(marker)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return ver.LessThanOrEqual(mar), nil
}

func VersionIsLessThan(version, marker string) (bool, error) {
	version = cleanupVersion(version)
	marker = cleanupVersion(marker)

	ver, err := goversion.NewVersion(version)
	if err != nil {
		return false, errors.WithStack(err)
	}
	mar, err := goversion.NewVersion(marker)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return ver.LessThan(mar), nil
}

func VersionIsEqualTo(version, marker string) (bool, error) {
	version = cleanupVersion(version)
	marker = cleanupVersion(marker)

	ver, err := goversion.NewVersion(version)
	if err != nil {
		return false, errors.WithStack(err)
	}
	mar, err := goversion.NewVersion(marker)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return ver.Equal(mar), nil
}

func OperatorLatestSupportedURL() string {
	return fmt.Sprintf("%s:%s", stosOperatorManifestsImageUrl, OperatorLatestSupportedVersion())
}

func ClusterLatestSupportedURL() string {
	return fmt.Sprintf("%s%s%s", newClusterYamlUrlPrefix, OperatorLatestSupportedVersion(), newClusterYamlUrlSuffix)
}

func EtcdOperatorLatestSupportedURL() string {
	// TODO add etcd-operator-version flag to return correct url
	return etcdOperatorYamlUrl
}

func EtcdClusterLatestSupportedURL() string {
	// TODO add etcd-operator-version flag to return correct url
	return etcdClusterYamlUrl
}
