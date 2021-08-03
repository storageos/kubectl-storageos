package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	etcdoperatorapi "github.com/improbable-eng/etcd-cluster-operator/api/v1alpha1"
	operatorapi "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClientConfig returns a client-go rest config
func NewClientConfig() (*rest.Config, error) {
	configFlags := &genericclioptions.ConfigFlags{}

	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetClientsetFromConfig returns a k8s clientset from a client-go rest config
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

// GetDefaultStorageClassName returns the name of the default storage class in the cluster, if more
// than one storage class is set to default, the first one discovered is returned. An error is returned
// if no default storage class is found.
func GetDefaultStorageClassName() (string, error) {
	restConfig, err := NewClientConfig()
	if err != nil {
		return "", err
	}

	storageV1Client, err := storagev1.NewForConfig(restConfig)
	if err != nil {
		return "", err
	}
	storageClasses, err := storageV1Client.StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, storageClass := range storageClasses.Items {
		if defaultSC, ok := storageClass.GetObjectMeta().GetAnnotations()["storageclass.kubernetes.io/is-default-class"]; ok && defaultSC == "true" {
			return storageClass.GetObjectMeta().GetName(), nil
		}
	}

	return "", fmt.Errorf("no default storage class discovered in cluster")
}

// WaitFor runs 'fn' every 'interval' for duration of 'limit', returning no error only if 'fn' returns no
// error inside 'limit'
func WaitFor(fn func() error, limit, interval time.Duration) error {
	timeout := time.After(time.Second * limit)
	ticker := time.NewTicker(time.Second * interval)
	defer ticker.Stop()
	var err error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout with error: %v", err)
		case <-ticker.C:
			err = fn()
			if err == nil {
				return nil
			}
		}
	}
}

// IsDeploymentReady attempts to `get` a deployment by name and namespace, the function returns no error
// if no deployment replicas are ready.
func IsDeploymentReady(config *rest.Config, name, namespace string) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}
	depClient := clientset.AppsV1().Deployments(namespace)

	dep, err := depClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if dep.Status.ReadyReplicas == 0 {
		return fmt.Errorf("no replicas are ready for deployment %s; %s", name, namespace)
	}
	return nil
}

// IsPodRunning attempts to `get` a pod by name and namespace, the function returns no error
// if the pod is in running phase.
func IsPodRunning(config *rest.Config, name, namespace string) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}
	podClient := clientset.CoreV1().Pods(namespace)
	pod, err := podClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("pod %s; %s is not in running phase", name, namespace)
	}
	return nil
}

// NoPodsInNS returns no error only if no pod exists in the provided namespace.
func NoPodsInNS(config *rest.Config, namespace string) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}
	podClient := clientset.CoreV1().Pods(namespace)
	pods, err := podClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(pods.Items) > 0 {
		return fmt.Errorf("pods still exist in namespace %s", namespace)
	}
	return nil
}

// GetNamespace return namespace object
func GetNamespace(config *rest.Config, namespace string) (*corev1.Namespace, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	nsClient := clientset.CoreV1().Namespaces()
	ns, err := nsClient.Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return ns, nil
}

// NamespaceDoesNotExist returns no error only if the specified namespace does not exist in the k8s cluster
func NamespaceDoesNotExist(config *rest.Config, namespace string) error {
	_, err := GetNamespace(config, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return fmt.Errorf("namespace %v exists in cluster", namespace)
}

// NamespaceExists returns no error only if the specified namespace exists in the k8s cluster
func NamespaceExists(config *rest.Config, namespace string) error {
	_, err := GetNamespace(config, namespace)
	if err != nil {
		return err
	}
	return nil
}

// GetSecret returns data of secret name/namespace
func GetSecret(config *rest.Config, name, namespace string) (*corev1.Secret, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	secretClient := clientset.CoreV1().Secrets(namespace)
	secret, err := secretClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// GetSecret returns data of secret name/namespace
func CreateSecret(config *rest.Config, secret *corev1.Secret, namespace string) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}
	secretClient := clientset.CoreV1().Secrets(namespace)
	_, err = secretClient.Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// SecretDoesNotExist returns no error only if the specified secret does not exist in the k8s cluster
func SecretDoesNotExist(config *rest.Config, name, namespace string) error {
	_, err := GetSecret(config, name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return fmt.Errorf("secret %v exists in cluster", namespace)
}

// SecretExists returns no error only if the specified secret exists in the k8s cluster
func SecretExists(config *rest.Config, name, namespace string) error {
	_, err := GetSecret(config, name, namespace)
	if err != nil {
		return err
	}
	return nil
}

// GetStorageOSCluster returns the storageoscluster object if it exists in the k8s cluster.
// Use 'List' to discover as there can only be one object per k8s cluster and 'List' does not
// require name/namespace - input namespace can optionally be passed to narrow the search.
func GetStorageOSCluster(config *rest.Config, namespace string) (operatorapi.StorageOSCluster, error) {
	scheme := runtime.NewScheme()
	operatorapi.AddToScheme(scheme)
	stosCluster := operatorapi.StorageOSCluster{}

	newClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return stosCluster, err
	}
	stosClusterList := &operatorapi.StorageOSClusterList{}
	listOption := (&client.ListOptions{}).ApplyOptions([]client.ListOption{client.InNamespace(namespace)})
	err = newClient.List(context.TODO(), stosClusterList, listOption)
	if err != nil {
		return stosCluster, err
	}

	if len(stosClusterList.Items) == 0 {
		return stosCluster, errors.NewNotFound(operatorapi.SchemeGroupVersion.WithResource("StorageOSCluster").GroupResource(), "")
	}

	stosCluster = stosClusterList.Items[0]
	return stosCluster, nil
}

// StorageOSClusterDoesNotExist return no error only if no storageoscluster object exists in k8s cluster
func StorageOSClusterDoesNotExist(config *rest.Config, namespace string) error {
	_, err := GetStorageOSCluster(config, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return fmt.Errorf("storageoscluster exists")
}

// GetEtcdCluster returns the etcdcluster object of name and namespace.
func GetEtcdCluster(config *rest.Config, name, namespace string) (*etcdoperatorapi.EtcdCluster, error) {
	scheme := runtime.NewScheme()
	etcdoperatorapi.AddToScheme(scheme)
	etcdCluster := &etcdoperatorapi.EtcdCluster{}

	newClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return etcdCluster, err
	}
	err = newClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, etcdCluster)
	if err != nil {
		return etcdCluster, err
	}
	return etcdCluster, nil
}

// EtcdClusterDoesNotExist return no error only if no etcdcluster object exists in k8s cluster
func EtcdClusterDoesNotExist(config *rest.Config, name, namespace string) error {
	_, err := GetEtcdCluster(config, name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return fmt.Errorf("etcdcluster exists")
}
