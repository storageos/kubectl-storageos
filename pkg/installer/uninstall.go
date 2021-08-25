package installer

import (
	"context"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"sigs.k8s.io/kustomize/api/krusty"
)

// Uninstall performs storageos and etcd uninstallation for kubectl-storageos
func (in *Installer) Uninstall() error {
	v := viper.GetViper()
	err := in.uninstallStorageOS(v.GetString(StosOperatorNSFlag), v.GetString(StosClusterNSFlag))
	if err != nil {
		return err
	}

	// return early if user only wishes to delete storageos, leaving etcd untouched
	if v.GetBool(SkipEtcdFlag) {
		return nil
	}
	err = in.uninstallEtcd(v.GetString(EtcdNamespaceFlag))
	if err != nil {
		return err
	}

	return nil
}

func (in *Installer) uninstallStorageOS(stosOperatorNS, stosClusterNS string) error {
	var err error
	// add changes to storageos kustomizations here before kustomizeAndDelete calls ie make changes
	// to storageos/operator/kustomization.yaml and/or storageos/cluster/kustomization.yaml
	// based on flags (or cli config file)
	if stosOperatorNS != "" {
		err := in.setFieldInFsManifest(filepath.Join(stosDir, operatorDir, kustomizationFile), stosOperatorNS, "namespace", "")
		if err != nil {
			return err
		}

	} else {
		stosOperatorNS = defaultStosOperatorNS
	}

	if stosClusterNS != "" {
		err = in.setFieldInFsManifest(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterNS, "namespace", "")
		if err != nil {
			return err
		}
	}

	err = in.kustomizeAndDelete(filepath.Join(stosDir, clusterDir), stosClusterFile)
	if err != nil {
		return err
	}
	// sleep to allow operator to terminate cluster's child objects
	// TODO: Add specific check instead of sleep
	time.Sleep(5 * time.Second)

	if stosClusterNS != "" {
		err = in.gracefullyDeleteNS(pluginutils.NamespaceYaml(stosClusterNS))
		if err != nil {
			return err
		}
	}

	err = in.kustomizeAndDelete(filepath.Join(stosDir, operatorDir), stosOperatorFile)
	if err != nil {
		return err
	}

	return nil
}

func (in *Installer) uninstallEtcd(etcdNamespace string) error {
	var err error
	// add changes to etcd kustomizations here before kustomizeAndDelete calls ie make changes
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

	} else {
		etcdNamespace = defaultEtcdClusterNS
	}

	err = in.kustomizeAndDelete(filepath.Join(etcdDir, clusterDir), etcdClusterFile)
	if err != nil {
		return err
	}
	// sleep to allow operator to terminate cluster's child objects
	// TODO: Add specific check instead of sleep
	time.Sleep(5 * time.Second)

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
// - delete dir/file.
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
		return pluginutils.NoPodsInNS(in.clientConfig, namespaceName)
	}, 120, 5)
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
