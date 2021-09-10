package installer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/replicatedhq/troubleshoot/cmd/util"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
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
