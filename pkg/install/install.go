package install

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	otkkubectl "github.com/darkowlzz/operator-toolkit/declarative/kubectl"
	"github.com/replicatedhq/troubleshoot/cmd/util"
	"github.com/spf13/viper"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
)

const (
	// URLs to installation manifests
	// TODO: move to storageos/kubectl-storageos.
	stosOperatorYamlUrl = "https://raw.githubusercontent.com/nolancon/placeholder/main/config/storageos/operator/storageos-operator.yaml"
	stosClusterYamlUrl  = "https://raw.githubusercontent.com/nolancon/placeholder/main/config/storageos/cluster/storageos-cluster.yaml"
	etcdOperatorYamlUrl = "https://github.com/nolancon/etcd-cluster-operator/releases/download/v0.2.1/etcd-cluster-operator.yaml"
	etcdClusterYamlUrl  = "https://raw.githubusercontent.com/nolancon/placeholder/main/config/etcd/cluster/etcd-cluster.yaml"

	// CLI flags
	StosOperatorYamlFlag = "stos-operator-yaml"
	StosClusterYamlFlag  = "stos-cluster-yaml"
	EtcdOperatorYamlFlag = "etcd-operator-yaml"
	EtcdClusterYamlFlag  = "etcd-cluster-yaml"
	SkipEtcdInstallFlag  = "skip-etcd-install"
	EtcdEndpointFlag     = "etcd-endpoints"
	ConfigPathFlag       = "config-path"
	EtcdNamespaceFlag    = "etcd-namespace"
	StosOperatorNSFlag   = "stos-operator-namespace"
	StosClusterNSFlag    = "stos-cluster-namespace"

	// config file fields
	StosOperatorYamlConfig = "storageOSOperatorYaml"
	StosClusterYamlConfig  = "storageOSClusterYaml"
	EtcdOperatorYamlConfig = "etcdOperatorYaml"
	EtcdClusterYamlConfig  = "etcdClusterYaml"
	SkipEtcdInstallConfig  = "skipEtcdInstall"
	EtcdEndpointConfig     = "etcdEndpoints"
	EtcdNamespaceConfig    = "etcdNamespace"
	StosOperatorNSConfig   = "storageOSOperatorNamespace"
	StosClusterNSConfig    = "storageOSClusterNamespace"

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
	defaultStosOperatorNS  = "storageos-system"
	defaultStosClusterNS   = "storageos"
	defaultStosClusterName = "storageoscluster-sample"
)

// fsData represents dir name, subdir name, file name and file data.
// It is used to create the Installer's in-memory file system.
type fsData map[string]map[string]map[string][]byte

type Installer struct {
	kubectlClient *otkkubectl.DefaultKubectl
	fileSys       filesys.FileSystem
}

func NewInstaller() (*Installer, error) {
	installer := &Installer{}
	kubectlClient := otkkubectl.New()
	fileSys, err := buildInstallerFileSys()
	if err != nil {
		return installer, err
	}

	installer = &Installer{
		kubectlClient: kubectlClient,
		fileSys:       fileSys,
	}

	return installer, nil
}

func (in *Installer) Install() error {
	v := viper.GetViper()
	if v.GetBool(SkipEtcdInstallFlag) {
		err := in.handleEndpointsInput(v.GetString(EtcdEndpointFlag))
		if err != nil {
			return err
		}
	} else {
		// add changes to etcd kustomizations here before kustomizeAndApply calls ie make changs
		// to etcd/operator/kustomization.yaml and/or etcd/cluster/kustomization.yaml
		// based on flags (or cli config file)
		err := in.setFieldInFsManifestIfValueExists(filepath.Join(etcdDir, operatorDir, kustomizationFile), v.GetString(EtcdNamespaceFlag), "namespace", "")
		if err != nil {
			return err
		}
		err = in.setFieldInFsManifestIfValueExists(filepath.Join(etcdDir, clusterDir, kustomizationFile), v.GetString(EtcdNamespaceFlag), "namespace", "")
		if err != nil {
			return err
		}

		err = in.kustomizeAndApply(filepath.Join(etcdDir, operatorDir))
		if err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
		err = in.kustomizeAndApply(filepath.Join(etcdDir, clusterDir))
		if err != nil {
			return err
		}
		// update endpoint for stos cluster based on input flag (just namespace can be set for now)
		err = in.updateEndpoint("", v.GetString(EtcdNamespaceFlag))
		if err != nil {
			return err
		}
	}
	// add changes to storageos kustomizations here before kustomizeAndApply calls ie make changs
	// to storageos/operator/kustomization.yaml and/or storageos/cluster/kustomization.yaml
	// based on flags (or cli config file)

	// set namespace for operator
	err := in.setFieldInFsManifestIfValueExists(filepath.Join(stosDir, operatorDir, kustomizationFile), v.GetString(StosOperatorNSFlag), "namespace", "")
	if err != nil {
		return err
	}

	// handle case where cluster ns has not been provided, creating default storageos ns for cluster
	var stosClusterNS string
	if v.GetString(StosClusterNSFlag) != "" {
		stosClusterNS = v.GetString(StosClusterNSFlag)

	} else {
		stosClusterNS = defaultStosClusterNS
	}
	err = in.kubectlClient.Apply(context.TODO(), "", NamespaceYaml(stosClusterNS), true)
	if err != nil {
		return err
	}

	// set namespace for storageos cluster and secret
	err = in.setFieldInFsManifestIfValueExists(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterNS, "namespace", "")
	if err != nil {
		return err
	}

	err = in.kustomizeAndApply(filepath.Join(stosDir, operatorDir))
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	err = in.kustomizeAndApply(filepath.Join(stosDir, clusterDir))
	if err != nil {
		return err
	}

	return nil
}

func (in *Installer) updateEndpoint(name, namespace string) error {
	var err error
	if name == "" {
		name, err = in.getFieldInFsManifest(path.Join(etcdDir, clusterDir, etcdClusterFile), "metadata", "name")
		if err != nil {
			return err
		}
	}
	if namespace == "" {
		namespace, err = in.getFieldInFsManifest(path.Join(etcdDir, clusterDir, etcdClusterFile), "metadata", "namespace")
		if err != nil {
			return err
		}
	}

	endpoint := fmt.Sprintf("%v%v%v%v%v", name, ".", namespace, ":", "2379")

	endpointPatch := KustomizePatch{
		Op:    "replace",
		Path:  "/spec/kvBackend/address",
		Value: endpoint,
	}

	err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, defaultStosClusterName, []KustomizePatch{endpointPatch})
	if err != nil {
		return err
	}

	return nil
}

func (in *Installer) addPatchesToFSKustomize(path, targetKind, targetName string, patches []KustomizePatch) error {
	kustFile, err := in.fileSys.ReadFile(path)
	if err != nil {
		return err
	}

	kustFileWithPatches, err := AddPatchesToKustomize(string(kustFile), targetKind, targetName, patches)
	if err != nil {
		return err
	}

	err = in.fileSys.WriteFile(filepath.Join(stosDir, clusterDir, kustomizationFile), []byte(kustFileWithPatches))
	if err != nil {
		return err
	}

	return nil
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
func buildInstallerFileSys() (filesys.FileSystem, error) {
	v := viper.GetViper()
	fs := filesys.MakeFsInMemory()
	fsData := make(fsData)
	stosSubDirs := make(map[string]map[string][]byte)

	// build storageos/operator
	stosOpFiles, err := createFileData(StosOperatorYamlFlag, stosOperatorYamlUrl, stosOperatorFile)
	if err != nil {
		return fs, err
	}
	stosSubDirs[operatorDir] = stosOpFiles

	// build storageos/cluster
	stosClusterFiles, err := createFileData(StosClusterYamlFlag, stosClusterYamlUrl, stosClusterFile)
	if err != nil {
		return fs, err
	}
	stosSubDirs[clusterDir] = stosClusterFiles

	fsData[stosDir] = stosSubDirs

	// if skip-etcd-install flag is set, create fs with storageos files and return early
	if v.GetBool(SkipEtcdInstallFlag) {
		fs, err = createDirAndFiles(fs, fsData)
		if err != nil {
			return fs, err
		}
		return fs, nil
	}

	etcdSubDirs := make(map[string]map[string][]byte)

	// build etcd/operator
	etcdOpFiles, err := createFileData(EtcdOperatorYamlFlag, etcdOperatorYamlUrl, etcdOperatorFile)
	if err != nil {
		return fs, err
	}
	etcdSubDirs[operatorDir] = etcdOpFiles

	// build etcd/cluster
	etcdClusterFiles, err := createFileData(EtcdClusterYamlFlag, etcdClusterYamlUrl, etcdClusterFile)
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
func createFileData(yamlFlag, yamlUrl, fileName string) (map[string][]byte, error) {
	v := viper.GetViper()
	files := make(map[string][]byte)
	yamlContents, err := readOrPullManifest(v.GetString(yamlFlag), yamlUrl)
	if err != nil {
		return files, err
	}
	files[fileName] = []byte(yamlContents)
	kustYamlContents, err := SetFieldInManifest(kustTemp, fmt.Sprintf("%s%s%s", "[", fileName, "]"), "resources", "")
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

// kustomize and apply performs kustomize run on the provided dir and kubect apply on the files in dir.
// It is the equivalent of:
// `kustomize build <dir> | kubectl apply -f -
func (in *Installer) kustomizeAndApply(dir string) error {
	kustomizer := krusty.MakeKustomizer(in.fileSys, krusty.MakeDefaultOptions())
	resMap, err := kustomizer.Run(dir)
	if err != nil {
		return err
	}
	resYaml, err := resMap.AsYaml()
	if err != nil {
		return err
	}
	err = in.kubectlClient.Apply(context.TODO(), "", string(resYaml), true)
	if err != nil {
		return err
	}

	return nil
}

// readOrPullManifest attempts readManifest first. If path is empty, pullManifest
// is called for url
func readOrPullManifest(path, url string) (string, error) {
	var contents string
	var err error
	if path == "" {
		contents, err = pullManifest(url)
		if err != nil {
			return contents, err
		}
		return contents, nil
	} else {
		contents, err = readManifest(path)
		if err != nil {
			return contents, err
		}
		return contents, nil
	}
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

// setFieldInFsManifest reads the the file at path of the in-memory filesystem, uses
// SetFieldInManiest internally to perform the update and then writes the returned file to path.
func (in *Installer) setFieldInFsManifest(path, value, valueName string, fields ...string) error {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return err
	}
	dataStr, err := SetFieldInManifest(string(data), value, valueName, fields...)
	if err != nil {
		return err
	}
	err = in.fileSys.WriteFile(path, []byte(dataStr))
	if err != nil {
		return err
	}
	return nil
}

// setFieldInFsManifestIfValueExists is the same as setFieldInFsManifest, except if value is ""
// the function returns early without setting any field.
func (in *Installer) setFieldInFsManifestIfValueExists(path, value, valueName string, fields ...string) error {
	if value == "" {
		return nil
	}
	err := in.setFieldInFsManifest(path, value, valueName, fields...)
	if err != nil {
		return err
	}
	return nil
}

// getFieldInFsManifest reads the the file at path of the in-memory filesystem, uses
// GetFieldInManiest internally to retrieve the specified field.
func (in *Installer) getFieldInFsManifest(path string, fields ...string) (string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return "", err
	}
	field, err := GetFieldInManifest(string(data), fields...)
	if err != nil {
		return "", err
	}
	return field, nil
}
