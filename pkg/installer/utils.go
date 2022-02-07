package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	gyaml "github.com/ghodss/yaml"
	"github.com/replicatedhq/troubleshoot/cmd/util"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	operatorapi "github.com/storageos/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	kstoragev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/kustomize/api/filesys"
)

const errFlagsNotSet = "The following flags have not been set and are required to perform this command:"

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

// getStringWithDefault returns primary string if set, secondary string otherwise
func getStringWithDefault(primaryString, secondaryString string) string {
	if primaryString != "" {
		return primaryString
	}
	return secondaryString
}

// createDirAndFiles is a helper function for buildInstallerFileSys, creating the in-memory
// file system for installer from the fsData provided.
func createDirAndFiles(fs filesys.FileSystem, fsData fsData) (filesys.FileSystem, error) {
	for dir, subDirs := range fsData {
		if err := fs.Mkdir(dir); err != nil {
			return fs, errors.WithStack(err)
		}

		for subDir, files := range subDirs {
			if err := fs.Mkdir(filepath.Join(dir, subDir)); err != nil {
				return fs, errors.WithStack(err)
			}
			for name, data := range files {
				if _, err := fs.Create(filepath.Join(dir, subDir, name)); err != nil {
					return fs, errors.WithStack(err)
				}
				if err := fs.WriteFile(filepath.Join(dir, subDir, name), data); err != nil {
					return fs, errors.WithStack(err)
				}
			}
		}
	}
	return fs, nil
}

// pullManifest returns a string of contents at url
func pullManifest(url string) (string, error) {
	if !util.IsURL(url) {
		return "", errors.WithStack(fmt.Errorf("%s is not a URL and was not found", url))
	}

	contents, err := pluginutils.FetchHttpContent(url, nil)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(contents), nil
}

func isDockerRepo(url string) bool {
	return strings.HasPrefix(url, "docker.io/")
}

// fetchImageAndExtractFromTarball creates a tarball from an OCI image and returns the file at filePath
func fetchImageAndExtractFileFromTarball(imageURL, filePath string) (string, error) {
	pulled, err := pluginutils.PullImage(imageURL)
	if err != nil {
		return "", errors.WithStack(err)
	}

	exported, err := pluginutils.ExportTarball(pulled)
	if err != nil {
		return "", errors.WithStack(err)
	}

	file, err := pluginutils.ExtractFile(filePath, bytes.NewReader(exported.Bytes()))
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(file), nil
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
	newStorageClass.SetResourceVersion(storageClass.GetResourceVersion())
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
		return nil, errors.WithStack(err)
	}
	data, err = gyaml.JSONToYAML(data)
	if err != nil {
		return nil, errors.WithStack(err)
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
	newStorageOSCluster.SetResourceVersion(storageOSCluster.GetResourceVersion())
	newStorageOSCluster.SetFinalizers(storageOSCluster.GetFinalizers())
	newStorageOSCluster.Spec = storageOSCluster.Spec

	data, err := json.Marshal(&newStorageOSCluster)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	data, err = gyaml.JSONToYAML(data)
	if err != nil {
		return nil, errors.WithStack(err)
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
	newSecret.SetResourceVersion(secret.GetResourceVersion())
	newSecret.SetFinalizers(secret.GetFinalizers())
	newSecret.Immutable = secret.Immutable
	newSecret.Data = secret.Data
	newSecret.StringData = secret.StringData
	newSecret.Type = secret.Type

	data, err := json.Marshal(&newSecret)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	data, err = gyaml.JSONToYAML(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return data, nil
}

// configMapToManifest returns a manifest for configmap
func configMapToManifest(configMap *corev1.ConfigMap) ([]byte, error) {
	newConfigMap := &corev1.ConfigMap{}
	newConfigMap.APIVersion = "v1"
	newConfigMap.Kind = "ConfigMap"
	newConfigMap.SetName(configMap.GetName())
	newConfigMap.SetNamespace(configMap.GetNamespace())
	newConfigMap.SetLabels(configMap.GetLabels())
	newConfigMap.SetAnnotations(configMap.GetAnnotations())
	newConfigMap.SetFinalizers(configMap.GetFinalizers())
	newConfigMap.SetResourceVersion(configMap.GetResourceVersion())
	newConfigMap.Immutable = configMap.Immutable
	newConfigMap.Data = configMap.Data
	newConfigMap.BinaryData = configMap.BinaryData

	data, err := json.Marshal(&newConfigMap)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	data, err = gyaml.JSONToYAML(data)
	if err != nil {
		return nil, errors.WithStack(err)
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

// configMapsToMultiDoc returns a multidoc manifest of configmaps from configMapList
func configMapsToMultiDoc(configMapList *corev1.ConfigMapList) ([]byte, error) {
	configMapManifests := make([]string, 0)
	for _, configMap := range configMapList.Items {
		configMapManifest, err := configMapToManifest(&configMap)
		if err != nil {
			return nil, err
		}
		configMapManifests = append(configMapManifests, string(configMapManifest))
	}
	return []byte(makeMultiDoc(configMapManifests...)), nil
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

// collectErrors collects all errors on the channel
func collectErrors(errChan <-chan error) error {
	mErr := multipleErrors{
		errors: []string{},
	}

	for {
		err, ok := <-errChan

		if !ok {
			switch len(mErr.errors) {
			case 0:
				return nil
			case 1:
				return errors.New(mErr.errors[0])
			default:
				return mErr
			}
		}

		if err != nil {
			mErr.errors = append(mErr.errors, err.Error())
		}
	}
}

// FlagsAreSet takes a map[string]string of flag-name:flag-value and returns an error listing
// all flag-names in 'flags' map that have not been set.
func FlagsAreSet(flags map[string]string) error {
	missingFlags := make([]string, 0)
	for flagName, flagValue := range flags {
		if flagValue == "" {
			missingFlags = append(missingFlags, flagName)
		}
	}
	if len(missingFlags) != 0 {
		return fmt.Errorf(errFlagsNotSet + strings.Join(missingFlags, ","))
	}
	return nil
}
