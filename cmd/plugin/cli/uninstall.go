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
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
)

func UninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "uninstall",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Uninstall StorageOS",
		Long:         `Uninstall StorageOS and/or ETCD`,
		SilenceUsage: true,
		PreRun:       func(cmd *cobra.Command, args []string) {},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := uninstallCmd(cmd)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().Bool(installer.SkipEtcdFlag, false, "uninstall storageos only, leaving ETCD untouched")
	cmd.Flags().String(installer.EtcdNamespaceFlag, "", "namespace of etcd operator and cluster to be uninstalled")
	cmd.Flags().String(installer.StosOperatorNSFlag, version.GetDefaultNamespace(), "namespace of storageos operator to be uninstalled")
	cmd.Flags().String(installer.StosClusterNSFlag, "", "namespace of storageos cluster to be uninstalled")
	cmd.Flags().String(installer.ConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func uninstallCmd(cmd *cobra.Command) error {
	v := viper.GetViper()

	logger.SetQuiet(v.GetBool("quiet"))

	ksConfig := &apiv1.KubectlStorageOSConfig{}

	err := setUninstallValues(cmd, ksConfig)
	if err != nil {
		return err
	}
	version, err := pluginversion.GetExistingOperatorVersion(ksConfig.Spec.Uninstall.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}
	fmt.Printf("Discovered StorageOS cluster and operator version %s...\n", version)

	err = setVersionSpecificValues(ksConfig, version)
	if err != nil {
		return err
	}

	cliInstaller, err := installer.NewInstaller(ksConfig, false)
	if err != nil {
		return err
	}

	err = cliInstaller.Uninstall(ksConfig, false)
	if err != nil {
		return err
	}

	return nil
}

func setUninstallValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
	viper.BindPFlag(installer.ConfigPathFlag, cmd.Flags().Lookup(installer.ConfigPathFlag))
	v := viper.GetViper()
	viper.SetConfigName("kubectl-storageos-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(v.GetString(installer.ConfigPathFlag))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; set fields in new config object directly
			config.Spec.Uninstall.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
			config.Spec.Uninstall.StorageOSClusterNamespace = cmd.Flags().Lookup(installer.StosClusterNSFlag).Value.String()
			config.Spec.Uninstall.EtcdNamespace = cmd.Flags().Lookup(installer.EtcdNamespaceFlag).Value.String()
			config.Spec.Uninstall.SkipEtcd, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipEtcdFlag).Value.String())
			// also set SkipEtcd value for Installer (used during installer fs build)
			config.Spec.Install.SkipEtcd, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipEtcdFlag).Value.String())

			return nil

		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error discovered in config file: %v", err)
		}
	}
	// config file read without error, set fields in new config object
	config.Spec.Uninstall.StorageOSOperatorNamespace = toString(viper.Get(installer.UninstallStosOperatorNSConfig))
	config.Spec.Uninstall.StorageOSClusterNamespace = toString(viper.Get(installer.UninstallStosClusterNSConfig))
	config.Spec.Uninstall.EtcdNamespace = toString(viper.Get(installer.UninstallEtcdNamespaceConfig))
	config.Spec.Uninstall.SkipEtcd = toBool(viper.Get(installer.UninstallSkipEtcdConfig))
	// also set SkipEtcd value for Installer (used during installer fs build)
	config.Spec.Install.SkipEtcd = toBool(viper.Get(installer.UninstallSkipEtcdConfig))
	return nil
}

func setVersionSpecificValues(config *apiv1.KubectlStorageOSConfig, version string) error {
	// set additional values to be used by Installer for in memory fs build
	operatorUrl, err := pluginversion.OperatorUrlByVersion(version)
	if err != nil {
		return err
	}
	clusterUrl, err := pluginversion.ClusterUrlByVersion(version)
	if err != nil {
		return err
	}
	secretUrl, err := pluginversion.SecretUrlByVersion(version)
	if err != nil {
		return err
	}

	config.Spec.Install.StorageOSOperatorYaml = operatorUrl
	config.Spec.Install.StorageOSClusterYaml = clusterUrl
	config.InstallerMeta.StorageOSSecretYaml = secretUrl

	return nil
}
