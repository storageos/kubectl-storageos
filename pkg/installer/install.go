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
func (in *Installer) Install(config *apiv1.KubectlStorageOSConfig) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, 2)
	if config.Spec.IncludeEtcd {
		wg.Add(1)
		go func() {
			defer wg.Done()

			errChan <- in.installEtcd(config.Spec.Install)
		}()
	} else {
		if err := in.handleEndpointsInput(config.Spec.Install.EtcdEndpoints); err != nil {
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
	go close(errChan)

	return collectErrors(errChan)
}

func (in *Installer) installEtcd(configInstall apiv1.Install) error {
	var err error
	// add changes to etcd kustomizations here before kustomizeAndApply calls ie make changes
	// to etcd/operator/kustomization.yaml and/or etcd/cluster/kustomization.yaml
	// based on flags (or cli config file)
	if configInstall.EtcdNamespace != "" {
		err = in.setFieldInFsManifest(filepath.Join(etcdDir, operatorDir, kustomizationFile), configInstall.EtcdNamespace, "namespace", "")
		if err != nil {
			return err
		}
		err = in.setFieldInFsManifest(filepath.Join(etcdDir, clusterDir, kustomizationFile), configInstall.EtcdNamespace, "namespace", "")
		if err != nil {
			return err
		}
		proxyUrlPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/template/spec/containers/0/args/1",
			Value: fmt.Sprintf("%s%s%s", "--proxy-url=storageos-proxy.", configInstall.EtcdNamespace, ".svc"),
		}
		err = in.addPatchesToFSKustomize(filepath.Join(etcdDir, operatorDir, kustomizationFile), "Deployment", "storageos-etcd-controller-manager", []pluginutils.KustomizePatch{proxyUrlPatch})
		if err != nil {
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

		err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{endpointsPatch})
		if err != nil {
			return err
		}
	}
	// get the cluster's default storage class if a storage class has not been provided. In any case, add patch
	// with desired storage class name to kustomization for etcd cluster
	if configInstall.StorageClassName == "" {
		configInstall.StorageClassName, err = pluginutils.GetDefaultStorageClassName()
		if err != nil {
			return err
		}
	}

	storageClassPatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/spec/storage/volumeClaimTemplate/storageClassName",
		Value: configInstall.StorageClassName,
	}
	err = in.addPatchesToFSKustomize(filepath.Join(etcdDir, clusterDir, kustomizationFile), etcdClusterKind, defaultEtcdClusterName, []pluginutils.KustomizePatch{storageClassPatch})
	if err != nil {
		return err
	}

	err = in.kustomizeAndApply(filepath.Join(etcdDir, operatorDir), etcdOperatorFile)
	if err != nil {
		return err
	}
	err = in.operatorDeploymentsAreReady(filepath.Join(etcdDir, operatorDir, etcdOperatorFile))
	if err != nil {
		return err
	}
	err = in.kustomizeAndApply(filepath.Join(etcdDir, clusterDir), etcdClusterFile)
	if err != nil {
		return err
	}

	return nil
}

func (in *Installer) installStorageOS(config *apiv1.KubectlStorageOSConfig) error {
	var err error
	// add changes to storageos kustomizations here before kustomizeAndApply calls ie make changes
	// to storageos/operator/kustomization.yaml and/or storageos/cluster/kustomization.yaml
	// based on flags (or cli config file)
	if config.Spec.Install.StorageOSOperatorNamespace != "" {
		err = in.setFieldInFsManifest(filepath.Join(stosDir, operatorDir, kustomizationFile), config.Spec.Install.StorageOSOperatorNamespace, "namespace", "")
		if err != nil {
			return err
		}
	}

	if config.Spec.Install.StorageOSClusterNamespace != "" {
		// apply the provided storageos cluster ns
		err = in.kubectlClient.Apply(context.TODO(), "", pluginutils.NamespaceYaml(config.Spec.Install.StorageOSClusterNamespace), true)
		if err != nil {
			return err
		}
		err = in.setFieldInFsManifest(filepath.Join(stosDir, clusterDir, kustomizationFile), config.Spec.Install.StorageOSClusterNamespace, "namespace", "")
		if err != nil {
			return err
		}
	}

	fsSecretName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), "Secret", "metadata", "name")
	if err != nil {
		return err
	}

	if config.InstallerMeta.SecretUsername != "" {
		usernamePatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/data/username",
			Value: config.InstallerMeta.SecretUsername,
		}

		err := in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{usernamePatch})
		if err != nil {
			return err
		}

	}

	if config.InstallerMeta.SecretPassword != "" {
		passwordPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/data/password",
			Value: config.InstallerMeta.SecretPassword,
		}

		err := in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), "Secret", fsSecretName, []pluginutils.KustomizePatch{passwordPatch})
		if err != nil {
			return err
		}
	}

	err = in.kustomizeAndApply(filepath.Join(stosDir, operatorDir), stosOperatorFile)
	if err != nil {
		return err
	}
	err = in.operatorDeploymentsAreReady(filepath.Join(stosDir, operatorDir, stosOperatorFile))
	if err != nil {
		return err
	}
	err = in.kustomizeAndApply(filepath.Join(stosDir, clusterDir), stosClusterFile)
	if err != nil {
		return err
	}

	return nil
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
		err = pluginutils.WaitFor(func() error {
			return pluginutils.IsDeploymentReady(in.clientConfig, deploymentName, deploymentNamespace)
		}, 90, 5)
		if err != nil {
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
	err = in.fileSys.WriteFile(filepath.Join(dir, file), resYaml)
	if err != nil {
		return err
	}

	removedNamespaces, err := in.omitAndReturnKindFromFSMultiDoc(filepath.Join(dir, file), "Namespace")
	if err != nil {
		return err
	}
	for _, removedNamespace := range removedNamespaces {
		err = in.gracefullyApplyNS(removedNamespace)
		if err != nil {
			return err
		}
	}

	manifest, err := in.fileSys.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return err
	}

	err = in.kubectlClient.Apply(context.TODO(), "", string(manifest), true)
	if err != nil {
		return err
	}

	return nil
}

// gracefullyApplyNS applies a namespace and then waits until it has been applied succesfully before
// returning no error
func (in *Installer) gracefullyApplyNS(namespaceManifest string) error {
	err := in.kubectlClient.Apply(context.TODO(), "", namespaceManifest, true)
	if err != nil {
		return err
	}

	namespace, err := pluginutils.GetFieldInManifest(namespaceManifest, "metadata", "name")
	if err != nil {
		return err
	}
	err = pluginutils.WaitFor(func() error {
		return pluginutils.NamespaceExists(in.clientConfig, namespace)
	}, 120, 5)
	if err != nil {
		return err
	}

	return nil
}
