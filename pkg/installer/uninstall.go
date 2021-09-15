package installer

import (
	"context"
	"fmt"
	"path/filepath"

	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/kustomize/api/krusty"
)

const skipNamespaceDeletionMessage = `Namespace %s still has resources.
Deletion of namespace has skipped.
Reason: %s
Please check the existing resources by executing the following command:
kubectl get all -n %s
To delete namespace please execute the command below:
kubectl delete namespace %s`

// Uninstall performs storageos and etcd uninstallation for kubectl-storageos
func (in *Installer) Uninstall(config *apiv1.KubectlStorageOSConfig, upgrade bool) error {
	err := in.uninstallStorageOS(config.Spec.Uninstall, upgrade)
	if err != nil {
		return err
	}

	// return early if user only wishes to delete storageos, leaving etcd untouched
	if config.Spec.Uninstall.SkipEtcd {
		return nil
	}
	err = in.uninstallEtcd(config.Spec.Uninstall.EtcdNamespace)
	if err != nil {
		return err
	}

	return nil
}

func (in *Installer) uninstallStorageOS(uninstallConfig apiv1.Uninstall, upgrade bool) error {
	// add changes to storageos kustomizations here before kustomizeAndDelete calls ie make changes
	// to storageos/operator/kustomization.yaml and/or storageos/cluster/kustomization.yaml
	// based on flags (or cli config file)
	if uninstallConfig.StorageOSOperatorNamespace != "" {
		err := in.setFieldInFsManifest(filepath.Join(stosDir, operatorDir, kustomizationFile), uninstallConfig.StorageOSOperatorNamespace, "namespace", "")
		if err != nil {
			return err
		}

	}

	fsClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "name")
	if err != nil {
		return err
	}

	storageOSCluster, err := pluginutils.GetStorageOSCluster(in.clientConfig, "")
	if err != nil {
		return err
	}

	clusterNamePatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/metadata/name",
		Value: storageOSCluster.GetObjectMeta().GetName(),
	}

	err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{clusterNamePatch})
	if err != nil {
		return err
	}

	clusterNamespacePatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/metadata/namespace",
		Value: storageOSCluster.GetObjectMeta().GetNamespace(),
	}

	err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{clusterNamespacePatch})
	if err != nil {
		return err
	}

	// edit storageos secret name and namespace via storageoscluster spec, apply patches to in-mem secret
	fsSecretName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), "Secret", "metadata", "name")
	if err != nil {
		return err
	}
	secretNamePatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/metadata/name",
		Value: storageOSCluster.Spec.SecretRefName,
	}

	err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{secretNamePatch})
	if err != nil {
		return err
	}

	secretNamespacePatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/metadata/namespace",
		Value: storageOSCluster.Spec.SecretRefNamespace,
	}

	err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{secretNamespacePatch})
	if err != nil {
		return err
	}

	// if this is not an upgrade, write manifests to disk before deletion
	if !upgrade {
		err = in.writeBackupFileSystem(storageOSCluster)
		if err != nil {
			return err
		}
	}
	err = in.kustomizeAndDelete(filepath.Join(stosDir, clusterDir), stosClusterFile)
	if err != nil {
		return err
	}

	// allow storageoscluster object to be deleted before continuing uninstall process
	err = in.waitForCustomResourceDeletion(func() error {
		return pluginutils.StorageOSClusterDoesNotExist(in.clientConfig, storageOSCluster.GetObjectMeta().GetNamespace())
	})
	if err != nil {
		return err
	}

	err = in.kustomizeAndDelete(filepath.Join(stosDir, operatorDir), stosOperatorFile)
	if err != nil {
		return err
	}

	err = in.gracefullyDeleteNS(pluginutils.NamespaceYaml(uninstallConfig.StorageOSClusterNamespace))
	if err != nil {
		if _, ok := err.(pluginutils.ResourcesStillExists); !ok {
			return err
		}
		println(fmt.Sprintf(skipNamespaceDeletionMessage, uninstallConfig.StorageOSClusterNamespace, err.Error(), uninstallConfig.StorageOSClusterNamespace, uninstallConfig.StorageOSClusterNamespace))
	}

	if uninstallConfig.StorageOSClusterNamespace != uninstallConfig.StorageOSOperatorNamespace {
		err = in.gracefullyDeleteNS(pluginutils.NamespaceYaml(uninstallConfig.StorageOSOperatorNamespace))
		if err != nil {
			if _, ok := err.(pluginutils.ResourcesStillExists); !ok {
				return err
			}
			println(fmt.Sprintf(skipNamespaceDeletionMessage, uninstallConfig.StorageOSOperatorNamespace, err.Error(), uninstallConfig.StorageOSOperatorNamespace, uninstallConfig.StorageOSOperatorNamespace))
		}
	}

	return nil
}

func (in *Installer) uninstallEtcd(etcdNamespace string) error {
	// add changes to etcd kustomizations here before kustomizeAndDelete calls ie make changes
	// to etcd/operator/kustomization.yaml and/or etcd/cluster/kustomization.yaml
	// based on flags (or cli config file)
	if etcdNamespace != "" {
		err := in.setFieldInFsManifest(filepath.Join(etcdDir, operatorDir, kustomizationFile), etcdNamespace, "namespace", "")
		if err != nil {
			return err
		}
		err = in.setFieldInFsManifest(filepath.Join(etcdDir, clusterDir, kustomizationFile), etcdNamespace, "namespace", "")
		if err != nil {
			return err
		}

	}

	err := in.kustomizeAndDelete(filepath.Join(etcdDir, clusterDir), etcdClusterFile)
	if err != nil {
		return err
	}

	fsEtcdName, err := in.getFieldInFsMultiDocByKind(filepath.Join(etcdDir, clusterDir, etcdClusterFile), etcdClusterKind, "metadata", "name")
	if err != nil {
		return err
	}

	fsEtcdNamespace, err := in.getFieldInFsMultiDocByKind(filepath.Join(etcdDir, clusterDir, etcdClusterFile), etcdClusterKind, "metadata", "namespace")
	if err != nil {
		return err
	}

	// allow etcdcluster object to be deleted before continuing uninstall process
	err = in.waitForCustomResourceDeletion(func() error {
		return pluginutils.EtcdClusterDoesNotExist(in.clientConfig, fsEtcdName, fsEtcdNamespace)
	})
	if err != nil {
		return err
	}

	err = in.kustomizeAndDelete(filepath.Join(etcdDir, operatorDir), etcdOperatorFile)
	if err != nil {
		return err
	}

	return nil
}

// kustomizeAndDelete performs the following in the order described:
// - kustomize run (build) on the provided 'dir'.
// - write the resulting kustomized manifest to dir/file of in-mem fs.
// - remove any namespaces from dir/file of in-mem fs.
// - delete objects by dir/file.
// - safely delete the removed namespaces.
func (in *Installer) kustomizeAndDelete(dir, file string) error {
	kustomizer := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := kustomizer.Run(in.fileSys, dir)
	if err != nil {
		return err
	}
	resYaml, err := resMap.AsYaml()
	if err != nil {
		return err
	}

	err = in.fileSys.WriteFile(filepath.Join(dir, file), resYaml)
	if err != nil {
		return err
	}

	removedNamespaces, err := in.omitAndReturnKindFromFSMultiDoc(filepath.Join(dir, file), "Namespace")
	if err != nil {
		return err
	}

	manifest, err := in.fileSys.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return err
	}

	err = in.kubectlClient.Delete(context.TODO(), "", string(manifest), true)
	if err != nil {
		return err
	}

	// gracefully delete removed namespaces (there is likely only one)
	for _, removedNamespace := range removedNamespaces {
		err = in.gracefullyDeleteNS(removedNamespace)
		if err != nil {
			return err
		}
	}

	return nil
}

// gracefullyDeleteNS deletes a k8s namespace only once there are no pod running in said namespace,
// then waits for the namespace to be removed from the cluster before returning no error
func (in *Installer) gracefullyDeleteNS(namespaceManifest string) error {
	namespaceName, err := pluginutils.GetFieldInManifest(namespaceManifest, "metadata", "name")
	if err != nil {
		return err
	}
	err = pluginutils.WaitFor(func() error {
		return pluginutils.NoResourcesInNS(in.clientConfig, namespaceName)
	}, 180, 10)
	if err != nil {
		return err
	}

	err = in.kubectlClient.Delete(context.TODO(), "", namespaceManifest, true)
	if err != nil {
		return err
	}

	err = pluginutils.WaitFor(func() error {
		return pluginutils.NamespaceDoesNotExist(in.clientConfig, namespaceName)
	}, 120, 5)
	if err != nil {
		return err
	}

	return nil
}

func (in *Installer) waitForCustomResourceDeletion(fn func() error) error {
	err := pluginutils.WaitFor(func() error {
		return fn()
	}, 120, 5)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}
