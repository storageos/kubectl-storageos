package installer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/logger"

	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
)

const (
	errSecretNotFound = `
	Unable to find etcd client secret %s in namespace %s for ETCD endpoint validation.

	Please create a k8s secret in the StorageOS cluster namespace like so:

	kubectl create secret generic <etcd-secret-name> -n <storageos-cluster-namespace> \
		--from-file=etcd-client-ca.crt=path/to/ca.crt \
		--from-file=etcd-client.crt=path/to/tls.crt \
		--from-file=etcd-client.key=path/to/tls.key


	If you have uninstalled StorageOS using kubectl-storageos since last creating your secret, check 
	
	$HOME/.kube/storageos/uninstall-<cluster-id>/storageos-secrets.yaml for a local backup.
	


	To skip ETCD endpoints validation during installation, set install flag --%s`

	errFailedToValidateTLSEndpoint = `
	Unable to validate ETCD endpoint %s 

	Please ensure this endpoint is TLS-enabled.

	Please note that, due to a known limitation, if your kubernetes cluster was provisioned using Google Anthos,

	kubectl storageos is unable to perform ETCD endpoint validation.

	To skip ETCD endpoints validation during installation, set install flag --%s

`

	errFailedToValidateEndpoint = `
	Unable to validate ETCD endpoint %s 

	If ETCD endpoints are TLS-enabled, please set install flag --%s

	Please note that, due to a known limitation, if your kubernetes cluster was provisioned using Google Anthos,

	kubectl storageos is unable to perform ETCD endpoint validation.

	To skip ETCD endpoints validation during installation, set install flag --%s
	
`

	endpointsValidatedMessage = `
	ETCD endpoint(s) %s successfully validated.

`
	etcdShellPodDeletionFailMessage = `
	Failed to cleanup etcd shell pod with error %v, 
	please delete pod manually after installaion is complete.
`

	etcdShellPod = `apiVersion: v1
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
func (in *Installer) handleEndpointsInput(configInstall apiv1.KubectlStorageOSConfigSpec) error {
	if !configInstall.Install.SkipEtcdEndpointsValidation {
		if err := in.validateEtcd(configInstall); err != nil {
			return err
		}
	}
	if configInstall.SkipStorageOSCluster {
		return nil
	}
	endpointPatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/spec/kvBackend/address",
		Value: configInstall.Install.EtcdEndpoints,
	}

	fsClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "name")
	if err != nil {
		return err
	}

	return in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{endpointPatch})
}

// validateEtcd:
// - creates the etcd-shell pod (TLS or non-TLS)
// - deletes the etcd-shell pod (deferred)
// - prompts the user for endpoints input if required
// - validates the endpoints using the etcd-shell-pod
func (in *Installer) validateEtcd(configSpec apiv1.KubectlStorageOSConfigSpec) error {
	var err error
	etcdShell := etcdShellPod

	etcdNS := configSpec.GetETCDValidationNamespace()
	etcdShell, err = pluginutils.SetFieldInManifest(etcdShell, etcdNS, "namespace", "metadata")
	if err != nil {
		return err
	}

	if configSpec.Install.EtcdTLSEnabled {
		etcdShell, err = in.tlsValidationPrep(etcdNS, configSpec.Install)
		if err != nil {
			return err
		}
	}

	if err = in.kubectlClient.Apply(context.TODO(), "", string(etcdShell), true); err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		if err = in.kubectlClient.Delete(context.TODO(), "", string(etcdShell), true); err != nil {
			// do nothing, etcd shell pod runs to completion even in unlikely event that delete fails
			logger.Printf(etcdShellPodDeletionFailMessage, err)
		}
	}()

	err = in.validateEndpoints(configSpec.Install.EtcdEndpoints, string(etcdShell), configSpec.Install.EtcdTLSEnabled)

	return err
}

// tlsValidationPrep:
// - searches for the etcd-secret
// - applies app=storageos label to secret
// - returns the tls equipped etcd-shell pod with storageos cluster namespace and secret name
func (in *Installer) tlsValidationPrep(namespace string, configInstall apiv1.Install) (string, error) {
	etcdSecret, err := pluginutils.GetSecret(in.clientConfig, configInstall.EtcdSecretName, namespace)
	if err != nil {
		return "", fmt.Errorf(errSecretNotFound, configInstall.EtcdSecretName, namespace, SkipEtcdEndpointsValFlag)
	}

	// apply app=storageos label to secret, this way it will be backed up locally during uninstall
	secretLabels := etcdSecret.GetLabels()
	secretLabels["app"] = "storageos"
	etcdSecret.SetLabels(secretLabels)
	etcdSecretManifest, err := secretToManifest(etcdSecret)
	if err != nil {
		return "", err
	}
	if err = in.kubectlClient.Apply(context.TODO(), namespace, string(etcdSecretManifest), true); err != nil {
		return "", errors.WithStack(err)
	}

	etcdShell := etcdShellPodTLS
	etcdShell, err = pluginutils.SetFieldInManifest(etcdShell, namespace, "namespace", "metadata")
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
		errStr := fmt.Sprintf(errFailedToValidateEndpoint, endpoint, EtcdTLSEnabledFlag, SkipEtcdEndpointsValFlag)
		if tls {
			errStr = fmt.Sprintf(errFailedToValidateTLSEndpoint, endpoint, SkipEtcdEndpointsValFlag)
		}

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
	}
	logger.Printf(endpointsValidatedMessage, strings.Join(endpoints, ","))

	return nil
}
