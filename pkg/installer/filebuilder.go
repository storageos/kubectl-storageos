package installer

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/troubleshoot/cmd/util"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kustomize/api/filesys"
)

type installerOptions struct {
	storageosOperator bool
	storageosCluster  bool
	portalClient      bool
	portalConfig      bool
	resourceQuota     bool
	etcdOperator      bool
	etcdCluster       bool
}

// fileBuilder is used to hold data required to build a file in the in-memory fs
type fileBuilder struct {
	// yamlPath is passed via plugin flag, it may be a local
	// file path, a URL or a docker repo URL.
	yamlPath string
	// yamlURL github release URL to yaml file
	yamlUrl string
	// yamlImage is a manifests image storing yaml file
	yamlImage string
	// fileName of yaml file
	fileName string
	// namespace of yaml file
	namespace string
}

func newFileBuilder(yamlPath, yamlUrl, yamlImage, fileName, namespace string) *fileBuilder {
	return &fileBuilder{
		yamlPath:  yamlPath,
		yamlUrl:   yamlUrl,
		yamlImage: yamlImage,
		fileName:  fileName,
		namespace: namespace,
	}
}

// buildInstallerFileSys builds an in-memory filesystem for installer with relevant storageos and
// etcd manifests based on installerOptions.
// - storageos
//   - operator
//     - storageos-operator.yaml
//     - kustomization.yaml
//   - cluster
//     - storageos-cluster.yaml
//     - kustomization.yaml
//   - portal-client
//     - kustomization.yaml
//   - portal-config
//     - portal-configmap.yaml
//     - kustomization.yaml
//   - resource-quota
//     - resource-quota.yaml
//     - kustomization.yaml
// - etcd
//   - operator
//     - etcd-operator.yaml
//     - kustomization.yaml
//   - cluster
//     - etcd-cluster.yaml
//     - kustomization.yaml
func (o *installerOptions) buildInstallerFileSys(config *apiv1.KubectlStorageOSConfig, clientConfig *rest.Config) (filesys.FileSystem, error) {
	fs := filesys.MakeFsInMemory()
	fsData := make(fsData)
	stosSubDirs := make(map[string]map[string][]byte)
	var err error

	// build storageos/operator
	if o.storageosOperator {
		stosOpFiles, err := newFileBuilder(getYamlPath(config.Spec.Install.StorageOSOperatorYaml, config.Spec.Uninstall.StorageOSOperatorYaml), pluginversion.OperatorLatestSupportedURL(), pluginversion.OperatorLatestSupportedImageURL(), stosOperatorFile, config.Spec.GetOperatorNamespace()).createFileWithKustPair(clientConfig)
		if err != nil {
			return fs, err
		}
		stosSubDirs[operatorDir] = stosOpFiles
	}

	// build storageos/cluster
	if o.storageosCluster {
		stosClusterFiles, err := newFileBuilder(getYamlPath(config.Spec.Install.StorageOSClusterYaml, config.Spec.Uninstall.StorageOSClusterYaml), pluginversion.ClusterLatestSupportedURL(), pluginversion.OperatorLatestSupportedImageURL(), stosClusterFile, config.Spec.GetOperatorNamespace()).createFileWithKustPair(clientConfig)
		if err != nil {
			return fs, err
		}
		stosSubDirs[clusterDir] = stosClusterFiles

		// append storageos secret yaml to cluster yaml if necessary. This will happen in the event of an
		// uninstall of storageos version < 2.5.0.
		if config.InstallerMeta.StorageOSSecretYaml != "" {
			stosSecretYaml, err := pullManifest(config.InstallerMeta.StorageOSSecretYaml)
			if err != nil {
				return fs, err
			}
			stosClusterMulti := makeMultiDoc(string(stosClusterFiles[stosClusterFile]), stosSecretYaml)
			stosClusterFiles[stosClusterFile] = []byte(stosClusterMulti)
		}
		stosSubDirs[clusterDir] = stosClusterFiles
	}

	// build resource quota
	if o.resourceQuota {
		resourceQuotaFiles, err := newFileBuilder(getYamlPath(config.Spec.Install.ResourceQuotaYaml, config.Spec.Uninstall.ResourceQuotaYaml), pluginversion.ResourceQuotaLatestSupportedURL(), "", resourceQuotaFile, config.Spec.GetOperatorNamespace()).createFileWithKustPair(clientConfig)
		if err != nil {
			return fs, err
		}
		stosSubDirs[resourceQuotaDir] = resourceQuotaFiles
	}

	// build storageos/portal-client this consists only of a kustomization file with a secret generator
	if o.portalClient {
		stosPortalClientFiles := make(map[string][]byte)

		stosPortalClientKust, err := newFileBuilder(getYamlPath(config.Spec.Install.StorageOSPortalClientSecretYaml, config.Spec.Uninstall.StorageOSPortalClientSecretYaml), pluginversion.PortalClientLatestSupportedURL(), pluginversion.PortalManagerLatestSupportedImageURL(), stosPortalClientFile, "").readOrPullManifest(clientConfig)
		if err != nil {
			return fs, err
		}
		stosPortalClientFiles[kustomizationFile] = []byte(stosPortalClientKust)
		stosSubDirs[portalClientDir] = stosPortalClientFiles
	}

	if o.portalConfig {
		// build storageos/portal-config
		stosPortalConfigFiles, err := newFileBuilder(getYamlPath(config.Spec.Install.StorageOSPortalConfigYaml, config.Spec.Uninstall.StorageOSPortalConfigYaml), pluginversion.PortalConfigLatestSupportedURL(), pluginversion.PortalManagerLatestSupportedImageURL(), stosPortalConfigFile, config.Spec.GetOperatorNamespace()).createFileWithKustPair(clientConfig)
		if err != nil {
			return fs, err
		}
		stosSubDirs[portalConfigDir] = stosPortalConfigFiles
	}
	fsData[stosDir] = stosSubDirs

	// if include-etcd flag is not set, create fs with storageos files and return early
	if !config.Spec.IncludeEtcd {
		fs, err = createDirAndFiles(fs, fsData)
		if err != nil {
			return fs, err
		}
		return fs, nil
	}

	etcdSubDirs := make(map[string]map[string][]byte)

	// build etcd/operator
	if o.etcdOperator {
		etcdOpFiles, err := newFileBuilder(getYamlPath(config.Spec.Install.EtcdOperatorYaml, config.Spec.Uninstall.EtcdOperatorYaml), "", pluginversion.EtcdOperatorLatestSupportedImageURL(), etcdOperatorFile, config.Spec.GetOperatorNamespace()).createFileWithKustPair(clientConfig)
		if err != nil {
			return fs, err
		}
		etcdSubDirs[operatorDir] = etcdOpFiles
	}

	if o.etcdCluster {
		// build etcd/cluster
		etcdClusterFiles, err := newFileBuilder(getYamlPath(config.Spec.Install.EtcdClusterYaml, config.Spec.Uninstall.EtcdClusterYaml), pluginversion.EtcdClusterLatestSupportedURL(), pluginversion.EtcdOperatorLatestSupportedImageURL(), etcdClusterFile, config.Spec.GetOperatorNamespace()).createFileWithKustPair(clientConfig)
		if err != nil {
			return fs, err
		}
		etcdSubDirs[clusterDir] = etcdClusterFiles
	}

	fsData[etcdDir] = etcdSubDirs
	fs, err = createDirAndFiles(fs, fsData)
	if err != nil {
		return fs, err
	}

	return fs, nil
}

// createFileWithKustPair creates a map of two files (file name to file data).
//
// The first file is that which has its address stored in fileBuilder as a
// local path, github release URL or manifests image repo
//
// The second file is the kustomization.yaml created from scratch.
// It's contents, to begin with are simply:
//
// resources:
// - <filename>
//
func (fb *fileBuilder) createFileWithKustPair(config *rest.Config) (map[string][]byte, error) {
	files, err := fb.createFileWithData(config)
	if err != nil {
		return files, err
	}

	kustYamlContents, err := pluginutils.SetFieldInManifest(kustTemp, fmt.Sprintf("%s%s%s", "[", fb.fileName, "]"), "resources", "")
	if err != nil {
		return files, err
	}

	files[kustomizationFile] = []byte(kustYamlContents)

	return files, nil
}

// createFileWithData returns a map with a single entry of [filename][filecontent]
func (fb *fileBuilder) createFileWithData(config *rest.Config) (map[string][]byte, error) {
	file := make(map[string][]byte)
	yamlContents, err := fb.readOrPullManifest(config)
	if err != nil {
		return file, err
	}
	file[fb.fileName] = []byte(yamlContents)

	return file, nil
}

// readOrPullManifest returns a string of the manifest from path, url or image provided
func (fb *fileBuilder) readOrPullManifest(config *rest.Config) (string, error) {
	location := fb.yamlPath
	if location == "" {
		location = fb.yamlImage
	}
	if location == "" {
		location = fb.yamlUrl
	}
	if location == "" {
		return "", errors.WithStack(errors.New("manifest location not set"))
	}

	if isDockerRepo(location) {
		contents, err := fetchImageAndExtractFileFromTarball(location, fb.fileName)
		if err == nil {
			return contents, nil
		}
		// could not get file from image (may not exist on provided version)
		// default to release URL
		location = fb.yamlUrl
	}

	if util.IsURL(location) {
		contents, err := pullManifest(location)
		if err != nil {
			return "", errors.WithStack(err)
		}
		return contents, nil
	}

	if _, err := os.Stat(location); err != nil {
		return "", errors.WithStack(err)
	}
	contents, err := ioutil.ReadFile(location)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(contents), nil
}

// getYamlPath returns whichever yamlPath is set (no more than one will ever be set)
func getYamlPath(installYamlPath, uninstallYamlPath string) string {
	if installYamlPath != "" {
		return installYamlPath
	}
	return uninstallYamlPath
}
