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

func InstallCmd() *cobra.Command {
	var err error
	var traceError bool
	cmd := &cobra.Command{
		Use:          "install",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Install StorageOS and (optionally) ETCD",
		Long:         `Install StorageOS and (optionally) ETCD`,
		SilenceUsage: true,
		PreRun:       func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			defer pluginutils.ConvertPanicToError(func(e error) {
				err = e
			})

			v := viper.GetViper()
			logger.SetQuiet(v.GetBool("quiet"))

			config := &apiv1.KubectlStorageOSConfig{}
			err = setInstallValues(cmd, config)
			if err != nil {
				return
			}

			traceError = config.Spec.StackTrace

			err = installCmd(config)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return pluginutils.HandleError("install", err, traceError)
		},
	}
	cmd.Flags().Bool(installer.StackTraceFlag, false, "print stack trace of error")
	cmd.Flags().Bool(installer.WaitFlag, false, "wait for storagos cluster to enter running phase")
	cmd.Flags().String(installer.VersionFlag, "", "version of storageos operator")
	cmd.Flags().String(installer.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url")
	cmd.Flags().String(installer.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url")
	cmd.Flags().String(installer.EtcdClusterYamlFlag, "", "etcd-cluster.yaml path or url")
	cmd.Flags().String(installer.EtcdOperatorYamlFlag, "", "etcd-operator.yaml path or url")
	cmd.Flags().Bool(installer.IncludeEtcdFlag, false, "install non-production etcd from github.com/storageos/etcd-cluster-operator")
	cmd.Flags().Bool(installer.EtcdTLSEnabledFlag, false, "etcd cluster is TLS enabled")
	cmd.Flags().Bool(installer.SkipEtcdEndpointsValFlag, false, "skip validation of ETCD endpoints")
	cmd.Flags().String(installer.EtcdEndpointsFlag, "", "etcd endpoints")
	cmd.Flags().String(installer.EtcdSecretNameFlag, consts.EtcdSecretName, "name of etcd secret in storageos cluster namespace")
	cmd.Flags().String(installer.ConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(installer.EtcdNamespaceFlag, consts.EtcdOperatorNamespace, "namespace of etcd operator and cluster to be installed")
	cmd.Flags().String(installer.StosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be installed")
	cmd.Flags().String(installer.StosClusterNSFlag, consts.NewOperatorNamespace, "namespace of storageos cluster to be installed")
	cmd.Flags().String(installer.StorageClassFlag, "", "name of storage class to be used by etcd cluster")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func installCmd(config *apiv1.KubectlStorageOSConfig) error {
	// user specified the version
	if config.Spec.Install.Version != "" {
		version.SetOperatorLatestSupportedVersion(config.Spec.Install.Version)
	}

	// if etcdEndpoints was not passed via flag or config, prompt user to enter manually
	if !config.Spec.IncludeEtcd && config.Spec.Install.EtcdEndpoints == "" {
		var err error
		config.Spec.Install.EtcdEndpoints, err = etcdEndpointsPrompt()
		if err != nil {
			return err
		}
	}

	cliInstaller, err := installer.NewInstaller(config, true)
	if err != nil {
		return err
	}

	err = cliInstaller.Install(config)

	return err
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
			config.Spec.StackTrace, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.StackTraceFlag).Value.String())
			config.Spec.IncludeEtcd, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.IncludeEtcdFlag).Value.String())
			config.Spec.Install.Wait, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.WaitFlag).Value.String())
			config.Spec.Install.Version = cmd.Flags().Lookup(installer.VersionFlag).Value.String()
			config.Spec.Install.StorageOSOperatorYaml = cmd.Flags().Lookup(installer.StosOperatorYamlFlag).Value.String()
			config.Spec.Install.StorageOSClusterYaml = cmd.Flags().Lookup(installer.StosClusterYamlFlag).Value.String()
			config.Spec.Install.EtcdOperatorYaml = cmd.Flags().Lookup(installer.EtcdOperatorYamlFlag).Value.String()
			config.Spec.Install.EtcdClusterYaml = cmd.Flags().Lookup(installer.EtcdClusterYamlFlag).Value.String()
			config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
			config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installer.StosClusterNSFlag).Value.String()
			config.Spec.Install.EtcdNamespace = cmd.Flags().Lookup(installer.EtcdNamespaceFlag).Value.String()
			config.Spec.Install.EtcdEndpoints = cmd.Flags().Lookup(installer.EtcdEndpointsFlag).Value.String()
			config.Spec.Install.SkipEtcdEndpointsValidation, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipEtcdEndpointsValFlag).Value.String())
			config.Spec.Install.EtcdTLSEnabled, _ = strconv.ParseBool(cmd.Flags().Lookup(installer.EtcdTLSEnabledFlag).Value.String())
			config.Spec.Install.EtcdSecretName = cmd.Flags().Lookup(installer.EtcdSecretNameFlag).Value.String()
			config.Spec.Install.StorageClassName = cmd.Flags().Lookup(installer.StorageClassFlag).Value.String()
			config.InstallerMeta.StorageOSSecretYaml = ""
			return nil

		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error discovered in config file: %v", err)
		}
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.SStackTraceConfig)
	config.Spec.IncludeEtcd = viper.GetBool(installer.IncludeEtcdConfig)
	config.Spec.Install.Wait = viper.GetBool(installer.InstallWaitConfig)
	config.Spec.Install.Version = viper.GetString(installer.InstallVersionConfig)
	config.Spec.Install.StorageOSOperatorYaml = viper.GetString(installer.StosOperatorYamlConfig)
	config.Spec.Install.StorageOSClusterYaml = viper.GetString(installer.StosClusterYamlConfig)
	config.Spec.Install.EtcdOperatorYaml = viper.GetString(installer.EtcdOperatorYamlConfig)
	config.Spec.Install.EtcdClusterYaml = viper.GetString(installer.EtcdClusterYamlConfig)
	config.Spec.Install.StorageOSOperatorNamespace = valueOrDefault(viper.GetString(installer.InstallStosOperatorNSConfig), consts.NewOperatorNamespace)
	config.Spec.Install.StorageOSClusterNamespace = viper.GetString(installer.InstallStosClusterNSConfig)
	config.Spec.Install.EtcdNamespace = valueOrDefault(viper.GetString(installer.InstallEtcdNamespaceConfig), consts.EtcdOperatorNamespace)
	config.Spec.Install.EtcdEndpoints = viper.GetString(installer.EtcdEndpointsConfig)
	config.Spec.Install.SkipEtcdEndpointsValidation = viper.GetBool(installer.SkipEtcdEndpointsValConfig)
	config.Spec.Install.EtcdTLSEnabled = viper.GetBool(installer.EtcdTLSEnabledConfig)
	config.Spec.Install.EtcdSecretName = viper.GetString(installer.EtcdSecretNameConfig)
	config.Spec.Install.StorageClassName = viper.GetString(installer.StorageClassConfig)
	config.InstallerMeta.StorageOSSecretYaml = ""
	return nil
}
