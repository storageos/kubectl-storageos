package installer

import (
	"fmt"
	"os"
	"path/filepath"

	otkkubectl "github.com/darkowlzz/operator-toolkit/declarative/kubectl"
	operatorapi "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	kstoragev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kustomize/api/filesys"
)

const (
	// CLI flags
	SkipNamespaceDeletionFlag = "skip-namespace-deletion"
	VersionFlag               = "version"
	StosOperatorYamlFlag      = "stos-operator-yaml"
	StosClusterYamlFlag       = "stos-cluster-yaml"
	StosSecretYamlFlag        = "stos-secret-yaml"
	EtcdOperatorYamlFlag      = "etcd-operator-yaml"
	EtcdClusterYamlFlag       = "etcd-cluster-yaml"
	SkipEtcdFlag              = "skip-etcd"
	EtcdEndpointsFlag         = "etcd-endpoints"
	ConfigPathFlag            = "config-path"
	EtcdNamespaceFlag         = "etcd-namespace"
	StosOperatorNSFlag        = "stos-operator-namespace"
	StosClusterNSFlag         = "stos-cluster-namespace"
	StorageClassFlag          = "storage-class"
	SecretUserFlag            = "secret-username"
	SecretPassFlag            = "secret-password"

	// config file fields - contain path delimiters for plugin interpretation of config manifest
	SkipNamespaceDeletionConfig   = "spec.skipNmespaceDeletion"
	InstallVersionConfig          = "spec.install.version"
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
	etcdDir              = "etcd"
	stosDir              = "storageos"
	operatorDir          = "operator"
	clusterDir           = "cluster"
	stosOperatorFile     = "storageos-operator.yaml"
	stosClusterFile      = "storageos-cluster.yaml"
	stosSecretsFile      = "storageos-secrets.yaml"
	csiSecretsFile       = "storageos-csi-secrets.yaml"
	stosStorageClassFile = "storageos-storageclass.yaml"
	etcdOperatorFile     = "etcd-operator.yaml"
	etcdClusterFile      = "etcd-cluster.yaml"
	kustomizationFile    = "kustomization.yaml"
	kubeDir              = ".kube"
	uninstallDirPrefix   = "uninstall-"

	// kustomization template
	kustTemp = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:`

	// other defaults
	stosClusterKind        = "StorageOSCluster"
	etcdClusterKind        = "EtcdCluster"
	defaultEtcdClusterName = "storageos-etcd"
	defaultEtcdClusterNS   = "storageos-etcd"
	stosFinalizer          = "storageos.com/finalizer"
	stosSCProvisioner      = "csi.storageos.com"
	stosAppLabel           = "app=storageos"
)

// fsData represents dir name, subdir name, file name and file data.
// It is used to create the Installer's in-memory file system.
type fsData map[string]map[string]map[string][]byte

// Installer holds the kubectl client and in-memory fs data used throughout the installation process
type Installer struct {
	kubectlClient *otkkubectl.DefaultKubectl
	clientConfig  *rest.Config
	kubeClusterID types.UID
	stosConfig    *apiv1.KubectlStorageOSConfig
	fileSys       filesys.FileSystem
	onDiskFileSys filesys.FileSystem
}

// NewInstaller returns an Installer object with the kubectl client and in-memory filesystem
func NewInstaller(config *apiv1.KubectlStorageOSConfig, ensureNamespace bool) (*Installer, error) {
	installer := &Installer{}

	clientConfig, err := pluginutils.NewClientConfig()
	if err != nil {
		return installer, err
	}

	if ensureNamespace {
		err = pluginutils.EnsureNamespace(clientConfig, config.Spec.GetNamespace())
		if err != nil {
			return installer, err
		}
	}

	kubesystemNS, err := pluginutils.GetNamespace(clientConfig, "kube-system")
	if err != nil {
		return installer, err
	}

	fileSys, err := buildInstallerFileSys(config, clientConfig)
	if err != nil {
		return installer, err
	}

	installer = &Installer{
		kubectlClient: otkkubectl.New(),
		clientConfig:  clientConfig,
		kubeClusterID: kubesystemNS.GetUID(),
		stosConfig:    config,
		fileSys:       fileSys,
		onDiskFileSys: filesys.MakeFsOnDisk(),
	}

	return installer, nil
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

// writeBackupFileSystem writes manifests of uninstalled secrets, storageoscluster and storageclass to disk
func (in *Installer) writeBackupFileSystem(storageOSCluster *operatorapi.StorageOSCluster) error {
	backupPath, err := in.getBackupPath()
	if err != nil {
		return err
	}
	err = in.onDiskFileSys.MkdirAll(backupPath)
	if err != nil {
		return err
	}

	storageOSClusterManifest, err := storageOSClusterToManifest(storageOSCluster)
	if err != nil {
		return err
	}
	err = in.onDiskFileSys.WriteFile(filepath.Join(backupPath, stosClusterFile), storageOSClusterManifest)
	if err != nil {
		return err
	}

	secretList, err := pluginutils.ListSecrets(in.clientConfig, metav1.ListOptions{LabelSelector: stosAppLabel})
	if err != nil {
		return err
	}
	stosSecretList, csiSecretList := separateSecrets(secretList)
	err = in.writeSecretsToDisk(stosSecretList, filepath.Join(backupPath, stosSecretsFile))
	if err != nil {
		return err
	}
	err = in.writeSecretsToDisk(csiSecretList, filepath.Join(backupPath, csiSecretsFile))
	if err != nil {
		return err
	}

	storageClassList, err := in.listStorageOSStorageClasses()
	if err != nil {
		return err
	}
	err = in.writeStorageClassesToDisk(storageClassList, filepath.Join(backupPath, stosStorageClassFile))
	if err != nil {
		return err
	}

	return nil
}

func (in *Installer) listStorageOSStorageClasses() (*kstoragev1.StorageClassList, error) {
	storageClassList, err := pluginutils.ListStorageClasses(in.clientConfig, metav1.ListOptions{LabelSelector: stosAppLabel})
	if err != nil {
		return nil, err
	}
	stosStorageClassList := &kstoragev1.StorageClassList{}
	for _, storageClass := range storageClassList.Items {
		if storageClass.Provisioner != stosSCProvisioner {
			continue
		}
		stosStorageClassList.Items = append(stosStorageClassList.Items, storageClass)
	}

	return stosStorageClassList, nil

}

// writeSecretsToDisk writes multidoc manifest of SecretList.Items to path of on-disk filesystem
func (in *Installer) writeSecretsToDisk(secretList *corev1.SecretList, path string) error {
	if len(secretList.Items) == 0 {
		return nil
	}
	secretsMultiDoc, err := secretsToMultiDoc(secretList)
	if err != nil {
		return err
	}
	err = in.onDiskFileSys.WriteFile(path, secretsMultiDoc)
	if err != nil {
		return err
	}

	return nil
}

// writeStorageClassesToDisk writes multidoc manifest of StorageClassList.Items to path of on-disk filesystem
func (in *Installer) writeStorageClassesToDisk(storageClassList *kstoragev1.StorageClassList, path string) error {
	if len(storageClassList.Items) == 0 {
		return nil
	}
	storageClassMultiDoc, err := storageClassesToMultiDoc(storageClassList)
	if err != nil {
		return err
	}
	err = in.onDiskFileSys.WriteFile(path, storageClassMultiDoc)
	if err != nil {
		return err
	}

	return nil
}

// getBackupPath returns the path to the on-disk directory where uninstalled manifests are stored.
func (in *Installer) getBackupPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, kubeDir, stosDir, fmt.Sprintf("%s%v", uninstallDirPrefix, in.kubeClusterID)), nil
}
