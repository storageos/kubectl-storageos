package cli

import (
	"fmt"
	"strconv"

	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/consts"
	"github.com/storageos/kubectl-storageos/pkg/installer"
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
	cmd.Flags().String(installer.EtcdNamespaceFlag, consts.EtcdOperatorNamespace, "namespace of etcd operator and cluster to be installed")
	cmd.Flags().String(installer.StosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be installed")
	cmd.Flags().String(installer.StosClusterNSFlag, consts.NewOperatorNamespace, "namespace of storageos cluster to be installed")
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

	// if etcdEndpoints was not passed via flag or config, prompt user to enter manually
	if ksConfig.Spec.SkipEtcd && ksConfig.Spec.Install.EtcdEndpoints == "" {
		ksConfig.Spec.Install.EtcdEndpoints, err = etcdEndpointsPrompt()
		if err != nil {
			return err
		}
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
			config.Spec.SkipEtcd, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipEtcdFlag).Value.String())
			config.Spec.Install.StorageOSOperatorYaml = cmd.Flags().Lookup(installer.StosOperatorYamlFlag).Value.String()
			config.Spec.Install.StorageOSClusterYaml = cmd.Flags().Lookup(installer.StosClusterYamlFlag).Value.String()
			config.Spec.Install.EtcdOperatorYaml = cmd.Flags().Lookup(installer.EtcdOperatorYamlFlag).Value.String()
			config.Spec.Install.EtcdClusterYaml = cmd.Flags().Lookup(installer.EtcdClusterYamlFlag).Value.String()
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
	config.Spec.SkipEtcd = viper.GetBool(installer.InstallSkipEtcdConfig)
	config.Spec.Install.StorageOSOperatorYaml = viper.GetString(installer.StosOperatorYamlConfig)
	config.Spec.Install.StorageOSClusterYaml = viper.GetString(installer.StosClusterYamlConfig)
	config.Spec.Install.EtcdOperatorYaml = viper.GetString(installer.EtcdOperatorYamlConfig)
	config.Spec.Install.EtcdClusterYaml = viper.GetString(installer.EtcdClusterYamlConfig)
	config.Spec.Install.StorageOSOperatorNamespace = valueOrDefault(viper.GetString(installer.InstallStosOperatorNSConfig), consts.NewOperatorNamespace)
	config.Spec.Install.StorageOSClusterNamespace = viper.GetString(installer.InstallStosClusterNSConfig)
	config.Spec.Install.EtcdNamespace = valueOrDefault(viper.GetString(installer.InstallEtcdNamespaceConfig), consts.EtcdOperatorNamespace)
	config.Spec.Install.EtcdEndpoints = viper.GetString(installer.EtcdEndpointsConfig)
	config.Spec.Install.StorageClassName = viper.GetString(installer.StorageClassConfig)
	config.InstallerMeta.StorageOSSecretYaml = ""
	return nil
}
