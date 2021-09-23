package cli

import (
	"strings"

	"github.com/replicatedhq/troubleshoot/pkg/k8sutil"
	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/storageos/kubectl-storageos/pkg/preflight"
)

const (
	// defaultCollectorImage can be removed once
	// https://github.com/replicatedhq/troubleshoot/pull/392 merges, then it
	// will default to "replicatedhq/troubleshoot:latest".
	defaultCollectorImage = "storageos/troubleshoot:c77d9dc"

	// defaultPreflightSpec will be used if not supplied by the user.
	defaultPreflightSpec = "https://raw.githubusercontent.com/storageos/kubectl-plugin/main/specs/preflight.yaml"
)

func PreflightCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "preflight [url]",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Test a k8s cluster for StorageOS pre-requisites",
		Long:         `A preflight check is a set of validations that can and should be run to ensure that a cluster meets the requirements to run StorageOS.`,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			logger.SetQuiet(v.GetBool("quiet"))

			spec := defaultPreflightSpec
			if len(args) > 0 {
				spec = args[0]
			}
			return preflight.Run(v, spec)
		},
	}

	cmd.Flags().Bool("interactive", true, "interactive preflights")
	cmd.Flags().String("format", "human", "output format, one of human, json, yaml. only used when interactive is set to false")
	cmd.Flags().String("collector-image", defaultCollectorImage, "the full name of the collector image to use")
	cmd.Flags().String("collector-pullpolicy", "", "the pull policy of the collector image")
	cmd.Flags().Bool("collect-without-permissions", false, "always run preflight checks even if some require permissions that preflight does not have")
	cmd.Flags().String("selector", "", "selector (label query) to filter remote collection nodes on.")
	cmd.Flags().String("since-time", "", "force pod logs collectors to return logs after a specific date (RFC3339)")
	cmd.Flags().String("since", "", "force pod logs collectors to return logs newer than a relative duration like 5s, 2m, or 3h.")

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	k8sutil.AddFlags(cmd.Flags())

	return cmd
}
