package install

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/manifoldco/promptui"
	"github.com/storageos/kubectl-storageos/pkg/logger"
)

const (
	etcdShellUrl = "https://raw.githubusercontent.com/nolancon/placeholder/main/storageos-etcd-shell.yaml"
)

// handleEndpointsInput:
// - pulls and creates the etcd-shell pod
// - prompts the user for endpoints input
// - validates the endpoints using the etcd-shell-pod
// - deletes the etcd-shell pod
// - adds validated endpoints to storageos-cluster.spec.kvBackend
func (in *Installer) handleEndpointsInput() error {
	etcdShell, err := pullManifest(etcdShellUrl)
	if err != nil {
		return err
	}
	err = in.kubectlClient.Apply(context.TODO(), "", string(etcdShell), true)
	if err != nil {
		return err
	}

	etcdEndpoints, err := etcdEndpointsPrompt()
	if err != nil {
		return err
	}

	err = validateEndpoints(etcdEndpoints, string(etcdShell))
	if err != nil {
		err = in.kubectlClient.Delete(context.TODO(), "", string(etcdShell), true)
		if err != nil {
			return err
		}

		return err
	}

	err = in.kubectlClient.Delete(context.TODO(), "", string(etcdShell), true)
	if err != nil {
		return err
	}

	err = in.setFieldInFsManifest(filepath.Join(stosDir, clusterDir, stosClusterFile), etcdEndpoints, "address", "spec", "kvBackend")
	if err != nil {
		return err
	}

	return nil
}

// validateEndpoints:
// - retrieves etcd-shell pod name and namespace
// - ensures the etcd-shell pod is in running state
// - execs into the etcd-shell pod and runs etcdctl to list the etcd cluster members
//   TODO: do read/write to etcd instead of members list
// - if no error has occurred, the endpoints are validated
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
	stdout, stderr, err := ExecToPod(config, etcdctlMemberListCmd(endpoints), "", etcdShellPodName, etcdShellPodNS, nil)
	if err != nil {
		return err
	}
	if stderr != "" {
		return fmt.Errorf(stderr)
	}
	logger.Printf("\n\nEndpoints succesfully validated, etcd members list:\n\n%s\n\n", stdout)

	return nil
}

// etcdctlMemberList returns a slice of strings representing the etcdctl command for members list to
// be interpreted by the pod exec:
// {`/bin/bash`, `-c`, `etcdctl --endpoints "http://<endpoints>" member list`}
func etcdctlMemberListCmd(endpoints string) []string {
	return []string{"/bin/bash", "-c", fmt.Sprintf("%s%s%s", "etcdctl --endpoints \"http://", endpoints, "\" member list")}
}

// etcdEndpointsPrompt uses promptui to prompt the user to enter etcd endpoints. The internal validate
// func is run on each character as it is entered as per the regexp - it does not refer to actual
// endpoint validation which is handled later.
func etcdEndpointsPrompt() (string, error) {
	logger.Printf("   Please enter ETCD endpoint(s). If more than one endpoint exists, enter endpoints as a comma-separated string.\n\n   Example: 10.42.15.23:2379,10.42.12.22:2379,10.42.13.16:2379\n\n")
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
