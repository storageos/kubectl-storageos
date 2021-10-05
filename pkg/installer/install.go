package installer

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"sigs.k8s.io/kustomize/api/krusty"
)

// Install performs storageos operator and etcd operator installation for kubectl-storageos
func (in *Installer) Install(config *apiv1.KubectlStorageOSConfig, upgrade bool) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, 3)
	if config.Spec.IncludeEtcd {
		wg.Add(1)
		go func() {
			defer wg.Done()

			errChan <- in.installEtcd(config.Spec.Install)
		}()
	} else if !config.Spec.IncludeEtcd && !upgrade {
		if err := in.handleEndpointsInput(config.Spec.Install); err != nil {
			return err
		}
	}

	if serialInstall {
		wg.Wait()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		errChan <- in.installStorageOS(config)
	}()

	wg.Wait()

	if config.Spec.Install.Wait {
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

func (in *Installer) installEtcd(configInstall apiv1.Install) error {
	var err error
	// add changes to etcd kustomizations here before kustomizeAndApply calls ie make changes
	// to etcd/operator/kustomization.yaml and/or etcd/cluster/kustomization.yaml
	// based on flags (or cli config file)
	fsEtcdClusterNamespace, err := in.getFieldInFsManifest(filepath.Join(etcdDir, clusterDir, etcdClusterFile), "metadata", "namespace")
	if err != nil {
		return err
	}

	if configInstall.EtcdNamespace != fsEtcdClusterNamespace {
		if err = in.setFieldInFsManifest(filepath.Join(etcdDir, operatorDir, kustomizationFile), configInstall.EtcdNamespace, "namespace", ""); err != nil {
			return err
		}
		if err = in.setFieldInFsManifest(filepath.Join(etcdDir, clusterDir, kustomizationFile), configInstall.EtcdNamespace, "namespace", ""); err != nil {
			return err
		}
		proxyUrlPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/template/spec/containers/0/args/1",
			Value: fmt.Sprintf("%s%s%s", "--proxy-url=storageos-proxy.", configInstall.EtcdNamespace, ".svc"),
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
			Value: fmt.Sprintf("%s%s%s%s", fsEtcdClusterName, ".", configInstall.EtcdNamespace, ":2379"),
		}

		if err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{endpointsPatch}); err != nil {
			return err
		}
	}
	// get the cluster's default storage class if a storage class has not been provided. In any case, add patch
	// with desired storage class name to kustomization for etcd cluster
	if configInstall.StorageClassName == "" {
		configInstall.StorageClassName, err = pluginutils.GetDefaultStorageClassName(in.clientConfig)
		if err != nil {
			return err
		}
	}

	storageClassPatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/spec/storage/volumeClaimTemplate/storageClassName",
		Value: configInstall.StorageClassName,
	}
	if err = in.addPatchesToFSKustomize(filepath.Join(etcdDir, clusterDir, kustomizationFile), etcdClusterKind, defaultEtcdClusterName, []pluginutils.KustomizePatch{storageClassPatch}); err != nil {
		return err
	}

	if configInstall.EtcdTLSEnabled {
		tlsEnabledPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tls/enabled",
			Value: "true",
		}
		storageOSClusterNSSpecPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tls/storageOSClusterNamespace",
			Value: configInstall.StorageOSClusterNamespace,
		}
		storageOSEtcdSecretNamePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tls/storageOSEtcdSecretName",
			Value: configInstall.EtcdSecretName,
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

func (in *Installer) installStorageOS(config *apiv1.KubectlStorageOSConfig) error {
	var err error
	// add changes to storageos kustomizations here before kustomizeAndApply calls ie make changes
	// to storageos/operator/kustomization.yaml and/or storageos/cluster/kustomization.yaml
	// based on flags (or cli config file)
	fsStosOperatorNamespace, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, operatorDir, stosOperatorFile), "Deployment", "metadata", "namespace")
	if err != nil {
		return err
	}
	if config.Spec.Install.StorageOSOperatorNamespace != fsStosOperatorNamespace {
		if err = in.setFieldInFsManifest(filepath.Join(stosDir, operatorDir, kustomizationFile), config.Spec.Install.StorageOSOperatorNamespace, "namespace", ""); err != nil {
			return err
		}
	}
	fsStosClusterNamespace, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "namespace")
	if err != nil {
		return err
	}

	if config.Spec.Install.StorageOSClusterNamespace != fsStosClusterNamespace {
		// apply the provided storageos cluster ns
		if err = in.kubectlClient.Apply(context.TODO(), "", pluginutils.NamespaceYaml(config.Spec.Install.StorageOSClusterNamespace), true); err != nil {
			return err
		}
		if err = in.setFieldInFsManifest(filepath.Join(stosDir, clusterDir, kustomizationFile), config.Spec.Install.StorageOSClusterNamespace, "namespace", ""); err != nil {
			return err
		}
	}

	fsStosClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "name")
	if err != nil {
		return err
	}

	if config.Spec.Install.EtcdTLSEnabled {
		tlsEtcdSecretRefNamePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tlsEtcdSecretRefName",
			Value: config.Spec.Install.EtcdSecretName,
		}
		tlsEtcdSecretRefNamespacePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/tlsEtcdSecretRefNamespace",
			Value: config.Spec.Install.StorageOSClusterNamespace,
		}

		if err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsStosClusterName, []pluginutils.KustomizePatch{tlsEtcdSecretRefNamePatch, tlsEtcdSecretRefNamespacePatch}); err != nil {
			return err
		}
	}

	fsSecretName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), "Secret", "metadata", "name")
	if err != nil {
		return err
	}

	if config.Spec.Install.AdminUsername != "" {
		usernamePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/data/username",
			Value: config.Spec.Install.AdminUsername,
		}

		if err := in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{usernamePatch}); err != nil {
			return err
		}
	}

	if config.Spec.Install.AdminPassword != "" {
		passwordPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/data/password",
			Value: config.Spec.Install.AdminPassword,
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

	err = in.kustomizeAndApply(filepath.Join(stosDir, clusterDir), stosClusterFile)

	return err
}

// operatorDeploymentsAreReady takes the path of an operator manifest and returns no error if all
// deployments in the manifest have the desired number of ready replicas
func (in *Installer) operatorDeploymentsAreReady(path string) error {
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
		}, 90, 5); err != nil {
			return err
		}
	}
	return nil
}

// operatorServicesAreReady takes the path of an operator manifest and returns no error if all
// services in the manifest have a ClusterIP and at least one endpoint that is ready.
func (in *Installer) operatorServicesAreReady(path string) error {
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

	removedNamespaces, err := in.omitAndReturnKindFromFSMultiDoc(filepath.Join(dir, file), "Namespace")
	if err != nil {
		return err
	}
	for _, removedNamespace := range removedNamespaces {
		if err = in.gracefullyApplyNS(removedNamespace); err != nil {
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
