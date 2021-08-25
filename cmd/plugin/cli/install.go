package cli

import (
	"fmt"

	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/storageos/kubectl-storageos/pkg/installer"
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

			cliInstaller, err := installer.NewInstaller()
			if err != nil {
				return err
			}

			err = cliInstaller.Install()
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().String(installer.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url")
	cmd.Flags().String(installer.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url")
	cmd.Flags().String(installer.EtcdClusterYamlFlag, "", "etcd-cluster.yaml path or url")
	cmd.Flags().String(installer.EtcdOperatorYamlFlag, "", "etcd-operator.yaml path or url")
	cmd.Flags().Bool(installer.SkipEtcdFlag, false, "skip etcd installation and enter endpoints manually")
	cmd.Flags().String(installer.EtcdEndpointsFlag, "", "etcd endpoints")
	cmd.Flags().String(installer.ConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(installer.EtcdNamespaceFlag, "", "namespace of etcd operator and cluster")
	cmd.Flags().String(installer.StosOperatorNSFlag, "", "namespace of storageos operator")
	cmd.Flags().String(installer.StosClusterNSFlag, "", "namespace of storageos cluster")
	cmd.Flags().String(installer.StorageClassFlag, "", "name of storage class to be used by etcd cluster")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func setFlags(cmd *cobra.Command) {
	viper.BindPFlag(installer.ConfigPathFlag, cmd.Flags().Lookup(installer.ConfigPathFlag))
	v := viper.GetViper()
	viper.SetConfigName("kubectl-storageos-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(v.GetString(installer.ConfigPathFlag))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; set flags directly
			viper.BindPFlag(installer.StosOperatorYamlFlag, cmd.Flags().Lookup(installer.StosOperatorYamlFlag))
			viper.BindPFlag(installer.StosClusterYamlFlag, cmd.Flags().Lookup(installer.StosClusterYamlFlag))
			viper.BindPFlag(installer.SkipEtcdFlag, cmd.Flags().Lookup(installer.SkipEtcdFlag))
			viper.BindPFlag(installer.EtcdEndpointsFlag, cmd.Flags().Lookup(installer.EtcdEndpointsFlag))
			viper.BindPFlag(installer.EtcdOperatorYamlFlag, cmd.Flags().Lookup(installer.EtcdOperatorYamlFlag))
			viper.BindPFlag(installer.EtcdClusterYamlFlag, cmd.Flags().Lookup(installer.EtcdClusterYamlFlag))
			viper.BindPFlag(installer.EtcdNamespaceFlag, cmd.Flags().Lookup(installer.EtcdNamespaceFlag))
			viper.BindPFlag(installer.StosOperatorNSFlag, cmd.Flags().Lookup(installer.StosOperatorNSFlag))
			viper.BindPFlag(installer.StosClusterNSFlag, cmd.Flags().Lookup(installer.StosClusterNSFlag))
			viper.BindPFlag(installer.StorageClassFlag, cmd.Flags().Lookup(installer.StorageClassFlag))

			return

		} else {
			// Config file was found but another error was produced
			panic(fmt.Errorf("error discovered in config file: %v", err))
		}
	}
	// config file read without error, set flags from config
	viper.Set(installer.SkipEtcdFlag, viper.Get(installer.SkipEtcdConfig))
	viper.Set(installer.EtcdEndpointsFlag, viper.Get(installer.EtcdEndpointsConfig))
	viper.Set(installer.StosOperatorYamlFlag, viper.Get(installer.StosOperatorYamlConfig))
	viper.Set(installer.StosClusterYamlFlag, viper.Get(installer.StosClusterYamlConfig))
	viper.Set(installer.EtcdOperatorYamlFlag, viper.Get(installer.EtcdOperatorYamlConfig))
	viper.Set(installer.EtcdNamespaceFlag, viper.Get(installer.EtcdNamespaceConfig))
	viper.Set(installer.StosOperatorNSFlag, viper.Get(installer.StosOperatorNSConfig))
	viper.Set(installer.StosClusterNSFlag, viper.Get(installer.StosClusterNSConfig))
	viper.Set(installer.StorageClassFlag, viper.Get(installer.StorageClassConfig))

	viper.BindPFlags(cmd.Flags())
}
