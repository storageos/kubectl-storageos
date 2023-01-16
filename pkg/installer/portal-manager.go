package installer

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"

	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"github.com/storageos/kubectl-storageos/pkg/version"
)

// NewPortalManagerInstaller returns an Installer used for all portal manager commands
func NewPortalManagerInstaller(config *apiv1.KubectlStorageOSConfig, manifestsRequired bool, log *logger.Logger) (*Installer, error) {
	version.SetPortalManagerLatestSupportedVersion(getStringWithDefault(config.Spec.Install.PortalManagerVersion, config.Spec.Uninstall.PortalManagerVersion))

	portalManagerInstaller, err := newCommonInstaller(config, log)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	portalManagerInstaller.installerOptions = &installerOptions{
		storageosOperator:    false,
		storageosCluster:     false,
		portalClient:         manifestsRequired, // manifests are required for install-portal, and uninstall-portal
		portalConfig:         manifestsRequired, // but not for enable-portal and disable-portal
		resourceQuota:        false,
		etcdOperator:         false,
		etcdCluster:          false,
		localPathProvisioner: false,
	}

	portalManagerInstaller.fileSys, err = portalManagerInstaller.installerOptions.buildInstallerFileSys(config, portalManagerInstaller.clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	portalManagerInstaller.storageOSCluster, err = pluginutils.GetFirstStorageOSCluster(portalManagerInstaller.clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return portalManagerInstaller, nil
}

// EnablePortalManager applies the existing storageoscluster with enablePortalManager set to value of 'enable'.
func (in *Installer) EnablePortalManager(enable bool) error {
	storageOSClusterManifest, err := storageOSClusterToManifest(in.storageOSCluster)
	if err != nil {
		return err
	}
	if err := in.fileSys.WriteFile(filepath.Join(stosDir, clusterDir, stosClusterFile), []byte(storageOSClusterManifest)); err != nil {
		return errors.WithStack(err)
	}
	kustYamlContents, err := pluginutils.SetFieldInManifest(kustTemp, fmt.Sprintf("%s%s%s", "[", stosClusterFile, "]"), "resources", "")
	if err != nil {
		return errors.WithStack(err)
	}
	if err := in.fileSys.WriteFile(filepath.Join(stosDir, clusterDir, kustomizationFile), []byte(kustYamlContents)); err != nil {
		return errors.WithStack(err)
	}

	if err := in.enablePortalManager(in.storageOSCluster.Name, enable); err != nil {
		return err
	}

	return in.kustomizeAndApply(filepath.Join(stosDir, clusterDir), stosClusterFile)
}

// InstallPortalManager installs portal manager necessary components.
func (in *Installer) InstallPortalManager() error {
	if err := in.installPortalManagerClient(in.storageOSCluster.Namespace); err != nil {
		return err
	}
	return in.installPortalManagerConfig(in.storageOSCluster.Namespace)
}

// UninstallPortalManager writes backup-filestem and uninstalls portal manager components.
func (in *Installer) UninstallPortalManager() error {
	if err := in.writeBackupFileSystem(); err != nil {
		return err
	}

	stosClusterNamespace := in.storageOSCluster.Namespace
	if stosClusterNamespace != "" {
		stosClusterNamespace = in.stosConfig.Spec.GetOperatorNamespace()
	}

	if err := in.uninstallPortalManagerConfig(stosClusterNamespace); err != nil {
		return err
	}

	return in.uninstallPortalManagerClient(stosClusterNamespace)
}

func (in *Installer) enablePortalManager(storageOSClusterName string, enable bool) error {
	enablePortalManagerPatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/spec/enablePortalManager",
		Value: strconv.FormatBool(enable),
	}

	return in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, storageOSClusterName, []pluginutils.KustomizePatch{enablePortalManagerPatch})
}

func (in *Installer) installPortalManagerConfig(stosClusterNamespace string) error {
	if !in.installerOptions.portalConfig {
		return nil
	}

	if err := in.setFieldInFsManifest(filepath.Join(stosDir, portalConfigDir, kustomizationFile), stosClusterNamespace, "namespace", ""); err != nil {
		return err
	}

	if in.stosConfig.Spec.Install.PortalHTTPSProxy != "" {
		portalConfig, err := in.getFieldInFsManifest(filepath.Join(stosDir, portalConfigDir, stosPortalConfigFile), "data", "portal_config.yaml")
		if err != nil {
			return err
		}

		portalConfig += fmt.Sprintf("\n  httpsProxy: %s", in.stosConfig.Spec.Install.PortalHTTPSProxy)

		if err := in.setFieldInFsManifest(filepath.Join(stosDir, portalConfigDir, stosPortalConfigFile), portalConfig, "portal_config.yaml", "data"); err != nil {
			return err
		}
	}

	return in.kustomizeAndApply(filepath.Join(stosDir, portalConfigDir), stosPortalConfigFile)
}

func (in *Installer) installPortalManagerClient(stosClusterNamespace string) error {
	if !in.installerOptions.portalClient {
		return nil
	}

	if err := in.setFieldInFsManifest(filepath.Join(stosDir, portalClientDir, kustomizationFile), stosClusterNamespace, "namespace", "secretGenerator", "0"); err != nil {
		return err
	}

	if err := in.setFieldInFsManifest(filepath.Join(stosDir, portalClientDir, kustomizationFile),
		buildStringForKustomize(in.stosConfig.Spec.Install.PortalClientID,
			in.stosConfig.Spec.Install.PortalSecret,
			in.stosConfig.Spec.Install.PortalAPIURL,
			in.stosConfig.Spec.Install.PortalTenantID),
		"literals", "secretGenerator", "0"); err != nil {
		return err
	}
	return in.kustomizeAndApply(filepath.Join(stosDir, portalClientDir), stosPortalClientFile)
}

func buildStringForKustomize(clientID, password, portalURL, tenantID string) string {
	return fmt.Sprint("[",
		"CLIENT_ID", "=", clientID, ",",
		"PASSWORD", "=", password, ",",
		"URL", "=", portalURL, ",",
		"TENANT_ID", "=", tenantID,
		"]")
}

func (in *Installer) uninstallPortalManagerClient(storageOSClusterNamespace string) error {
	if !in.installerOptions.portalClient {
		return nil
	}
	if err := in.setFieldInFsManifest(filepath.Join(stosDir, portalClientDir, kustomizationFile), storageOSClusterNamespace, "namespace", "secretGenerator", "0"); err != nil {
		return err
	}

	return in.kustomizeAndDelete(filepath.Join(stosDir, portalClientDir), stosPortalClientFile)
}

func (in *Installer) uninstallPortalManagerConfig(storageOSClusterNamespace string) error {
	if !in.installerOptions.portalConfig {
		return nil
	}

	if err := in.setFieldInFsManifest(filepath.Join(stosDir, portalConfigDir, kustomizationFile), storageOSClusterNamespace, "namespace", ""); err != nil {
		return err
	}

	return in.kustomizeAndDelete(filepath.Join(stosDir, portalConfigDir), stosPortalConfigFile)
}
