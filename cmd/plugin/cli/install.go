package cli

import (
	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/storageos/kubectl-storageos/pkg/install"
)

func InstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "install",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Install StorageOS Cluster Operator",
		Long:         `Install StorageOS Cluster Operator`,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag(install.StosOperatorYamlFlag, cmd.Flags().Lookup(install.StosOperatorYamlFlag))
			viper.BindPFlag(install.StosClusterYamlFlag, cmd.Flags().Lookup(install.StosClusterYamlFlag))
			viper.BindPFlag(install.SkipEtcdInstallFlag, cmd.Flags().Lookup(install.SkipEtcdInstallFlag))
			viper.BindPFlag(install.EtcdOperatorYamlFlag, cmd.Flags().Lookup(install.EtcdOperatorYamlFlag))
			viper.BindPFlag(install.EtcdClusterYamlFlag, cmd.Flags().Lookup(install.EtcdClusterYamlFlag))

		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			logger.SetQuiet(v.GetBool("quiet"))

			installer, err := install.NewInstaller()
			if err != nil {
				return err
			}

			err = installer.Install()
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().String(install.StosOperatorYamlFlag, "", "path to storageos-operator.yaml")
	cmd.Flags().String(install.StosClusterYamlFlag, "", "path to storageos-cluster.yaml")
	cmd.Flags().String(install.EtcdClusterYamlFlag, "", "path to etcd-cluster.yaml")
	cmd.Flags().String(install.EtcdOperatorYamlFlag, "", "path to etcd-operator.yaml")
	cmd.Flags().Bool(install.SkipEtcdInstallFlag, false, "skip etcd installation and enter endpoints manually")

	viper.BindPFlags(cmd.Flags())

	return cmd
}
