package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/mattn/go-isatty"
	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/consts"
	"github.com/storageos/kubectl-storageos/pkg/installer"
	"github.com/storageos/kubectl-storageos/pkg/version"
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
	cmd.Flags().Bool(installer.WaitFlag, false, "wait for storagos cluster to enter running phase")
	cmd.Flags().String(installer.VersionFlag, "", "version of storageos operator")
	cmd.Flags().Bool(installer.SkipNamespaceDeletionFlag, false, "leaving namespaces untouched")
	cmd.Flags().String(installer.ConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(uninstallStosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be uninstalled")
	cmd.Flags().String(uninstallStosClusterNSFlag, consts.NewOperatorNamespace, "namespace of storageos cluster to be uninstalled")
	cmd.Flags().String(installStosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be installed")
	cmd.Flags().String(installStosClusterNSFlag, consts.NewOperatorNamespace, "namespace of storageos cluster to be installed")
	cmd.Flags().String(installer.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url to be installed")
	cmd.Flags().String(installer.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url to be installed")
	cmd.Flags().String(installer.EtcdEndpointsFlag, "", "etcd endpoints")
	cmd.Flags().String(installer.EtcdSecretNameFlag, consts.EtcdSecretName, "name of etcd secret in storageos cluster namespace")
	cmd.Flags().Bool(installer.SkipEtcdEndpointsValFlag, false, "skip validation of ETCD endpoints")
	cmd.Flags().Bool(installer.EtcdTLSEnabledFlag, false, "etcd cluster is TLS enabled")
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

	// if skip namespace delete was not passed via flag or config, prompt user to enter manually
	if !ksUninstallConfig.Spec.SkipNamespaceDeletion && isatty.IsTerminal(os.Stdout.Fd()) {
		ksUninstallConfig.Spec.SkipNamespaceDeletion, err = skipNamespaceDeletionPrompt()
		if err != nil {
			return err
		}
	}

	existingVersion, err := pluginversion.GetExistingOperatorVersion(ksUninstallConfig.Spec.Uninstall.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}

	noUpgrade, err := pluginversion.VersionIsEqualTo(existingVersion, version.OperatorLatestSupportedVersion())
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

	// user specified the version
	if ksInstallConfig.Spec.Install.Version != "" {
		version.SetOperatorLatestSupportedVersion(ksInstallConfig.Spec.Install.Version)
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
			config.Spec.IncludeEtcd = false
			// Config file not found; set fields in new config object directly
			config.Spec.Install.Wait, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.WaitFlag).Value.String())
			config.Spec.Install.Version = cmd.Flags().Lookup(installer.VersionFlag).Value.String()
			config.Spec.Install.StorageOSOperatorYaml = cmd.Flags().Lookup(installer.StosOperatorYamlFlag).Value.String()
			config.Spec.Install.StorageOSClusterYaml = cmd.Flags().Lookup(installer.StosClusterYamlFlag).Value.String()
			config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installStosOperatorNSFlag).Value.String()
			config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installStosClusterNSFlag).Value.String()
			config.Spec.Install.EtcdEndpoints = cmd.Flags().Lookup(installer.EtcdEndpointsFlag).Value.String()
			config.Spec.Install.SkipEtcdEndpointsValidation, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipEtcdEndpointsValFlag).Value.String())
			config.Spec.Install.EtcdTLSEnabled, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.EtcdTLSEnabledFlag).Value.String())
			config.Spec.Install.EtcdSecretName = cmd.Flags().Lookup(installer.EtcdSecretNameFlag).Value.String()
			config.InstallerMeta.StorageOSSecretYaml = ""
			return nil

		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error discovered in config file: %v", err)
		}
	}
	// config file read without error, set fields in new config object
	config.Spec.IncludeEtcd = false
	config.Spec.Install.Wait = viper.GetBool(installer.InstallWaitConfig)
	config.Spec.Install.Version = viper.GetString(installer.InstallVersionConfig)
	config.Spec.Install.StorageOSOperatorYaml = viper.GetString(installer.StosOperatorYamlConfig)
	config.Spec.Install.StorageOSClusterYaml = viper.GetString(installer.StosClusterYamlConfig)
	config.Spec.Install.EtcdEndpoints = viper.GetString(installer.EtcdEndpointsConfig)
	config.Spec.Install.SkipEtcdEndpointsValidation = viper.GetBool(installer.SkipEtcdEndpointsValConfig)
	config.Spec.Install.EtcdTLSEnabled = viper.GetBool(installer.EtcdTLSEnabledConfig)
	config.Spec.Install.EtcdSecretName = viper.GetString(installer.EtcdSecretNameConfig)
	config.Spec.Install.StorageOSOperatorNamespace = valueOrDefault(viper.GetString(installer.InstallStosOperatorNSConfig), consts.NewOperatorNamespace)
	config.Spec.Install.StorageOSClusterNamespace = viper.GetString(installer.InstallStosClusterNSConfig)
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
			config.Spec.SkipNamespaceDeletion, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipNamespaceDeletionFlag).Value.String())
			if err != nil {
				return err
			}
			config.Spec.IncludeEtcd = false
			config.Spec.Uninstall.StorageOSOperatorNamespace = cmd.Flags().Lookup(uninstallStosOperatorNSFlag).Value.String()
			config.Spec.Uninstall.StorageOSClusterNamespace = cmd.Flags().Lookup(uninstallStosClusterNSFlag).Value.String()
			return nil
		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error discovered in config file: %v", err)
		}
	}
	// config file read without error, set fields in new config object
	config.Spec.SkipNamespaceDeletion = viper.GetBool(installer.SkipNamespaceDeletionConfig)
	config.Spec.IncludeEtcd = false
	config.Spec.Uninstall.StorageOSOperatorNamespace = viper.GetString(installer.UninstallStosOperatorNSConfig)
	config.Spec.Uninstall.StorageOSClusterNamespace = viper.GetString(installer.UninstallStosClusterNSConfig)
	return nil
}
