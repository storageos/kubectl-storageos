package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gyaml "github.com/ghodss/yaml"
	"github.com/replicatedhq/troubleshoot/cmd/util"
	operatorapi "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
	corev1 "k8s.io/api/core/v1"
	kstoragev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kustomize/api/filesys"
)

// buildInstallerFileSys builds an in-memory filesystem for installer with relevant storageos and
// etcd manifests. If '--skip-etcd-install' flag is set, etcd dir is not created.
// - storageos
//   - operator
//     - storageos-operator.yaml
//     - kustomization.yaml
//   - cluster
//     - storageos-cluster.yaml
//     - kustomization.yaml
// - etcd
//   - operator
//     - etcd-operator.yaml
//     - kustomization.yaml
//   - cluster
//     - etcd-cluster.yaml
//     - kustomization.yaml
func buildInstallerFileSys(config *apiv1.KubectlStorageOSConfig, clientConfig *rest.Config) (filesys.FileSystem, error) {
	fs := filesys.MakeFsInMemory()
	fsData := make(fsData)
	stosSubDirs := make(map[string]map[string][]byte)

	// build storageos/operator
	stosOpFiles, err := createFileData(config.Spec.Install.StorageOSOperatorYaml, pluginversion.OperatorLatestSupportedURL(), stosOperatorFile, clientConfig, config.Spec.GetNamespace())
	if err != nil {
		return fs, err
	}
	stosSubDirs[operatorDir] = stosOpFiles

	// build storageos/cluster
	stosClusterFiles, err := createFileData(config.Spec.Install.StorageOSClusterYaml, pluginversion.ClusterLatestSupportedURL(), stosClusterFile, clientConfig, config.Spec.GetNamespace())
	if err != nil {
		return fs, err
	}

	// append storageos secret yaml to cluster yaml if necessary. This will happen in the event of an
	// uninstall of storageos version < 2.5.0.
	if config.InstallerMeta.StorageOSSecretYaml != "" {
		stosSecretYaml, err := pullManifest(config.InstallerMeta.StorageOSSecretYaml)
		if err != nil {
			return fs, err
		}
		stosClusterMulti := makeMultiDoc(string(stosClusterFiles[stosClusterFile]), stosSecretYaml)
		stosClusterFiles[stosClusterFile] = []byte(stosClusterMulti)
	}
	stosSubDirs[clusterDir] = stosClusterFiles
	fsData[stosDir] = stosSubDirs

	// if skip-etcd-install flag is set, create fs with storageos files and return early
	if config.Spec.Install.SkipEtcd {
		fs, err = createDirAndFiles(fs, fsData)
		if err != nil {
			return fs, err
		}
		return fs, nil
	}

	etcdSubDirs := make(map[string]map[string][]byte)

	// build etcd/operator
	etcdOpFiles, err := createFileData(config.Spec.Install.EtcdOperatorYaml, pluginversion.EtcdOperatorLatestSupportedURL(), etcdOperatorFile, clientConfig, config.Spec.GetNamespace())
	if err != nil {
		return fs, err
	}
	etcdSubDirs[operatorDir] = etcdOpFiles

	// build etcd/cluster
	etcdClusterFiles, err := createFileData(config.Spec.Install.EtcdClusterYaml, pluginversion.EtcdClusterLatestSupportedURL(), etcdClusterFile, clientConfig, config.Spec.GetNamespace())
	if err != nil {
		return fs, err
	}
	etcdSubDirs[clusterDir] = etcdClusterFiles

	fsData[etcdDir] = etcdSubDirs
	fs, err = createDirAndFiles(fs, fsData)
	if err != nil {
		return fs, err
	}

	return fs, nil
}

//splitMultiDoc splits a single multidoc manifest into multiple manifests
func splitMultiDoc(multidoc string) []string {
	return strings.Split(multidoc, "\n---\n")
}

// makeMultiDoc creates a single multidoc manifest from multiple manifests
func makeMultiDoc(manifests ...string) string {
	manifestsSlice := make([]string, 0)
	manifestsSlice = append(manifestsSlice, manifests...)

	return strings.Join(manifestsSlice, "\n---\n")
}

// createFileData creates a map of two files (file name to file data).
//
// The first file is that passed to the function and its contents are either pulled or read
// (depending on flag).
//
// The second file is the kustomization.yaml created from scratch.
// It's contents, to begin with are simply:
//
// resources:
// - <filename>
//
func createFileData(yamlPath, yamlUrl, fileName string, config *rest.Config, namespace string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	yamlContents, err := readOrPullManifest(yamlPath, yamlUrl, config, namespace)
	if err != nil {
		return files, err
	}
	files[fileName] = []byte(yamlContents)
	kustYamlContents, err := pluginutils.SetFieldInManifest(kustTemp, fmt.Sprintf("%s%s%s", "[", fileName, "]"), "resources", "")
	if err != nil {
		return files, err
	}

	files[kustomizationFile] = []byte(kustYamlContents)
	return files, nil
}

// createDirAndFiles is a helper function for buildInstallerFileSys, creating the in-memory
// file system for installer from the fsData provided.
func createDirAndFiles(fs filesys.FileSystem, fsData fsData) (filesys.FileSystem, error) {
	for dir, subDirs := range fsData {
		err := fs.Mkdir(dir)
		if err != nil {
			return fs, err
		}

		for subDir, files := range subDirs {
			err := fs.Mkdir(filepath.Join(dir, subDir))
			if err != nil {
				return fs, err
			}
			for name, data := range files {
				_, err := fs.Create(filepath.Join(dir, subDir, name))
				if err != nil {
					return fs, err
				}
				err = fs.WriteFile(filepath.Join(dir, subDir, name), data)
				if err != nil {
					return fs, err
				}
			}
		}
	}
	return fs, nil
}

// readOrPullManifest returns a string of the manifest from path or url provided
func readOrPullManifest(path, url string, config *rest.Config, namespace string) (string, error) {
	location := path
	if location == "" {
		location = url
	}
	if location == "" {
		return "", errors.New("manifest location not set")
	}

	if util.IsURL(location) {
		contents, err := pullManifest(location)
		if err != nil {
			return "", err
		}
		return contents, nil
	}

	if _, err := os.Stat(location); err == nil {
		contents, err := ioutil.ReadFile(location)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	jobName := "storageos-operator-manifests-fetch-" + strconv.FormatInt(time.Now().Unix(), 10)
	return pluginutils.CreateJobAndFetchResult(config, jobName, namespace, location)
}

// pullManifest returns a string of contents at url
func pullManifest(url string) (string, error) {
	if !util.IsURL(url) {
		return "", fmt.Errorf("%s is not a URL and was not found", url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

// storageClassToManifest returns a manifest for storageClass
func storageClassToManifest(storageClass *kstoragev1.StorageClass) ([]byte, error) {
	newStorageClass := &kstoragev1.StorageClass{}
	newStorageClass.APIVersion = "storage.k8s.io/v1"
	newStorageClass.Kind = "StorageClass"
	newStorageClass.SetName(storageClass.Name)
	newStorageClass.SetNamespace(storageClass.Namespace)
	newStorageClass.SetLabels(storageClass.Labels)
	newStorageClass.SetAnnotations(storageClass.Annotations)
	newStorageClass.SetFinalizers(storageClass.GetFinalizers())
	newStorageClass.Provisioner = storageClass.Provisioner
	newStorageClass.Parameters = storageClass.Parameters
	newStorageClass.ReclaimPolicy = storageClass.ReclaimPolicy
	newStorageClass.MountOptions = storageClass.MountOptions
	newStorageClass.AllowVolumeExpansion = storageClass.AllowVolumeExpansion
	newStorageClass.VolumeBindingMode = storageClass.VolumeBindingMode
	newStorageClass.AllowedTopologies = storageClass.AllowedTopologies

	data, err := json.Marshal(&newStorageClass)
	if err != nil {
		return nil, err
	}
	data, err = gyaml.JSONToYAML(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// storageClassToManifest returns a manifest for storageOSCluster
func storageOSClusterToManifest(storageOSCluster *operatorapi.StorageOSCluster) ([]byte, error) {
	newStorageOSCluster := &operatorapi.StorageOSCluster{}
	newStorageOSCluster.APIVersion = storageOSCluster.APIVersion
	newStorageOSCluster.Kind = storageOSCluster.Kind
	newStorageOSCluster.SetName(storageOSCluster.GetName())
	newStorageOSCluster.SetNamespace(storageOSCluster.GetNamespace())
	newStorageOSCluster.SetLabels(storageOSCluster.GetLabels())
	newStorageOSCluster.SetAnnotations(storageOSCluster.GetAnnotations())
	newStorageOSCluster.SetFinalizers(storageOSCluster.GetFinalizers())
	newStorageOSCluster.Spec = storageOSCluster.Spec

	data, err := json.Marshal(&newStorageOSCluster)
	if err != nil {
		return nil, err
	}
	data, err = gyaml.JSONToYAML(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// secretToManifest returns a manifest for secret
func secretToManifest(secret *corev1.Secret) ([]byte, error) {
	newSecret := &corev1.Secret{}
	newSecret.APIVersion = "v1"
	newSecret.Kind = "Secret"
	newSecret.SetName(secret.GetName())
	newSecret.SetNamespace(secret.GetNamespace())
	newSecret.SetLabels(secret.GetLabels())
	newSecret.SetAnnotations(secret.GetAnnotations())
	newSecret.SetFinalizers(secret.GetFinalizers())
	newSecret.Immutable = secret.Immutable
	newSecret.Data = secret.Data
	newSecret.StringData = secret.StringData
	newSecret.Type = secret.Type

	data, err := json.Marshal(&newSecret)
	if err != nil {
		return nil, err
	}
	data, err = gyaml.JSONToYAML(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// secretsToMultiDoc returns a multidoc manifest of secrets from secretList
func secretsToMultiDoc(secretList *corev1.SecretList) ([]byte, error) {
	secretManifests := make([]string, 0)
	for _, secret := range secretList.Items {
		secretManifest, err := secretToManifest(&secret)
		if err != nil {
			return nil, err
		}
		secretManifests = append(secretManifests, string(secretManifest))
	}
	return []byte(makeMultiDoc(secretManifests...)), nil
}

// storageClassesToMultiDoc returns a multidoc manifest of secrets from secretList
func storageClassesToMultiDoc(storageClassList *kstoragev1.StorageClassList) ([]byte, error) {
	storageClassManifests := make([]string, 0)
	for _, storageClass := range storageClassList.Items {
		storageClassManifest, err := storageClassToManifest(&storageClass)
		if err != nil {
			return nil, err
		}
		storageClassManifests = append(storageClassManifests, string(storageClassManifest))
	}
	return []byte(makeMultiDoc(storageClassManifests...)), nil
}

// separateSecrets returns two SecretLists, one of CSI storageos secrets and one of non-CSI
// storageos secrets
func separateSecrets(secretList *corev1.SecretList) (*corev1.SecretList, *corev1.SecretList) {
	stosSecretList := &corev1.SecretList{}
	csiSecretList := &corev1.SecretList{}
	for _, secret := range secretList.Items {
		if strings.HasPrefix(secret.GetName(), "csi-") {
			csiSecretList.Items = append(csiSecretList.Items, secret)
			continue
		}
		stosSecretList.Items = append(stosSecretList.Items, secret)
	}
	return stosSecretList, csiSecretList
}
