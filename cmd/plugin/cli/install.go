package cli

import (
	"fmt"

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
			setFlags(cmd)
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
	cmd.Flags().String(install.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url")
	cmd.Flags().String(install.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url")
	cmd.Flags().String(install.EtcdClusterYamlFlag, "", "etcd-cluster.yaml path or url")
	cmd.Flags().String(install.EtcdOperatorYamlFlag, "", "etcd-operator.yaml path or url")
	cmd.Flags().Bool(install.SkipEtcdInstallFlag, false, "skip etcd installation and enter endpoints manually")
	cmd.Flags().String(install.EtcdEndpointsFlag, "", "etcd endpoints")
	cmd.Flags().String(install.ConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(install.EtcdNamespaceFlag, "", "namespace of etcd operator and cluster")
	cmd.Flags().String(install.StosOperatorNSFlag, "", "namespace of storageos operator")
	cmd.Flags().String(install.StosClusterNSFlag, "", "namespace of storageos cluster")
	cmd.Flags().String(install.StorageClassFlag, "", "name of storage class to be used by etcd cluster")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func setFlags(cmd *cobra.Command) {
	viper.BindPFlag(install.ConfigPathFlag, cmd.Flags().Lookup(install.ConfigPathFlag))
	v := viper.GetViper()
	viper.SetConfigName("kubectl-storageos-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(v.GetString(install.ConfigPathFlag))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; set flags directly
			viper.BindPFlag(install.StosOperatorYamlFlag, cmd.Flags().Lookup(install.StosOperatorYamlFlag))
			viper.BindPFlag(install.StosClusterYamlFlag, cmd.Flags().Lookup(install.StosClusterYamlFlag))
			viper.BindPFlag(install.SkipEtcdInstallFlag, cmd.Flags().Lookup(install.SkipEtcdInstallFlag))
			viper.BindPFlag(install.EtcdEndpointsFlag, cmd.Flags().Lookup(install.EtcdEndpointsFlag))
			viper.BindPFlag(install.EtcdOperatorYamlFlag, cmd.Flags().Lookup(install.EtcdOperatorYamlFlag))
			viper.BindPFlag(install.EtcdClusterYamlFlag, cmd.Flags().Lookup(install.EtcdClusterYamlFlag))
			viper.BindPFlag(install.EtcdNamespaceFlag, cmd.Flags().Lookup(install.EtcdNamespaceFlag))
			viper.BindPFlag(install.StosOperatorNSFlag, cmd.Flags().Lookup(install.StosOperatorNSFlag))
			viper.BindPFlag(install.StosClusterNSFlag, cmd.Flags().Lookup(install.StosClusterNSFlag))
			viper.BindPFlag(install.StorageClassFlag, cmd.Flags().Lookup(install.StorageClassFlag))

			return

		}
		// Config file was found but another error was produced
		panic(fmt.Errorf("error discovered in config file: %v", err))
	}
	// config file read without error, set flags from config
	viper.Set(install.SkipEtcdInstallFlag, viper.Get(install.SkipEtcdInstallConfig))
	viper.Set(install.EtcdEndpointsFlag, viper.Get(install.EtcdEndpointsConfig))
	viper.Set(install.StosOperatorYamlFlag, viper.Get(install.StosOperatorYamlConfig))
	viper.Set(install.StosClusterYamlFlag, viper.Get(install.StosClusterYamlConfig))
	viper.Set(install.EtcdOperatorYamlFlag, viper.Get(install.EtcdOperatorYamlConfig))
	viper.Set(install.EtcdNamespaceFlag, viper.Get(install.EtcdNamespaceConfig))
	viper.Set(install.StosOperatorNSFlag, viper.Get(install.StosOperatorNSConfig))
	viper.Set(install.StosClusterNSFlag, viper.Get(install.StosClusterNSConfig))
	viper.Set(install.StorageClassFlag, viper.Get(install.StorageClassConfig))

	viper.BindPFlags(cmd.Flags())
}
