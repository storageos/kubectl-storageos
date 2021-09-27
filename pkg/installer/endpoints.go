package installer

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/logger"

	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
)

const (
	errNSForSecretNotFound = `
	Namespace %s not found for %s while attempting to validate ETCD endpoints.

	To skip ETCD endpoints validation during installation, set the --%s flag`

	errSecretNotFound = `
	Unable to find etcd client secret %s in namespace %s for ETCD endpoint validation.

	Please create a k8s secret in the StorageOS cluster namespace like so:

	kubectl create secret generic <etcd-secret-name> -n <storageos-cluster-namespace> \
		--from-file=etcd-client-ca.crt=path/to/ca.crt \
		--from-file=etcd-client.crt=path/to/tls.crt \
		--from-file=etcd-client.key=path/to/tls.key


	If you have uninstalled StorageOS using kubectl-storageos since last creating your secret, check 
	
	$HOME/.kube/storageos/uninstall-<cluster-id>/storageos-secrets.yaml for a local backup.
	


	To skip ETCD endpoints validation during installation, set the --%s flag`

	etcdShellPod = `apiVersion: v1
kind: Pod
metadata:
  name: storageos-etcd-shell
  namespace: default
spec:
  restartPolicy: OnFailure      
  containers:
    - name: storageos-etcd-shell
      image: gcr.io/etcd-development/etcd:v3.5.0
      # pod completes and is not restarted after 3m, this is in case
      # the plugin crashes and is unable to delete this pod after health check
      command: [ "sleep" ]
      args: [ "3m" ]
`
	etcdShellPodTLS = `apiVersion: v1
kind: Pod
metadata:
  name: storageos-etcd-shell
  namespace: storageos
spec:
  restartPolicy: OnFailure
  containers:
    - name: storageos-etcd-shell
      image: gcr.io/etcd-development/etcd:v3.5.0
      # pod completes and is not restarted after 3m, this is in case
      # the plugin crashes and is unable to delete this pod after health check
      command: [ "sleep" ]
      args: [ "infinity" ]
      volumeMounts:
      - mountPath: /run/storageos/pki
        name: etcd-certs
        readOnly: true
  volumes:
  - name: etcd-certs
    secret:
      secretName: storageos-etcd-secret
`
)

// handleEndpointsInput adds validated (or not validated) endpoints patch to kustomization file
// for storageos-cluster.yaml
func (in *Installer) handleEndpointsInput(configInstall apiv1.Install) error {
	if !configInstall.SkipEtcdEndpointsValidation {
		if err := in.validateEtcd(configInstall); err != nil {
			return err
		}
	}
	endpointPatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/spec/kvBackend/address",
		Value: configInstall.EtcdEndpoints,
	}

	fsClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "name")
	if err != nil {
		return err
	}

	err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{endpointPatch})

	return err
}

// validateEtcd:
// - creates the etcd-shell pod (TLS or non-TLS)
// - deletes the etcd-shell pod (deferred)
// - prompts the user for endpoints input if required
// - validates the endpoints using the etcd-shell-pod
func (in *Installer) validateEtcd(configInstall apiv1.Install) error {
	var err error
	etcdShell := etcdShellPod
	if configInstall.EtcdTLSEnabled {
		etcdShell, err = in.tlsValidationPrep(configInstall)
		if err != nil {
			return err
		}
	}

	if err = in.kubectlClient.Apply(context.TODO(), "", string(etcdShell), true); err != nil {
		return errors.WithStack(err)
	}

	defer func() error {
		if err = in.kubectlClient.Delete(context.TODO(), "", string(etcdShell), true); err != nil {
			return err
		}
		return nil
	}()

	err = in.validateEndpoints(configInstall.EtcdEndpoints, string(etcdShell), configInstall.EtcdTLSEnabled)

	return err
}

// tlsValidationPrep:
// - searches for the etcd-secret
// - applies app=storageos label to secret
// - returns the tls equipped etcd-shell pod with storageos cluster namespace and secret name
func (in *Installer) tlsValidationPrep(configInstall apiv1.Install) (string, error) {
	if err := pluginutils.NamespaceExists(in.clientConfig, configInstall.StorageOSClusterNamespace); err != nil {
		return "", errors.WithStack(fmt.Errorf(errNSForSecretNotFound, configInstall.StorageOSClusterNamespace, configInstall.EtcdSecretName, SkipEtcdEndpointsValFlag))
	}
	etcdSecret, err := pluginutils.GetSecret(in.clientConfig, configInstall.EtcdSecretName, configInstall.StorageOSClusterNamespace)
	if err != nil {
		return "", fmt.Errorf(errSecretNotFound, configInstall.EtcdSecretName, configInstall.StorageOSClusterNamespace, SkipEtcdEndpointsValFlag)
	}

	// apply app=storageos label to secret, this way it will be backed up locally during uninstall
	secretLabels := etcdSecret.GetLabels()
	secretLabels["app"] = "storageos"
	etcdSecret.SetLabels(secretLabels)
	etcdSecretManifest, err := secretToManifest(etcdSecret)
	if err != nil {
		return "", err
	}
	if err = in.kubectlClient.Apply(context.TODO(), configInstall.StorageOSClusterNamespace, string(etcdSecretManifest), true); err != nil {
		return "", errors.WithStack(err)
	}

	etcdShell := etcdShellPodTLS
	etcdShell, err = pluginutils.SetFieldInManifest(etcdShell, configInstall.StorageOSClusterNamespace, "namespace", "metadata")
	if err != nil {
		return "", err
	}
	etcdShell, err = pluginutils.SetFieldInManifest(etcdShell, configInstall.EtcdSecretName, "secretName", "spec", "volumes", "[name=etcd-certs]", "secret")
	if err != nil {
		return "", err
	}

	return etcdShell, nil
}

// validateEndpoints:
// - retrieves etcd-shell pod name and namespace
// - ensures the etcd-shell pod is in running state
// - performs etcdctlHealthCheck
// - if no error has occurred during health check, the endpoints are validated
func (in *Installer) validateEndpoints(endpoints, etcdShell string, tlsEnabled bool) error {
	etcdShellPodName, err := pluginutils.GetFieldInManifest(etcdShell, "metadata", "name")
	if err != nil {
		return err
	}
	etcdShellPodNS, err := pluginutils.GetFieldInManifest(etcdShell, "metadata", "namespace")
	if err != nil {
		return err
	}

	if err = pluginutils.WaitFor(func() error {
		return pluginutils.IsPodRunning(in.clientConfig, etcdShellPodName, etcdShellPodNS)
	}, 60, 5); err != nil {
		return err
	}
	err = in.etcdctlHealthCheck(etcdShellPodName, etcdShellPodNS, endpointsSplitter(endpoints, tlsEnabled), tlsEnabled)

	return err
}

// etcdctlHealthCheck performs write, read, delete of key/value to etcd endpoints, returning an error
// if any step fails.
func (in *Installer) etcdctlHealthCheck(etcdShellPodName, etcdShellPodNS string, endpoints []string, tls bool) error {
	for _, endpoint := range endpoints {
		errStr := fmt.Sprintf("%s%s", "failed to validate ETCD endpoint: ", endpoint)

		// use dummy key/value pair 'foo'/'bar' to write to, read from & delete from etcd
		// in order to validate each endpoint
		key, value := "foo", "bar"
		_, stderr, err := pluginutils.ExecToPod(in.clientConfig, etcdctlPutCmd(endpoint, key, value, tls), "", etcdShellPodName, etcdShellPodNS, nil)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))
		}
		if stderr != "" {
			return errors.WithStack(fmt.Errorf(stderr))
		}

		_, stderr, err = pluginutils.ExecToPod(in.clientConfig, etcdctlGetCmd(endpoint, key, tls), "", etcdShellPodName, etcdShellPodNS, nil)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))
		}
		if stderr != "" {
			return errors.WithStack(fmt.Errorf(stderr))
		}

		_, stderr, err = pluginutils.ExecToPod(in.clientConfig, etcdctlDelCmd(endpoint, key, tls), "", etcdShellPodName, etcdShellPodNS, nil)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))
		}
		if stderr != "" {
			return errors.WithStack(fmt.Errorf(stderr))
		}
		logger.Printf("\nETCD endpoint %s successfully validated\n\n", endpoint)
	}
	return nil
}
