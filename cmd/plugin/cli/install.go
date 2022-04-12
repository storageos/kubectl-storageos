package cli

import (
	"fmt"

	"github.com/coreos/go-semver/semver"
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
	cmd.Flags().Bool(installer.DryRunFlag, false, "no installation performed, installation manifests stored locally at \"./storageos-dry-run\"")
	cmd.Flags().String(installer.StosVersionFlag, "", "version of storageos operator")
	cmd.Flags().String(installer.EtcdOperatorVersionFlag, "", "version of etcd operator")
	cmd.Flags().String(installer.K8sVersionFlag, "", "version of kubernetes cluster")
	cmd.Flags().String(installer.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url")
	cmd.Flags().String(installer.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url")
	cmd.Flags().String(installer.StosPortalConfigYamlFlag, "", "storageos-portal-manager-configmap.yaml path or url")
	cmd.Flags().String(installer.StosPortalClientSecretYamlFlag, "", "storageos-portal-manager-client-secret.yaml path or url")
	cmd.Flags().String(installer.EtcdClusterYamlFlag, "", "etcd-cluster.yaml path or url")
	cmd.Flags().String(installer.EtcdOperatorYamlFlag, "", "etcd-operator.yaml path or url")
	cmd.Flags().String(installer.ResourceQuotaYamlFlag, "", "resource-quota.yaml path or url")
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
	cmd.Flags().String(installer.EtcdDockerRepositoryFlag, "", "the docker repository to use for the etcd docker image")
	cmd.Flags().String(installer.EtcdVersionTag, "", "the docker tag for the version of etcd to use - must be in the format 1.2.3")
	cmd.Flags().String(installer.AdminUsernameFlag, "", "storageos admin username (plaintext)")
	cmd.Flags().String(installer.AdminPasswordFlag, "", "storageos admin password (plaintext)")
	cmd.Flags().String(installer.PortalClientIDFlag, "", "storageos portal client id (plaintext)")
	cmd.Flags().String(installer.PortalSecretFlag, "", "storageos portal secret (plaintext)")
	cmd.Flags().String(installer.PortalTenantIDFlag, "", "storageos portal tenant id")
	cmd.Flags().String(installer.PortalAPIURLFlag, "", "storageos portal api url")
	cmd.Flags().Bool(installer.IncludeLocalPathProvisionerFlag, false, "install the local path provisioner storage class")
	cmd.Flags().String(installer.LocalPathProvisionerVersionFlag, "", "version of the local path provisioner storage class to use")
	cmd.Flags().String(installer.LocalPathProvisionerYamlFlag, "", "local-path-provisioner.yaml path or url")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func installCmd(config *apiv1.KubectlStorageOSConfig) error {
	if config.Spec.Install.StorageOSVersion == "" {
		config.Spec.Install.StorageOSVersion = version.OperatorLatestSupportedVersion()
	}
	version.SetOperatorLatestSupportedVersion(config.Spec.Install.StorageOSVersion)

	if config.Spec.IncludeEtcd {
		if config.Spec.Install.EtcdOperatorVersion == "" {
			config.Spec.Install.EtcdOperatorVersion = version.EtcdOperatorLatestSupportedVersion()
		}
		version.SetEtcdOperatorLatestSupportedVersion(config.Spec.Install.EtcdOperatorVersion)
	}

	if config.Spec.Install.LocalPathProvisionerVersion == "" {
		config.Spec.Install.LocalPathProvisionerVersion = version.LocalPathProvisionerLatestSupportVersion()
	}

	if config.Spec.Install.EnablePortalManager {
		if err := versionSupportsPortal(config.Spec.Install.StorageOSVersion); err != nil {
			return err
		}
		if err := installer.FlagsAreSet(map[string]string{
			installer.PortalClientIDFlag: config.Spec.Install.PortalClientID,
			installer.PortalSecretFlag:   config.Spec.Install.PortalSecret,
			installer.PortalTenantIDFlag: config.Spec.Install.PortalTenantID,
			installer.PortalAPIURLFlag:   config.Spec.Install.PortalAPIURL,
		}); err != nil {
			return err
		}
		// TODO: Do we need to add a --portal-manager-version flag?
		// for now, there is no released version so default to 'develop'
		version.SetPortalManagerLatestSupportedVersion("develop")
	}

	var err error
	// if etcdEndpoints was not passed via flag or config, prompt user to enter manually
	if !config.Spec.IncludeEtcd && config.Spec.Install.EtcdEndpoints == "" {
		config.Spec.Install.EtcdEndpoints, err = etcdEndpointsPrompt()
		if err != nil {
			return err
		}
	}
	if config.Spec.Install.DryRun {
		if config.Spec.Install.KubernetesVersion == "" {
			config.Spec.Install.KubernetesVersion, err = k8sVersionPrompt()
			if err != nil {
				return err
			}
		}
		if config.Spec.IncludeEtcd && config.Spec.Install.EtcdStorageClassName == "" {
			config.Spec.Install.EtcdStorageClassName, err = storageClassPrompt()
			if err != nil {
				return err
			}
		}
		config.Spec.Install.SkipEtcdEndpointsValidation = true
		cliInstaller, err := installer.NewDryRunInstaller(config)
		if err != nil {
			return err
		}
		return cliInstaller.Install(false)
	}

	cliInstaller, err := installer.NewInstaller(config)
	if err != nil {
		return err
	}

	return cliInstaller.Install(false)
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
		config.Spec.StackTrace, err = cmd.Flags().GetBool(installer.StackTraceFlag)
		if err != nil {
			return err
		}
		config.Spec.IncludeEtcd, err = cmd.Flags().GetBool(installer.IncludeEtcdFlag)
		if err != nil {
			return err
		}
		config.Spec.SkipStorageOSCluster, err = cmd.Flags().GetBool(installer.SkipStosClusterFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.EnablePortalManager, err = cmd.Flags().GetBool(installer.EnablePortalManagerFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.Wait, err = cmd.Flags().GetBool(installer.WaitFlag)
		if err != nil {
			return err
		}
		config.Spec.Install.DryRun, err = cmd.Flags().GetBool(installer.DryRunFlag)
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
		config.Spec.IncludeLocalPathProvisioner, err = cmd.Flags().GetBool(installer.IncludeLocalPathProvisionerFlag)
		if err != nil {
			return err
		}

		config.Spec.Install.StorageOSVersion = cmd.Flags().Lookup(installer.StosVersionFlag).Value.String()
		config.Spec.Install.EtcdOperatorVersion = cmd.Flags().Lookup(installer.EtcdOperatorVersionFlag).Value.String()
		config.Spec.Install.KubernetesVersion = cmd.Flags().Lookup(installer.K8sVersionFlag).Value.String()
		config.Spec.Install.StorageOSOperatorYaml = cmd.Flags().Lookup(installer.StosOperatorYamlFlag).Value.String()
		config.Spec.Install.StorageOSClusterYaml = cmd.Flags().Lookup(installer.StosClusterYamlFlag).Value.String()
		config.Spec.Install.StorageOSPortalConfigYaml = cmd.Flags().Lookup(installer.StosPortalConfigYamlFlag).Value.String()
		config.Spec.Install.StorageOSPortalClientSecretYaml = cmd.Flags().Lookup(installer.StosPortalClientSecretYamlFlag).Value.String()
		config.Spec.Install.EtcdOperatorYaml = cmd.Flags().Lookup(installer.EtcdOperatorYamlFlag).Value.String()
		config.Spec.Install.EtcdClusterYaml = cmd.Flags().Lookup(installer.EtcdClusterYamlFlag).Value.String()
		config.Spec.Install.ResourceQuotaYaml = cmd.Flags().Lookup(installer.ResourceQuotaYamlFlag).Value.String()
		config.Spec.Install.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
		config.Spec.Install.StorageOSClusterNamespace = cmd.Flags().Lookup(installer.StosClusterNSFlag).Value.String()
		config.Spec.Install.EtcdNamespace = cmd.Flags().Lookup(installer.EtcdNamespaceFlag).Value.String()
		config.Spec.Install.EtcdEndpoints = cmd.Flags().Lookup(installer.EtcdEndpointsFlag).Value.String()
		config.Spec.Install.EtcdSecretName = cmd.Flags().Lookup(installer.EtcdSecretNameFlag).Value.String()
		config.Spec.Install.EtcdStorageClassName = cmd.Flags().Lookup(installer.EtcdStorageClassFlag).Value.String()
		config.Spec.Install.EtcdDockerRepository = cmd.Flags().Lookup(installer.EtcdDockerRepositoryFlag).Value.String()
		config.Spec.Install.AdminUsername = cmd.Flags().Lookup(installer.AdminUsernameFlag).Value.String()
		config.Spec.Install.AdminPassword = cmd.Flags().Lookup(installer.AdminPasswordFlag).Value.String()
		config.Spec.Install.PortalClientID = cmd.Flags().Lookup(installer.PortalClientIDFlag).Value.String()
		config.Spec.Install.PortalSecret = cmd.Flags().Lookup(installer.PortalSecretFlag).Value.String()
		config.Spec.Install.PortalTenantID = cmd.Flags().Lookup(installer.PortalTenantIDFlag).Value.String()
		config.Spec.Install.PortalAPIURL = cmd.Flags().Lookup(installer.PortalAPIURLFlag).Value.String()
		config.Spec.Install.LocalPathProvisionerVersion = cmd.Flags().Lookup(installer.LocalPathProvisionerVersionFlag).Value.String()
		config.Spec.Install.LocalPathProvisionerYaml = cmd.Flags().Lookup(installer.LocalPathProvisionerYamlFlag).Value.String()

		config.InstallerMeta.StorageOSSecretYaml = ""

		config.Spec.Install.EtcdVersionTag = cmd.Flags().Lookup(installer.EtcdVersionTag).Value.String()

		if config.Spec.Install.EtcdVersionTag != "" {
			// Perform the same validation as the etcd operator does, to ensure the install will succeed
			_, err := semver.NewVersion(config.Spec.Install.EtcdVersionTag)
			if err != nil {
				return fmt.Errorf("etcd version provided is not valid: %w", err)
			}
		}
		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.StackTraceConfig)
	config.Spec.IncludeEtcd = viper.GetBool(installer.IncludeEtcdConfig)
	config.Spec.SkipStorageOSCluster = viper.GetBool(installer.SkipStosClusterConfig)
	config.Spec.Install.EnablePortalManager = viper.GetBool(installer.EnablePortalManagerConfig)
	config.Spec.Install.Wait = viper.GetBool(installer.WaitConfig)
	config.Spec.Install.DryRun = viper.GetBool(installer.DryRunConfig)
	config.Spec.Install.StorageOSVersion = viper.GetString(installer.StosVersionConfig)
	config.Spec.Install.EtcdOperatorVersion = viper.GetString(installer.EtcdOperatorVersionConfig)
	config.Spec.Install.KubernetesVersion = viper.GetString(installer.K8sVersionConfig)
	config.Spec.Install.StorageOSOperatorYaml = viper.GetString(installer.InstallStosOperatorYamlConfig)
	config.Spec.Install.StorageOSClusterYaml = viper.GetString(installer.InstallStosClusterYamlConfig)
	config.Spec.Install.StorageOSPortalConfigYaml = viper.GetString(installer.InstallStosPortalConfigYamlConfig)
	config.Spec.Install.StorageOSPortalClientSecretYaml = viper.GetString(installer.InstallStosPortalClientSecretYamlConfig)
	config.Spec.Install.EtcdOperatorYaml = viper.GetString(installer.InstallEtcdOperatorYamlConfig)
	config.Spec.Install.EtcdClusterYaml = viper.GetString(installer.InstallEtcdClusterYamlConfig)
	config.Spec.Install.ResourceQuotaYaml = viper.GetString(installer.InstallResourceQuotaYamlConfig)
	config.Spec.Install.StorageOSOperatorNamespace = valueOrDefault(viper.GetString(installer.InstallStosOperatorNSConfig), consts.NewOperatorNamespace)
	config.Spec.Install.StorageOSClusterNamespace = valueOrDefault(viper.GetString(installer.StosClusterNSConfig), consts.NewOperatorNamespace)
	config.Spec.Install.EtcdNamespace = valueOrDefault(viper.GetString(installer.InstallEtcdNamespaceConfig), consts.EtcdOperatorNamespace)
	config.Spec.Install.EtcdEndpoints = viper.GetString(installer.EtcdEndpointsConfig)
	config.Spec.Install.SkipEtcdEndpointsValidation = viper.GetBool(installer.SkipEtcdEndpointsValConfig)
	config.Spec.Install.EtcdTLSEnabled = viper.GetBool(installer.EtcdTLSEnabledConfig)
	config.Spec.Install.EtcdSecretName = viper.GetString(installer.EtcdSecretNameConfig)
	config.Spec.Install.EtcdStorageClassName = viper.GetString(installer.EtcdStorageClassConfig)
	config.Spec.Install.EtcdDockerRepository = viper.GetString(installer.EtcdDockerRepositoryFlag)
	config.Spec.Install.EtcdVersionTag = viper.GetString(installer.EtcdVersionTag)
	config.Spec.Install.AdminUsername = viper.GetString(installer.AdminUsernameConfig)
	config.Spec.Install.AdminPassword = viper.GetString(installer.AdminPasswordConfig)
	config.Spec.Install.PortalClientID = viper.GetString(installer.PortalClientIDConfig)
	config.Spec.Install.PortalSecret = viper.GetString(installer.PortalSecretConfig)
	config.Spec.Install.PortalTenantID = viper.GetString(installer.PortalTenantIDConfig)
	config.Spec.Install.PortalAPIURL = viper.GetString(installer.PortalAPIURLConfig)
	config.InstallerMeta.StorageOSSecretYaml = ""
	config.Spec.IncludeLocalPathProvisioner = viper.GetBool(installer.IncludeLocalPathProvisionerFlag)
	config.Spec.Install.LocalPathProvisionerVersion = viper.GetString(installer.LocalPathProvisionerVersionFlag)
	config.Spec.Install.LocalPathProvisionerYaml = viper.GetString(installer.LocalPathProvisionerYamlFlag)

	return nil
}
