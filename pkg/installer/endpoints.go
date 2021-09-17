package installer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"k8s.io/client-go/rest"
)

const (
	etcdShellUrl = "https://raw.githubusercontent.com/nolancon/placeholder/main/storageos-etcd-shell.yaml"
	httpsPrefix  = "https://"
)

// handleEndpointsInput:
// - pulls and creates the etcd-shell pod
// - deletes the etcd-shell pod (deferred)
// - prompts the user for endpoints input if required
// - validates the endpoints using the etcd-shell-pod
// - adds validated endpoints patch to kustomization file for storageos-cluster.yaml
func (in *Installer) handleEndpointsInput(etcdEndpoints string) error {
	fmt.Println("Warning: TLS endpoints are not supported")
	if strings.HasPrefix(etcdEndpoints, httpsPrefix) {
		return fmt.Errorf("TLS endpoint discovered")
	}

	etcdShell, err := pullManifest(etcdShellUrl)
	if err != nil {
		return err
	}
	err = in.kubectlClient.Apply(context.TODO(), "", string(etcdShell), true)
	if err != nil {
		return err
	}

	defer func() error {
		err = in.kubectlClient.Delete(context.TODO(), "", string(etcdShell), true)
		if err != nil {
			return err
		}
		return nil
	}()

	err = validateEndpoints(etcdEndpoints, string(etcdShell))
	if err != nil {
		return err
	}

	endpointPatch := pluginutils.KustomizePatch{
		Op:    "replace",
		Path:  "/spec/kvBackend/address",
		Value: etcdEndpoints,
	}

	fsClusterName, err := in.getFieldInFsMultiDocByKind(filepath.Join(stosDir, clusterDir, stosClusterFile), stosClusterKind, "metadata", "name")
	if err != nil {
		return err
	}

	err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, fsClusterName, []pluginutils.KustomizePatch{endpointPatch})
	if err != nil {
		return err
	}

	return nil
}

// validateEndpoints:
// - retrieves etcd-shell pod name and namespace
// - ensures the etcd-shell pod is in running state
// - performs etcdctlHealthCheck
// - if no error has occurred during health check, the endpoints are validated
func validateEndpoints(endpoints, etcdShell string) error {
	etcdShellPodName, err := pluginutils.GetFieldInManifest(etcdShell, "metadata", "name")
	if err != nil {
		return err
	}
	etcdShellPodNS, err := pluginutils.GetFieldInManifest(etcdShell, "metadata", "namespace")
	if err != nil {
		return err
	}

	config, err := pluginutils.NewClientConfig()
	if err != nil {
		return err
	}

	err = pluginutils.WaitFor(func() error {
		return pluginutils.IsPodRunning(config, etcdShellPodName, etcdShellPodNS)
	}, 60, 5)
	if err != nil {
		return err
	}

	err = etcdctlHealthCheck(config, etcdShellPodName, etcdShellPodNS, endpointsSplitter(endpoints), "foo", "bar")
	if err != nil {
		return err
	}

	logger.Printf("\nETCD endpoints successfully validated\n\n")

	return nil
}

// etcdctlHelathCheck performs write, read, delete of key/value to etcd endpoints, returning an error
// if any step fails.
func etcdctlHealthCheck(config *rest.Config, etcdShellPodName, etcdShellPodNS, endpoints, key, value string) error {
	errStr := fmt.Sprintf("%s%s", "failed to validate ETCD endpoints: ", endpoints)
	_, stderr, err := pluginutils.ExecToPod(config, etcdctlPutCmd(endpoints, key, value), "", etcdShellPodName, etcdShellPodNS, nil)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))
	}
	if stderr != "" {
		return fmt.Errorf(stderr)
	}

	_, stderr, err = pluginutils.ExecToPod(config, etcdctlGetCmd(endpoints, key), "", etcdShellPodName, etcdShellPodNS, nil)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))

	}
	if stderr != "" {
		return fmt.Errorf(stderr)
	}

	_, stderr, err = pluginutils.ExecToPod(config, etcdctlDelCmd(endpoints, key), "", etcdShellPodName, etcdShellPodNS, nil)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))
	}
	if stderr != "" {
		return fmt.Errorf(stderr)
	}

	return nil
}
