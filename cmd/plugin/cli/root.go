package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kubectl-storageos",
		Aliases: []string{"kubectl storageos"},
		Short:   "StorageOS",
		Long:    `StorageOS kubectl plugin`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(0)
			}

			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	cmd.AddCommand(PreflightCmd())
	cmd.AddCommand(BundleCmd())
	cmd.AddCommand(InstallCmd())
	cmd.AddCommand(UninstallCmd())
	cmd.AddCommand(UpgradeCmd())
	cmd.AddCommand(VersionCmd())
	cmd.AddCommand(InstallPortalCmd())
	cmd.AddCommand(UninstallPortalCmd())
	cmd.AddCommand(EnablePortalCmd())
	cmd.AddCommand(DisablePortalCmd())
	cmd.AddCommand(CompletionCmd)

	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
}
