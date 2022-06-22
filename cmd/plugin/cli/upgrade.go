package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/consts"
	"github.com/storageos/kubectl-storageos/pkg/installer"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"github.com/storageos/kubectl-storageos/pkg/version"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
)

const (
	upgrade = "upgrade"

	uninstallStosOperatorNSFlag = installer.UninstallPrefix + installer.StosOperatorNSFlag

	installStosOperatorNSFlag = installer.InstallPrefix + installer.StosOperatorNSFlag
	installStosClusterNSFlag  = installer.InstallPrefix + installer.StosClusterNSFlag

	installStosOperatorYamlFlag           = installer.InstallPrefix + installer.StosOperatorYamlFlag
	installStosClusterYamlFlag            = installer.InstallPrefix + installer.StosClusterYamlFlag
	installStosPortalConfigYamlFlag       = installer.InstallPrefix + installer.StosPortalConfigYamlFlag
	installStosPortalClientSecretYamlFlag = installer.InstallPrefix + installer.StosPortalClientSecretYamlFlag
	installResourceQuotaYamlFlag          = installer.InstallPrefix + installer.ResourceQuotaYamlFlag

	uninstallStosOperatorYamlFlag           = installer.UninstallPrefix + installer.StosOperatorYamlFlag
	uninstallStosClusterYamlFlag            = installer.UninstallPrefix + installer.StosClusterYamlFlag
	uninstallStosPortalConfigYamlFlag       = installer.UninstallPrefix + installer.StosPortalConfigYamlFlag
	uninstallStosPortalClientSecretYamlFlag = installer.UninstallPrefix + installer.StosPortalClientSecretYamlFlag
	uninstallResourceQuotaYamlFlag          = installer.UninstallPrefix + installer.ResourceQuotaYamlFlag
)

func UpgradeCmd() *cobra.Command {
	var err error
	var traceError bool
	pluginLogger := logger.NewLogger()
	cmd := &cobra.Command{
		Use:          upgrade,
		Args:         cobra.MinimumNArgs(0),
		Short:        "Ugrade StorageOS",
		Long:         `Upgrade StorageOS operator and cluster version`,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			defer pluginutils.ConvertPanicToError(func(e error) {
				err = e
			})

			uninstallConfig := &apiv1.KubectlStorageOSConfig{}
			if err = setUpgradeUninstallValues(cmd, uninstallConfig); err != nil {
				return
			}

			installConfig := &apiv1.KubectlStorageOSConfig{}
			if err = setUpgradeInstallValues(cmd, installConfig); err != nil {
				return
			}

			traceError = installConfig.Spec.StackTrace

			err = upgradeCmd(uninstallConfig, installConfig, pluginutils.HasFlagSet(installer.SkipNamespaceDeletionFlag), pluginLogger)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if err := pluginutils.HandleError(upgrade, err, traceError); err != nil {
				pluginLogger.Error(fmt.Sprintf("%s%s", upgrade, " has failed"))
				return err
			}
			pluginLogger.Success("StorageOS upgraded successfully.")
			return nil
		},
	}
	cmd.Flags().Bool(installer.StackTraceFlag, false, "print stack trace of error")
	cmd.Flags().Bool(installer.WaitFlag, false, "wait for storageos cluster to enter running phase")
	cmd.Flags().Bool(installer.SkipExistingWorkloadCheckFlag, false, "skip check for PVCs using storageos storage class during upgrade")
	cmd.Flags().String(installer.StosVersionFlag, "", "version of storageos operator")
	cmd.Flags().String(installer.K8sVersionFlag, "", "version of kubernetes cluster")
	cmd.Flags().Bool(installer.SkipNamespaceDeletionFlag, false, "leaving namespaces untouched")
	cmd.Flags().Bool(installer.EnablePortalManagerFlag, false, "enable storageos portal manager during upgrade")
	cmd.Flags().String(installer.StosConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(uninstallStosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be uninstalled")
	cmd.Flags().String(installStosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be installed")
	cmd.Flags().String(installStosClusterNSFlag, "", "namespace of storageos cluster to be installed")
	cmd.Flags().String(installStosOperatorYamlFlag, "", "storageos-operator.yaml path or url to be installed")
	cmd.Flags().String(installStosClusterYamlFlag, "", "storageos-cluster.yaml path or url to be installed")
	cmd.Flags().String(installStosPortalConfigYamlFlag, "", "storageos-portal-manager-configmap.yaml path or url to be installer")
	cmd.Flags().String(installStosPortalClientSecretYamlFlag, "", "storageos-portal-manager-client-secret.yaml path or url to be installed")
	cmd.Flags().String(installResourceQuotaYamlFlag, "", "resource-quota.yaml path or url to be installed")
	cmd.Flags().String(uninstallStosOperatorYamlFlag, "", "storageos-operator.yaml path or url to be uninstalled")
	cmd.Flags().String(uninstallStosClusterYamlFlag, "", "storageos-cluster.yaml path or url to be uninstalled")
	cmd.Flags().String(uninstallStosPortalConfigYamlFlag, "", "storageos-portal-manager-configmap.yaml path or url to be uninstaller")
	cmd.Flags().String(uninstallStosPortalClientSecretYamlFlag, "", "storageos-portal-manager-client-secret.yaml path or url to be uninstalled")
	cmd.Flags().String(uninstallResourceQuotaYamlFlag, "", "resource-quota.yaml path or url to be uninstalled")

	cmd.Flags().String(installer.EtcdEndpointsFlag, "", "endpoints of pre-existing etcd backend for storageos (implies not --include-etcd)")
	cmd.Flags().String(installer.EtcdSecretNameFlag, consts.EtcdSecretName, "name of etcd secret in storageos cluster namespace")
	cmd.Flags().Bool(installer.SkipEtcdEndpointsValFlag, false, "skip validation of etcd endpoints")
	cmd.Flags().Bool(installer.SkipStosClusterFlag, false, "skip storageos cluster during upgrade")
	cmd.Flags().Bool(installer.EtcdTLSEnabledFlag, false, "etcd cluster is tls enabled")
	cmd.Flags().String(installer.AdminUsernameFlag, "", "storageos admin username (plaintext)")
	cmd.Flags().String(installer.AdminPasswordFlag, "", "storageos admin password (plaintext)")
	cmd.Flags().String(installer.PortalClientIDFlag, "", "storageos portal client id (plaintext)")
	cmd.Flags().String(installer.PortalSecretFlag, "", "storageos portal secret (plaintext)")
	cmd.Flags().String(installer.PortalAPIURLFlag, "", "storageos portal api url")
	cmd.Flags().String(installer.PortalTenantIDFlag, "", "storageos portal tenant id")
	cmd.Flags().Bool(installer.EnableMetricsFlag, false, "enable metrics exporter")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func upgradeCmd(uninstallConfig *apiv1.KubectlStorageOSConfig, installConfig *apiv1.KubectlStorageOSConfig, skipNamespaceDeletionHasSet bool, log *logger.Logger) error {
	if installConfig.Spec.Install.StorageOSVersion == "" {
		installConfig.Spec.Install.StorageOSVersion = version.OperatorLatestSupportedVersion()
	}

	if installConfig.Spec.Install.EnablePortalManager {
		if err := versionSupportsFeature(installConfig.Spec.Install.StorageOSVersion, consts.PortalManagerFirstSupportedVersion); err != nil {
			return fmt.Errorf("failed to enable portal manager: %w", err)
		}
	}

	if installConfig.Spec.Install.EnableMetrics != nil && *installConfig.Spec.Install.EnableMetrics {
		if err := versionSupportsFeature(installConfig.Spec.Install.StorageOSVersion, consts.MetricsExporterFirstSupportedVersion); err != nil {
			return fmt.Errorf("failed to enable metrics exporter: %w", err)
		}
	}

	version.SetOperatorLatestSupportedVersion(installConfig.Spec.Install.StorageOSVersion)

	// if skip namespace delete was not passed via flag or config, prompt user to enter manually
	if !uninstallConfig.Spec.SkipNamespaceDeletion && !skipNamespaceDeletionHasSet {
		var err error
		uninstallConfig.Spec.SkipNamespaceDeletion, err = skipNamespaceDeletionPrompt(log)
		if err != nil {
			return err
		}
	}

	existingVersion, err := pluginversion.GetExistingOperatorVersion(uninstallConfig.Spec.Uninstall.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}

	noUpgrade, err := pluginversion.VersionIsEqualTo(existingVersion, version.OperatorLatestSupportedVersion())
	if err != nil {
		return err
	}
	if noUpgrade {
		log.Successf("StorageOS cluster and operator %s are already installed. No action required.", existingVersion)
		return nil
	}
	log.Warnf("Discovered StorageOS cluster and operator version %s.", existingVersion)

	err = setVersionSpecificValues(uninstallConfig, existingVersion)
	if err != nil {
		return err
	}

	log.Commencing(upgrade)
	return installer.Upgrade(uninstallConfig, installConfig, existingVersion, log)
}

func setUpgradeInstallValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
	viper.BindPFlag(installer.StosConfigPathFlag, cmd.Flags().Lookup(installer.StosConfigPathFlag))
	v := viper.GetViper()
	viper.SetConfigName("kubectl-storageos-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(v.GetString(installer.StosConfigPathFlag))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return fmt.Errorf("error discovered in config file: %v", err)
		}
		// Config file not found; set fields in new config object directly
		config.Spec.IncludeEtcd = false
		config.Spec.StackTrace, err = cmd.Flags().GetBool(installer.StackTraceFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.Wait, err = cmd.Flags().GetBool(installer.WaitFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.EnablePortalManager, err = cmd.Flags().GetBool(installer.EnablePortalManagerFlag)
		if err != nil {
			return err
		}
		config.Spec.SkipExistingWorkloadCheck, err = cmd.Flags().GetBool(installer.SkipExistingWorkloadCheckFlag)
		if err != nil {
			return err
		}
		config.Spec.SkipStorageOSCluster, err = cmd.Flags().GetBool(installer.SkipStosClusterFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.SkipEtcdEndpointsValidation, err = cmd.Flags().GetBool(installer.SkipEtcdEndpointsValFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.EtcdTLSEnabled, err = cmd.Flags().GetBool(installer.EtcdTLSEnabledFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.EnableMetrics, err = GetBoolIfFlagSet(cmd.Flags(), installer.EnableMetricsFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.StorageOSVersion = cmd.Flags().Lookup(installer.StosVersionFlag).Value.String()
		config.Spec.Install.StorageOSOperatorYaml = cmd.Flags().Lookup(installStosOperatorYamlFlag).Value.String()
		config.Spec.Install.StorageOSClusterYaml = cmd.Flags().Lookup(installStosClusterYamlFlag).Value.String()
		config.Spec.Install.StorageOSPortalConfigYaml = cmd.Flags().Lookup(installStosPortalConfigYamlFlag).Value.String()
		config.Spec.Install.StorageOSPortalClientSecretYaml = cmd.Flags().Lookup(installStosPortalClientSecretYamlFlag).Value.String()
		config.Spec.Install.ResourceQuotaYaml = cmd.Flags().Lookup(installResourceQuotaYamlFlag).Value.String()
		config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installStosOperatorNSFlag).Value.String()
		config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installStosClusterNSFlag).Value.String()
		config.Spec.Install.EtcdEndpoints = cmd.Flags().Lookup(installer.EtcdEndpointsFlag).Value.String()
		config.Spec.Install.EtcdSecretName = cmd.Flags().Lookup(installer.EtcdSecretNameFlag).Value.String()
		config.Spec.Install.AdminUsername = cmd.Flags().Lookup(installer.AdminUsernameFlag).Value.String()
		config.Spec.Install.AdminPassword = cmd.Flags().Lookup(installer.AdminPasswordFlag).Value.String()
		config.Spec.Install.PortalClientID = cmd.Flags().Lookup(installer.PortalClientIDFlag).Value.String()
		config.Spec.Install.PortalSecret = cmd.Flags().Lookup(installer.PortalSecretFlag).Value.String()
		config.Spec.Install.PortalAPIURL = cmd.Flags().Lookup(installer.PortalAPIURLFlag).Value.String()
		config.Spec.Install.PortalTenantID = cmd.Flags().Lookup(installer.PortalTenantIDFlag).Value.String()
		config.InstallerMeta.StorageOSSecretYaml = ""
		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.StackTraceConfig)
	config.Spec.IncludeEtcd = false
	config.Spec.SkipExistingWorkloadCheck = viper.GetBool(installer.SkipExistingWorkloadCheckConfig)
	config.Spec.SkipStorageOSCluster = viper.GetBool(installer.SkipStosClusterConfig)
	config.Spec.Install.EnablePortalManager = viper.GetBool(installer.EnablePortalManagerConfig)
	config.Spec.Install.EnableMetrics = GetBoolIfConfigSet(installer.EnableMetricsConfig)
	config.Spec.Install.Wait = viper.GetBool(installer.WaitConfig)
	config.Spec.Install.StorageOSVersion = viper.GetString(installer.StosVersionConfig)
	config.Spec.Install.StorageOSOperatorYaml = viper.GetString(installer.InstallStosOperatorYamlConfig)
	config.Spec.Install.StorageOSClusterYaml = viper.GetString(installer.InstallStosClusterYamlConfig)
	config.Spec.Install.StorageOSPortalConfigYaml = viper.GetString(installer.InstallStosPortalConfigYamlConfig)
	config.Spec.Install.StorageOSPortalClientSecretYaml = viper.GetString(installer.InstallStosPortalClientSecretYamlConfig)
	config.Spec.Install.ResourceQuotaYaml = viper.GetString(installer.InstallResourceQuotaYamlConfig)
	config.Spec.Install.EtcdEndpoints = viper.GetString(installer.EtcdEndpointsConfig)
	config.Spec.Install.SkipEtcdEndpointsValidation = viper.GetBool(installer.SkipEtcdEndpointsValConfig)
	config.Spec.Install.EtcdTLSEnabled = viper.GetBool(installer.EtcdTLSEnabledConfig)
	config.Spec.Install.EtcdSecretName = viper.GetString(installer.EtcdSecretNameConfig)
	config.Spec.Install.StorageOSOperatorNamespace = valueOrDefault(viper.GetString(installer.InstallStosOperatorNSConfig), consts.NewOperatorNamespace)
	config.Spec.Install.StorageOSClusterNamespace = viper.GetString(installer.StosClusterNSConfig)
	config.Spec.Install.AdminUsername = viper.GetString(installer.AdminUsernameConfig)
	config.Spec.Install.AdminPassword = viper.GetString(installer.AdminPasswordConfig)
	config.Spec.Install.PortalClientID = viper.GetString(installer.PortalClientIDConfig)
	config.Spec.Install.PortalSecret = viper.GetString(installer.PortalSecretConfig)
	config.Spec.Install.PortalAPIURL = viper.GetString(installer.PortalAPIURLConfig)
	config.Spec.Install.PortalTenantID = viper.GetString(installer.PortalTenantIDConfig)
	config.InstallerMeta.StorageOSSecretYaml = ""
	return nil
}

func setUpgradeUninstallValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
	viper.BindPFlag(installer.StosConfigPathFlag, cmd.Flags().Lookup(installer.StosConfigPathFlag))
	v := viper.GetViper()
	viper.SetConfigName("kubectl-storageos-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(v.GetString(installer.StosConfigPathFlag))
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return fmt.Errorf("error discovered in config file: %v", err)
		}

		// Config file not found; set fields in new config object directly
		config.Spec.SkipNamespaceDeletion, err = cmd.Flags().GetBool(installer.SkipNamespaceDeletionFlag)
		if err != nil {
			return err
		}
		config.Spec.SkipStorageOSCluster, err = cmd.Flags().GetBool(installer.SkipStosClusterFlag)
		if err != nil {
			return err
		}

		config.Spec.IncludeEtcd = false
		config.Spec.Uninstall.StorageOSOperatorNamespace = cmd.Flags().Lookup(uninstallStosOperatorNSFlag).Value.String()
		config.Spec.Uninstall.StorageOSOperatorYaml = cmd.Flags().Lookup(uninstallStosOperatorYamlFlag).Value.String()
		config.Spec.Uninstall.StorageOSClusterYaml = cmd.Flags().Lookup(uninstallStosClusterYamlFlag).Value.String()
		config.Spec.Uninstall.StorageOSPortalConfigYaml = cmd.Flags().Lookup(uninstallStosPortalConfigYamlFlag).Value.String()
		config.Spec.Uninstall.StorageOSPortalClientSecretYaml = cmd.Flags().Lookup(uninstallStosPortalClientSecretYamlFlag).Value.String()
		config.Spec.Uninstall.ResourceQuotaYaml = cmd.Flags().Lookup(uninstallResourceQuotaYamlFlag).Value.String()

		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.SkipNamespaceDeletion = viper.GetBool(installer.SkipNamespaceDeletionConfig)
	config.Spec.IncludeEtcd = false
	config.Spec.SkipStorageOSCluster = viper.GetBool(installer.SkipStosClusterConfig)
	config.Spec.Uninstall.StorageOSOperatorNamespace = viper.GetString(installer.UninstallStosOperatorNSConfig)
	config.Spec.Uninstall.StorageOSOperatorYaml = viper.GetString(installer.UninstallStosOperatorYamlConfig)
	config.Spec.Uninstall.StorageOSClusterYaml = viper.GetString(installer.UninstallStosClusterYamlConfig)
	config.Spec.Uninstall.StorageOSPortalConfigYaml = viper.GetString(installer.UninstallStosPortalConfigYamlConfig)
	config.Spec.Uninstall.StorageOSPortalClientSecretYaml = viper.GetString(installer.UninstallStosPortalClientSecretYamlConfig)
	config.Spec.Uninstall.ResourceQuotaYaml = viper.GetString(installer.UninstallResourceQuotaYamlConfig)

	return nil
}
