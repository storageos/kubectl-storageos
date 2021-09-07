package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	etcdoperatorapi "github.com/improbable-eng/etcd-cluster-operator/api/v1alpha1"
	operatorapi "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
		return stosCluster, kerrors.NewNotFound(operatorapi.SchemeGroupVersion.WithResource("StorageOSCluster").GroupResource(), "")
	}

	stosCluster = stosClusterList.Items[0]
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

// EnsureNamespace Creates namespace if not exists.
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
			println(fmt.Sprintf(`
Unable to delete helper job.
Reason: %s
Please delete it manually by executing the following command:
kubectl delete job -n %s %s`, delErr.Error(), namespace, job.Name))
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
				println(fmt.Sprintf(`
Unable to delete helper pod.
Reason: %s
Please delete it manually by executing the following command:
kubectl delete pod -n %s %s`, delErr.Error(), namespace, pod.Name))
			}
		}()

		return FetchPodLogs(config, pod.Name, namespace)
	}
}
