package cli

import (
	"strings"

	"github.com/replicatedhq/troubleshoot/pkg/k8sutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/storageos/kubectl-storageos/pkg/troubleshoot"
)

const (
	defaultSpec = "https://raw.githubusercontent.com/storageos/kubectl-storageos/master/config/bundle.yaml"
)

func BundleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle [url]",
		Args:  cobra.MinimumNArgs(0),
		Short: "Generate a support bundle",
		Long: `A support bundle is an archive of files, output, metrics and state
from a server that can be used to assist when troubleshooting a Kubernetes cluster.`,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			// logger.SetQuiet(v.GetBool("quiet"))

			spec := defaultSpec
			if len(args) > 0 {
				spec = args[0]
			}
			return troubleshoot.Run(v, spec)
		},
	}

	// cobra.OnInitialize(initConfig)

	cmd.Flags().StringSlice("redactors", []string{}, "names of the additional redactors to use")
	cmd.Flags().Bool("redact", true, "enable/disable default redactions")
	cmd.Flags().Bool("collect-without-permissions", false, "always generate a support bundle, even if it some require additional permissions")

	// hidden in favor of the `insecure-skip-tls-verify` flag
	cmd.Flags().Bool("allow-insecure-connections", false, "when set, do not verify TLS certs when retrieving spec and reporting results")
	cmd.Flags().MarkHidden("allow-insecure-connections")

	viper.BindPFlags(cmd.Flags())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	k8sutil.AddFlags(cmd.Flags())

	return cmd
}
