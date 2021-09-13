package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	etcdoperatorapi "github.com/improbable-eng/etcd-cluster-operator/api/v1alpha1"
	operatorapi "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kstoragev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

const helperDeletionErrorMessage = `Unable to delete helper %s.
Reason: %s
Please delete it manually by executing the following command:
kubectl delete %s -n %s %s`

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

// FetchPodLogs fetches logs of the given pod.
func FetchPodLogs(config *rest.Config, name, namespace string) (string, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return "", err
	}

	logs := clientset.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{})
	logs.Timeout(time.Minute)
	result := logs.Do(context.Background())
	if result.Error() != nil {
		return "", fmt.Errorf("unable to read job logs: %s", result.Error())
	}

	raw, err := result.Raw()
	if err != nil {
		return "", fmt.Errorf("unable to read job output: %s", err.Error())
	}

	return string(raw), nil
}

// FindFirstPodByLabel finds first pod by label or returns error.
func FindFirstPodByLabel(config *rest.Config, namespace, label string) (*corev1.Pod, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list job pods: %s", err.Error())
	}
	if len(pods.Items) == 0 {
		return nil, errors.New("unable to find job pod")
	}

	return &pods.Items[0], nil
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

// NoResourcesInNS returns no error only if no resource exists in the provided namespace.
func NoResourcesInNS(config *rest.Config, namespace string) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}

	resources, err := clientset.DiscoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		return err
	}

	for _, resource := range resources {
		apiClient := getClient(clientset, resource.GroupVersion)
		if apiClient == nil {
			continue
		}

		for _, apiResource := range resource.APIResources {
			if !apiResource.Namespaced {
				continue
			}

			res := apiClient.Get().Namespace(namespace).Name(apiResource.Name).Do(context.Background())
			if res.Error() != nil {
				if kerrors.IsMethodNotSupported(res.Error()) {
					continue
				}
				return err
			}

			obj, err := res.Get()
			if err != nil {
				return err
			}

			items, err := countItems(obj)
			if err != nil {
				return err
			}

			if items > 0 {
				return fmt.Errorf("%s/%s still exists in namespace %s", resource.GroupVersion, apiResource.Name, namespace)
			}
		}
	}

	return nil
}

type itemList struct {
	Items []interface{} `json:"items,omitempty"`
}

func countItems(obj runtime.Object) (int, error) {
	marshalled, err := json.Marshal(obj)
	if err != nil {
		return 0, err
	}

	unmarshalled := &itemList{}
	err = json.Unmarshal(marshalled, unmarshalled)
	if err != nil {
		return 0, err
	}

	return len(unmarshalled.Items), nil
}

func getClient(clientset *kubernetes.Clientset, groupVersion string) rest.Interface {
	switch groupVersion {
	case "apps/v1":
		return clientset.AppsV1().RESTClient()
	case "app/v1beta1":
		return clientset.AppsV1beta1().RESTClient()
	case "app/v1beta2":
		return clientset.AppsV1beta2().RESTClient()
	case "authorization.k8s.io/v1":
		return clientset.AuthorizationV1().RESTClient()
	case "authorization.k8s.io/v1beta1":
		return clientset.AuthorizationV1beta1().RESTClient()
	case "autoscaling/v1":
		return clientset.AutoscalingV1().RESTClient()
	case "autoscaling/v2beta1":
		return clientset.AutoscalingV2beta1().RESTClient()
	case "autoscaling/v2beta2":
		return clientset.AutoscalingV2beta2().RESTClient()
	case "batch/v1":
		return clientset.BatchV1().RESTClient()
	case "batch/v1beta1":
		return clientset.BatchV1beta1().RESTClient()
	case "coordination.k8s.io/v1":
		return clientset.CoordinationV1().RESTClient()
	case "coordination.k8s.io/v1beta1":
		return clientset.CoordinationV1beta1().RESTClient()
	case "v1":
		return clientset.CoreV1().RESTClient()
	case "events.k8s.io/v1":
		return nil
	case "events.k8s.io/v1beta1":
		return nil
	case "extensions/v1beta1":
		return clientset.ExtensionsV1beta1().RESTClient()
	case "networking.k8s.io/v1":
		return clientset.NetworkingV1().RESTClient()
	case "networking.k8s.io/v1beta1":
		return clientset.NetworkingV1beta1().RESTClient()
	case "policy/v1":
		return clientset.PolicyV1().RESTClient()
	case "policy/v1beta1":
		return clientset.PolicyV1beta1().RESTClient()
	default:
		return nil
	}
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
		if kerrors.IsNotFound(err) {
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

// GetStorageClass returns storage class of name.
func GetStorageClass(config *rest.Config, name string) (*kstoragev1.StorageClass, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	scClient := clientset.StorageV1().StorageClasses()
	storageClass, err := scClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return storageClass, nil
}

// ListStorageClasses returns StorageClassList
func ListStorageClasses(config *rest.Config, listOptions metav1.ListOptions) (*kstoragev1.StorageClassList, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	storageClasses, err := clientset.StorageV1().StorageClasses().List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}

	return storageClasses, nil
}

// CreateStorageClass creates k8s storage class.
func CreateStorageClass(config *rest.Config, storageClass *kstoragev1.StorageClass) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}
	scClient := clientset.StorageV1().StorageClasses()
	_, err = scClient.Create(context.TODO(), storageClass, metav1.CreateOptions{})
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

// ListSecrets returns SecretList
func ListSecrets(config *rest.Config, listOptions metav1.ListOptions) (*corev1.SecretList, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	secrets, err := clientset.CoreV1().Secrets("").List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

// CreateSecret creates k8s secret.
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
		if kerrors.IsNotFound(err) {
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
func GetStorageOSCluster(config *rest.Config, namespace string) (*operatorapi.StorageOSCluster, error) {
	scheme := runtime.NewScheme()
	operatorapi.AddToScheme(scheme)
	stosCluster := &operatorapi.StorageOSCluster{}

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
		return stosCluster, kerrors.NewNotFound(operatorapi.SchemeGroupVersion.WithResource("StorageOSCluster").GroupResource(), "")
	}

	stosCluster = &stosClusterList.Items[0]
	return stosCluster, nil
}

// StorageOSClusterDoesNotExist return no error only if no storageoscluster object exists in k8s cluster
func StorageOSClusterDoesNotExist(config *rest.Config, namespace string) error {
	_, err := GetStorageOSCluster(config, namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
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
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return fmt.Errorf("etcdcluster exists")
}

// EnsureNamespace Creates namespace if it does not exists.
func EnsureNamespace(config *rest.Config, name string) error {
	err := NamespaceExists(config, name)
	if err == nil {
		return nil
	}

	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	err = WaitFor(func() error {
		return NamespaceExists(config, name)
	}, 120, 5)
	if err != nil {
		return err
	}

	return nil
}

// CreateJobAndFetchResult Creates a job, fetches the output of the job and deletes the created resources.
func CreateJobAndFetchResult(config *rest.Config, name, namespace, image string) (string, error) {
	jobMeta := metav1.ObjectMeta{
		Name: name,
	}
	job := &batchv1.Job{
		ObjectMeta: jobMeta,
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return "", err
	}

	jobClient := clientset.BatchV1().Jobs(namespace)

	_, err = jobClient.Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	defer func() {
		delErr := jobClient.Delete(context.Background(), job.Name, metav1.DeleteOptions{})
		if delErr != nil {
			println(fmt.Sprintf(helperDeletionErrorMessage, "job", delErr.Error(), "job", namespace, job.Name))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watch, err := jobClient.Watch(ctx, metav1.SingleObject(jobMeta))
	if err != nil {
		return "", err
	}

	for {
		res, ok := <-watch.ResultChan()
		if !ok {
			return "", errors.New("unable to read job events")
		}

		job, ok := res.Object.(*batchv1.Job)
		if !ok {
			return "", errors.New("unable to convert event to job")
		}

		if job.Status.CompletionTime == nil {
			continue
		}

		if job.Status.Failed > 0 {
			return "", errors.New("unable to fetch manifests")
		}

		pod, err := FindFirstPodByLabel(config, namespace, "job-name="+name)
		if err != nil {
			return "", err
		}
		defer func() {
			delErr := clientset.CoreV1().Pods(namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
			if delErr != nil {
				println(fmt.Sprintf(helperDeletionErrorMessage, "pod", delErr.Error(), "pod", namespace, pod.Name))
			}
		}()

		return FetchPodLogs(config, pod.Name, namespace)
	}
}
