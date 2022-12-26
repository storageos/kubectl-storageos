package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/storageos/kubectl-storageos/cmd"
)

func main() {
	if err := RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func RootCmd() *cobra.Command {
	cobracmd := &cobra.Command{
		Use:     "kubectl-storageos",
		Aliases: []string{"kubectl storageos"},
		Short:   "StorageOS",
		Long:    `StorageOS kubectl plugin`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				_ = cmd.Help()
				os.Exit(0)
			}
		},
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	cobracmd.AddCommand(cmd.PreflightCmd())
	cobracmd.AddCommand(cmd.BundleCmd())
	cobracmd.AddCommand(cmd.InstallCmd())
	cobracmd.AddCommand(cmd.UninstallCmd())
	cobracmd.AddCommand(cmd.UpgradeCmd())
	cobracmd.AddCommand(cmd.VersionCmd())
	cobracmd.AddCommand(cmd.InstallPortalCmd())
	cobracmd.AddCommand(cmd.UninstallPortalCmd())
	cobracmd.AddCommand(cmd.EnablePortalCmd())
	cobracmd.AddCommand(cmd.DisablePortalCmd())
	cobracmd.AddCommand(cmd.ApplyCmd())
	cobracmd.AddCommand(cmd.UpdateCmd())
	cobracmd.AddCommand(cmd.CreateCmd())
	cobracmd.AddCommand(cmd.GetCmd())
	cobracmd.AddCommand(cmd.DescribeCmd())
	cobracmd.AddCommand(cmd.AttachCmd())
	cobracmd.AddCommand(cmd.DetachCmd())
	cobracmd.AddCommand(cmd.NfsCmd())
	cobracmd.AddCommand(cmd.CordonCmd())
	cobracmd.AddCommand(cmd.UncordonCmd())
	cobracmd.AddCommand(cmd.CompletionCmd)

	return cobracmd
}
