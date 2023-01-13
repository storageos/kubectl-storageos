package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/util/retry"

	apiv1 "github.com/storageos/kubectl-storageos/api/v1"
	"github.com/storageos/kubectl-storageos/pkg/consts"
	"github.com/storageos/kubectl-storageos/pkg/installer"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"github.com/storageos/kubectl-storageos/pkg/version"
)

const uninstallPortal = "uninstall-portal"

func UninstallPortalCmd() *cobra.Command {
	var err error
	var traceError bool
	pluginLogger := logger.NewLogger()
	cmd := &cobra.Command{
		Use:          uninstallPortal,
		Args:         cobra.MinimumNArgs(0),
		Short:        "Uninstall StorageOS Portal Manager",
		Long:         `Uninstall StorageOS Portal Manager`,
		SilenceUsage: true,
		PreRun:       func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			defer pluginutils.ConvertPanicToError(func(e error) {
				err = e
			})

			config := &apiv1.KubectlStorageOSConfig{}
			if err = setUninstallPortalValues(cmd, config); err != nil {
				return
			}

			traceError = config.Spec.StackTrace

			err = uninstallPortalCmd(config, pluginLogger)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if err := pluginutils.HandleError(uninstallPortal, err, traceError); err != nil {
				pluginLogger.Error(fmt.Sprintf("%s%s", uninstallPortal, " has failed"))

				return err
			}
			pluginLogger.Success("Portal Manager uninstalled successfully.")
			return nil
		},
	}
	cmd.Flags().Bool(installer.StackTraceFlag, false, "print stack trace of error")
	cmd.Flags().BoolP(installer.VerboseFlag, "v", false, "verbose logging")
	cmd.Flags().String(installer.StosConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(installer.StosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator")
	cmd.Flags().String(installer.PortalManagerVersionFlag, "", "version of portal manager")
	cmd.Flags().String(installer.StosPortalConfigYamlFlag, "", "storageos-portal-manager-configmap.yaml path or url")
	cmd.Flags().String(installer.StosPortalClientSecretYamlFlag, "", "storageos-portal-manager-client-secret.yaml path or url")
	cmd.Flags().Bool(installer.AirGapFlag, false, "uninstall portal manger in an air gapped environment")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func uninstallPortalCmd(config *apiv1.KubectlStorageOSConfig, log *logger.Logger) error {
	log.Verbose = config.Spec.Verbose

	existingOperatorVersion, err := version.GetExistingOperatorVersion(config.Spec.Uninstall.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}

	if err := versionSupportsFeature(existingOperatorVersion, consts.PortalManagerFirstSupportedVersion); err != nil {
		return err
	}

	if config.Spec.Uninstall.PortalManagerVersion == "" {
		config.Spec.Uninstall.PortalManagerVersion, err = version.GetExistingPortalManagerVersion()
		if err != nil {
			return err
		}
		log.Successf("Discovered Portal Manager version '%s'.", config.Spec.Uninstall.PortalManagerVersion)
	}

	if config.Spec.AirGap && config.Spec.Uninstall.PortalManagerVersion == "" {
		// if no portal manager version could be discovered, portal-manager-version becomes a required
		// flag for an air-gap uninstall to avoid querying for the latest version.
		if err := installer.FlagsAreSet(map[string]string{installer.PortalManagerVersionFlag: config.Spec.Install.PortalManagerVersion}); err != nil {
			return err
		}
	}

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cliInstaller, err := installer.NewPortalManagerInstaller(config, true, log)
		if err != nil {
			return err
		}

		if err = cliInstaller.EnablePortalManager(false); err != nil {
			return err
		}

		log.Commencing(uninstallPortal)
		return cliInstaller.UninstallPortalManager()
	})
}

func setUninstallPortalValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
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
		config.Spec.StackTrace, err = cmd.Flags().GetBool(installer.StackTraceFlag)
		if err != nil {
			return err
		}
		config.Spec.Verbose, err = cmd.Flags().GetBool(installer.VerboseFlag)
		if err != nil {
			return err
		}
		config.Spec.AirGap, err = cmd.Flags().GetBool(installer.AirGapFlag)
		if err != nil {
			return err
		}
		config.Spec.Uninstall.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
		config.Spec.Uninstall.PortalManagerVersion = cmd.Flags().Lookup(installer.PortalManagerVersionFlag).Value.String()
		config.Spec.Uninstall.StorageOSPortalConfigYaml = cmd.Flags().Lookup(installer.StosPortalConfigYamlFlag).Value.String()
		config.Spec.Uninstall.StorageOSPortalClientSecretYaml = cmd.Flags().Lookup(installer.StosPortalClientSecretYamlFlag).Value.String()
		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.StackTraceConfig)
	config.Spec.Verbose = viper.GetBool(installer.VerboseConfig)
	config.Spec.AirGap = viper.GetBool(installer.AirGapConfig)
	config.Spec.Uninstall.StorageOSOperatorNamespace = viper.GetString(installer.UninstallStosOperatorNSConfig)
	config.Spec.Uninstall.PortalManagerVersion = viper.GetString(installer.UninstallPortalManagerVersionConfig)
	config.Spec.Uninstall.StorageOSPortalConfigYaml = viper.GetString(installer.UninstallStosPortalConfigYamlConfig)
	config.Spec.Uninstall.StorageOSPortalClientSecretYaml = viper.GetString(installer.UninstallStosPortalClientSecretYamlConfig)

	return nil
}
