package cli

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
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

func stringToBase64(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}
