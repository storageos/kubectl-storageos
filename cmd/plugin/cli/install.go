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
			if err = setInstallValues(cmd, config); err != nil {
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
	cmd.Flags().Bool(installer.WaitFlag, false, "wait for storageos cluster to enter running phase")
	cmd.Flags().String(installer.StosVersionFlag, "", "version of storageos operator")
	cmd.Flags().String(installer.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url")
	cmd.Flags().String(installer.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url")
	cmd.Flags().String(installer.StosPortalConfigYamlFlag, "", "storageos-portal-manager-configmap.yaml path or url")
	cmd.Flags().String(installer.EtcdClusterYamlFlag, "", "etcd-cluster.yaml path or url")
	cmd.Flags().String(installer.EtcdOperatorYamlFlag, "", "etcd-operator.yaml path or url")
	cmd.Flags().Bool(installer.IncludeEtcdFlag, false, "install non-production etcd from github.com/storageos/etcd-cluster-operator")
	cmd.Flags().Bool(installer.EtcdTLSEnabledFlag, false, "etcd cluster is tls enabled")
	cmd.Flags().Bool(installer.SkipEtcdEndpointsValFlag, false, "skip validation of etcd endpoints")
	cmd.Flags().Bool(installer.SkipStosClusterFlag, false, "skip storageos cluster installation")
	cmd.Flags().Bool(installer.EnablePortalManagerFlag, false, "enable storageos portal manager during installation")
	cmd.Flags().String(installer.EtcdEndpointsFlag, "", "endpoints of pre-existing etcd backend for storageos (implies not --include-etcd)")
	cmd.Flags().String(installer.EtcdSecretNameFlag, consts.EtcdSecretName, "name of etcd secret in storageos cluster namespace")
	cmd.Flags().String(installer.StosConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(installer.EtcdNamespaceFlag, consts.EtcdOperatorNamespace, "namespace of etcd operator and cluster to be installed")
	cmd.Flags().String(installer.StosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be installed")
	cmd.Flags().String(installer.StosClusterNSFlag, consts.NewOperatorNamespace, "namespace of storageos cluster to be installed")
	cmd.Flags().String(installer.EtcdStorageClassFlag, "", "name of storage class to be used by etcd cluster")
	cmd.Flags().String(installer.AdminUsernameFlag, "", "storageos admin username (plaintext)")
	cmd.Flags().String(installer.AdminPasswordFlag, "", "storageos admin password (plaintext)")
	cmd.Flags().String(installer.PortalUsernameFlag, "", "storageos portal username (plaintext)")
	cmd.Flags().String(installer.PortalPasswordFlag, "", "storageos portal password (plaintext)")
	cmd.Flags().String(installer.TenantIDFlag, "", "storageos portal tenant id")
	cmd.Flags().String(installer.PortalAPIURLFlag, "", "storageos portal api url")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func installCmd(config *apiv1.KubectlStorageOSConfig) error {
	if config.Spec.Install.StorageOSVersion == "" {
		config.Spec.Install.StorageOSVersion = version.OperatorLatestSupportedVersion()
	}

	if config.Spec.Install.EnablePortalManager {
		if err := versionSupportsPortal(config.Spec.Install.StorageOSVersion); err != nil {
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
	}
	version.SetOperatorLatestSupportedVersion(config.Spec.Install.StorageOSVersion)

	if !config.Spec.Install.SkipEtcdEndpointsValidation {
		// if etcdEndpoints was not passed via flag or config, prompt user to enter manually
		if !config.Spec.IncludeEtcd && config.Spec.Install.EtcdEndpoints == "" {
			var err error
			config.Spec.Install.EtcdEndpoints, err = etcdEndpointsPrompt()
			if err != nil {
				return err
			}
		}
	}
	cliInstaller, err := installer.NewInstaller(config, true, true)
	if err != nil {
		return err
	}

	err = cliInstaller.Install(false)

	return err
}

func setInstallValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
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
		config.Spec.IncludeEtcd, err = strconv.ParseBool(cmd.Flags().Lookup(installer.IncludeEtcdFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.SkipStorageOSCluster, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipStosClusterFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.Install.EnablePortalManager, err = strconv.ParseBool(cmd.Flags().Lookup(installer.EnablePortalManagerFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.Install.Wait, err = strconv.ParseBool(cmd.Flags().Lookup(installer.WaitFlag).Value.String())
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
		config.Spec.Install.StorageOSPortalConfigYaml = cmd.Flags().Lookup(installer.StosPortalConfigYamlFlag).Value.String()
		config.Spec.Install.EtcdOperatorYaml = cmd.Flags().Lookup(installer.EtcdOperatorYamlFlag).Value.String()
		config.Spec.Install.EtcdClusterYaml = cmd.Flags().Lookup(installer.EtcdClusterYamlFlag).Value.String()
		config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
		config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installer.StosClusterNSFlag).Value.String()
		config.Spec.Install.EtcdNamespace = cmd.Flags().Lookup(installer.EtcdNamespaceFlag).Value.String()
		config.Spec.Install.EtcdEndpoints = cmd.Flags().Lookup(installer.EtcdEndpointsFlag).Value.String()
		config.Spec.Install.EtcdSecretName = cmd.Flags().Lookup(installer.EtcdSecretNameFlag).Value.String()
		config.Spec.Install.EtcdStorageClassName = cmd.Flags().Lookup(installer.EtcdStorageClassFlag).Value.String()
		config.Spec.Install.AdminUsername = cmd.Flags().Lookup(installer.AdminUsernameFlag).Value.String()
		config.Spec.Install.AdminPassword = cmd.Flags().Lookup(installer.AdminPasswordFlag).Value.String()
		config.Spec.Install.PortalUsername = cmd.Flags().Lookup(installer.PortalUsernameFlag).Value.String()
		config.Spec.Install.PortalPassword = cmd.Flags().Lookup(installer.PortalPasswordFlag).Value.String()
		config.Spec.Install.TenantID = cmd.Flags().Lookup(installer.TenantIDFlag).Value.String()
		config.Spec.Install.PortalAPIURL = cmd.Flags().Lookup(installer.PortalAPIURLFlag).Value.String()
		config.InstallerMeta.StorageOSSecretYaml = ""
		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.StackTraceConfig)
	config.Spec.IncludeEtcd = viper.GetBool(installer.IncludeEtcdConfig)
	config.Spec.SkipStorageOSCluster = viper.GetBool(installer.SkipStosClusterConfig)
	config.Spec.Install.EnablePortalManager = viper.GetBool(installer.EnablePortalManagerConfig)
	config.Spec.Install.Wait = viper.GetBool(installer.WaitConfig)
	config.Spec.Install.StorageOSVersion = viper.GetString(installer.StosVersionConfig)
	config.Spec.Install.StorageOSOperatorYaml = viper.GetString(installer.StosOperatorYamlConfig)
	config.Spec.Install.StorageOSClusterYaml = viper.GetString(installer.StosClusterYamlConfig)
	config.Spec.Install.StorageOSPortalConfigYaml = viper.GetString(installer.StosPortalConfigYamlConfig)
	config.Spec.Install.EtcdOperatorYaml = viper.GetString(installer.EtcdOperatorYamlConfig)
	config.Spec.Install.EtcdClusterYaml = viper.GetString(installer.EtcdClusterYamlConfig)
	config.Spec.Install.StorageOSOperatorNamespace = valueOrDefault(viper.GetString(installer.InstallStosOperatorNSConfig), consts.NewOperatorNamespace)
	config.Spec.Install.StorageOSClusterNamespace = viper.GetString(installer.StosClusterNSConfig)
	config.Spec.Install.EtcdNamespace = valueOrDefault(viper.GetString(installer.InstallEtcdNamespaceConfig), consts.EtcdOperatorNamespace)
	config.Spec.Install.EtcdEndpoints = viper.GetString(installer.EtcdEndpointsConfig)
	config.Spec.Install.SkipEtcdEndpointsValidation = viper.GetBool(installer.SkipEtcdEndpointsValConfig)
	config.Spec.Install.EtcdTLSEnabled = viper.GetBool(installer.EtcdTLSEnabledConfig)
	config.Spec.Install.EtcdSecretName = viper.GetString(installer.EtcdSecretNameConfig)
	config.Spec.Install.EtcdStorageClassName = viper.GetString(installer.EtcdStorageClassConfig)
	config.Spec.Install.AdminUsername = viper.GetString(installer.AdminUsernameConfig)
	config.Spec.Install.AdminPassword = viper.GetString(installer.AdminPasswordConfig)
	config.Spec.Install.PortalUsername = viper.GetString(installer.PortalUsernameConfig)
	config.Spec.Install.PortalPassword = viper.GetString(installer.PortalPasswordConfig)
	config.Spec.Install.TenantID = viper.GetString(installer.TenantIDConfig)
	config.Spec.Install.PortalAPIURL = viper.GetString(installer.PortalAPIURLConfig)
	config.InstallerMeta.StorageOSSecretYaml = ""
	return nil
}
