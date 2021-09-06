package cli

import (
	"fmt"

	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/installer"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
)

const (
	uninstallStosOperatorNSFlag = "uninstall-" + installer.StosOperatorNSFlag
	installStosOperatorNSFlag   = "install-" + installer.StosOperatorNSFlag
	uninstallStosClusterNSFlag  = "uninstall-" + installer.StosClusterNSFlag
	installStosClusterNSFlag    = "install-" + installer.StosClusterNSFlag
)

func UpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "upgrade",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Ugrade StorageOS",
		Long:         `Upgrade StorageOS operator and cluster version`,
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := upgradeCmd(cmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().String(installer.ConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(uninstallStosOperatorNSFlag, "", "namespace of storageos operator to be uninstalled")
	cmd.Flags().String(uninstallStosClusterNSFlag, "", "namespace of storageos cluster to be uninstalled")
	cmd.Flags().String(installStosOperatorNSFlag, "", "namespace of storageos operator to be installed")
	cmd.Flags().String(installStosClusterNSFlag, "", "namespace of storageos cluster to be installed")
	cmd.Flags().String(installer.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url to be installed")
	cmd.Flags().String(installer.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url to be installed")
	cmd.Flags().String(installer.EtcdEndpointsFlag, "", "etcd endpoints")
	viper.BindPFlags(cmd.Flags())

	return cmd
}

func upgradeCmd(cmd *cobra.Command) error {
	v := viper.GetViper()

	logger.SetQuiet(v.GetBool("quiet"))

	ksUninstallConfig := &apiv1.KubectlStorageOSConfig{}
	err := setUpgradeUninstallValues(cmd, ksUninstallConfig)
	if err != nil {
		return err
	}
	existingVersion, err := pluginversion.GetExistingOperatorVersion(ksUninstallConfig.Spec.Uninstall.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}

	noUpgrade, err := pluginversion.VersionIsEqualTo(existingVersion, pluginversion.OperatorLatestSupportedVersion())
	if err != nil {
		return err
	}
	if noUpgrade {
		fmt.Println("Latest version of StorageOS cluster and operator are already installed...")
		return nil
	}
	fmt.Printf("Discovered StorageOS cluster and operator version %s...\n", existingVersion)

	err = setVersionSpecificValues(ksUninstallConfig, existingVersion)
	if err != nil {
		return err
	}

	ksInstallConfig := &apiv1.KubectlStorageOSConfig{}
	err = setUpgradeInstallValues(cmd, ksInstallConfig)
	if err != nil {
		return err
	}

	err = installer.Upgrade(ksUninstallConfig, ksInstallConfig, existingVersion)
	if err != nil {
		return err
	}
	return nil
}

func setUpgradeInstallValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
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
			config.Spec.Install.SkipEtcd = true
			config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installStosOperatorNSFlag).Value.String()
			config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installStosClusterNSFlag).Value.String()
			config.Spec.Install.EtcdEndpoints = cmd.Flags().Lookup(installer.EtcdEndpointsFlag).Value.String()
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
	config.Spec.Install.SkipEtcd = true
	config.Spec.Install.EtcdEndpoints = toString(viper.Get(installer.EtcdEndpointsConfig))
	config.Spec.Install.StorageOSOperatorNamespace = toString(viper.Get(installer.InstallStosOperatorNSConfig))
	config.Spec.Install.StorageOSClusterNamespace = toString(viper.Get(installer.InstallStosClusterNSConfig))
	config.InstallerMeta.StorageOSSecretYaml = ""
	return nil
}

func setUpgradeUninstallValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
	viper.BindPFlag(installer.ConfigPathFlag, cmd.Flags().Lookup(installer.ConfigPathFlag))
	v := viper.GetViper()
	viper.SetConfigName("kubectl-storageos-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(v.GetString(installer.ConfigPathFlag))
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; set fields in new config object directly
			config.Spec.Uninstall.SkipEtcd = true
			// Also set install skip-etcd to ignore unnecessary fs building
			config.Spec.Install.SkipEtcd = true
			config.Spec.Uninstall.StorageOSOperatorNamespace = cmd.Flags().Lookup(uninstallStosOperatorNSFlag).Value.String()
			config.Spec.Uninstall.StorageOSClusterNamespace = cmd.Flags().Lookup(uninstallStosClusterNSFlag).Value.String()
			return nil
		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error discovered in config file: %v", err)
		}
	}
	// config file read without error, set fields in new config object
	config.Spec.Uninstall.SkipEtcd = true
	// Also set install skip-etcd to ignore unnecessary fs building
	config.Spec.Install.SkipEtcd = true
	config.Spec.Uninstall.StorageOSOperatorNamespace = toString(viper.Get(installer.UninstallStosOperatorNSConfig))
	config.Spec.Uninstall.StorageOSClusterNamespace = toString(viper.Get(installer.UninstallStosClusterNSConfig))
	return nil
}
