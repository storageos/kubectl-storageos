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
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"github.com/storageos/kubectl-storageos/pkg/version"
	pluginversion "github.com/storageos/kubectl-storageos/pkg/version"
)

const (
	uninstallStosOperatorNSFlag = "uninstall-" + installer.StosOperatorNSFlag
	installStosOperatorNSFlag   = "install-" + installer.StosOperatorNSFlag
	installStosClusterNSFlag    = "install-" + installer.StosClusterNSFlag
)

func UpgradeCmd() *cobra.Command {
	var err error
	var traceError bool
	cmd := &cobra.Command{
		Use:          "upgrade",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Ugrade StorageOS",
		Long:         `Upgrade StorageOS operator and cluster version`,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			defer pluginutils.ConvertPanicToError(func(e error) {
				err = e
			})

			v := viper.GetViper()
			logger.SetQuiet(v.GetBool("quiet"))

			uninstallConfig := &apiv1.KubectlStorageOSConfig{}
			if err = setUpgradeUninstallValues(cmd, uninstallConfig); err != nil {
				return
			}

			installConfig := &apiv1.KubectlStorageOSConfig{}
			if err = setUpgradeInstallValues(cmd, installConfig); err != nil {
				return
			}

			traceError = installConfig.Spec.StackTrace

			err = upgradeCmd(uninstallConfig, installConfig)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return pluginutils.HandleError("upgrade", err, traceError)
		},
	}
	cmd.Flags().Bool(installer.StackTraceFlag, false, "print stack trace of error")
	cmd.Flags().Bool(installer.WaitFlag, false, "wait for storageos cluster to enter running phase")
	cmd.Flags().Bool(installer.SkipExistingWorkloadCheckFlag, false, "skip check for PVCs using storageos storage class during upgrade")
	cmd.Flags().String(installer.StosVersionFlag, "", "version of storageos operator")
	cmd.Flags().Bool(installer.SkipNamespaceDeletionFlag, false, "leaving namespaces untouched")
	cmd.Flags().Bool(installer.EnablePortalManagerFlag, false, "enable storageos portal manager during upgrade")
	cmd.Flags().String(installer.StosConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(uninstallStosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be uninstalled")
	cmd.Flags().String(installStosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be installed")
	cmd.Flags().String(installStosClusterNSFlag, consts.NewOperatorNamespace, "namespace of storageos cluster to be installed")
	cmd.Flags().String(installer.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url to be installed")
	cmd.Flags().String(installer.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url to be installed")
	cmd.Flags().String(installer.EtcdEndpointsFlag, "", "endpoints of pre-existing etcd backend for storageos (implies not --include-etcd)")
	cmd.Flags().String(installer.EtcdSecretNameFlag, consts.EtcdSecretName, "name of etcd secret in storageos cluster namespace")
	cmd.Flags().Bool(installer.SkipEtcdEndpointsValFlag, false, "skip validation of etcd endpoints")
	cmd.Flags().Bool(installer.SkipStosClusterFlag, false, "skip storageos cluster during upgrade")
	cmd.Flags().Bool(installer.EtcdTLSEnabledFlag, false, "etcd cluster is tls enabled")
	cmd.Flags().String(installer.AdminUsernameFlag, "", "storageos admin username (plaintext)")
	cmd.Flags().String(installer.AdminPasswordFlag, "", "storageos admin password (plaintext)")
	cmd.Flags().String(installer.PortalAPIURLFlag, "", "storageos portal api url")
	cmd.Flags().String(installer.TenantIDFlag, "", "storageos tenant id")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func upgradeCmd(uninstallConfig *apiv1.KubectlStorageOSConfig, installConfig *apiv1.KubectlStorageOSConfig) error {
	// if skip namespace delete was not passed via flag or config, prompt user to enter manually
	if !uninstallConfig.Spec.SkipNamespaceDeletion && isatty.IsTerminal(os.Stdout.Fd()) {
		var err error
		uninstallConfig.Spec.SkipNamespaceDeletion, err = skipNamespaceDeletionPrompt()
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
		fmt.Println("Latest version of StorageOS cluster and operator are already installed...")
		return nil
	}
	fmt.Printf("Discovered StorageOS cluster and operator version %s...\n", existingVersion)

	err = setVersionSpecificValues(uninstallConfig, existingVersion)
	if err != nil {
		return err
	}

	// Let's start install

	// user specified the version
	if installConfig.Spec.Install.StorageOSVersion != "" {
		if installConfig.Spec.Install.EnablePortalManager {
			if err := versionSupportsPortal(installConfig.Spec.Install.StorageOSVersion); err != nil {
				return err
			}
			if err := portalFlagsExist(installConfig); err != nil {
				return err
			}
		}
		version.SetOperatorLatestSupportedVersion(installConfig.Spec.Install.StorageOSVersion)
	}

	err = installer.Upgrade(uninstallConfig, installConfig, existingVersion)

	return err
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
		config.Spec.StackTrace, err = strconv.ParseBool(cmd.Flags().Lookup(installer.StackTraceFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.Install.Wait, err = strconv.ParseBool(cmd.Flags().Lookup(installer.WaitFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.Install.EnablePortalManager, err = strconv.ParseBool(cmd.Flags().Lookup(installer.EnablePortalManagerFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.SkipExistingWorkloadCheck, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipExistingWorkloadCheckFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.SkipStorageOSCluster, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipStosClusterFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.Install.SkipEtcdEndpointsValidation, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipEtcdEndpointsValFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.Install.EtcdTLSEnabled, err = strconv.ParseBool(cmd.Flags().Lookup(installer.EtcdTLSEnabledFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.Install.StorageOSVersion = cmd.Flags().Lookup(installer.StosVersionFlag).Value.String()
		config.Spec.Install.StorageOSOperatorYaml = cmd.Flags().Lookup(installer.StosOperatorYamlFlag).Value.String()
		config.Spec.Install.StorageOSClusterYaml = cmd.Flags().Lookup(installer.StosClusterYamlFlag).Value.String()
		config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installStosOperatorNSFlag).Value.String()
		config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installStosClusterNSFlag).Value.String()
		config.Spec.Install.EtcdEndpoints = cmd.Flags().Lookup(installer.EtcdEndpointsFlag).Value.String()
		config.Spec.Install.EtcdSecretName = cmd.Flags().Lookup(installer.EtcdSecretNameFlag).Value.String()
		config.Spec.Install.AdminUsername = stringToBase64(cmd.Flags().Lookup(installer.AdminUsernameFlag).Value.String())
		config.Spec.Install.AdminPassword = stringToBase64(cmd.Flags().Lookup(installer.AdminPasswordFlag).Value.String())
		config.Spec.Install.PortalAPIURL = cmd.Flags().Lookup(installer.PortalAPIURLFlag).Value.String()
		config.Spec.Install.TenantID = cmd.Flags().Lookup(installer.TenantIDFlag).Value.String()
		config.InstallerMeta.StorageOSSecretYaml = ""
		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.StackTraceConfig)
	config.Spec.IncludeEtcd = false
	config.Spec.SkipExistingWorkloadCheck = viper.GetBool(installer.SkipExistingWorkloadCheckConfig)
	config.Spec.SkipStorageOSCluster = viper.GetBool(installer.SkipStosClusterConfig)
	config.Spec.Install.EnablePortalManager = viper.GetBool(installer.EnablePortalManagerConfig)
	config.Spec.Install.Wait = viper.GetBool(installer.WaitConfig)
	config.Spec.Install.StorageOSVersion = viper.GetString(installer.StosVersionConfig)
	config.Spec.Install.StorageOSOperatorYaml = viper.GetString(installer.StosOperatorYamlConfig)
	config.Spec.Install.StorageOSClusterYaml = viper.GetString(installer.StosClusterYamlConfig)
	config.Spec.Install.EtcdEndpoints = viper.GetString(installer.EtcdEndpointsConfig)
	config.Spec.Install.SkipEtcdEndpointsValidation = viper.GetBool(installer.SkipEtcdEndpointsValConfig)
	config.Spec.Install.EtcdTLSEnabled = viper.GetBool(installer.EtcdTLSEnabledConfig)
	config.Spec.Install.EtcdSecretName = viper.GetString(installer.EtcdSecretNameConfig)
	config.Spec.Install.StorageOSOperatorNamespace = valueOrDefault(viper.GetString(installer.InstallStosOperatorNSConfig), consts.NewOperatorNamespace)
	config.Spec.Install.StorageOSClusterNamespace = viper.GetString(installer.StosClusterNSConfig)
	config.Spec.Install.AdminUsername = stringToBase64(viper.GetString(installer.AdminUsernameConfig))
	config.Spec.Install.AdminPassword = stringToBase64(viper.GetString(installer.AdminPasswordConfig))
	config.Spec.Install.PortalAPIURL = viper.GetString(installer.PortalAPIURLConfig)
	config.Spec.Install.TenantID = viper.GetString(installer.TenantIDConfig)
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
		config.Spec.SkipNamespaceDeletion, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipNamespaceDeletionFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.SkipStorageOSCluster, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipStosClusterFlag).Value.String())
		if err != nil {
			return err
		}

		config.Spec.IncludeEtcd = false
		config.Spec.Uninstall.StorageOSOperatorNamespace = cmd.Flags().Lookup(uninstallStosOperatorNSFlag).Value.String()
		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.SkipNamespaceDeletion = viper.GetBool(installer.SkipNamespaceDeletionConfig)
	config.Spec.IncludeEtcd = false
	config.Spec.SkipStorageOSCluster = viper.GetBool(installer.SkipStosClusterConfig)
	config.Spec.Uninstall.StorageOSOperatorNamespace = viper.GetString(installer.UninstallStosOperatorNSConfig)
	return nil
}
