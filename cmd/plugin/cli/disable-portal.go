package cli

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

const disablePortal = "disable-portal"

func DisablePortalCmd() *cobra.Command {
	var err error
	var traceError bool
	pluginLogger := logger.NewLogger()
	cmd := &cobra.Command{
		Use:          disablePortal,
		Args:         cobra.MinimumNArgs(0),
		Short:        "Disable StorageOS Portal Manager",
		Long:         `Disable StorageOS Portal Manager`,
		SilenceUsage: true,
		PreRun:       func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			defer pluginutils.ConvertPanicToError(func(e error) {
				err = e
			})

			config := &apiv1.KubectlStorageOSConfig{}
			if err = setDisablePortalValues(cmd, config); err != nil {
				return
			}

			traceError = config.Spec.StackTrace

			err = disablePortalCmd(config, pluginLogger)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if err := pluginutils.HandleError(disablePortal, err, traceError); err != nil {
				pluginLogger.Error(fmt.Sprintf("%s%s", disablePortal, " has failed"))
				return err
			}
			pluginLogger.Success("Portal Manager successfully disabled.")
			return nil
		},
	}
	cmd.Flags().Bool(installer.StackTraceFlag, false, "print stack trace of error")
	cmd.Flags().BoolP(installer.VerboseFlag, "v", false, "verbose logging")
	cmd.Flags().String(installer.StosConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(installer.StosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func disablePortalCmd(config *apiv1.KubectlStorageOSConfig, log *logger.Logger) error {
	log.Verbose = config.Spec.Verbose
	existingOperatorVersion, err := version.GetExistingOperatorVersion(config.Spec.Install.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}

	if err := versionSupportsFeature(existingOperatorVersion, consts.PortalManagerFirstSupportedVersion); err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cliInstaller, err := installer.NewPortalManagerInstaller(config, false, log)
		if err != nil {
			return err
		}

		log.Commencing(disablePortal)
		return cliInstaller.EnablePortalManager(false)
	})
}

func setDisablePortalValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
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
		config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.StackTraceConfig)
	config.Spec.Verbose = viper.GetBool(installer.VerboseConfig)
	config.Spec.Install.StorageOSOperatorNamespace = viper.GetString(installer.InstallStosOperatorNSConfig)
	return nil
}
