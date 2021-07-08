package install

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/manifoldco/promptui"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	"k8s.io/client-go/rest"
)

const (
	etcdShellUrl = "https://raw.githubusercontent.com/nolancon/placeholder/main/storageos-etcd-shell.yaml"
)

// handleEndpointsInput:
// - pulls and creates the etcd-shell pod
// - deletes the etcd-shell pod (deferred)
// - prompts the user for endpoints input if required
// - validates the endpoints using the etcd-shell-pod
// - adds validated endpoints patch to kustomization file for storageos-cluster.yaml
func (in *Installer) handleEndpointsInput(etcdEndpoints string) error {
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

	// if etcdEndpoints were not passed via flag or config, prompt user to enter manually
	if etcdEndpoints == "" {
		etcdEndpoints, err = etcdEndpointsPrompt()
		if err != nil {
			return err
		}
	}
	err = validateEndpoints(etcdEndpoints, string(etcdShell))
	if err != nil {
		return err
	}

	endpointPatch := KustomizePatch{
		Op:    "replace",
		Path:  "/spec/kvBackend/address",
		Value: etcdEndpoints,
	}

	err = in.addPatchesToFSKustomize(filepath.Join(stosDir, clusterDir, kustomizationFile), stosClusterKind, defaultStosClusterName, []KustomizePatch{endpointPatch})
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
	etcdShellPodName, err := GetFieldInManifest(etcdShell, "metadata", "name")
	if err != nil {
		return err
	}
	etcdShellPodNS, err := GetFieldInManifest(etcdShell, "metadata", "namespace")
	if err != nil {
		return err
	}

	config, err := NewClientConfig()
	if err != nil {
		return err
	}

	err = PodIsRunning(config, etcdShellPodName, etcdShellPodNS)
	if err != nil {
		return err
	}

	err = etcdctlHealthCheck(config, etcdShellPodName, etcdShellPodNS, endpointsSplitter(endpoints), "foo", "bar")
	if err != nil {
		return err
	}

	logger.Printf("\nETCD endpoints succesfully validated\n\n")

	return nil
}

// etcdctlHelathCheck performs write, read, delete of key/value to etcd endpoints, returning an error
// if any step fails.
func etcdctlHealthCheck(config *rest.Config, etcdShellPodName, etcdShellPodNS, endpoints, key, value string) error {
	errStr := fmt.Sprintf("%s%s", "failed to validate ETCD endpoints: ", endpoints)
	_, stderr, err := ExecToPod(config, etcdctlSetCmd(endpoints, key, value), "", etcdShellPodName, etcdShellPodNS, nil)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))
	}
	if stderr != "" {
		return fmt.Errorf(stderr)
	}

	_, stderr, err = ExecToPod(config, etcdctlGetCmd(endpoints, key), "", etcdShellPodName, etcdShellPodNS, nil)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))

	}
	if stderr != "" {
		return fmt.Errorf(stderr)
	}

	_, stderr, err = ExecToPod(config, etcdctlRmCmd(endpoints, key), "", etcdShellPodName, etcdShellPodNS, nil)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s%v", errStr, err))
	}
	if stderr != "" {
		return fmt.Errorf(stderr)
	}

	return nil
}

// etcdEndpointsPrompt uses promptui to prompt the user to enter etcd endpoints. The internal validate
// func is run on each character as it is entered as per the regexp - it does not refer to actual
// endpoint validation which is handled later.
func etcdEndpointsPrompt() (string, error) {
	logger.Printf("   Please enter ETCD endpoints. If more than one endpoint exists, enter endpoints as a comma-delimited list of machine addresses in the cluster.\n\n   Example: 10.42.15.23:2379,10.42.12.22:2379,10.42.13.16:2379\n\n")
	validate := func(input string) error {
		match, _ := regexp.MatchString("^[a-z0-9,.:-]+$", input)
		if !match {
			return errors.New("Invalid entry")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "ETCD endpoint(s)",
		Validate: validate,
	}

	result, err := prompt.Run()
	if err != nil {
		logger.Printf("Prompt failed %v\n", err)
		return "", err
	}

	return result, nil
}