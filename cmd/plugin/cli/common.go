package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"github.com/storageos/kubectl-storageos/pkg/version"
	"k8s.io/apimachinery/pkg/api/resource"
)

// etcdEndpointsPrompt uses promptui to prompt the user to enter etcd endpoints. The internal validate
// func is run on each character as it is entered as per the regexp - it does not refer to actual
// endpoint validation which is handled later.
func etcdEndpointsPrompt() (string, error) {
	logger.Printf("   Please enter ETCD endpoints. If more than one endpoint exists, enter endpoints as a comma-delimited list of machine addresses in the cluster.\n\n   Example: 10.42.15.23:2379,10.42.12.22:2379,10.42.13.16:2379\n\n")
	validate := func(input string) error {
		match, _ := regexp.MatchString("^[a-z0-9,.:-]+$", input)
		if !match {
			return errors.New("invalid entry")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "ETCD endpoint(s)",
		Validate: validate,
	}

	return pluginutils.AskUser(prompt)
}

// skipNamespaceDeletionPrompt uses promptui to prompt the user to enter decision of skipping namespace deletion
func skipNamespaceDeletionPrompt() (bool, error) {
	logger.Printf("   Please confirm namespace deletion.\n   Warning: protected namespaces (default, kube-system, kube-node-lease, kube-public) cannot be deleted by kubectl-storageos.")
	yesValues := map[string]bool{
		"y":   true,
		"yes": true,
	}
	noValues := map[string]bool{
		"":   true,
		"n":  true,
		"no": true,
	}

	validate := func(input string) error {
		ilc := strings.ToLower(input)
		_, yes := yesValues[ilc]
		_, no := noValues[ilc]

		if !yes && !no {
			return errors.New("invalid input")
		}

		return nil
	}
	prompt := promptui.Prompt{
		Label:    "Skip namespace deletion [y/N]",
		Validate: validate,
	}

	input, err := pluginutils.AskUser(prompt)
	if err != nil {
		return false, err
	}

	ilc := strings.ToLower(input)
	_, yes := yesValues[ilc]

	return yes, nil
}

// storageClassPrompt uses promptui the user to enter the etcd storage class name
func storageClassPrompt() (string, error) {
	logger.Printf("   Please enter the name of the storage class used by the ETCD cluster\n\n")
	validate := func(input string) error {
		match, _ := regexp.MatchString("^[a-z0-9.-]+$", input)
		if !match {
			return errors.New("invalid entry - must consist only of lowercase alphanumeric characters, '-', or '.'")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "ETCD storage class name",
		Validate: validate,
	}

	return pluginutils.AskUser(prompt)
}

// k8sVersionPrompt uses promptui the user to enter the kubernetes version of the target cluster
func k8sVersionPrompt() (string, error) {
	logger.Printf("   Please enter the version of the target Kubernetes cluster\n\n")
	validate := func(input string) error {
		match, _ := regexp.MatchString("v[0-9]+.[0-9]+.[0-9]+", input)
		if !match {
			return errors.New("invalid entry, example of valid entry: 'v1.23.0'")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Version of target Kubernetes cluster",
		Validate: validate,
	}

	return pluginutils.AskUser(prompt)
}

func valueOrDefault(value string, def string) string {
	if value != "" {
		return value
	}
	return def
}

func versionSupportsFeature(existingOperatorVersion, featureFirstSupportedVersion string) error {
	if version.IsDevelop(existingOperatorVersion) {
		return nil
	}

	supported, err := version.IsSupported(existingOperatorVersion, featureFirstSupportedVersion)
	if err != nil {
		return err
	}
	if !supported {
		return fmt.Errorf("feature is not supported in StorageOS %s, requires minimum version %s. ", existingOperatorVersion, featureFirstSupportedVersion)
	}

	return nil
}

func validateResourceLimit(resourceLimit string) error {
	_, err := resource.ParseQuantity(resourceLimit)
	if err != nil {
		return err
	}
	return nil
}
