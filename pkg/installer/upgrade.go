package installer

import (
	"context"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
)

const (
	outputCopyingPortalData  = "Attempting to copy portal manager data from existing storageos-portal-client secret."
	errPortalManagerNotFound = `
	Portal manager data necessary to perform upgrade was not found locally.

	Please use the following flags to configure portal manager:

	--portal-client-id
	--portal-secret
	--portal-api-url
	--portal-tenant-id
	`
)

type Upgrader struct {
	installer   *Installer
	uninstaller *Installer
}

func NewUpgrader(uninstallConfig *apiv1.KubectlStorageOSConfig, installConfig *apiv1.KubectlStorageOSConfig, log *logger.Logger) (*Upgrader, error) {
	// create new installer with in-mem fs of operator and cluster to be installed
	installer, err := NewInstaller(installConfig, log)
	if err != nil {
		return nil, err
	}

	// create uninstaller with in-mem fs of operator and cluster to be uninstalled
	uninstaller, err := NewUninstaller(uninstallConfig, log)
	if err != nil {
		return nil, err
	}

	upgrader := &Upgrader{
		installer:   installer,
		uninstaller: uninstaller,
	}

	return upgrader, nil
}

func (u *Upgrader) Upgrade() error {
	// if etcdEndpoints was not passed via config, use that of existing cluster
	if u.installer.stosConfig.Spec.Install.EtcdEndpoints == "" {
		u.installer.stosConfig.Spec.Install.EtcdEndpoints = u.uninstaller.storageOSCluster.Spec.KVBackend.Address
	}

	// if storageOSClusterNamespace was not passed via config, use that of existing cluster
	if u.installer.stosConfig.Spec.Install.StorageOSClusterNamespace == "" {
		// First, check spec.namespace which defines the namespace for storageos installation by storageos/cluster-operator,
		// (this field is deprecated in storageos/operator). Otherwise, use metadata.Namespace for storageos installation
		// (default behaviour for storageos/operator).
		if u.uninstaller.storageOSCluster.Spec.Namespace != "" {
			u.installer.stosConfig.Spec.Install.StorageOSClusterNamespace = u.uninstaller.storageOSCluster.Spec.Namespace
		} else {
			u.installer.stosConfig.Spec.Install.StorageOSClusterNamespace = u.uninstaller.storageOSCluster.Namespace
		}
	}

	if err := u.installer.handleEndpointsInput(); err != nil {
		return err
	}

	if err := u.prepareForUpgrade(); err != nil {
		return err
	}

	// uninstall existing storageos operator and cluster
	if err := u.uninstaller.Uninstall(true); err != nil {
		return err
	}

	// sleep to allow CRDs to be removed
	// TODO: Add specific check instead of sleep
	time.Sleep(30 * time.Second)

	// install new storageos operator and cluster
	return u.installer.Install(true)
}

// prepareForUpgrade performs necessary steps before upgrade commences
func (u *Upgrader) prepareForUpgrade() error {
	// write storageoscluster, secret and storageclass manifests to disk
	if err := u.uninstaller.writeBackupFileSystem(); err != nil {
		return errors.WithStack(err)
	}

	// apply the storageclass manifest written to disk (now with finalizer to prevent deletion by operator)
	if err := u.applyBackupManifestWithFinalizer(stosStorageClassFile); err != nil {
		return err
	}

	// if the version being uninstalled during upgrade is that of the 'old' operator (pre v2.5) existing
	// CSI secrets are applied with finalizer to prevent deletion by operator
	oldVersion, err := pluginversion.VersionIsLessThanOrEqual(u.uninstaller.stosConfig.Spec.Uninstall.StorageOSVersion, pluginversion.ClusterOperatorLastVersion())
	if err != nil {
		return err
	}
	if !pluginversion.IsDevelop(u.uninstaller.stosConfig.Spec.Uninstall.StorageOSVersion) && oldVersion {
		if err = u.applyBackupManifestWithFinalizer(csiSecretsFile); err != nil {
			return err
		}
	}

	// if no storageos-cluster.yaml has been passed to the cli for installation, use the backed-up storageos cluster.
	if u.installer.stosConfig.Spec.Install.StorageOSClusterYaml == "" {
		if err := u.copyStorageOSClusterToMemory(); err != nil {
			return err
		}
	}
	// discover uninstalled secret username and password for upgrade. Here we use (1) the (un)installer
	// as it contains the on-disk FS of the uninstalled secrets and (2) the installConfig so we can
	// set secret username and password in the secret manifest to be installed later
	err = u.copyStorageOSSecretData()

	return err
}

// copyStorageOSClusterToMemory takes the (uninstalled) on-disk storageos-cluster manifest and combines it with
// the installer's in-memory storageos-api secret to create a multi-doc storageos-cluster.yaml. This manifest
// is written to the installer's in-memory fs for installation. Thus maintaining the original cluster's specs.
func (u *Upgrader) copyStorageOSClusterToMemory() error {
	backupPath, err := u.uninstaller.getBackupPath()
	if err != nil {
		return err
	}

	onDiskStosClusterManifest, err := u.uninstaller.onDiskFileSys.ReadFile(filepath.Join(backupPath, stosClusterFile))
	if err != nil {
		return errors.WithStack(err)
	}

	inMemStosClusterManifest, err := u.uninstaller.fileSys.ReadFile(filepath.Join(stosDir, clusterDir, stosClusterFile))
	if err != nil {
		return errors.WithStack(err)
	}

	inMemStosAPISecret, err := pluginutils.GetManifestFromMultiDocByKind(string(inMemStosClusterManifest), "Secret")
	if err != nil {
		return err
	}

	stosClusterManifest := makeMultiDoc(string(onDiskStosClusterManifest), inMemStosAPISecret)

	// write uninstalled manifest to installer filesystem
	if err = u.installer.fileSys.WriteFile(filepath.Join(stosDir, clusterDir, stosClusterFile), []byte(stosClusterManifest)); err != nil {
		return err
	}

	// create patch to remove list of existing images from storageos-cluster
	clusterName, err := pluginutils.GetFieldInMultiDocByKind(stosClusterManifest, stosClusterKind, "metadata", "name")
	if err != nil {
		return err
	}
	removeImagesPatch := pluginutils.KustomizePatch{
		Op:   "remove",
		Path: "/spec/images",
	}

	return u.installer.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, clusterName, []pluginutils.KustomizePatch{removeImagesPatch})
}

// copyStorageOSAPIData reads the username and password of the storageos secret which is to be uninstalled.
// This data is then copied to the installConfig so that it can be added to the new storageos secret to be
// created during the install phase of the upgrade
func (u *Upgrader) copyStorageOSAPIData(stosSecrets string) error {
	storageosAPISecret, err := pluginutils.GetManifestFromMultiDocByName(stosSecrets, "storageos-api")
	if err != nil {
		return err
	}

	// need to search both apiUsername (pre 2.5.0) and username
	decodedAdminUsername, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifestMultiSearch(
			storageosAPISecret, [][]string{{"data", "apiUsername"}, {"data", "username"}})
	})
	if err != nil {
		return err
	}
	if u.installer.stosConfig.Spec.Install.AdminUsername == "" {
		u.installer.stosConfig.Spec.Install.AdminUsername = decodedAdminUsername
	}

	// need to search both apiPassword (pre 2.5.0) and password
	decodedAdminPassword, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifestMultiSearch(
			storageosAPISecret, [][]string{{"data", "apiPassword"}, {"data", "password"}})
	})
	if err != nil {
		return err
	}
	if u.installer.stosConfig.Spec.Install.AdminPassword == "" {
		u.installer.stosConfig.Spec.Install.AdminPassword = decodedAdminPassword
	}

	return nil
}

// copyStorageOSPortalClientData reads the portal-username, portal-password, tenant-id and portal-api-url
// of the portal client secret which is to be uninstalled.
// This data is then copied to the installConfig so that it can be added to the new storageos portal client
// to be created during the install phase of the upgrade
func (u *Upgrader) copyStorageOSPortalClientData(stosSecrets string) error {
	storageosPortalClientSecret, err := pluginutils.GetManifestFromMultiDocByName(stosSecrets, "storageos-portal-client")
	if err != nil {
		return errors.Wrap(err, errPortalManagerNotFound)
	}

	decodedPortalClientID, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifest(
			storageosPortalClientSecret, "data", "CLIENT_ID")
	})
	if err != nil {
		return err
	}
	if u.installer.stosConfig.Spec.Install.PortalClientID == "" {
		u.installer.stosConfig.Spec.Install.PortalClientID = decodedPortalClientID
	}

	decodedPortalSecret, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifest(
			storageosPortalClientSecret, "data", "PASSWORD")
	})
	if err != nil {
		return err
	}
	if u.installer.stosConfig.Spec.Install.PortalSecret == "" {
		u.installer.stosConfig.Spec.Install.PortalSecret = decodedPortalSecret
	}

	decodedPortalTenantID, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifest(
			storageosPortalClientSecret, "data", "TENANT_ID")
	})
	if err != nil {
		return err
	}
	if u.installer.stosConfig.Spec.Install.PortalTenantID == "" {
		u.installer.stosConfig.Spec.Install.PortalTenantID = decodedPortalTenantID
	}

	decodedPortalAPIURL, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifest(
			storageosPortalClientSecret, "data", "URL")
	})
	if err != nil {
		return err
	}
	if u.installer.stosConfig.Spec.Install.PortalAPIURL == "" {
		u.installer.stosConfig.Spec.Install.PortalAPIURL = decodedPortalAPIURL
	}

	return nil
}

// copyStorageOSSecretData discover uninstalled secret username and password for upgrade.
func (u *Upgrader) copyStorageOSSecretData() error {
	backupPath, err := u.uninstaller.getBackupPath()
	if err != nil {
		return err
	}
	stosSecrets, err := u.uninstaller.onDiskFileSys.ReadFile(filepath.Join(backupPath, stosSecretsFile))
	if err != nil {
		return errors.WithStack(err)
	}

	if err := u.copyStorageOSAPIData(string(stosSecrets)); err != nil {
		return err
	}

	// return early if enable-portal-manager is not set
	if !u.installer.stosConfig.Spec.Install.EnablePortalManager {
		return nil
	}
	// if all portal-manager flags have been set, return without reading back-up secret for portal data
	// as values passed by flag take precedent
	if err = FlagsAreSet(map[string]string{
		PortalClientIDFlag: u.uninstaller.stosConfig.Spec.Install.PortalClientID,
		PortalSecretFlag:   u.uninstaller.stosConfig.Spec.Install.PortalSecret,
		PortalTenantIDFlag: u.uninstaller.stosConfig.Spec.Install.PortalTenantID,
		PortalAPIURLFlag:   u.uninstaller.stosConfig.Spec.Install.PortalAPIURL,
	}); err == nil {
		return nil
	}

	// log this error as a warning only, as the user may have intentionally
	// not set portal flags so that existing data is used instead.
	u.uninstaller.log.Warn(err.Error())

	// warn user that existing portal data will be copied.
	u.uninstaller.log.Warn(outputCopyingPortalData)

	return u.copyStorageOSPortalClientData(string(stosSecrets))
}

// applyBackupManifest applies file from the uninstaller's on-disk filesystem with finalizer
func (u *Upgrader) applyBackupManifestWithFinalizer(file string) error {
	backupPath, err := u.uninstaller.getBackupPath()
	if err != nil {
		return err
	}

	multidoc, err := u.uninstaller.onDiskFileSys.ReadFile(filepath.Join(backupPath, file))
	if err != nil {
		return errors.WithStack(err)
	}

	manifests := splitMultiDoc(string(multidoc))
	for _, manifest := range manifests {
		// if a finalizer already exists for this object, continue.
		// This may be the case if an upgrade has already occurred.
		existingFinalizers, err := pluginutils.GetFieldInManifest(string(manifest), "metadata", "finalizers")
		if err != nil {
			return err
		}
		if existingFinalizers != "" {
			continue
		}

		// add finalizer to manifest (mutated manifest is not written to disk)
		manifestWithFinaliser, err := pluginutils.SetFieldInManifest(string(manifest), "- "+stosFinalizer, "finalizers", "metadata")
		if err != nil {
			return err
		}

		if err = u.uninstaller.kubectlClient.Apply(context.TODO(), "", string(manifestWithFinaliser), true); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
