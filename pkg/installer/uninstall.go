package installer

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	operatorapi "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"github.com/storageos/kubectl-storageos/pkg/version"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/krusty"
)

const (
	skipNamespaceDeletionMessage = `Namespace %s still has resources.
	Skipped namespace removal.
	Reason: %s
	Check for resources remaining in the namespace with:
	kubectl get all -n %s
	To remove the namespace and all remaining resources within it, run:
	kubectl delete namespace %s`

	errEtcdUninstallAborted = `
	ETCD uninstall aborted`

	errStosUninstallAborted = `
	StorageOS uninstall aborted`

	errPVCsExist = `
	Discovered bound PVC [%s/%s] provisioned by StorageOS storageclass provisioner [` + stosSCProvisioner + `].
	No PVCs should be bound to StorageOS volumes before uninstalling ETCD.
	Re-run with --skip-existing-workload-check to ignore.`

	errWorkloadsExist = `
	Discovered workload [%s/%s] using PVC provisioned by StorageOS storageclass provisioner [` + stosSCProvisioner + `].
	All workloads that rely on StorageOS volumes should be stopped before uninstalling StorageOS.
	Re-run with --skip-existing-workload-check to ignore.`

	removingFinalizersMessage = `Attempting to remove any existing finalizers from object [%s] to allow object deletion.`

	errDuringStosUninstall = `
	An error has occurred during StorageOS uninstallation. Please delete StorageOS components manually.`

	errDuringEtcdUninstall = `
	An error has occurred during Etcd uninstallation. Please delete Etcd components manually.`
)

var protectedNamespaces = map[string]bool{
	"kube-system": true,
}

// Uninstall performs storageos and etcd uninstallation for kubectl-storageos. Bool 'upgrade'
// indicates whether or not this uninstallation is part of an upgrade.
func (in *Installer) Uninstall(upgrade bool, currentVersion string) error {
	stosPVCs := &corev1.PersistentVolumeClaimList{}
	var err error
	if !in.stosConfig.Spec.SkipExistingWorkloadCheck {
		stosPVCs, err = in.storageOSPVCs()
		if err != nil {
			return errors.Wrap(err, errStosUninstallAborted)
		}
		if err := in.storageOSWorkloadsExist(stosPVCs); err != nil {
			return errors.Wrap(err, errStosUninstallAborted)
		}
	}

	wg := sync.WaitGroup{}
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()

		errChan <- in.uninstallStorageOS(upgrade, currentVersion)
	}()

	if serialInstall {
		wg.Wait()
	}

	if in.stosConfig.Spec.IncludeEtcd {
		if !in.stosConfig.Spec.SkipExistingWorkloadCheck && len(stosPVCs.Items) > 0 {
			return errors.Wrap(fmt.Errorf(errPVCsExist, stosPVCs.Items[0].Namespace, stosPVCs.Items[0].Name), errEtcdUninstallAborted)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			errChan <- in.uninstallEtcd()
		}()
	}

	wg.Wait()
	go close(errChan)

	return collectErrors(errChan)
}

func (in *Installer) uninstallStorageOS(upgrade bool, currentVersion string) error {
	storageOSCluster, err := pluginutils.GetFirstStorageOSCluster(in.clientConfig)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	}

	if !upgrade && in.distribution == pluginutils.DistributionGKE {
		lessThanOrEqual, err := version.VersionIsLessThanOrEqual(currentVersion, version.ClusterOperatorLastVersion())
		if err != nil {
			return err
		}
		if !lessThanOrEqual {
			if err = in.uninstallResourceQuota(storageOSCluster); err != nil {
				return err
			}
		}
	}

	if storageOSCluster.Name != "" {
		if err := in.uninstallStorageOSCluster(storageOSCluster, upgrade); err != nil {
			return err
		}
		if err := in.ensureStorageOSClusterRemoved(); err != nil {
			return errors.WithStack(err)
		}
	}

	// StorageOS cluster resources should be in a different namespace, on that case need to delete
	if storageOSCluster.Namespace != in.stosConfig.Spec.Uninstall.StorageOSOperatorNamespace {
		if err = in.gracefullyDeleteNS(storageOSCluster.Namespace); err != nil {
			return err
		}
	}

	err = in.uninstallStorageOSOperator()

	return err
}

func (in *Installer) uninstallStorageOSCluster(storageOSCluster *operatorapi.StorageOSCluster, upgrade bool) error {
	// make changes to storageos/cluster/kustomization.yaml based on flags (or cli config file) before
	// kustomizeAndDelete call
	fsClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "name")
	if err != nil {
		return err
	}

	clusterNamePatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/metadata/name",
		Value: storageOSCluster.Name,
	}

	if err := in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{clusterNamePatch}); err != nil {
		return err
	}

	clusterNamespacePatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/metadata/namespace",
		Value: storageOSCluster.Namespace,
	}

	if err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{clusterNamespacePatch}); err != nil {
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

	if err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{secretNamePatch}); err != nil {
		return err
	}

	secretNamespacePatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/metadata/namespace",
		Value: storageOSCluster.Spec.SecretRefNamespace,
	}

	if err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{secretNamespacePatch}); err != nil {
		return err
	}

	// if this is not an upgrade, write manifests to disk before deletion
	if !upgrade {
		if err = in.writeBackupFileSystem(storageOSCluster); err != nil {
			return errors.WithStack(err)
		}
	}

	if !in.stosConfig.Spec.SkipNamespaceDeletion {
		if storageOSCluster.Namespace == in.stosConfig.Spec.Uninstall.StorageOSOperatorNamespace {
			// postpone namespace deletion as storageos cluster and operator are in the same namespace
			// the namespace will be deleted later by storageos operator uninstallation
			err := in.postponeNamespaceKustomizeAndDelete(filepath.Join(stosDir, clusterDir), stosClusterFile)
			return err
		}
	}

	if err = in.kustomizeAndDelete(filepath.Join(stosDir, clusterDir), stosClusterFile); err != nil {
		return err
	}

	return err
}

func (in *Installer) uninstallResourceQuota(storageOSCluster *operatorapi.StorageOSCluster) error {
	// make changes to storageos/resource-quota/kustomization.yaml based on flags (or cli config file) before
	// kustomizeAndDelete call
	fsResourceQuotaName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, resourceQuotaDir, resourceQuotaFile), resourceQuotaKind, "metadata", "name")
	if err != nil {
		return err
	}

	clusterNamespacePatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/metadata/namespace",
		Value: storageOSCluster.Namespace,
	}

	if err := in.addPatchesToFSKustomize(filepath.Join(stosDir, resourceQuotaDir, kustomizationFile), resourceQuotaKind, fsResourceQuotaName, []pluginutils.KustomizePatch{clusterNamespacePatch}); err != nil {
		return err
	}

	return in.kustomizeAndDelete(filepath.Join(stosDir, resourceQuotaDir), resourceQuotaFile)
}

func (in *Installer) uninstallStorageOSOperator() error {
	// make changes to storageos/operator/kustomization.yaml based on flags (or cli config file) before
	// kustomizeAndDelete call
	if err := in.setFieldInFsManifest(filepath.Join(stosDir, operatorDir, kustomizationFile), in.stosConfig.Spec.Uninstall.StorageOSOperatorNamespace, "namespace", ""); err != nil {
		return err
	}

	err := in.kustomizeAndDelete(filepath.Join(stosDir, operatorDir), stosOperatorFile)

	return err
}

func (in *Installer) uninstallEtcd() error {
	fsEtcdName, err := in.getFieldInFsMultiDocByKind(filepath.Join(etcdDir, clusterDir, etcdClusterFile), etcdClusterKind, "metadata", "name")
	if err != nil {
		return err
	}
	etcdCluster, err := pluginutils.GetEtcdCluster(in.clientConfig, fsEtcdName, in.stosConfig.Spec.Uninstall.EtcdNamespace)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	}
	if etcdCluster.Name != "" {
		if err := in.uninstallEtcdCluster(); err != nil {
			return err
		}
		if err := in.ensureEtcdClusterRemoved(fsEtcdName); err != nil {
			return err
		}
	}
	err = in.uninstallEtcdOperator()

	return err
}

func (in *Installer) uninstallEtcdCluster() error {
	// make changes etcd/cluster/kustomization.yaml based on flags (or cli config file) before
	//kustomizeAndDelete call
	if err := in.setFieldInFsManifest(filepath.Join(etcdDir, clusterDir, kustomizationFile), in.stosConfig.Spec.Uninstall.EtcdNamespace, "namespace", ""); err != nil {
		return err
	}

	if !in.stosConfig.Spec.SkipNamespaceDeletion {
		// postpone namespace deletion as etcd cluster and operator are in the same namespace
		// the namespace will be deleted later by etcd operator uninstallation
		err := in.postponeNamespaceKustomizeAndDelete(filepath.Join(etcdDir, clusterDir), etcdClusterFile)
		return err
	}
	err := in.kustomizeAndDelete(filepath.Join(etcdDir, clusterDir), etcdClusterFile)

	return err
}

func (in *Installer) uninstallEtcdOperator() error {
	// make changes etcd/operator/kustomization.yaml based on flags (or cli config file) before
	//kustomizeAndDelete call
	if err := in.setFieldInFsManifest(filepath.Join(etcdDir, operatorDir, kustomizationFile), in.stosConfig.Spec.Uninstall.EtcdNamespace, "namespace", ""); err != nil {
		return err
	}

	err := in.kustomizeAndDelete(filepath.Join(etcdDir, operatorDir), etcdOperatorFile)

	return err
}

// storageOSPVCs returns a PersistenVolumeClaimList of bound PVCs provisioned by storageos.
func (in *Installer) storageOSPVCs() (*corev1.PersistentVolumeClaimList, error) {
	stosPVCs := &corev1.PersistentVolumeClaimList{}
	pvcList, err := pluginutils.ListPersistentVolumeClaims(in.clientConfig, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pvc := range pvcList.Items {
		if pvc.Status.Phase != corev1.ClaimBound {
			continue
		}
		isStosPVC, err := pluginutils.IsProvisionedPVC(in.clientConfig, &pvc, stosSCProvisioner)
		if err != nil {
			return nil, err
		}
		if isStosPVC {
			stosPVCs.Items = append(stosPVCs.Items, pvc)
		}
	}

	return stosPVCs, nil
}

// storageOSWorkloadsExist return error if a pod is discovered using a storageos pvc.
func (in *Installer) storageOSWorkloadsExist(stosPVCs *corev1.PersistentVolumeClaimList) error {
	pods, err := pluginutils.ListPods(in.clientConfig, "", "")
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		for _, stosPVC := range stosPVCs.Items {
			if pluginutils.PodHasPVC(&pod, stosPVC.Name) {
				return fmt.Errorf(errWorkloadsExist, pod.Namespace, pod.Name)
			}
		}
	}

	return nil
}

// kustomizeAndDelete performs the following in the order described:
// - kustomize run (build) on the provided 'dir'.
// - write the resulting kustomized manifest to dir/file of in-mem fs.
// - remove any namespaces from dir/file of in-mem fs.
// - delete objects by dir/file.
// - safely delete the removed namespaces and returns them.
func (in *Installer) kustomizeAndDelete(dir, file string) error {
	kustomizer := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := kustomizer.Run(in.fileSys, dir)
	if err != nil {
		return errors.WithStack(err)
	}
	resYaml, err := resMap.AsYaml()
	if err != nil {
		return errors.WithStack(err)
	}

	if err = in.fileSys.WriteFile(filepath.Join(dir, file), resYaml); err != nil {
		return errors.WithStack(err)
	}

	removedNamespaces, err := in.omitAndReturnKindFromFSMultiDoc(filepath.Join(dir, file), "Namespace")
	if err != nil {
		return err
	}

	manifest, err := in.fileSys.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return errors.WithStack(err)
	}

	if err = in.kubectlClient.Delete(context.TODO(), "", string(manifest), true); err != nil {
		return errors.WithStack(err)
	}

	if in.stosConfig.Spec.SkipNamespaceDeletion {
		return nil
	}

	// gracefully delete removed namespaces (there is likely only one)
	for _, removedNamespace := range removedNamespaces {
		namespace, err := pluginutils.GetFieldInManifest(removedNamespace, "metadata", "name")
		if err != nil {
			return err
		}

		if err = in.gracefullyDeleteNS(namespace); err != nil {
			return err
		}
	}

	return nil
}

// postponeNamespaceKustomizeAndDelete sets SkipNamespaceDeletion to true, performs kustomizeAndDelete
// before resetting SkipNamespaceDeletion to original value.
func (in *Installer) postponeNamespaceKustomizeAndDelete(dir, file string) error {
	skipNamespaceDeletion := in.stosConfig.Spec.SkipNamespaceDeletion
	in.stosConfig.Spec.SkipNamespaceDeletion = true
	defer func() {
		in.restoreSkipNamespaceDeletion(skipNamespaceDeletion)
	}()
	err := in.kustomizeAndDelete(dir, file)
	return err
}

// restoreSkipNamespaceDeletion is a helper function deferred by postponeNSKustomizeAndDelete
func (in *Installer) restoreSkipNamespaceDeletion(skipNamespaceDeletion bool) {
	in.stosConfig.Spec.SkipNamespaceDeletion = skipNamespaceDeletion
}

// gracefullyDeleteNS deletes a k8s namespace only once there are no resources running in said namespace,
// then waits for the namespace to be removed from the cluster before returning no error
func (in *Installer) gracefullyDeleteNS(namespace string) error {
	if _, ok := protectedNamespaces[namespace]; ok || in.stosConfig.Spec.SkipNamespaceDeletion {
		return nil
	}

	if err := pluginutils.DeleteNamespace(in.clientConfig, namespace); err != nil {
		return err
	}

	if err := pluginutils.WaitFor(func() error {
		return pluginutils.NamespaceDoesNotExist(in.clientConfig, namespace)
	}, 120, 5); err != nil {
		parentErr := errors.Unwrap(err)
		if _, ok := parentErr.(pluginutils.ResourcesStillExists); !ok {
			return err
		}

		println(fmt.Sprintf(skipNamespaceDeletionMessage, namespace, err.Error(), namespace, namespace))
	}

	return nil
}

// ensureStorageOSClusterDeletion returns no error if storageoscluster has been removed from k8s cluster.
func (in *Installer) ensureStorageOSClusterRemoved() error {
	// allow storageoscluster object to be deleted before continuing uninstall process
	if err := in.waitForCustomResourceDeletion(func() error {
		return pluginutils.StorageOSClusterDoesNotExist(in.clientConfig)
	}); err == nil {
		return nil
	}
	// storageoscluster still exists at this point, it may be stuck in deleting phase with finalizer. So we
	// rediscover the object, remove any finlaizers and update (known issue on k8s 1.18)
	storageOSCluster, err := pluginutils.GetFirstStorageOSCluster(in.clientConfig)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(errors.WithStack(err), errDuringStosUninstall)
	}
	fmt.Println(fmt.Sprintf(removingFinalizersMessage, storageOSCluster.Name))
	if err := pluginutils.UpdateStorageOSClusterWithoutFinalizers(in.clientConfig, storageOSCluster); err != nil {
		return errors.Wrap(errors.WithStack(err), errDuringStosUninstall)
	}
	// once again, wait to see if object is deleted.
	if err = in.waitForCustomResourceDeletion(func() error {
		return pluginutils.StorageOSClusterDoesNotExist(in.clientConfig)
	}); err != nil {
		return errors.Wrap(errors.WithStack(err), errDuringEtcdUninstall)
	}

	return nil
}

// ensureEtcdClusterRemoved returns no error if etcdcluster has been removed from k8s cluster.
func (in *Installer) ensureEtcdClusterRemoved(etcdName string) error {
	// allow etcdcluster object to be deleted before continuing uninstall process
	if err := in.waitForCustomResourceDeletion(func() error {
		return pluginutils.EtcdClusterDoesNotExist(in.clientConfig, etcdName, in.stosConfig.Spec.Uninstall.EtcdNamespace)
	}); err == nil {
		return nil
	}
	// etcdcluster still exists at this point, it may be stuck in deleting phase with finalizer. So we
	// rediscover the object, remove any finlaizers and update (known issue on k8s 1.18)
	etcdCluster, err := pluginutils.GetEtcdCluster(in.clientConfig, etcdName, in.stosConfig.Spec.Uninstall.EtcdNamespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(errors.WithStack(err), errDuringEtcdUninstall)
	}
	fmt.Println(fmt.Sprintf(removingFinalizersMessage, etcdCluster.Name))
	if err := pluginutils.UpdateEtcdClusterWithoutFinalizers(in.clientConfig, etcdCluster); err != nil {
		return errors.Wrap(errors.WithStack(err), errDuringEtcdUninstall)
	}
	// once again, wait to see if object is deleted.
	if err = in.waitForCustomResourceDeletion(func() error {
		return pluginutils.EtcdClusterDoesNotExist(in.clientConfig, etcdName, in.stosConfig.Spec.Uninstall.EtcdNamespace)
	}); err != nil {
		return errors.Wrap(errors.WithStack(err), errDuringEtcdUninstall)
	}

	return nil
}

func (in *Installer) waitForCustomResourceDeletion(fn func() error) error {
	if err := pluginutils.WaitFor(func() error {
		return fn()
	}, 45, 5); err != nil {
		return errors.Wrap(err, "timeout waiting for custom resource deletion during uninstall")
	}
	return nil
}
