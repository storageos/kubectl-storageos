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

const (
	outputCopyingPortalData  = "Attempting to copy portal manager data from existing storageos-portal-client secret..."
	errPortalManagerNotFound = `
	Portal manager data necessary to perform upgrade was not found locally.

	Please use the following flags to configure portal manager:

	--portal-client-id
	--portal-secret
	--portal-api-url
	--tenant-id
	`
)

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

	// if storageOSClusterNamespace was not passed via config, use that of existing cluster
	if installConfig.Spec.Install.StorageOSClusterNamespace == "" {
		// First, check spec.namespace which defines the namespace for storageos installation by storageos/cluster-operator,
		// (this field is deprecated in storageos/operator). Otherwise, use metadata.Namespace for storageos installation
		// (default behaviour for storageos/operator).
		if storageOSCluster.Spec.Namespace != "" {
			installConfig.Spec.Install.StorageOSClusterNamespace = storageOSCluster.Spec.Namespace
		} else {
			installConfig.Spec.Install.StorageOSClusterNamespace = storageOSCluster.Namespace
		}
	}

	if err = installer.handleEndpointsInput(installConfig.Spec.Install); err != nil {
		return err
	}

	// create (un)installer with in-mem fs of operator and cluster to be uninstalled
	uninstaller, err := NewInstaller(uninstallConfig, false, false)
	if err != nil {
		return err
	}

	if err = uninstaller.prepareForUpgrade(installConfig, storageOSCluster, versionToUninstall, installer); err != nil {
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
func (in *Installer) prepareForUpgrade(installConfig *apiv1.KubectlStorageOSConfig, storageOSCluster *operatorapi.StorageOSCluster, versionToUninstall string, installer *Installer) error {
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

	// if no storageos-cluster.yaml has been passed to the cli, use the backed-up storageos cluster.
	if installConfig.Spec.Install.StorageOSClusterYaml == "" {
		if err := in.copyStorageOSClusterToMemory(installer); err != nil {
			return err
		}
	}
	// discover uninstalled secret username and password for upgrade. Here we use (1) the (un)installer
	// as it contains the on-disk FS of the uninstalled secrets and (2) the installConfig so we can
	// set secret username and password in the secret manifest to be installed later
	err = in.copyStorageOSSecretData(installConfig)

	return err
}

// copyStorageOSClusterToMemory takes the (uninstalled) on-disk storageos-cluster manifest and combines it with
// the installer's in-memory storageos-api secret to create a multi-doc storageos-cluster.yaml. This manifest
// is written to the installer's in-memory fs for installation. Thus maintaining the original cluster's specs.
func (in *Installer) copyStorageOSClusterToMemory(installer *Installer) error {
	backupPath, err := in.getBackupPath()
	if err != nil {
		return err
	}

	onDiskStosClusterManifest, err := in.onDiskFileSys.ReadFile(filepath.Join(backupPath, stosClusterFile))
	if err != nil {
		return errors.WithStack(err)
	}

	inMemStosClusterManifest, err := in.fileSys.ReadFile(filepath.Join(stosDir, clusterDir, stosClusterFile))
	if err != nil {
		return errors.WithStack(err)
	}

	inMemStosAPISecret, err := pluginutils.GetManifestFromMultiDocByKind(string(inMemStosClusterManifest), "Secret")
	if err != nil {
		return err
	}

	stosClusterManifest := makeMultiDoc(string(onDiskStosClusterManifest), inMemStosAPISecret)

	// write unistalled manifest to instller filesystem
	return installer.fileSys.WriteFile(filepath.Join(stosDir, clusterDir, stosClusterFile), []byte(stosClusterManifest))
}

// copyStorageOSAPIData uses the (un)installer's on-disk filesystem to read the username and password
// of the storageos secret which is to be uninstalled. This data is then copied to the installConfig so
// that it can be added to the new storageos secret to be created during the install phase of the upgrade
func (in *Installer) copyStorageOSAPIData(installConfig *apiv1.KubectlStorageOSConfig, stosSecrets string) error {
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
	if installConfig.Spec.Install.AdminUsername == "" {
		installConfig.Spec.Install.AdminUsername = decodedAdminUsername
	}

	// need to search both apiPassword (pre 2.5.0) and password
	decodedAdminPassword, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifestMultiSearch(
			storageosAPISecret, [][]string{{"data", "apiPassword"}, {"data", "password"}})
	})
	if err != nil {
		return err
	}
	if installConfig.Spec.Install.AdminPassword == "" {
		installConfig.Spec.Install.AdminPassword = decodedAdminPassword
	}

	return nil
}

// copyStorageOSPortalClientData uses the (un)installer's on-disk filesystem to read the portal-username,
// portal-password, tenant-id and portal-api-url of the portal client secret which is to be uninstalled.
// This data is then copied to the installConfig so that it can be added to the new storageos portal client
// to be created during the install phase of the upgrade
func (in *Installer) copyStorageOSPortalClientData(installConfig *apiv1.KubectlStorageOSConfig, stosSecrets string) error {
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
	if installConfig.Spec.Install.PortalClientID == "" {
		installConfig.Spec.Install.PortalClientID = decodedPortalClientID
	}

	decodedPortalSecret, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifest(
			storageosPortalClientSecret, "data", "PASSWORD")
	})
	if err != nil {
		return err
	}
	if installConfig.Spec.Install.PortalSecret == "" {
		installConfig.Spec.Install.PortalSecret = decodedPortalSecret
	}

	decodedPortalTenantID, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifest(
			storageosPortalClientSecret, "data", "TENANT_ID")
	})
	if err != nil {
		return err
	}
	if installConfig.Spec.Install.PortalTenantID == "" {
		installConfig.Spec.Install.PortalTenantID = decodedPortalTenantID
	}

	decodedPortalAPIURL, err := pluginutils.GetDecodedManifestField(func() (string, error) {
		return pluginutils.GetFieldInManifest(
			storageosPortalClientSecret, "data", "URL")
	})
	if err != nil {
		return err
	}
	if installConfig.Spec.Install.PortalAPIURL == "" {
		installConfig.Spec.Install.PortalAPIURL = decodedPortalAPIURL
	}

	return nil
}

// copyStorageOSSecretData
func (in *Installer) copyStorageOSSecretData(installConfig *apiv1.KubectlStorageOSConfig) error {
	backupPath, err := in.getBackupPath()
	if err != nil {
		return err
	}
	stosSecrets, err := in.onDiskFileSys.ReadFile(filepath.Join(backupPath, stosSecretsFile))
	if err != nil {
		return errors.WithStack(err)
	}

	if err := in.copyStorageOSAPIData(installConfig, string(stosSecrets)); err != nil {
		return err
	}

	// return early if enable-portal-manager is not set
	if !installConfig.Spec.Install.EnablePortalManager {
		return nil
	}
	// if all portal-manager flags have been set, return without reading back-up secret for portal data
	// as values passed by flag take precedent
	if err = FlagsAreSet(map[string]string{
		PortalClientIDFlag: in.stosConfig.Spec.Install.PortalClientID,
		PortalSecretFlag:   in.stosConfig.Spec.Install.PortalSecret,
		PortalTenantIDFlag: in.stosConfig.Spec.Install.PortalTenantID,
		PortalAPIURLFlag:   in.stosConfig.Spec.Install.PortalAPIURL,
	}); err == nil {
		return nil
	}

	fmt.Println("Warning: " + err.Error())
	fmt.Println(outputCopyingPortalData)

	return in.copyStorageOSPortalClientData(installConfig, string(stosSecrets))
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
