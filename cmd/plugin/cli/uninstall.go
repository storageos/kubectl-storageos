package cli

import (
	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/storageos/kubectl-storageos/pkg/installer"
)

func UninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "uninstall",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Uninstall StorageOS",
		Long:         `Uninstall StorageOS and/or ETCD`,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag(installer.StosOperatorNSFlag, cmd.Flags().Lookup(installer.StosOperatorNSFlag))
			viper.BindPFlag(installer.StosClusterNSFlag, cmd.Flags().Lookup(installer.StosClusterNSFlag))
			viper.BindPFlag(installer.EtcdNamespaceFlag, cmd.Flags().Lookup(installer.EtcdNamespaceFlag))
			viper.BindPFlag(installer.SkipEtcdFlag, cmd.Flags().Lookup(installer.SkipEtcdFlag))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			logger.SetQuiet(v.GetBool("quiet"))

			cliInstaller, err := installer.NewInstaller()
			if err != nil {
				return err
			}

			err = cliInstaller.Uninstall()
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().Bool(installer.SkipEtcdFlag, false, "uninstall storageos only, leaving ETCD untouched")
	cmd.Flags().String(installer.EtcdNamespaceFlag, "", "namespace of etcd operator and cluster")
	cmd.Flags().String(installer.StosOperatorNSFlag, "", "namespace of storageos operator")
	cmd.Flags().String(installer.StosClusterNSFlag, "", "namespace of storageos cluster")

	viper.BindPFlags(cmd.Flags())

	return cmd
}
