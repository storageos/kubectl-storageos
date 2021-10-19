package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"

	etcdoperatorapi "github.com/improbable-eng/etcd-cluster-operator/api/v1alpha1"
	operatorapi "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/kubectl-storageos/pkg/consts"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kstoragev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kversion "k8s.io/apimachinery/pkg/version"
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

const jobTimeout = time.Minute

// ResourcesStillExists contains all the existing resource types in namespace
type ResourcesStillExists struct {
	namespace string
	resources []string
}

// Error generates error message
func (e ResourcesStillExists) Error() string {
	return fmt.Sprintf("resource(s) still found in namespace %s: %s", e.namespace, strings.Join(e.resources, ", "))
}

// NewClientConfig returns a client-go rest config
func NewClientConfig() (*rest.Config, error) {
	configFlags := &genericclioptions.ConfigFlags{}

	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, errors.Wrap(err, consts.ErrUnableToConstructClientConfig)
	}
	return config, nil
}

// GetClientsetFromConfig returns a k8s clientset from a client-go rest config
func GetClientsetFromConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, consts.ErrUnableToContructClientFromConfig)
	}
	return clientset, nil
}

// GetKubernetesVersion fetches Kubernetes version.
func GetKubernetesVersion(config *rest.Config) (*kversion.Info, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, consts.ErrUnableToContructClientFromConfig)
	}

	info, err := clientset.DiscoveryClient.ServerVersion()
	if err != nil {
		err = errors.Wrap(err, "unable to fetch Kubernetes version")
	}

	return info, err
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
		return "", "", errors.WithStack(fmt.Errorf("error adding to scheme: %v", err))
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
		return "", "", errors.WithStack(fmt.Errorf("error while creating Executor: %v", err))
	}

	var stdout, stderr bytes.Buffer
	if err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	}); err != nil {
		return "", "", errors.WithStack(fmt.Errorf("error in Stream: %v", err))
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
		return "", errors.WithStack(fmt.Errorf("unable to read job logs: %s", result.Error()))
	}

	raw, err := result.Raw()
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("unable to read job output: %s", err.Error()))
	}

	return string(raw), nil
}

// FindFirstPodByLabel finds first pod by label or returns error.
func FindFirstPodByLabel(config *rest.Config, namespace, label string) (*corev1.Pod, error) {
	pods, err := ListPods(config, namespace, label)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("unable to list pods: %s", err.Error()))
	}
	if len(pods.Items) == 0 {
		return nil, errors.WithStack(errors.New("no pods found"))
	}

	return &pods.Items[0], nil
}

//ListPods returns PodList discovered by namespace and label.
func ListPods(config *rest.Config, namespace, label string) (*corev1.PodList, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return nil, err
	}

	return pods, nil
}

// PodHasPVC returns true if the pod has the pvc.
func PodHasPVC(pod *corev1.Pod, pvcName string) bool {
	for _, vol := range pod.Spec.Volumes {
		if VolumeHasPVC(&vol, pvcName) {
			return true
		}
	}
	return false
}

// VolumeHasPVC returns true if the volume has the pvc.
func VolumeHasPVC(vol *corev1.Volume, pvcName string) bool {
	return vol.PersistentVolumeClaim != nil && vol.PersistentVolumeClaim.ClaimName == pvcName
}

// GetStorageClass returns storageclass of name.
func GetStorageClass(config *rest.Config, name string) (*kstoragev1.StorageClass, error) {
	storageV1Client, err := storagev1.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, consts.ErrUnableToContructClientFromConfig)
	}
	storageClass, err := storageV1Client.StorageClasses().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return storageClass, nil
}

// GetDefaultStorage returns the the default storage class in the cluster, if more than one storage
// class is set to default, the first one discovered is returned. An error is returned if no default
// storage class is found.
func GetDefaultStorageClass(config *rest.Config) (*kstoragev1.StorageClass, error) {
	storageV1Client, err := storagev1.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, consts.ErrUnableToContructClientFromConfig)
	}
	storageClasses, err := storageV1Client.StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, storageClass := range storageClasses.Items {
		if defaultSC, ok := storageClass.GetObjectMeta().GetAnnotations()["storageclass.kubernetes.io/is-default-class"]; ok && defaultSC == "true" {
			return &storageClass, nil
		}
	}

	return nil, fmt.Errorf("no default storage class discovered in cluster")
}

// GetDefaultStorageClassName returns the name of the default storage class in the cluster, if more
// than one storage class is set to default, the first one discovered is returned. An error is returned
// if no default storage class is found.
func GetDefaultStorageClassName(config *rest.Config) (string, error) {
	defaultSC, err := GetDefaultStorageClass(config)
	if err != nil {
		return "", err
	}
	return defaultSC.Name, nil
}

// IsProvisionedStorageClass returns true if the StorageClass has one of the given provisioners.
func IsProvisionedStorageClass(sc *kstoragev1.StorageClass, provisioners ...string) bool {
	for _, provisioner := range provisioners {
		if sc.Provisioner == provisioner {
			return true
		}
	}
	return false
}

// PVCStorageClassName returns the PVC provisioner name.
func PVCStorageClassName(pvc *corev1.PersistentVolumeClaim) string {
	// The beta annotation should still be supported since even latest versions
	// of Kubernetes still allow it.
	if pvc.Spec.StorageClassName != nil && len(*pvc.Spec.StorageClassName) > 0 {
		return *pvc.Spec.StorageClassName
	}
	if val, ok := pvc.Annotations["volume.beta.kubernetes.io/storage-provisioner"]; ok {
		return val
	}
	return ""
}

// IsProvisionedPVC returns true if the PVC was provided by one of the given provisioners.
func IsProvisionedPVC(config *rest.Config, pvc *corev1.PersistentVolumeClaim, provisioners ...string) (bool, error) {
	// Get the StorageClass that provisioned the volume.
	sc, err := StorageClassForPVC(config, pvc)
	if err != nil {
		return false, err
	}

	return IsProvisionedStorageClass(sc, provisioners...), nil
}

// StorageClassForPVC returns the StorageClass of the PVC. If no StorageClass
// was specified, returns the cluster default if set.
func StorageClassForPVC(config *rest.Config, pvc *corev1.PersistentVolumeClaim) (*kstoragev1.StorageClass, error) {
	name := PVCStorageClassName(pvc)
	if name == "" {
		sc, err := GetDefaultStorageClass(config)
		if err != nil {
			return nil, err
		}
		return sc, nil
	}
	sc, err := GetStorageClass(config, name)
	if err != nil {
		return nil, err
	}

	return sc, nil
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
			return errors.WithStack(errors.Wrap(err, "timeout with error"))
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

// IsServiceReady attempts to `get` a service by name and namespace, the function returns no error
// if the service doesn't have a ClusterIP or any ready endpoints.
func IsServiceReady(config *rest.Config, name, namespace string) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}
	svcClient := clientset.CoreV1().Services(namespace)

	svc, err := svcClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if svc.Spec.ClusterIP == "" {
		return fmt.Errorf("no cluster ip for service %s; %s", name, namespace)
	}

	epClient := clientset.CoreV1().Endpoints(namespace)
	ep, err := epClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, subset := range ep.Subsets {
		if len(subset.Addresses) > 0 {
			return nil
		}
	}
	return fmt.Errorf("no endpoints are ready for service %s; %s", name, namespace)
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

// GetNamespace return namespace object
func GetNamespace(config *rest.Config, namespace string) (*corev1.Namespace, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	nsClient := clientset.CoreV1().Namespaces()
	ns, err := nsClient.Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ns, nil
}

// DeleteNamespace deletes the given namespace
func DeleteNamespace(config *rest.Config, namespace string) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}

	err = clientset.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
	if err != nil && kerrors.IsNotFound(err) {
		return errors.WithStack(err)
	}

	return nil
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
	return fmt.Errorf("namespace %s exists in cluster", namespace)
}

// NamespaceExists returns no error only if the specified namespace exists in the k8s cluster
func NamespaceExists(config *rest.Config, namespace string) error {
	_, err := GetNamespace(config, namespace)

	return err
}

// ListStorageClasses returns StorageClassList
func ListStorageClasses(config *rest.Config, listOptions metav1.ListOptions) (*kstoragev1.StorageClassList, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	storageClasses, err := clientset.StorageV1().StorageClasses().List(context.TODO(), listOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return storageClasses, nil
}

// ListPersistentVolumeClaims returns PersistentVolumeClaimList
func ListPersistentVolumeClaims(config *rest.Config, listOptions metav1.ListOptions) (*corev1.PersistentVolumeClaimList, error) {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(context.TODO(), listOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return pvcs, nil
}

// CreateStorageClass creates k8s storage class.
func CreateStorageClass(config *rest.Config, storageClass *kstoragev1.StorageClass) error {
	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}
	scClient := clientset.StorageV1().StorageClasses()
	_, err = scClient.Create(context.TODO(), storageClass, metav1.CreateOptions{})

	return err
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
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
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

	return err
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

	return err
}

// GetFirstStorageOSCluster returns the storageoscluster object if it exists in the k8s cluster.
// Use 'List' to discover as there can only be one object per k8s cluster and 'List' does not
// require name/namespace.
func GetFirstStorageOSCluster(config *rest.Config) (*operatorapi.StorageOSCluster, error) {
	stosCluster := &operatorapi.StorageOSCluster{}
	stosClusterList := &operatorapi.StorageOSClusterList{}
	newClient, err := storageOSOperatorClient(config)
	if err != nil {
		return stosCluster, err
	}
	err = newClient.List(context.TODO(), stosClusterList, &client.ListOptions{})
	if err != nil {
		return stosCluster, errors.WithStack(err)
	}

	if len(stosClusterList.Items) == 0 {
		return stosCluster, kerrors.NewNotFound(operatorapi.SchemeGroupVersion.WithResource("StorageOSCluster").GroupResource(), "")
	}

	stosCluster = &stosClusterList.Items[0]
	return stosCluster, nil
}

func storageOSOperatorClient(config *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := operatorapi.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "failed to add to scheme")
	}
	return client.New(config, client.Options{Scheme: scheme})
}

// StorageOSClusterDoesNotExist return no error only if no storageoscluster object exists in k8s cluster
func StorageOSClusterDoesNotExist(config *rest.Config) error {
	stosCluster, err := GetFirstStorageOSCluster(config)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return fmt.Errorf("storageoscluster [" + stosCluster.Name + ";" + stosCluster.Namespace + "] exists in k8s cluster")
}

// UpdateStorageOSClusterWithoutFinalizers updates the storageos cluster without any finalizers.
func UpdateStorageOSClusterWithoutFinalizers(config *rest.Config, storageosCluster *operatorapi.StorageOSCluster) error {
	if len(storageosCluster.Finalizers) == 0 {
		return nil
	}
	newClient, err := storageOSOperatorClient(config)
	if err != nil {
		return err
	}
	storageosCluster.SetFinalizers(nil)
	return newClient.Update(context.TODO(), storageosCluster)
}

// GetEtcdCluster returns the etcdcluster object of name and namespace.
func GetEtcdCluster(config *rest.Config, name, namespace string) (*etcdoperatorapi.EtcdCluster, error) {
	etcdCluster := &etcdoperatorapi.EtcdCluster{}
	newClient, err := etcdOperatorClient(config)
	if err != nil {
		return etcdCluster, err
	}
	if err = newClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, etcdCluster); err != nil {
		return etcdCluster, err
	}
	return etcdCluster, nil
}

func etcdOperatorClient(config *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := etcdoperatorapi.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "failed to add to scheme")
	}
	return client.New(config, client.Options{Scheme: scheme})
}

// EtcdClusterDoesNotExist return no error only if no etcdcluster object exists in k8s cluster
func EtcdClusterDoesNotExist(config *rest.Config, name, namespace string) error {
	if _, err := GetEtcdCluster(config, name, namespace); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return fmt.Errorf("etcdcluster [" + name + ";" + namespace + "] exists in k8s cluster")
}

// UpdateEtcdClusterWithoutFinalizers returns the etcdcluster object of name and namespace.
func UpdateEtcdClusterWithoutFinalizers(config *rest.Config, etcdCluster *etcdoperatorapi.EtcdCluster) error {
	if len(etcdCluster.Finalizers) == 0 {
		return nil
	}
	newClient, err := etcdOperatorClient(config)
	if err != nil {
		return err
	}
	etcdCluster.SetFinalizers(nil)
	return newClient.Update(context.TODO(), etcdCluster)
}

// EnsureNamespace Creates namespace if it does not exists.
func EnsureNamespace(config *rest.Config, name string) error {
	if err := NamespaceExists(config, name); err == nil {
		return nil
	}

	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return err
	}

	if _, err = clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, metav1.CreateOptions{}); err != nil {
		return errors.WithStack(err)
	}

	err = WaitFor(func() error {
		return NamespaceExists(config, name)
	}, 120, 5)

	return err
}

// CreateJobAndFetchResult Creates a job, fetches the output of the job and deletes the created resources.
func CreateJobAndFetchResult(config *rest.Config, name, namespace, image, cmd string) (string, error) {
	jobMeta := metav1.ObjectMeta{
		Name: name,
	}

	bofl := int32(1)
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
			BackoffLimit: &bofl,
		},
	}

	if cmd != "" {
		job.Spec.Template.Spec.Containers[0].Command = strings.Split(cmd, " ")
	}

	clientset, err := GetClientsetFromConfig(config)
	if err != nil {
		return "", err
	}

	jobClient := clientset.BatchV1().Jobs(namespace)

	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	_, err = jobClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer func() {
		delErr := jobClient.Delete(context.Background(), job.Name, metav1.DeleteOptions{})
		if delErr != nil {
			println(fmt.Sprintf(helperDeletionErrorMessage, "job", delErr.Error(), "job", namespace, job.Name))
		}
	}()

	watch, err := jobClient.Watch(ctx, metav1.SingleObject(jobMeta))
	if err != nil {
		return "", errors.WithStack(err)
	}

	for {
		res, ok := <-watch.ResultChan()
		if !ok {
			return "", errors.WithStack(fmt.Errorf("unable to read job events of %s", image))
		}

		job, ok := res.Object.(*batchv1.Job)
		if !ok {
			return "", errors.WithStack(errors.New("unable to convert event to job"))
		}

		if job.Status.CompletionTime == nil {
			continue
		}

		if job.Status.Failed > 0 {
			return "", errors.WithStack(errors.New("unable to fetch manifests"))
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
