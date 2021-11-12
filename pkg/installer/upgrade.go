package installer

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	operatorapi "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
)

const outputCopyingPortalData = "Attempting to copy portal manager data from existing storageos-portal-client secret..."

func Upgrade(uninstallConfig *apiv1.KubectlStorageOSConfig, installConfig *apiv1.KubectlStorageOSConfig, versionToUninstall string) error {
	// create new installer with in-mem fs of operator and cluster to be installed
	// use installer to validate etcd-endpoints before going any further
	installer, err := NewInstaller(installConfig, true, true)
	if err != nil {
		return err
	}
	storageOSCluster, err := pluginutils.GetFirstStorageOSCluster(installer.clientConfig)
	if err != nil {
		return err
	}

	// if etcdEndpoints was not passed via config, use that of existing cluster
	if installConfig.Spec.Install.EtcdEndpoints == "" {
		installConfig.Spec.Install.EtcdEndpoints = storageOSCluster.Spec.KVBackend.Address
	}

	if err = installer.handleEndpointsInput(installConfig.Spec.Install); err != nil {
		return err
	}

	// create (un)installer with in-mem fs of operator and cluster to be uninstalled
	uninstaller, err := NewInstaller(uninstallConfig, false, false)
	if err != nil {
		return err
	}

	if err = uninstaller.prepareForUpgrade(installConfig, storageOSCluster, versionToUninstall); err != nil {
		return err
	}

	// uninstall existing storageos operator and cluster
	if err = uninstaller.Uninstall(true, versionToUninstall); err != nil {
		return err
	}

	// sleep to allow CRDs to be removed
	// TODO: Add specific check instead of sleep
	time.Sleep(30 * time.Second)

	// install new storageos operator and cluster
	err = installer.Install(true)

	return err
}

// prepareForUpgrade performs necessary steps before upgrade commences
func (in *Installer) prepareForUpgrade(installConfig *apiv1.KubectlStorageOSConfig, storageOSCluster *operatorapi.StorageOSCluster, versionToUninstall string) error {
	// write storageoscluster, secret and storageclass manifests to disk
	if err := in.writeBackupFileSystem(storageOSCluster); err != nil {
		return errors.WithStack(err)
	}

	// apply the storageclass manifest written to disk (now with finalizer to prevent deletion by operator)
	if err := in.applyBackupManifestWithFinalizer(stosStorageClassFile); err != nil {
		return err
	}

	// if the version being uninstalled during upgrade is that of the 'old' operator (pre v2.5) existing
	// CSI secrets are applied with finalizer to prevent deletion by operator
	oldVersion, err := pluginversion.VersionIsLessThanOrEqual(versionToUninstall, pluginversion.ClusterOperatorLastVersion())
	if err != nil {
		return err
	}
	if !pluginversion.IsDevelop(versionToUninstall) && oldVersion {
		if err = in.applyBackupManifestWithFinalizer(csiSecretsFile); err != nil {
			return err
		}
	}

	// discover uninstalled secret username and password for upgrade. Here we use (1) the (un)installer
	// as it contains the on-disk FS of the uninstalled secrets and (2) the installConfig so we can
	// set secret username and password in the secret manifest to be installed later
	err = in.copyStorageOSSecretData(installConfig)

	return err
}

// copyStorageOSSecretData uses the (un)installer's on-disk filesystem to read the username and password
// of the storageos secret which is to be uninstalled. This data is then copied to the installConfig so
// that it can be added to the new storageos secret to be created during the install phase of the upgrade
func (in *Installer) copyStorageOSSecretData(installConfig *apiv1.KubectlStorageOSConfig) error {
	backupPath, err := in.getBackupPath()
	if err != nil {
		return err
	}
	stosSecrets, err := in.onDiskFileSys.ReadFile(filepath.Join(backupPath, stosSecretsFile))
	if err != nil {
		return errors.WithStack(err)
	}
	storageosAPISecret, err := pluginutils.GetManifestFromMultiDocByName(string(stosSecrets), "storageos-api")
	if err != nil {
		return err
	}
	secretUsername, err := pluginutils.GetFieldInManifest(storageosAPISecret, "data", "username")
	if err != nil {
		return err
	}
	if secretUsername == "" {
		// also check for apiUsername (pre 2.5.0)
		secretUsername, err = pluginutils.GetFieldInManifest(storageosAPISecret, "data", "apiUsername")
		if err != nil {
			return err
		}
	}
	secretPassword, err := pluginutils.GetFieldInManifest(storageosAPISecret, "data", "password")
	if err != nil {
		return err
	}
	if secretPassword == "" {
		// also check for apiPassword (pre 2.5.0)
		secretPassword, err = pluginutils.GetFieldInManifest(storageosAPISecret, "data", "apiPassword")
		if err != nil {
			return err
		}
	}
	if installConfig.Spec.Install.AdminUsername == "" {
		installConfig.Spec.Install.AdminUsername = secretUsername
	}
	if installConfig.Spec.Install.AdminPassword == "" {
		installConfig.Spec.Install.AdminPassword = secretPassword
	}

	// return early if enable-portal-manager is not set
	if !installConfig.Spec.Install.EnablePortalManager {
		return nil
	}
	// if all portal flags have been set, return without reading back-up secret for portal data
	// as values passed by flag take precedent
	if err := PortalFlagsExist(installConfig); err == nil {
		return nil
	}

	fmt.Println("Warning: " + err.Error())
	fmt.Println(outputCopyingPortalData)

	storageosPortalClientSecret, err := pluginutils.GetManifestFromMultiDocByName(string(stosSecrets), "storageos-portal-client")
	if err != nil {
		return err
	}
	secretPortalAPIURL, err := pluginutils.GetFieldInManifest(storageosPortalClientSecret, "data", "URL")
	if err != nil {
		return err
	}
	secretTenantID, err := pluginutils.GetFieldInManifest(storageosPortalClientSecret, "data", "TENANT_ID")
	if err != nil {
		return err
	}

	if installConfig.Spec.Install.PortalAPIURL == "" {
		installConfig.Spec.Install.PortalAPIURL = secretPortalAPIURL
	}
	if installConfig.Spec.Install.TenantID == "" {
		installConfig.Spec.Install.TenantID = secretTenantID
	}

	return nil
}

// applyBackupManifest applies file from the (un)installer's on-disk filesystem with finalizer
func (in *Installer) applyBackupManifestWithFinalizer(file string) error {
	backupPath, err := in.getBackupPath()
	if err != nil {
		return err
	}

	multidoc, err := in.onDiskFileSys.ReadFile(filepath.Join(backupPath, file))
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
		if err = in.kubectlClient.Apply(context.TODO(), "", string(manifestWithFinaliser), true); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
