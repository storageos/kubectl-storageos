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
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
	"github.com/storageos/kubectl-storageos/pkg/version"
)

func InstallPortalCmd() *cobra.Command {
	var err error
	var traceError bool
	cmd := &cobra.Command{
		Use:          "install-portal",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Install StorageOS Portal Manager",
		Long:         `Install StorageOS Portal Manager`,
		SilenceUsage: true,
		PreRun:       func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			defer pluginutils.ConvertPanicToError(func(e error) {
				err = e
			})

			v := viper.GetViper()
			logger.SetQuiet(v.GetBool("quiet"))

			config := &apiv1.KubectlStorageOSConfig{}
			if err = setInstallPortalValues(cmd, config); err != nil {
				return
			}

			traceError = config.Spec.StackTrace

			err = installPortalCmd(config)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return pluginutils.HandleError("install-portal", err, traceError)
		},
	}
	cmd.Flags().Bool(installer.StackTraceFlag, false, "print stack trace of error")
	cmd.Flags().String(installer.StosConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(installer.StosPortalConfigYamlFlag, "", "storageos-portal-configmap.yaml path or url")
	cmd.Flags().String(installer.StosClusterNSFlag, consts.NewOperatorNamespace, "namespace of storageos cluster")
	cmd.Flags().String(installer.StosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator")
	cmd.Flags().String(installer.PortalUsernameFlag, "", "storageos portal username (plaintext)")
	cmd.Flags().String(installer.PortalPasswordFlag, "", "storageos portal password (plaintext)")
	cmd.Flags().String(installer.TenantIDFlag, "", "storageos portal tenant id")
	cmd.Flags().String(installer.PortalAPIURLFlag, "", "storageos portal url")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func installPortalCmd(config *apiv1.KubectlStorageOSConfig) error {
	existingOperatorVersion, err := version.GetExistingOperatorVersion(config.Spec.Install.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}

	if err := versionSupportsPortal(existingOperatorVersion); err != nil {
		return err
	}
	if err := installer.FlagsAreSet(map[string]string{
		installer.PortalUsernameFlag: config.Spec.Install.PortalUsername,
		installer.PortalPasswordFlag: config.Spec.Install.PortalPassword,
		installer.TenantIDFlag:       config.Spec.Install.TenantID,
		installer.PortalAPIURLFlag:   config.Spec.Install.PortalAPIURL,
	}); err != nil {
		return err
	}

	cliInstaller, err := installer.NewInstaller(config, true, true)
	if err != nil {
		return err
	}

	if err := cliInstaller.InstallPortalManager(); err != nil {
		return err
	}

	return cliInstaller.EnablePortalManager(true)
}

func setInstallPortalValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
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
		config.Spec.StackTrace, err = strconv.ParseBool(cmd.Flags().Lookup(installer.StackTraceFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.Install.StorageOSPortalConfigYaml = cmd.Flags().Lookup(installer.StosPortalConfigYamlFlag).Value.String()
		config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installer.StosClusterNSFlag).Value.String()
		config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
		config.Spec.Install.PortalUsername = cmd.Flags().Lookup(installer.PortalUsernameFlag).Value.String()
		config.Spec.Install.PortalPassword = cmd.Flags().Lookup(installer.PortalPasswordFlag).Value.String()
		config.Spec.Install.PortalAPIURL = cmd.Flags().Lookup(installer.PortalAPIURLFlag).Value.String()
		config.Spec.Install.TenantID = cmd.Flags().Lookup(installer.TenantIDFlag).Value.String()
		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.StackTraceConfig)
	config.Spec.Install.StorageOSPortalConfigYaml = viper.GetString(installer.StosPortalConfigYamlConfig)
	config.Spec.Install.StorageOSClusterNamespace = viper.GetString(installer.StosClusterNSConfig)
	config.Spec.Install.StorageOSOperatorNamespace = viper.GetString(installer.InstallStosOperatorNSConfig)
	config.Spec.Install.PortalUsername = viper.GetString(installer.PortalUsernameConfig)
	config.Spec.Install.PortalPassword = viper.GetString(installer.PortalPasswordConfig)
	config.Spec.Install.PortalAPIURL = viper.GetString(installer.PortalAPIURLConfig)
	config.Spec.Install.TenantID = viper.GetString(installer.TenantIDConfig)
	return nil
}
