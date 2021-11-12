package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/consts"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"github.com/storageos/kubectl-storageos/pkg/version"
)

const (
	errNoUsername     = "admin-username not provided"
	errNoPassword     = "admin-password not provided"
	errNoPortalAPIURL = "portal-api-url not provided"
	errNoTenantID     = "tenant-id not provided"
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
	logger.Printf("   Please confirm namespace deletion.")
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

func valueOrDefault(value string, def string) string {
	if value != "" {
		return value
	}
	return def
}

func versionSupportsPortal(existingOperatorVersion string) error {
	// TODO: remove develop check after 2.6 release
	if !version.IsDevelop(existingOperatorVersion) {
		supported, err := version.IsSupported(existingOperatorVersion, consts.PortalManagerFirstSupportedVersion)
		if err != nil {
			return err
		}
		if !supported {
			return fmt.Errorf("Portal Manager is not supported in StorageOS %s", existingOperatorVersion)
		}
	}

	return nil
}

func portalFlagsExist(config *apiv1.KubectlStorageOSConfig) error {
	missingFlags := make([]string, 0)
	if config.Spec.Install.AdminUsername == "" {
		missingFlags = append(missingFlags, errNoUsername)
	}
	if config.Spec.Install.AdminPassword == "" {
		missingFlags = append(missingFlags, errNoPassword)
	}
	if config.Spec.Install.PortalAPIURL == "" {
		missingFlags = append(missingFlags, errNoPortalAPIURL)
	}
	if config.Spec.Install.TenantID == "" {
		missingFlags = append(missingFlags, errNoTenantID)
	}

	if len(missingFlags) != 0 {
		return fmt.Errorf(strings.Join(missingFlags, ", "))
	}
	return nil
}
