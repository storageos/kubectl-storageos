package installer

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"sigs.k8s.io/kustomize/api/krusty"
)

// Install performs storageos operator and etcd operator installation for kubectl-storageos
func (in *Installer) Install() error {
	var err error
	v := viper.GetViper()
	if v.GetBool(SkipEtcdFlag) {
		err = in.handleEndpointsInput(v.GetString(EtcdEndpointsFlag))
		if err != nil {
			return err
		}
	} else {
		err = in.installEtcd(v.GetString(EtcdNamespaceFlag), v.GetString(StorageClassFlag))
		if err != nil {
			return err
		}
	}

	err = in.installStorageOS(v.GetString(StosClusterNSFlag), v.GetString(StosOperatorNSFlag))
	if err != nil {
		return err
	}
	return nil
}

func (in *Installer) installEtcd(etcdNamespace, storageClass string) error {
	var err error
	// add changes to etcd kustomizations here before kustomizeAndApply calls ie make changes
	// to etcd/operator/kustomization.yaml and/or etcd/cluster/kustomization.yaml
	// based on flags (or cli config file)
	if etcdNamespace != "" {
		err = in.setFieldInFsManifest(filepath.Join(etcdDir, operatorDir, kustomizationFile), etcdNamespace, "namespace", "")
		if err != nil {
			return err
		}
		err = in.setFieldInFsManifest(filepath.Join(etcdDir, clusterDir, kustomizationFile), etcdNamespace, "namespace", "")
		if err != nil {
			return err
		}
		err = in.addPatchesToFSKustomize(filepath.Join(etcdDir, operatorDir, kustomizationFile), "Deployment", "storageos-etcd-controller-manager", []pluginutils.KustomizePatch{pluginutils.KustomizePatch{Op: "replace", Path: "/spec/template/spec/containers/0/args/1", Value: fmt.Sprintf("%s%s%s", "--proxy-url=storageos-proxy.", etcdNamespace, ".svc")}})
		if err != nil {
			return err
		}

		// update endpoint for stos cluster based on etcd namespace flag
		endpointsPatch := pluginutils.KustomizePatch{
			Op:    "replace",
			Path:  "/spec/kvBackend/address",
			Value: fmt.Sprintf("%s%s%s%s", defaultEtcdClusterNS, ".", etcdNamespace, ":2379"),
		}
		err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, defaultStosClusterName, []pluginutils.KustomizePatch{endpointsPatch})
		if err != nil {
			return err
		}
	}
	// get the cluster's default storage class if a storage class has not been provided. In any case, add patch
	// with desired storage class name to kustomization for etcd cluster
	if storageClass == "" {
		storageClass, err = pluginutils.GetDefaultStorageClassName()
		if err != nil {
			return err
		}
	}
	err = in.addPatchesToFSKustomize(filepath.Join(etcdDir, clusterDir, kustomizationFile), etcdClusterKind, defaultEtcdClusterName, []pluginutils.KustomizePatch{pluginutils.KustomizePatch{Op: "replace", Path: "/spec/storage/volumeClaimTemplate/storageClassName", Value: storageClass}})
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

func (in *Installer) installStorageOS(stosClusterNS, stosOperatorNS string) error {
	var err error
	// add changes to storageos kustomizations here before kustomizeAndApply calls ie make changes
	// to storageos/operator/kustomization.yaml and/or storageos/cluster/kustomization.yaml
	// based on flags (or cli config file)
	if stosOperatorNS != "" {
		err = in.setFieldInFsManifest(filepath.Join(stosDir, operatorDir, kustomizationFile), stosOperatorNS, "namespace", "")
		if err != nil {
			return err
		}
	}

	if stosClusterNS != "" {
		// apply the provided storageos cluster ns
		err = in.kubectlClient.Apply(context.TODO(), "", pluginutils.NamespaceYaml(stosClusterNS), true)
		if err != nil {
			return err
		}
		err = in.setFieldInFsManifest(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterNS, "namespace", "")
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
			return pluginutils.DeploymentIsReady(in.clientConfig, deploymentName, deploymentNamespace)
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

	namespaceName, err := pluginutils.GetFieldInManifest(namespaceManifest, "metadata", "name")
	if err != nil {
		return err
	}
	err = pluginutils.WaitFor(func() error {
		return pluginutils.NamespaceExists(in.clientConfig, namespaceName)
	}, 120, 5)
	if err != nil {
		return err
	}

	return nil
}
