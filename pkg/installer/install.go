package installer

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"sigs.k8s.io/kustomize/api/krusty"
)

// Install performs storageos operator and etcd operator installation for kubectl-storageos
func (in *Installer) Install(upgrade bool) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, 3)
	if in.stosConfig.Spec.IncludeEtcd {
		wg.Add(1)
		go func() {
			defer wg.Done()

			errChan <- in.installEtcd()
		}()
	} else if !in.stosConfig.Spec.Install.DryRun && !upgrade {
		if err := in.handleEndpointsInput(in.stosConfig.Spec.Install); err != nil {
			return err
		}
	}

	if serialInstall {
		wg.Wait()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		errChan <- in.installStorageOS()
	}()

	wg.Wait()

	if in.stosConfig.Spec.Install.Wait {
		once := sync.Once{}
		errChan <- pluginutils.WaitFor(func() error {
			cluster, err := pluginutils.GetFirstStorageOSCluster(in.clientConfig)
			if err != nil {
				return err
			}

			once.Do(func() {
				fmt.Printf("waiting for %s to be ready\n", cluster.Name)
			})

			if cluster.Status.Phase != "Running" {
				return fmt.Errorf("cluster %s not ready", cluster.Name)
			}

			return nil
		}, 300, 5)
	}

	go close(errChan)

	return collectErrors(errChan)
}

func (in *Installer) installEtcd() error {
	var err error
	// add changes to etcd kustomizations here before kustomizeAndApply calls ie make changes
	// to etcd/operator/kustomization.yaml and/or etcd/cluster/kustomization.yaml
	// based on flags (or cli in.stosConfig file)
	fsEtcdClusterNamespace, err := in.getFieldInFsManifest(filepath.Join(etcdDir, clusterDir, etcdClusterFile), "metadata", "namespace")
	if err != nil {
		return err
	}

	if in.stosConfig.Spec.Install.EtcdNamespace != fsEtcdClusterNamespace {
		if err = in.setFieldInFsManifest(filepath.Join(etcdDir, operatorDir, kustomizationFile), in.stosConfig.Spec.Install.EtcdNamespace, "namespace", ""); err != nil {
			return err
		}
		if err = in.setFieldInFsManifest(filepath.Join(etcdDir, clusterDir, kustomizationFile), in.stosConfig.Spec.Install.EtcdNamespace, "namespace", ""); err != nil {
			return err
		}
		proxyUrlPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/template/spec/containers/0/args/1",
			Value: fmt.Sprintf("%s%s%s", "--proxy-url=storageos-proxy.", in.stosConfig.Spec.Install.EtcdNamespace, ".svc"),
		}
		if err = in.addPatchesToFSKustomize(filepath.Join(etcdDir, operatorDir, kustomizationFile), "Deployment", "storageos-etcd-controller-manager", []pluginutils.KustomizePatch{proxyUrlPatch}); err != nil {
			return err
		}

		fsEtcdClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(etcdDir, clusterDir, etcdClusterFile), etcdClusterKind, "metadata", "name")
		if err != nil {
			return err
		}
		fsClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "name")
		if err != nil {
			return err
		}
		// update endpoint for stos cluster based on etcd namespace flag
		endpointsPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/kvBackend/address",
			Value: fmt.Sprintf("%s%s%s%s", fsEtcdClusterName, ".", in.stosConfig.Spec.Install.EtcdNamespace, ":2379"),
		}

		if err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{endpointsPatch}); err != nil {
			return err
		}
	}
	// get the cluster's default storage class if a storage class has not been provided. In any case, add patch
	// with desired storage class name to kustomization for etcd cluster
	if in.stosConfig.Spec.Install.EtcdStorageClassName == "" {
		in.stosConfig.Spec.Install.EtcdStorageClassName, err = pluginutils.GetDefaultStorageClassName(in.clientConfig)
		if err != nil {
			return err
		}
	}

	storageClassPatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/spec/storage/volumeClaimTemplate/storageClassName",
		Value: in.stosConfig.Spec.Install.EtcdStorageClassName,
	}
	if err = in.addPatchesToFSKustomize(filepath.Join(etcdDir, clusterDir, kustomizationFile), etcdClusterKind, defaultEtcdClusterName, []pluginutils.KustomizePatch{storageClassPatch}); err != nil {
		return err
	}

	if in.stosConfig.Spec.Install.EtcdTLSEnabled {
		tlsEnabledPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tls/enabled",
			Value: "true",
		}
		storageOSClusterNSSpecPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tls/storageOSClusterNamespace",
			Value: in.stosConfig.Spec.Install.StorageOSClusterNamespace,
		}
		storageOSEtcdSecretNamePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tls/storageOSEtcdSecretName",
			Value: in.stosConfig.Spec.Install.EtcdSecretName,
		}

		if err = in.addPatchesToFSKustomize(filepath.Join(etcdDir, clusterDir, kustomizationFile), etcdClusterKind, defaultEtcdClusterName, []pluginutils.KustomizePatch{tlsEnabledPatch, storageOSClusterNSSpecPatch, storageOSEtcdSecretNamePatch}); err != nil {
			return err
		}
	}

	if err = in.kustomizeAndApply(filepath.Join(etcdDir, operatorDir), etcdOperatorFile); err != nil {
		return err
	}
	if err = in.operatorDeploymentsAreReady(filepath.Join(etcdDir, operatorDir, etcdOperatorFile)); err != nil {
		return err
	}
	err = in.kustomizeAndApply(filepath.Join(etcdDir, clusterDir), etcdClusterFile)

	return err
}

func (in *Installer) installStorageOS() error {
	var err error
	// add changes to storageos kustomizations here before kustomizeAndApply calls ie make changes
	// to storageos/operator/kustomization.yaml and/or storageos/cluster/kustomization.yaml
	// based on flags (or cli in.stosConfig file)
	fsStosOperatorNamespace, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, operatorDir, stosOperatorFile), "Deployment", "metadata", "namespace")
	if err != nil {
		return err
	}
	if in.stosConfig.Spec.Install.StorageOSOperatorNamespace != fsStosOperatorNamespace {
		if err = in.setFieldInFsManifest(filepath.Join(stosDir, operatorDir, kustomizationFile), in.stosConfig.Spec.Install.StorageOSOperatorNamespace, "namespace", ""); err != nil {
			return err
		}
	}

	if in.stosConfig.Spec.Install.StorageOSClusterNamespace != in.stosConfig.Spec.Install.StorageOSOperatorNamespace {
		// apply the provided storageos cluster ns
		if err = in.kubectlClient.Apply(context.TODO(), "", pluginutils.NamespaceYaml(in.stosConfig.Spec.Install.StorageOSClusterNamespace), true); err != nil {
			return err
		}
		if err = in.setFieldInFsManifest(filepath.Join(stosDir, clusterDir, kustomizationFile), in.stosConfig.Spec.Install.StorageOSClusterNamespace, "namespace", ""); err != nil {
			return err
		}
	}

	fsStosClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "name")
	if err != nil {
		return err
	}

	if in.stosConfig.Spec.Install.EtcdTLSEnabled {
		tlsEtcdSecretRefNamePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tlsEtcdSecretRefName",
			Value: in.stosConfig.Spec.Install.EtcdSecretName,
		}
		tlsEtcdSecretRefNamespacePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tlsEtcdSecretRefNamespace",
			Value: in.stosConfig.Spec.Install.StorageOSClusterNamespace,
		}

		if err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsStosClusterName, []pluginutils.KustomizePatch{tlsEtcdSecretRefNamePatch, tlsEtcdSecretRefNamespacePatch}); err != nil {
			return err
		}
	}

	fsSecretName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), "Secret", "metadata", "name")
	if err != nil {
		return err
	}

	if in.stosConfig.Spec.Install.AdminUsername != "" {
		usernamePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/data/username",
			Value: base64.StdEncoding.EncodeToString([]byte(in.stosConfig.Spec.Install.AdminUsername)),
		}

		if err := in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{usernamePatch}); err != nil {
			return err
		}
	}

	if in.stosConfig.Spec.Install.AdminPassword != "" {
		passwordPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/data/password",
			Value: base64.StdEncoding.EncodeToString([]byte(in.stosConfig.Spec.Install.AdminPassword)),
		}

		if err := in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{passwordPatch}); err != nil {
			return err
		}
	}
	if err = in.kustomizeAndApply(filepath.Join(stosDir, operatorDir), stosOperatorFile); err != nil {
		return err
	}
	if err = in.operatorDeploymentsAreReady(filepath.Join(stosDir, operatorDir, stosOperatorFile)); err != nil {
		return err
	}
	if err = in.operatorServicesAreReady(filepath.Join(stosDir, operatorDir, stosOperatorFile)); err != nil {
		return err
	}

	if in.distribution == pluginutils.DistributionGKE {
		fsResourceQuotaName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, resourceQuotaDir, resourceQuotaFile), resourceQuotaKind, "metadata", "name")
		if err != nil {
			return err
		}

		clusterNamespacePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/metadata/namespace",
			Value: in.stosConfig.Spec.Install.StorageOSClusterNamespace,
		}

		if err := in.addPatchesToFSKustomize(filepath.Join(stosDir, resourceQuotaDir, kustomizationFile), resourceQuotaKind, fsResourceQuotaName, []pluginutils.KustomizePatch{clusterNamespacePatch}); err != nil {
			return err
		}

		if err = in.kustomizeAndApply(filepath.Join(stosDir, resourceQuotaDir), resourceQuotaFile); err != nil {
			return err
		}
	}

	if in.stosConfig.Spec.Install.EnablePortalManager {
		if err := in.installPortalManagerClient(in.stosConfig.Spec.Install.StorageOSClusterNamespace); err != nil {
			return err
		}
		if err := in.installPortalManagerConfig(in.stosConfig.Spec.Install.StorageOSClusterNamespace); err != nil {
			return err
		}
		if err := in.enablePortalManager(fsStosClusterName, true); err != nil {
			return err
		}
	}

	if in.stosConfig.Spec.SkipStorageOSCluster {
		return nil
	}

	return in.kustomizeAndApply(filepath.Join(stosDir, clusterDir), stosClusterFile)
}

// operatorDeploymentsAreReady takes the path of an operator manifest and returns no error if all
// deployments in the manifest have the desired number of ready replicas
func (in *Installer) operatorDeploymentsAreReady(path string) error {
	// return early for dry-run
	if in.stosConfig.Spec.Install.DryRun {
		return nil
	}
	operatorDeployments, err := in.getAllManifestsOfKindFromFsMultiDoc(path, "Deployment")
	if err != nil {
		return err
	}

	for _, deployment := range operatorDeployments {
		deploymentName, err := pluginutils.GetFieldInManifest(deployment, "metadata", "name")
		if err != nil {
			return err
		}
		deploymentNamespace, err := pluginutils.GetFieldInManifest(deployment, "metadata", "namespace")
		if err != nil {
			return err
		}
		if err = pluginutils.WaitFor(func() error {
			return pluginutils.IsDeploymentReady(in.clientConfig, deploymentName, deploymentNamespace)
		}, 120, 5); err != nil {
			return err
		}
	}
	return nil
}

// operatorServicesAreReady takes the path of an operator manifest and returns no error if all
// services in the manifest have a ClusterIP and at least one endpoint that is ready.
func (in *Installer) operatorServicesAreReady(path string) error {
	// return early for dry-run
	if in.stosConfig.Spec.Install.DryRun {
		return nil
	}
	operatorServices, err := in.getAllManifestsOfKindFromFsMultiDoc(path, "Service")
	if err != nil {
		return err
	}

	for _, service := range operatorServices {
		serviceName, err := pluginutils.GetFieldInManifest(service, "metadata", "name")
		if err != nil {
			return err
		}
		serviceNamespace, err := pluginutils.GetFieldInManifest(service, "metadata", "namespace")
		if err != nil {
			return err
		}
		if err = pluginutils.WaitFor(func() error {
			return pluginutils.IsServiceReady(in.clientConfig, serviceName, serviceNamespace)
		}, 90, 5); err != nil {
			return err
		}
	}
	return nil
}

// kustomizeAndApply performs the following in the order described:
// - kustomize run (build) on the provided 'dir'.
// - write the resulting kustomized manifest to dir/file of in-mem fs.
// - remove any namespaces from dir/file of in-mem fs.
// - safely apply the removed namespaces.
// - apply dir/file (once removed namespaces have been applied  successfully).
func (in *Installer) kustomizeAndApply(dir, file string) error {
	kustomizer := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := kustomizer.Run(in.fileSys, dir)
	if err != nil {
		return err
	}
	resYaml, err := resMap.AsYaml()
	if err != nil {
		return err
	}
	if err = in.fileSys.WriteFile(filepath.Join(dir, file), resYaml); err != nil {
		return err
	}

	if in.stosConfig.Spec.Install.DryRun {
		if err := writeDryRunManifests(file, resYaml); err != nil {
			return err
		}
		// return early for dry-run without applying manifest
		return nil
	}

	namespaces, err := in.omitAndReturnKindFromFSMultiDoc(filepath.Join(dir, file), "Namespace")
	if err != nil {
		return err
	}
	for _, namespace := range namespaces {
		if err = in.gracefullyApplyNS(namespace); err != nil {
			return err
		}
	}

	manifest, err := in.fileSys.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return err
	}

	err = in.kubectlClient.Apply(context.TODO(), "", string(manifest), true)

	return err
}

// gracefullyApplyNS applies a namespace and then waits until it has been applied successfully before
// returning no error
func (in *Installer) gracefullyApplyNS(namespaceManifest string) error {
	if err := in.kubectlClient.Apply(context.TODO(), "", namespaceManifest, true); err != nil {
		return err
	}

	namespace, err := pluginutils.GetFieldInManifest(namespaceManifest, "metadata", "name")
	if err != nil {
		return err
	}
	err = pluginutils.WaitFor(func() error {
		return pluginutils.NamespaceExists(in.clientConfig, namespace)
	}, 120, 5)

	return err
}

func writeDryRunManifests(filename string, fileData []byte) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err = os.Stat(filepath.Join(cwd, stosDryRunDir)); err != nil {
		if err = os.Mkdir(filepath.Join(cwd, stosDryRunDir), 0770); err != nil {
			return errors.WithStack(err)
		}
	}
	if err = os.WriteFile(filepath.Join(cwd, stosDryRunDir, filename), fileData, 0640); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
