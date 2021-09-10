package installer

import (
	"fmt"
	"time"

	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
	corev1 "k8s.io/api/core/v1"
	kstoragev1 "k8s.io/api/storage/v1"
)

func Upgrade(uninstallConfig *apiv1.KubectlStorageOSConfig, installConfig *apiv1.KubectlStorageOSConfig, versionToUninstall string) error {
	// create new installer with in-mem fs of operator and cluster to be installed
	// use installer to validate etcd-endpoints before going any further
	installer, err := NewInstaller(installConfig, true)
	if err != nil {
		return err
	}
	err = installer.handleEndpointsInput(installConfig.Spec.Install.EtcdEndpoints)
	if err != nil {
		return err
	}

	// create (un)installer with in-mem fs of operator and cluster to be uninstalled
	uninstaller, err := NewInstaller(uninstallConfig, false)
	if err != nil {
		return err
	}

	// discover existing secret username and password for upgrade. Here we use the (un)installer
	// as it contains the manifests to be uninstalled, and the installConfig so we can set existing
	// secret username and password in the secret manifest to be installed
	err = uninstaller.storeExistingStorageOSSecretData(installConfig)
	if err != nil {
		return err
	}

	// if the version being uninstalled during upgrade is that of the 'old' operator (pre v2.5) existing
	// CSI secrets and 'fast' storage class must be stored so that they can be recreated after upgrade.
	storeData, err := pluginversion.VersionIsLessThanOrEqual(versionToUninstall, pluginversion.ClusterOperatorLastVersion())
	if err != nil {
		return err
	}
	var csiSecrets []*corev1.Secret
	var storageClass *kstoragev1.StorageClass
	if storeData {
		// discover and store the old (pre v2.5) CSI secrets in kube-system
		csiSecrets, err = uninstaller.storeExistingCSISecrets()
		if err != nil {
			return err
		}
		// get old (pre v2.5) storage class 'fast'
		storageClass, err = pluginutils.GetStorageClass(uninstaller.clientConfig, "fast")
		if err != nil {
			return err
		}
	}
	// uninstall existing storageos operator and cluster
	err = uninstaller.Uninstall(uninstallConfig)
	if err != nil {
		return err
	}

	// sleep to allow CRDs to be removed
	// TODO: Add specific check instead of sleep
	time.Sleep(30 * time.Second)

	// install new storageos operator and cluster
	err = installer.installStorageOS(installConfig)
	if err != nil {
		return err
	}

	if storeData {
		// recreate previously stored storage class from uninstalled version
		err = installer.recreateStorageClass(storageClass)
		if err != nil {
			return err
		}

		// recreate previously stored CSI secrets from uninstalled version
		err = installer.recreateCSISecrets(csiSecrets)
		if err != nil {
			return err
		}
	}
	return nil
}

func (in *Installer) storeExistingStorageOSSecretData(installConfig *apiv1.KubectlStorageOSConfig) error {
	stosCluster, err := pluginutils.GetStorageOSCluster(in.clientConfig, "")
	if err != nil {
		return err
	}

	secret, err := pluginutils.GetSecret(in.clientConfig, stosCluster.Spec.SecretRefName, stosCluster.Spec.SecretRefNamespace)
	if err != nil {
		return err
	}

	installConfig.InstallerMeta.SecretUsername = string(secret.Data["username"])
	installConfig.InstallerMeta.SecretPassword = string(secret.Data["password"])
	return nil
}

func (in *Installer) storeExistingCSISecrets() ([]*corev1.Secret, error) {
	fmt.Println("Storing existing CSI secrets...")
	secrets := make([]*corev1.Secret, 0)
	secretNames := []string{
		"csi-controller-expand-secret",
		"csi-controller-publish-secret",
		"csi-node-publish-secret",
		"csi-provisioner-secret",
	}
	for _, name := range secretNames {
		secret, err := pluginutils.GetSecret(in.clientConfig, name, "kube-system")
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

func (in *Installer) recreateStorageClass(storageClass *kstoragev1.StorageClass) error {
	fmt.Println("Recreating StorageClass...")
	storageClass.SetResourceVersion("")
	err := pluginutils.CreateStorageClass(in.clientConfig, storageClass)
	if err != nil {
		return err
	}
	return nil
}

func (in *Installer) recreateCSISecrets(secrets []*corev1.Secret) error {
	fmt.Println("Recreating CSI secrets...")
	for _, secret := range secrets {
		// In the event that the secret already has finalizer(s), continue,
		// as it will not have been deleted during the operator uninstall
		// phase of the upgrade. This might happen if there has previously
		// been an upgrade and finalizers have been added to the secret.
		finalizers := secret.GetFinalizers()
		if len(finalizers) != 0 {
			continue
		}
		secret.SetFinalizers([]string{stosFinalizer})
		secret.SetResourceVersion("")
		err := in.gracefullyCreateSecret(secret)
		if err != nil {
			return err
		}
	}
	return nil
}

func (in *Installer) gracefullyCreateSecret(secret *corev1.Secret) error {
	err := pluginutils.WaitFor(func() error {
		return pluginutils.SecretDoesNotExist(in.clientConfig, secret.GetObjectMeta().GetName(), secret.GetObjectMeta().GetNamespace())
	}, 180, 5)
	if err != nil {
		return err
	}

	err = pluginutils.CreateSecret(in.clientConfig, secret, secret.GetObjectMeta().GetNamespace())
	if err != nil {
		return err
	}
	err = pluginutils.WaitFor(func() error {
		return pluginutils.SecretExists(in.clientConfig, secret.GetObjectMeta().GetName(), secret.GetObjectMeta().GetNamespace())
	}, 180, 5)
	if err != nil {
		return err
	}

	return nil
}
