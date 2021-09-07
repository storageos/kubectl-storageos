package cli

import (
	"fmt"
	"strconv"

	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/installer"
	"github.com/storageos/kubectl-storageos/pkg/version"
)

func InstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "install",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Install StorageOS Cluster Operator",
		Long:         `Install StorageOS Cluster Operator`,
		SilenceUsage: true,
		PreRun:       func(cmd *cobra.Command, args []string) {},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := installCmd(cmd)
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
	cmd.Flags().String(installer.EtcdNamespaceFlag, "", "namespace of etcd operator and cluster to be installed")
	cmd.Flags().String(installer.StosOperatorNSFlag, version.GetDefaultNamespace(), "namespace of storageos operator to be installed")
	cmd.Flags().String(installer.StosClusterNSFlag, "", "namespace of storageos cluster to be installed")
	cmd.Flags().String(installer.StorageClassFlag, "", "name of storage class to be used by etcd cluster")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func installCmd(cmd *cobra.Command) error {
	v := viper.GetViper()

	logger.SetQuiet(v.GetBool("quiet"))
	ksConfig := &apiv1.KubectlStorageOSConfig{}
	err := setInstallValues(cmd, ksConfig)
	if err != nil {
		return err
	}
	cliInstaller, err := installer.NewInstaller(ksConfig, true)
	if err != nil {
		return err
	}

	err = cliInstaller.Install(ksConfig)
	if err != nil {
		return err
	}

	return nil
}

func setInstallValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
	viper.BindPFlag(installer.ConfigPathFlag, cmd.Flags().Lookup(installer.ConfigPathFlag))
	v := viper.GetViper()
	viper.SetConfigName("kubectl-storageos-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(v.GetString(installer.ConfigPathFlag))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; set fields in new config object directly
			config.Spec.Install.StorageOSOperatorYaml = cmd.Flags().Lookup(installer.StosOperatorYamlFlag).Value.String()
			config.Spec.Install.StorageOSClusterYaml = cmd.Flags().Lookup(installer.StosClusterYamlFlag).Value.String()
			config.Spec.Install.EtcdOperatorYaml = cmd.Flags().Lookup(installer.EtcdOperatorYamlFlag).Value.String()
			config.Spec.Install.EtcdClusterYaml = cmd.Flags().Lookup(installer.EtcdClusterYamlFlag).Value.String()
			config.Spec.Install.SkipEtcd, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipEtcdFlag).Value.String())
			config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
			config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installer.StosClusterNSFlag).Value.String()
			config.Spec.Install.EtcdNamespace = cmd.Flags().Lookup(installer.EtcdNamespaceFlag).Value.String()
			config.Spec.Install.EtcdEndpoints = cmd.Flags().Lookup(installer.EtcdEndpointsFlag).Value.String()
			config.Spec.Install.StorageClassName = cmd.Flags().Lookup(installer.StorageClassFlag).Value.String()
			config.InstallerMeta.StorageOSSecretYaml = ""
			return nil

		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error discovered in config file: %v", err)
		}
	}
	// config file read without error, set fields in new config object
	config.Spec.Install.StorageOSOperatorYaml = toString(viper.Get(installer.StosOperatorYamlConfig))
	config.Spec.Install.StorageOSClusterYaml = toString(viper.Get(installer.StosClusterYamlConfig))
	config.Spec.Install.EtcdOperatorYaml = toString(viper.Get(installer.EtcdOperatorYamlConfig))
	config.Spec.Install.EtcdClusterYaml = toString(viper.Get(installer.EtcdClusterYamlConfig))
	config.Spec.Install.SkipEtcd = toBool(viper.Get(installer.InstallSkipEtcdConfig))
	config.Spec.Install.StorageOSOperatorNamespace = toStringOrdefault(viper.Get(installer.InstallStosOperatorNSConfig), version.GetDefaultNamespace())
	config.Spec.Install.StorageOSClusterNamespace = toString(viper.Get(installer.InstallStosClusterNSConfig))
	config.Spec.Install.EtcdNamespace = toString(viper.Get(installer.InstallEtcdNamespaceConfig))
	config.Spec.Install.EtcdEndpoints = toString(viper.Get(installer.EtcdEndpointsConfig))
	config.Spec.Install.StorageClassName = toString(viper.Get(installer.StorageClassConfig))
	config.InstallerMeta.StorageOSSecretYaml = ""
	return nil
}

func toBool(value interface{}) bool {
	if value != nil {
		return value.(bool)
	}
	return false
}

func toString(value interface{}) string {
	if value != nil {
		return value.(string)
	}
	return ""
}

func toStringOrdefault(value interface{}, def string) string {
	if value != nil {
		return value.(string)
	}
	return def
}
