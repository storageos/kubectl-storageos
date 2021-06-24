package install

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func NewClientConfig() (*rest.Config, error) {
	configFlags := &genericclioptions.ConfigFlags{}

	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func GetClientsetFromConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// ExecToPod execs into a pod and executes command from inside that pod.
// containerName can be "" if the pod contains only a single container.
// Returned are strings represent STDOUT and STDERR respectively.
// Also returned is any error encountered.
func ExecToPod(config *rest.Config, command []string, containerName, podName, namespace string, stdin io.Reader) (string, string, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return "", "", err
	}
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", "", fmt.Errorf("error adding to scheme: %v", err)
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     stdin != nil,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", "", fmt.Errorf("error in Stream: %v", err)
	}

	return stdout.String(), stderr.String(), nil
}

func SetFieldInManifest(manifest, value, valueName string, fields ...string) (string, error) {
	obj, err := kyaml.Parse(manifest)
	if err != nil {
		return "", err
	}

	parsedVal, err := kyaml.Parse(value)
	if err != nil {
		return "", err
	}

	_, err = obj.Pipe(kyaml.LookupCreate(kyaml.MappingNode, fields...), kyaml.SetField(valueName, parsedVal))
	if err != nil {
		return "", err
	}
	return obj.MustString(), nil

}

func GetFieldInManifest(manifest string, fields ...string) (string, error) {
	obj, err := kyaml.Parse(manifest)
	if err != nil {
		return "", err
	}

	val, err := obj.Pipe(kyaml.Lookup(fields...))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(val.MustString()), nil
}

// KustomizePatch is useed to pass a new patch to a kustomization file, see AddPatchesToKustomize
type KustomizePatch struct {
	Op    string
	Path  string
	Value string
}

// AddPatchesToKustomize adds any number of patches (via []KustomizePatch{}) to kustomizationFile string,
// returning the updated kustomization file as a string.

// Example
//*******************************************************
// Input kustomization file:
//*******************************************************
// apiVersion: kustomize.config.k8s.io/v1beta1
// kind: Kustomization
//
// resources:
// - storageos-cluster.yaml
//******************************************************
// Other inputs:
// targetKind: "StorageOSCluster"
// targetName: "storageoscluster-sample"
// patches: []KustomizePatch{
//	{
//		Op: "replace",
//		Path: "/spec/kvBackend/address",
//		Value: 	"storageos.storageos-etcd:2379",
//	},
// }
//*******************************************************
// Results in the following output kustomization file:
//*******************************************************
// apiVersion: kustomize.config.k8s.io/v1beta1
// kind: Kustomization
//
// resources:
// - storageos-cluster.yaml
//
// patches:
// - target:
//     kind: StorageOSCluster
//     name: storageoscluster-sample
//   patch: |
//     - op: replace
//       path: /spec/kvBackend/address
//       value: storageos.storageos-etcd:2379
//*******************************************************
func AddPatchesToKustomize(kustomizationFile, targetKind, targetName string, patches []KustomizePatch) (string, error) {
	obj, err := kyaml.Parse(string(kustomizationFile))
	if err != nil {
		return "", err
	}

	patchStrings := make([]string, 0)
	for _, patch := range patches {
		patchString := fmt.Sprintf("%s%s%s%s%s%s", `
    - op: `, patch.Op, `
      path: `, patch.Path, `
      value: `, patch.Value)
		patchStrings = append(patchStrings, patchString)

	}

	allPatchesStr := strings.Join(patchStrings, "")

	targetString := fmt.Sprintf("%s%s%s%s%s", `
- target:
    kind: `, targetKind, `
    name: `, targetName, `
  patch: |`)

	patch, err := kyaml.Parse(strings.Join([]string{targetString, allPatchesStr}, ""))

	_, err = obj.Pipe(
		kyaml.LookupCreate(kyaml.SequenceNode, "patches"),
		kyaml.Append(patch.YNode().Content...))
	if err != nil {
		return "", err
	}

	return obj.MustString(), nil
}

// NamespaceYaml returns a yaml string for a namespace object based on the namespace name
func NamespaceYaml(namespace string) string {
	return fmt.Sprintf("%v%v", `apiVersion: v1
kind: Namespace
metadata:
  name: `, namespace)

}

// PodIsRunning attempts to `get` a pod by name and namespace, the function returns no error
// if the pod is in running phase. If an error occurs during `get`, the error is returned.
// If the pod is a phase other than running, `get` is executed again after 5 seconds.
// After 60 seconds, the function times out and returns timeout error.
func PodIsRunning(config *rest.Config, name, namespace string) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}
	podClient := clientset.CoreV1().Pods(namespace)
	timeout := time.After(time.Second * 60)
	errs, ctx := errgroup.WithContext(context.TODO())
	errs.Go(func() error {
		for {
			select {
			case <-timeout:
				return fmt.Errorf("timeout attempting to reach pod %s;%s", name, namespace)
			default:
				pod, err := podClient.Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if pod.Status.Phase == corev1.PodRunning {
					return nil
				}
				time.Sleep(5 * time.Second)
			}
		}
	})
	return errs.Wait()
}
