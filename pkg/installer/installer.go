package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	otkkubectl "github.com/ondat/operator-toolkit/declarative/kubectl"
	"github.com/pkg/errors"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
	operatorapi "github.com/storageos/operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	kstoragev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kustomize/api/filesys"
)

const (
	// CLI flags
	StackTraceFlag                  = "stack-trace"
	SkipNamespaceDeletionFlag       = "skip-namespace-deletion"
	SkipExistingWorkloadCheckFlag   = "skip-existing-workload-check"
	StosVersionFlag                 = "stos-version"
	EtcdOperatorVersionFlag         = "etcd-operator-version"
	K8sVersionFlag                  = "k8s-version"
	WaitFlag                        = "wait"
	DryRunFlag                      = "dry-run"
	StosOperatorYamlFlag            = "stos-operator-yaml"
	StosClusterYamlFlag             = "stos-cluster-yaml"
	StosPortalConfigYamlFlag        = "stos-portal-config-yaml"
	StosPortalClientSecretYamlFlag  = "stos-portal-client-secret-yaml"
	EtcdOperatorYamlFlag            = "etcd-operator-yaml"
	EtcdClusterYamlFlag             = "etcd-cluster-yaml"
	EtcdDockerRepositoryFlag        = "etcd-docker-repository"
	EtcdVersionTag                  = "etcd-version-tag"
	ResourceQuotaYamlFlag           = "resource-quota-yaml"
	IncludeEtcdFlag                 = "include-etcd"
	EtcdEndpointsFlag               = "etcd-endpoints"
	SkipEtcdEndpointsValFlag        = "skip-etcd-endpoints-validation"
	SkipStosClusterFlag             = "skip-stos-cluster"
	EtcdTLSEnabledFlag              = "etcd-tls-enabled"
	EtcdSecretNameFlag              = "etcd-secret-name"
	StosConfigPathFlag              = "stos-config-path"
	EtcdNamespaceFlag               = "etcd-namespace"
	StosOperatorNSFlag              = "stos-operator-namespace"
	StosClusterNSFlag               = "stos-cluster-namespace"
	EtcdStorageClassFlag            = "etcd-storage-class"
	AdminUsernameFlag               = "admin-username"
	AdminPasswordFlag               = "admin-password"
	PortalClientIDFlag              = "portal-client-id"
	PortalSecretFlag                = "portal-secret"
	PortalTenantIDFlag              = "portal-tenant-id"
	PortalAPIURLFlag                = "portal-api-url"
	EnablePortalManagerFlag         = "enable-portal-manager"
	IncludeLocalPathProvisionerFlag = "include-local-path-storage-class"
	LocalPathProvisionerYamlFlag    = "local-path-provisioner-yaml"
	EtcdTopologyKeyFlag             = "etcd-topology-key"
	EtcdCPULimitFlag                = "etcd-cpu-limit"
	EtcdMemoryLimitFlag             = "etcd-memory-limit"
	EtcdReplicasFlag                = "etcd-replicas"
	EnableMetricsFlag               = "enable-metrics"
	HiddenTestCluster               = "test-cluster"

	// config file fields - contain path delimiters for plugin interpretation of config manifest
	StackTraceConfig                          = "spec.stackTrace"
	SkipNamespaceDeletionConfig               = "spec.skipNamespaceDeletion"
	SkipExistingWorkloadCheckConfig           = "spec.skipExistingWorkloadCheck"
	SkipStosClusterConfig                     = "spec.skipStorageOSCluster"
	IncludeEtcdConfig                         = "spec.includeEtcd"
	WaitConfig                                = "spec.install.wait"
	DryRunConfig                              = "spec.install.dryRun"
	StosVersionConfig                         = "spec.install.storageOSVersion"
	EtcdOperatorVersionConfig                 = "spec.install.etcdOperatorVersion"
	K8sVersionConfig                          = "spec.install.kubernetesVersion"
	InstallEtcdNamespaceConfig                = "spec.install.etcdNamespace"
	InstallStosOperatorNSConfig               = "spec.install.storageOSOperatorNamespace"
	StosClusterNSConfig                       = "spec.install.storageOSClusterNamespace"
	InstallStosOperatorYamlConfig             = "spec.install.storageOSOperatorYaml"
	InstallStosClusterYamlConfig              = "spec.install.storageOSClusterYaml"
	InstallStosPortalConfigYamlConfig         = "spec.install.storageOSPortalConfigYaml"
	InstallStosPortalClientSecretYamlConfig   = "spec.install.storageOSPortalClientSecretYaml"
	InstallEtcdOperatorYamlConfig             = "spec.install.etcdOperatorYaml"
	InstallEtcdClusterYamlConfig              = "spec.install.etcdClusterYaml"
	InstallResourceQuotaYamlConfig            = "spec.install.resourceQuotaYaml"
	EtcdEndpointsConfig                       = "spec.install.etcdEndpoints"
	SkipEtcdEndpointsValConfig                = "spec.install.skipEtcdEndpointsValidation"
	EtcdTLSEnabledConfig                      = "spec.install.etcdTLSEnabled"
	EtcdSecretNameConfig                      = "spec.install.etcdSecretName"
	EtcdStorageClassConfig                    = "spec.install.etcdStorageClassName"
	AdminUsernameConfig                       = "spec.install.adminUsername"
	AdminPasswordConfig                       = "spec.install.adminPassword"
	PortalClientIDConfig                      = "spec.install.portalClientID"
	PortalSecretConfig                        = "spec.install.portalSecret"
	PortalTenantIDConfig                      = "spec.install.portalTenantID"
	PortalAPIURLConfig                        = "spec.install.portalAPIURL"
	EnablePortalManagerConfig                 = "spec.install.enablePortalManager"
	UninstallEtcdNSConfig                     = "spec.uninstall.etcdNamespace"
	UninstallStosOperatorNSConfig             = "spec.uninstall.storageOSOperatorNamespace"
	UninstallStosOperatorYamlConfig           = "spec.uninstall.storageOSOperatorYaml"
	UninstallStosClusterYamlConfig            = "spec.uninstall.storageOSClusterYaml"
	UninstallStosPortalConfigYamlConfig       = "spec.uninstall.storageOSPortalConfigYaml"
	UninstallStosPortalClientSecretYamlConfig = "spec.uninstall.storageOSPortalClientSecretYaml"
	UninstallEtcdOperatorYamlConfig           = "spec.uninstall.etcdOperatorYaml"
	UninstallEtcdClusterYamlConfig            = "spec.uninstall.etcdClusterYaml"
	UninstallResourceQuotaYamlConfig          = "spec.uninstall.resourceQuotaYaml"
	IncludeLocalPathProvisionerConfig         = "spec.includeLocalPathProvisioner"
	InstallLocalPathProvisionerYamlConfig     = "spec.install.localPathProvisionerYamlConfig"
	UninstallLocalPathProvisionerYamlConfig   = "spec.uninstall.localPathProvisionerYamlConfig"
	EtcdVersionTagConfig                      = "spec.install.etcdVersionTag"
	EtcdDockerRepositoryConfig                = "spec.install.etcdDockerRepository"
	EtcdTopologyKeyConfig                     = "spec.install.etcdTopologyKey"
	EtcdCPULimitConfig                        = "spec.install.etcdCPULimit"
	EtcdMemoryLimitConfig                     = "spec.install.etcdMemoryLimit"
	EtcdReplicasConfig                        = "spec.install.etcdReplicas"
	EnableMetricsConfig                       = "spec.install.enableMetrics"
	MarkTestCluster                           = "spec.install.markTestCluster"

	// dir and file names for in memory fs
	etcdDir                  = "etcd"
	stosDir                  = "storageos"
	localPathProvisionerDir  = "local-path-provisioner"
	operatorDir              = "operator"
	clusterDir               = "cluster"
	storageclassDir          = "storage-class"
	resourceQuotaDir         = "resource-quota"
	portalClientDir          = "portal-client"
	portalConfigDir          = "portal-config"
	stosOperatorFile         = "storageos-operator.yaml"
	stosClusterFile          = "storageos-cluster.yaml"
	resourceQuotaFile        = "resource-quota.yaml"
	stosPortalClientFile     = "storageos-portal-client.yaml"
	stosPortalConfigFile     = "storageos-portal-configmap.yaml"
	stosSecretsFile          = "storageos-secrets.yaml"
	csiSecretsFile           = "storageos-csi-secrets.yaml"
	stosStorageClassFile     = "storageos-storageclass.yaml"
	stosConfigMapsFile       = "storageos-configmaps.yaml"
	etcdOperatorFile         = "etcd-operator.yaml"
	etcdClusterFile          = "etcd-cluster.yaml"
	kustomizationFile        = "kustomization.yaml"
	localPathProvisionerFile = "local-path-provisioner-storage-class.yaml"
	kubeDir                  = ".kube"
	InstallPrefix            = "install-"
	UninstallPrefix          = "uninstall-"

	// kustomization template
	kustTemp = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:`

	// other defaults
	stosClusterKind   = "StorageOSCluster"
	resourceQuotaKind = "ResourceQuota"
	etcdClusterKind   = "EtcdCluster"
	stosFinalizer     = "storageos.com/finalizer"
	stosSCProvisioner = "csi.storageos.com"
	stosAppLabel      = "app=storageos"
)

var (
	// SerialInstall allows the installer to install operators serially not parallel.
	// This could be change with build flag:
	// -X github.com/storageos/kubectl-storageos/pkg/installer.SerialInstall=ANY_VALUE
	SerialInstall string
	serialInstall bool
)

func init() {
	serialInstall = SerialInstall != ""
}

type multipleErrors struct {
	errors []string
}

func (me multipleErrors) Error() string {
	return "Multiple errors: \n" + strings.Join(me.errors, "\n---\n")
}

// fsData represents dir name, subdir name, file name and file data.
// It is used to create the Installer's in-memory file system.
type fsData map[string]map[string]map[string][]byte

// Installer holds the kubectl client and in-memory fs data used throughout the installation process
type Installer struct {
	distribution      pluginutils.Distribution
	kubectlClient     *otkkubectl.DefaultKubectl
	clientConfig      *rest.Config
	kubeClusterID     types.UID
	stosConfig        *apiv1.KubectlStorageOSConfig
	fileSys           filesys.FileSystem
	onDiskFileSys     filesys.FileSystem
	installerOptions  *installerOptions
	dryRunFileCounter int
	storageOSCluster  *operatorapi.StorageOSCluster
}

// NewInstaller returns an Installer used for install command
func NewInstaller(config *apiv1.KubectlStorageOSConfig) (*Installer, error) {
	in, err := newCommonInstaller(config)
	if err != nil {
		return in, errors.WithStack(err)
	}

	installerOptions := &installerOptions{
		storageosOperator:    true,
		storageosCluster:     !config.Spec.SkipStorageOSCluster,
		portalClient:         config.Spec.Install.EnablePortalManager,
		portalConfig:         config.Spec.Install.EnablePortalManager,
		resourceQuota:        (in.distribution == pluginutils.DistributionGKE),
		etcdOperator:         config.Spec.IncludeEtcd,
		etcdCluster:          config.Spec.IncludeEtcd,
		localPathProvisioner: config.Spec.IncludeLocalPathProvisioner,
	}
	in.installerOptions = installerOptions

	fileSys, err := installerOptions.buildInstallerFileSys(config, in.clientConfig)
	if err != nil {
		return in, errors.WithStack(err)
	}

	in.fileSys = fileSys

	return in, nil
}

// NewPortalManagerInstaller returns an Installer used for all portal manager commands
func NewPortalManagerInstaller(config *apiv1.KubectlStorageOSConfig, manifestsRequired bool) (*Installer, error) {
	in, err := newCommonInstaller(config)
	if err != nil {
		return in, errors.WithStack(err)
	}

	installerOptions := &installerOptions{
		storageosOperator:    false,
		storageosCluster:     false,
		portalClient:         manifestsRequired, // manifests are required for install-portal, and uninstall-portal
		portalConfig:         manifestsRequired, // but not for enable-portal and disable-portal
		resourceQuota:        false,
		etcdOperator:         false,
		etcdCluster:          false,
		localPathProvisioner: false,
	}
	in.installerOptions = installerOptions

	fileSys, err := installerOptions.buildInstallerFileSys(config, in.clientConfig)
	if err != nil {
		return in, errors.WithStack(err)
	}

	in.fileSys = fileSys

	stosCluster, err := pluginutils.GetFirstStorageOSCluster(in.clientConfig)
	if err != nil {
		return in, errors.WithStack(err)
	}

	in.storageOSCluster = stosCluster

	return in, nil
}

// newCommonInstaller contains logic that is common to NewInstaller and NewPortalManagerInstaller
func newCommonInstaller(config *apiv1.KubectlStorageOSConfig) (*Installer, error) {
	installer := &Installer{}
	clientConfig, err := pluginutils.NewClientConfig()
	if err != nil {
		return installer, errors.WithStack(err)
	}

	if err := pluginutils.EnsureNamespace(clientConfig, config.Spec.GetOperatorNamespace()); err != nil {
		return installer, errors.WithStack(err)
	}

	if etcdNS := config.Spec.GetETCDValidationNamespace(); etcdNS != "" && etcdNS != config.Spec.GetOperatorNamespace() {
		err = pluginutils.EnsureNamespace(clientConfig, etcdNS)
		if err != nil {
			return installer, errors.WithStack(err)
		}
	}

	currentVersionStr := config.Spec.Install.KubernetesVersion
	if currentVersionStr == "" {
		currentVersion, err := pluginutils.GetKubernetesVersion(clientConfig)
		if err != nil {
			return installer, errors.WithStack(err)
		}
		currentVersionStr = currentVersion.String()
	}

	distribution := pluginutils.DetermineDistribution(currentVersionStr)

	minVersion, err := fetchImageAndExtractFileFromTarball(pluginversion.OperatorLatestSupportedImageURL(), "MIN_KUBE_VERSION")
	// Version 2.5.0-beta.1 doesn't contains the version file. After 2.5.0 has released error handling needs here.
	if err == nil && minVersion != "" {
		supported, err := pluginversion.IsSupported(currentVersionStr, minVersion)
		if err != nil {
			return installer, errors.WithStack(err)
		} else if !supported {
			return installer, errors.WithStack(fmt.Errorf("current version of Kubernetes is lower than required minimum version [%s]", minVersion))
		}
	}

	kubesystemNS, err := pluginutils.GetNamespace(clientConfig, "kube-system")
	if err != nil {
		return installer, errors.WithStack(err)
	}

	installer = &Installer{
		distribution:  distribution,
		kubectlClient: otkkubectl.New(),
		clientConfig:  clientConfig,
		kubeClusterID: kubesystemNS.GetUID(),
		stosConfig:    config,
		onDiskFileSys: filesys.MakeFsOnDisk(),
	}

	return installer, nil
}

// NewDryRunInstaller returns a lightweight Installer object for '--dry-run' enabled commands
func NewDryRunInstaller(config *apiv1.KubectlStorageOSConfig) (*Installer, error) {
	installer := &Installer{}

	clientConfig, err := pluginutils.NewClientConfig()
	if err != nil {
		return installer, errors.WithStack(err)
	}

	distribution := pluginutils.DetermineDistribution(config.Spec.Install.KubernetesVersion)

	installerOptions := &installerOptions{
		storageosOperator:    true,
		storageosCluster:     !config.Spec.SkipStorageOSCluster,
		portalClient:         config.Spec.Install.EnablePortalManager,
		portalConfig:         config.Spec.Install.EnablePortalManager,
		resourceQuota:        (distribution == pluginutils.DistributionGKE),
		etcdOperator:         config.Spec.IncludeEtcd,
		etcdCluster:          config.Spec.IncludeEtcd,
		localPathProvisioner: config.Spec.IncludeLocalPathProvisioner,
	}

	fileSys, err := installerOptions.buildInstallerFileSys(config, clientConfig)
	if err != nil {
		return installer, errors.WithStack(err)
	}

	installer = &Installer{
		distribution:      distribution,
		clientConfig:      clientConfig,
		stosConfig:        config,
		fileSys:           fileSys,
		onDiskFileSys:     filesys.MakeFsOnDisk(),
		installerOptions:  installerOptions,
		dryRunFileCounter: 0,
	}

	return installer, nil
}

// NewUninstaller returns an Installer used for uninstall command
func NewUninstaller(config *apiv1.KubectlStorageOSConfig) (*Installer, error) {
	uninstaller := &Installer{}

	clientConfig, err := pluginutils.NewClientConfig()
	if err != nil {
		return uninstaller, errors.WithStack(err)
	}

	currentVersion, err := pluginutils.GetKubernetesVersion(clientConfig)
	if err != nil {
		return uninstaller, errors.WithStack(err)
	}

	distribution := pluginutils.DetermineDistribution(currentVersion.String())

	kubesystemNS, err := pluginutils.GetNamespace(clientConfig, "kube-system")
	if err != nil {
		return uninstaller, errors.WithStack(err)
	}

	stosCluster, err := pluginutils.GetFirstStorageOSCluster(clientConfig)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return uninstaller, errors.WithStack(err)
		}
	}

	uninstallPortal := false
	if stosCluster != nil {
		uninstallPortal = stosCluster.Spec.EnablePortalManager
	}

	uninstallerOptions := &installerOptions{
		storageosOperator:    true,
		storageosCluster:     !config.Spec.SkipStorageOSCluster,
		portalClient:         uninstallPortal,
		portalConfig:         uninstallPortal,
		resourceQuota:        distribution == pluginutils.DistributionGKE,
		etcdOperator:         config.Spec.IncludeEtcd,
		etcdCluster:          config.Spec.IncludeEtcd,
		localPathProvisioner: config.Spec.IncludeLocalPathProvisioner,
	}
	uninstaller.installerOptions = uninstallerOptions

	fileSys, err := uninstallerOptions.buildInstallerFileSys(config, clientConfig)
	if err != nil {
		return uninstaller, errors.WithStack(err)
	}

	uninstaller = &Installer{
		distribution:     distribution,
		kubectlClient:    otkkubectl.New(),
		clientConfig:     clientConfig,
		kubeClusterID:    kubesystemNS.GetUID(),
		stosConfig:       config,
		fileSys:          fileSys,
		onDiskFileSys:    filesys.MakeFsOnDisk(),
		installerOptions: uninstallerOptions,
		storageOSCluster: stosCluster,
	}

	return uninstaller, nil
}

// addPatchesToFSKustomize uses AddPatchesToKustomize internally to add a list of patches to a kustomization file
// at path of in-memory fs.
func (in *Installer) addPatchesToFSKustomize(path, targetKind, targetName string, patches []pluginutils.KustomizePatch) error {
	kustFile, err := in.fileSys.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}

	kustFileWithPatches, err := pluginutils.AddPatchesToKustomize(string(kustFile), targetKind, targetName, patches)
	if err != nil {
		return err
	}

	err = in.fileSys.WriteFile(path, []byte(kustFileWithPatches))

	return errors.WithStack(err)
}

// setFieldInFsManifest reads the file at path of the in-memory filesystem, uses
// SetFieldInManiest internally to perform the update and then writes the returned file to path.
func (in *Installer) setFieldInFsManifest(path, value, valueName string, fields ...string) error {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}
	dataStr, err := pluginutils.SetFieldInManifest(string(data), value, valueName, fields...)
	if err != nil {
		return err
	}
	err = in.fileSys.WriteFile(path, []byte(dataStr))

	return errors.WithStack(err)
}

// getFieldInFsManifest reads the file at path of the in-memory filesystem, uses
// GetFieldInManiest internally to perform the update and then writes the returned file to path.
func (in *Installer) getFieldInFsManifest(path string, fields ...string) (string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return "", errors.WithStack(err)
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
		return "", errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}
	manifests, err := pluginutils.GetAllManifestsOfKindFromMultiDoc(string(data), kind)
	if err != nil {
		return nil, err
	}
	return manifests, nil
}

// omitKindFromMultiDoc reads the file at path of the in-memory filesystem, uses
// OmitKindFromMultiDoc internally to perform the update and then writes the returned file to path,
// also returninng a string of the manifests omitted.
func (in *Installer) omitKindFromMultiDoc(path, kind string) (string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return "", errors.WithStack(err)
	}
	dataStr, err := pluginutils.OmitKindFromMultiDoc(string(data), kind)
	if err != nil {
		return "", err
	}
	err = in.fileSys.WriteFile(path, []byte(dataStr))
	if err != nil {
		return "", errors.WithStack(err)
	}
	return dataStr, nil
}

// omitAndReturnKindFromFSMultiDoc reads the file at path of the in-memory filesystem, uses
// OmitAndReturnKindFromMultiDoc internally to perform the update and then writes the returned file to path,
// also returninng a []string of the objects omitted.
func (in *Installer) omitAndReturnKindFromFSMultiDoc(path, kind string) ([]string, error) {
	data, err := in.fileSys.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	dataStr, objsOfKind, err := pluginutils.OmitAndReturnKindFromMultiDoc(string(data), kind)
	if err != nil {
		return nil, err
	}
	err = in.fileSys.WriteFile(path, []byte(dataStr))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return objsOfKind, nil
}

// writeBackupFileSystem writes manifests of uninstalled secrets, configmaps, storageoscluster and storageclass to disk
func (in *Installer) writeBackupFileSystem(storageOSCluster *operatorapi.StorageOSCluster) error {
	backupPath, err := in.getBackupPath()
	if err != nil {
		return err
	}
	if err = in.onDiskFileSys.MkdirAll(backupPath); err != nil {
		return errors.WithStack(err)
	}

	storageOSClusterManifest, err := storageOSClusterToManifest(storageOSCluster)
	if err != nil {
		return err
	}
	if err = in.onDiskFileSys.WriteFile(filepath.Join(backupPath, stosClusterFile), storageOSClusterManifest); err != nil {
		return errors.WithStack(err)
	}

	secretList, err := pluginutils.ListSecrets(in.clientConfig, metav1.ListOptions{LabelSelector: stosAppLabel})
	if err != nil {
		return err
	}
	stosSecretList, csiSecretList := separateSecrets(secretList)
	if err = in.writeSecretsToDisk(stosSecretList, filepath.Join(backupPath, stosSecretsFile)); err != nil {
		return errors.WithStack(err)
	}
	if err = in.writeSecretsToDisk(csiSecretList, filepath.Join(backupPath, csiSecretsFile)); err != nil {
		return errors.WithStack(err)
	}

	storageClassList, err := in.listStorageOSStorageClasses()
	if err != nil {
		return err
	}
	if err := in.writeStorageClassesToDisk(storageClassList, filepath.Join(backupPath, stosStorageClassFile)); err != nil {
		return errors.WithStack(err)
	}

	configMapList, err := pluginutils.ListConfigMaps(in.clientConfig, metav1.ListOptions{LabelSelector: stosAppLabel})
	if err != nil {
		return err
	}
	err = in.writeConfigMapsToDisk(configMapList, filepath.Join(backupPath, stosConfigMapsFile))

	return errors.WithStack(err)
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

	return errors.WithStack(err)
}

// writeConfigMapsToDisk writes multidoc manifest of ConfigMapList.Items to path of on-disk filesystem
func (in *Installer) writeConfigMapsToDisk(configMapList *corev1.ConfigMapList, path string) error {
	if len(configMapList.Items) == 0 {
		return nil
	}
	configMapMultiDoc, err := configMapsToMultiDoc(configMapList)
	if err != nil {
		return err
	}
	err = in.onDiskFileSys.WriteFile(path, configMapMultiDoc)

	return errors.WithStack(err)
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

	return errors.WithStack(err)
}

// getBackupPath returns the path to the on-disk directory where uninstalled manifests are stored.
func (in *Installer) getBackupPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return filepath.Join(homeDir, kubeDir, stosDir, fmt.Sprintf("%s%v", UninstallPrefix, in.kubeClusterID)), nil
}
