package installer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	otkkubectl "github.com/darkowlzz/operator-toolkit/declarative/kubectl"
	"github.com/replicatedhq/troubleshoot/cmd/util"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/kustomize/api/filesys"
)

const (
	// CLI flags
	StosOperatorYamlFlag = "stos-operator-yaml"
	StosClusterYamlFlag  = "stos-cluster-yaml"
	StosSecretYamlFlag   = "stos-secret-yaml"
	EtcdOperatorYamlFlag = "etcd-operator-yaml"
	EtcdClusterYamlFlag  = "etcd-cluster-yaml"
	SkipEtcdFlag         = "skip-etcd"
	EtcdEndpointsFlag    = "etcd-endpoints"
	ConfigPathFlag       = "config-path"
	EtcdNamespaceFlag    = "etcd-namespace"
	StosOperatorNSFlag   = "stos-operator-namespace"
	StosClusterNSFlag    = "stos-cluster-namespace"
	StorageClassFlag     = "storage-class"
	SecretUserFlag       = "secret-username"
	SecretPassFlag       = "secret-password"

	// config file fields - contain path delimiters for plugin interpretation of config manifest
	InstallEtcdNamespaceConfig    = "spec.install.etcdNamespace"
	InstallStosOperatorNSConfig   = "spec.install.storageOSOperatorNamespace"
	InstallStosClusterNSConfig    = "spec.install.storageOSClusterNamespace"
	StosOperatorYamlConfig        = "spec.install.storageOSOperatorYaml"
	StosClusterYamlConfig         = "spec.install.storageOSClusterYaml"
	EtcdOperatorYamlConfig        = "spec.install.etcdOperatorYaml"
	EtcdClusterYamlConfig         = "spec.install.etcdClusterYaml"
	InstallSkipEtcdConfig         = "spec.install.skipEtcd"
	EtcdEndpointsConfig           = "spec.install.etcdEndpoints"
	StorageClassConfig            = "spec.install.storageClassName"
	SecretUserConfig              = "spec.install.secretUsername"
	SecretPassConfig              = "spec.install.secretPassword"
	UninstallEtcdNamespaceConfig  = "spec.uninstall.etcdNamespace"
	UninstallStosOperatorNSConfig = "spec.uninstall.storageOSOperatorNamespace"
	UninstallStosClusterNSConfig  = "spec.uninstall.storageOSClusterNamespace"
	UninstallSkipEtcdConfig       = "spec.uninstall.skipEtcd"

	// dir and file names for in memory fs
	etcdDir           = "etcd"
	stosDir           = "storageos"
	operatorDir       = "operator"
	clusterDir        = "cluster"
	stosOperatorFile  = "storageos-operator.yaml"
	stosClusterFile   = "storageos-cluster.yaml"
	etcdOperatorFile  = "etcd-operator.yaml"
	etcdClusterFile   = "etcd-cluster.yaml"
	kustomizationFile = "kustomization.yaml"

	// kustomization template
	kustTemp = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:`

	stosClusterKind        = "StorageOSCluster"
	etcdClusterKind        = "EtcdCluster"
	defaultEtcdClusterName = "storageos-etcd"
	defaultEtcdClusterNS   = "storageos-etcd"
	stosFinalizer          = "storageos.com/finalizer"
)

// fsData represents dir name, subdir name, file name and file data.
// It is used to create the Installer's in-memory file system.
type fsData map[string]map[string]map[string][]byte

// Installer holds the kubectl client and in-memory fs data used throughout the installation process
type Installer struct {
	kubectlClient *otkkubectl.DefaultKubectl
	clientConfig  *rest.Config
	fileSys       filesys.FileSystem
}

// NewInstaller returns an Installer object with the kubectl client and in-memory filesystem
func NewInstaller(config *apiv1.KubectlStorageOSConfig) (*Installer, error) {
	installer := &Installer{}
	kubectlClient := otkkubectl.New()
	fileSys, err := buildInstallerFileSys(config)
	if err != nil {
		return installer, err
	}
	clientConfig, err := pluginutils.NewClientConfig()
	if err != nil {
		return installer, err
	}

	installer = &Installer{
		kubectlClient: kubectlClient,
		clientConfig:  clientConfig,
		fileSys:       fileSys,
	}

	return installer, nil
}

// buildInstallerFileSys builds an in-memory filesystem for installer with relevant storageos and
// etcd manifests. If '--skip-etcd-install' flag is set, etcd dir is not created.
// - storageos
//   - operator
//     - storageos-operator.yaml
//     - kustomization.yaml
//   - cluster
//     - storageos-cluster.yaml
//     - kustomization.yaml
// - etcd
//   - operator
//     - etcd-operator.yaml
//     - kustomization.yaml
//   - cluster
//     - etcd-cluster.yaml
//     - kustomization.yaml
func buildInstallerFileSys(config *apiv1.KubectlStorageOSConfig) (filesys.FileSystem, error) {
	fs := filesys.MakeFsInMemory()
	fsData := make(fsData)
	stosSubDirs := make(map[string]map[string][]byte)

	// build storageos/operator
	stosOpFiles, err := createFileData(config.Spec.Install.StorageOSOperatorYaml, pluginversion.OperatorLatestSupportedURL(), stosOperatorFile)
	if err != nil {
		return fs, err
	}
	stosSubDirs[operatorDir] = stosOpFiles

	// build storageos/cluster
	stosClusterFiles, err := createFileData(config.Spec.Install.StorageOSClusterYaml, pluginversion.ClusterLatestSupportedURL(), stosClusterFile)
	if err != nil {
		return fs, err
	}

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
	fsData[stosDir] = stosSubDirs

	// if skip-etcd-install flag is set, create fs with storageos files and return early
	if config.Spec.Install.SkipEtcd {
		fs, err = createDirAndFiles(fs, fsData)
		if err != nil {
			return fs, err
		}
		return fs, nil
	}

	etcdSubDirs := make(map[string]map[string][]byte)

	// build etcd/operator
	etcdOpFiles, err := createFileData(config.Spec.Install.EtcdOperatorYaml, pluginversion.EtcdOperatorLatestSupportedURL(), etcdOperatorFile)
	if err != nil {
		return fs, err
	}
	etcdSubDirs[operatorDir] = etcdOpFiles

	// build etcd/cluster
	etcdClusterFiles, err := createFileData(config.Spec.Install.EtcdClusterYaml, pluginversion.EtcdClusterLatestSupportedURL(), etcdClusterFile)
	if err != nil {
		return fs, err
	}
	etcdSubDirs[clusterDir] = etcdClusterFiles

	fsData[etcdDir] = etcdSubDirs
	fs, err = createDirAndFiles(fs, fsData)
	if err != nil {
		return fs, err
	}

	return fs, nil
}

func makeMultiDoc(manifests ...string) string {
	manifestsSlice := make([]string, 0)
	for _, manifest := range manifests {
		manifestsSlice = append(manifestsSlice, manifest)
	}

	return strings.Join(manifestsSlice, "\n---\n")
}

// createFileData creates a map of two files (file name to file data).
//
// The first file is that passed to the function and its contents are either pulled or read
// (depending on flag).
//
// The second file is the kustomization.yaml created from scratch.
// It's contents, to begin with are simply:
//
// resources:
// - <filename>
//
func createFileData(yamlPath, yamlUrl, fileName string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	yamlContents, err := readOrPullManifest(yamlPath, yamlUrl)
	if err != nil {
		return files, err
	}
	files[fileName] = []byte(yamlContents)
	kustYamlContents, err := pluginutils.SetFieldInManifest(kustTemp, fmt.Sprintf("%s%s%s", "[", fileName, "]"), "resources", "")
	if err != nil {
		return files, err
	}

	files[kustomizationFile] = []byte(kustYamlContents)
	return files, nil
}

// createDirAndFiles is a helper function for buildInstallerFileSys, creating the in-memory
// file system for installer from the fsData provided.
func createDirAndFiles(fs filesys.FileSystem, fsData fsData) (filesys.FileSystem, error) {
	for dir, subDirs := range fsData {
		err := fs.Mkdir(dir)
		if err != nil {
			return fs, err
		}

		for subDir, files := range subDirs {
			err := fs.Mkdir(filepath.Join(dir, subDir))
			if err != nil {
				return fs, err
			}
			for name, data := range files {
				_, err := fs.Create(filepath.Join(dir, subDir, name))
				if err != nil {
					return fs, err
				}
				err = fs.WriteFile(filepath.Join(dir, subDir, name), data)
				if err != nil {
					return fs, err
				}
			}
		}
	}
	return fs, nil
}

// readOrPullManifest returns a string of the manifest from path or url provided
func readOrPullManifest(path, url string) (string, error) {
	var contents string
	var err error
	if path == "" {
		contents, err = pullManifest(url)
		if err != nil {
			return contents, err
		}
		return contents, nil
	} else if util.IsURL(path) {
		contents, err = pullManifest(path)
		if err != nil {
			return contents, err
		}
		return contents, nil
	}
	contents, err = readManifest(path)
	if err != nil {
		return contents, err
	}
	return contents, nil
}

// readManifest returns string of contents at path
func readManifest(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("%s was not found", path)
	}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

// pullManifest returns a string of contents at url
func pullManifest(url string) (string, error) {
	if !util.IsURL(url) {
		return "", fmt.Errorf("%s is not a URL and was not found", url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

// addPatchesToFSKustomize uses AddPatchesToKustomize internally to add a list of patches to a kustomization file
// at path of in-memory fs.
func (in *Installer) addPatchesToFSKustomize(path, targetKind, targetName string, patches []pluginutils.KustomizePatch) error {
	kustFile, err := in.fileSys.ReadFile(path)
	if err != nil {
		return err
	}

	kustFileWithPatches, err := pluginutils.AddPatchesToKustomize(string(kustFile), targetKind, targetName, patches)
	if err != nil {
		return err
	}

	err = in.fileSys.WriteFile(path, []byte(kustFileWithPatches))
	if err != nil {
		return err
	}

	return nil
}

// setFieldInFsManifest reads the file at path of the in-memory filesystem, uses
// SetFieldInManiest internally to perform the update and then writes the returned file to path.
func (in *Installer) setFieldInFsManifest(path, value, valueName string, fields ...string) error {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return err
	}
	dataStr, err := pluginutils.SetFieldInManifest(string(data), value, valueName, fields...)
	if err != nil {
		return err
	}
	err = in.fileSys.WriteFile(path, []byte(dataStr))
	if err != nil {
		return err
	}
	return nil
}

// getFieldInFsManifest reads the file at path of the in-memory filesystem, uses
// GetFieldInManiest internally to retrieve the specified field.
func (in *Installer) getFieldInFsManifest(path string, fields ...string) (string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return "", err
	}
	field, err := pluginutils.GetFieldInManifest(string(data), fields...)
	if err != nil {
		return "", err
	}
	return field, nil
}

// getFieldInFsMultiDocByKind reads the file at path of the in-memory filesystem, uses
// GetFieldInMultiiDocByKind internally to retrieve the specified field.
func (in *Installer) getFieldInFsMultiDocByKind(path, kind string, fields ...string) (string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return "", err
	}
	field, err := pluginutils.GetFieldInMultiDocByKind(string(data), kind, fields...)
	if err != nil {
		return "", err
	}
	return field, nil
}

// getManifestFromFsMultiDoc reads the file at path of the in-memory filesystem, uses
// GetManifestFromMultiDoc internally to retrieve the individual manifest by kind.
func (in *Installer) getManifestFromFsMultiDoc(path, kind string) (string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return "", err
	}
	singleManifest, err := pluginutils.GetManifestFromMultiDoc(string(data), kind)
	if err != nil {
		return "", err
	}
	return singleManifest, nil
}

// getAllManifestsOfKindFromFsMultiDoc reads the file at path of the in-memory filesystem, uses
// GetManifestFromMultiDoc internally to retrieve all manifests ok 'kind'.
func (in *Installer) getAllManifestsOfKindFromFsMultiDoc(path, kind string) ([]string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return nil, err
	}
	manifests, err := pluginutils.GetAllManifestsOfKindFromMultiDoc(string(data), kind)
	if err != nil {
		return nil, err
	}
	return manifests, nil
}

// omitAndReturnKindFromFSMultiDoc reads the file at path of the in-memory filesystem, uses
// OmitAndReturnKindFromMultiDoc internally to perform the update and then writes the returned file to path,
// also returninng a []string of the objects omitted.
func (in *Installer) omitAndReturnKindFromFSMultiDoc(path, kind string) ([]string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dataStr, objsOfKind, err := pluginutils.OmitAndReturnKindFromMultiDoc(string(data), kind)
	if err != nil {
		return nil, err
	}
	err = in.fileSys.WriteFile(path, []byte(dataStr))
	if err != nil {
		return nil, err
	}
	return objsOfKind, nil
}
