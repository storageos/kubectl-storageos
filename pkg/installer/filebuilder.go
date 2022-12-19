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
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

const errManifestNotFoundFromImage = `
   An error occurred attempting to fetch the manifest image "%s" with manifest "%s" from the docker daemon. 
   Possible causes:
      - The manifest image is not stored locally (please ensure you have specified the correct version).
      - The docker daemon is not running.
      - Some other error occurred while retrieving the manifest from the image.

   Alternatively, the manifest itself can be saved locally and passed to kubectl-storageos using the flag:
   
      --%s=/path/to/%s
`
const errNoURLForAirGap = `
   Air gapped operation - the source of the manifest "%s" passed to "--%s" cannot be a URL.

   It must be a local file or locally stored manifest image.
`

type installerOptions struct {
	storageosOperator    bool
	storageosCluster     bool
	portalClient         bool
	portalConfig         bool
	resourceQuota        bool
	etcdOperator         bool
	etcdCluster          bool
	localPathProvisioner bool
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
	// flagToFile is a map of flags to their corresponding files.
	// Only used for more helpful error messages
	flagToFile map[string]string
}

func newFileBuilder(yamlPath, yamlUrl, yamlImage, fileName, namespace string) *fileBuilder {
	return &fileBuilder{
		yamlPath:  yamlPath,
		yamlUrl:   yamlUrl,
		yamlImage: yamlImage,
		fileName:  fileName,
		namespace: namespace,
		flagToFile: map[string]string{
			stosOperatorFile:         StosOperatorYamlFlag,
			stosClusterFile:          StosClusterYamlFlag,
			stosPortalClientFile:     StosPortalClientSecretYamlFlag,
			stosPortalConfigFile:     StosPortalConfigYamlFlag,
			resourceQuotaFile:        ResourceQuotaYamlFlag,
			etcdOperatorFile:         EtcdOperatorYamlFlag,
			etcdClusterFile:          EtcdClusterYamlFlag,
			localPathProvisionerFile: LocalPathProvisionerYamlFlag,
		},
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
// - local-path-provisioner
//   - storageclass
//     - local-path-provisioner.yaml
//     - kustomization.yaml
func (o *installerOptions) buildInstallerFileSys(config *apiv1.KubectlStorageOSConfig, clientConfig *rest.Config) (filesys.FileSystem, error) {
	fs := filesys.MakeFsInMemory()
	fsData := make(fsData)
	stosSubDirs := make(map[string]map[string][]byte)
	var err error

	// build storageos/operator
	if o.storageosOperator {
		stosOpFiles, err := newFileBuilder(
			// path to storageos-operator.yaml as passed by --stos-operator-yaml can be a local path, a docker image or url.
			getStringWithDefault(config.Spec.Install.StorageOSOperatorYaml, config.Spec.Uninstall.StorageOSOperatorYaml),
			// url for latest storageos-operator.yaml
			pluginversion.OperatorLatestSupportedURL(),
			// latest docker image docker.io/storageos/operator-manifests
			pluginversion.OperatorLatestSupportedImageURL(),
			// filename storageos-operator.yaml
			stosOperatorFile,
			// storageos operator namespace
			config.Spec.GetOperatorNamespace()).createFileWithKustPair(config)
		if err != nil {
			return fs, err
		}
		stosSubDirs[operatorDir] = stosOpFiles
	}

	// build storageos/cluster
	if o.storageosCluster {
		stosClusterFiles, err := newFileBuilder(
			// path to storageos-cluster.yaml as passed by --stos-cluster-yaml can be a local path, a docker image or url.
			getStringWithDefault(config.Spec.Install.StorageOSClusterYaml, config.Spec.Uninstall.StorageOSClusterYaml),
			// url for latest storageos-cluster.yaml
			pluginversion.ClusterLatestSupportedURL(),
			// latest docker image docker.io/storageos/operator-manifests
			pluginversion.OperatorLatestSupportedImageURL(),
			// filename storageos-cluster.yaml
			stosClusterFile,
			// storageos operator namespace
			config.Spec.GetOperatorNamespace()).createFileWithKustPair(config)
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
		resourceQuotaFiles, err := newFileBuilder(
			// path to resource-quota.yaml as passed by --resource-quota-yaml can be a local path, a docker image or url.
			getStringWithDefault(config.Spec.Install.ResourceQuotaYaml, config.Spec.Uninstall.ResourceQuotaYaml),
			// url for latest resource-quota.yaml
			pluginversion.ResourceQuotaLatestSupportedURL(),
			// docker image does not exist for this file
			"",
			// filename resource-quota.yaml
			resourceQuotaFile,
			// storageos operator namespace
			config.Spec.GetOperatorNamespace()).createFileWithKustPair(config)
		if err != nil {
			return fs, err
		}
		stosSubDirs[resourceQuotaDir] = resourceQuotaFiles
	}

	// build storageos/portal-client this consists only of a kustomization file with a secret generator
	if o.portalClient {
		stosPortalClientFiles := make(map[string][]byte)
		stosPortalClientKust, err := newFileBuilder(
			// path to storageos-portal-client.yaml as passed by --stos-portal-client-secret-yaml can be a local path, a docker image or url.
			getStringWithDefault(config.Spec.Install.StorageOSPortalClientSecretYaml, config.Spec.Uninstall.StorageOSPortalClientSecretYaml),
			// url for latest storageos-portal-client.yaml
			pluginversion.PortalClientLatestSupportedURL(),
			// latest docker image docker.io/storageos/portal-manager-manifests
			pluginversion.PortalManagerLatestSupportedImageURL(),
			// filename storageos-portal-client.yaml
			stosPortalClientFile,
			// namespace not applicable
			"").readOrPullManifest(config)
		if err != nil {
			return fs, err
		}
		stosPortalClientFiles[kustomizationFile] = []byte(stosPortalClientKust)
		stosSubDirs[portalClientDir] = stosPortalClientFiles
	}

	if o.portalConfig {
		// build storageos/portal-config
		stosPortalConfigFiles, err := newFileBuilder(
			// path to storageos-portal-configmap.yaml as passed by --stos-portal-config-yaml can be a local path, a docker image or url.
			getStringWithDefault(config.Spec.Install.StorageOSPortalConfigYaml, config.Spec.Uninstall.StorageOSPortalConfigYaml),
			// url for latest storageos-portal-configmap.yaml
			pluginversion.PortalConfigLatestSupportedURL(),
			// latest docker image docker.io/storageos/portal-manager-manifests
			pluginversion.PortalManagerLatestSupportedImageURL(),
			// filename storageos-portal-configmap.yaml
			stosPortalConfigFile,
			// storageos operator namespace
			config.Spec.GetOperatorNamespace()).createFileWithKustPair(config)
		if err != nil {
			return fs, err
		}
		stosSubDirs[portalConfigDir] = stosPortalConfigFiles
	}
	fsData[stosDir] = stosSubDirs

	if o.localPathProvisioner {
		localPathProvisionerFiles, err := newFileBuilder(
			// path to local-path-provisioner-storage-class.yaml as passed by --local-path-provisioner-yaml can be a local path, a docker image or url.
			getStringWithDefault(config.Spec.Install.LocalPathProvisionerYaml, config.Spec.Uninstall.LocalPathProvisionerYaml),
			// url for latest local-path-provisioner-storage-class.yaml
			pluginversion.LocalPathProvisionerLatestSupportVersion(),
			// docker images does not exist for this file
			"",
			// filename local-path-provisioner-storage-class.yaml
			localPathProvisionerFile,
			// namespace not applicable
			"").createFileWithKustPair(config)
		if err != nil {
			return fs, err
		}

		localPathProvisionerSubDirs := make(map[string]map[string][]byte)
		localPathProvisionerSubDirs[storageclassDir] = localPathProvisionerFiles
		fsData[localPathProvisionerDir] = localPathProvisionerSubDirs
	}

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
		etcdOpFiles, err := newFileBuilder(
			// path to etcd-operator.yaml as passed by --etcd-operator-yaml can be a local path, a docker image or url.
			getStringWithDefault(config.Spec.Install.EtcdOperatorYaml, config.Spec.Uninstall.EtcdOperatorYaml),
			// no url for latest etcd-operator.yaml
			"",
			// latest docker image docker.io/storageos/etcd-cluster-operator-manifests
			pluginversion.EtcdOperatorLatestSupportedImageURL(),
			// filename etcd-operator.yaml
			etcdOperatorFile,
			// storageos operator namespace
			config.Spec.GetOperatorNamespace()).createFileWithKustPair(config)
		if err != nil {
			return fs, err
		}
		etcdSubDirs[operatorDir] = etcdOpFiles
	}

	if o.etcdCluster {
		// build etcd/cluster
		etcdClusterFiles, err := newFileBuilder(
			// path to etcd-cluster.yaml as passed by --etcd-cluster-yaml can be a local path, a docker image or url.
			getStringWithDefault(config.Spec.Install.EtcdClusterYaml, config.Spec.Uninstall.EtcdClusterYaml),
			// url for latest etcd-cluster.yaml
			pluginversion.EtcdClusterLatestSupportedURL(),
			// latest docker image docker.io/storageos/etcd-cluster-operator-manifests
			pluginversion.EtcdOperatorLatestSupportedImageURL(),
			// filename etcd-cluster.yaml
			etcdClusterFile,
			config.Spec.GetOperatorNamespace()).createFileWithKustPair(config)
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
func (fb *fileBuilder) createFileWithKustPair(config *apiv1.KubectlStorageOSConfig) (map[string][]byte, error) {
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
func (fb *fileBuilder) createFileWithData(config *apiv1.KubectlStorageOSConfig) (map[string][]byte, error) {
	file := make(map[string][]byte)
	yamlContents, err := fb.readOrPullManifest(config)
	if err != nil {
		return file, err
	}
	file[fb.fileName] = []byte(yamlContents)

	return file, nil
}

// readOrPullManifest returns a string of the manifest from path, url or image provided
func (fb *fileBuilder) readOrPullManifest(config *apiv1.KubectlStorageOSConfig) (string, error) {
	// At this point the 'yamlPath' could be a local has been passed by the user and
	// can be a local path, a docker image or a url. Attempt to retrieve the manifest
	// from whatever has been specified.
	location := fb.yamlPath
	if location != "" {
		if !isDockerRepo(location) && !util.IsURL(location) {
			// not docker repo or url, must be a local path
			return fb.getManifestFromPath(location)
		} else if util.IsURL(location) {
			return fb.getManifestFromURL(config.Spec.AirGap, location)
		} else if isDockerRepo(location) {
			return fb.getManifestFromImage(config.Spec.AirGap, location)
		}
	}

	// the user did not specify a location for the manifest, so
	// we will attempt to retrieve it from the manifest image.
	location = fb.yamlImage
	if location != "" && isDockerRepo(location) {
		contents, err := fb.getManifestFromImage(config.Spec.AirGap, location)
		if err == nil {
			return contents, nil
		} else if err != nil && config.Spec.AirGap {
			return "", err
		}
	}

	// could not get the manifest from the manifest image either,
	// last resort is to pull from the default url.
	location = fb.yamlUrl
	if location != "" && util.IsURL(location) {
		contents, err := fb.getManifestFromURL(config.Spec.AirGap, location)
		if err == nil {
			return contents, nil
		}
	}

	return "", errors.WithStack(fmt.Errorf("could not retrieve file %s from local path, remote url or docker repo", fb.fileName))
}

func (fb *fileBuilder) getManifestFromPath(location string) (string, error) {
	if _, err := os.Stat(location); err != nil {
		return "", errors.WithStack(err)
	}
	contents, err := ioutil.ReadFile(location)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(contents), nil
}

func (fb *fileBuilder) getManifestFromURL(airGap bool, location string) (string, error) {
	// getting the file from the default url equires an internet connection, so return at this point if
	// --air-gap flag has been set.
	if airGap {
		return "", errors.WithStack(fmt.Errorf(errNoURLForAirGap, fb.fileName, fb.flagToFile[fb.fileName]))
	}

	contents, err := pullManifest(location)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return contents, nil
}

func (fb *fileBuilder) getManifestFromImage(airGap bool, location string) (string, error) {
	// attempt to fetch image locally
	image, err := pluginutils.Image(location)
	if err == nil {
		contents, err := extractFileFromImage(image, fb.fileName)
		if err == nil {
			return contents, nil
		}
	}
	// pulling a remote image requires an internet connection, so return at this point if
	// --air-gap flag has been set.
	if airGap {
		return "", errors.WithStack(fmt.Errorf(errManifestNotFoundFromImage, location, fb.fileName, fb.flagToFile[fb.fileName], fb.fileName))
	}

	// attempt to pull image remotely
	image, err = pluginutils.PullImage(location)
	if err != nil {
		return "", err
	}

	return extractFileFromImage(image, fb.fileName)
}
