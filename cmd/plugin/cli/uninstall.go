package cli

import (
	"fmt"

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

func UninstallCmd() *cobra.Command {
	var err error
	var traceError bool
	cmd := &cobra.Command{
		Use:          "uninstall",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Uninstall StorageOS and (optionally) ETCD",
		Long:         `Uninstall StorageOS and (optionally) ETCD`,
		SilenceUsage: true,
		PreRun:       func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			defer pluginutils.ConvertPanicToError(func(e error) {
				err = e
			})

			v := viper.GetViper()
			logger.SetQuiet(v.GetBool("quiet"))

			config := &apiv1.KubectlStorageOSConfig{}
			err = setUninstallValues(cmd, config)
			if err != nil {
				return
			}

			traceError = config.Spec.StackTrace

			err = uninstallCmd(config, pluginutils.HasFlagSet(installer.SkipNamespaceDeletionFlag))
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return pluginutils.HandleError("uninstall", err, traceError)
		},
	}
	cmd.Flags().Bool(installer.StackTraceFlag, false, "print stack trace of error")
	cmd.Flags().Bool(installer.SkipNamespaceDeletionFlag, false, "leaving namespaces untouched")
	cmd.Flags().Bool(installer.SkipExistingWorkloadCheckFlag, false, "skip check for PVCs using storageos storage class during uninstall")
	cmd.Flags().Bool(installer.SkipStosClusterFlag, false, "skip storageos cluster uninstallation")
	cmd.Flags().Bool(installer.IncludeEtcdFlag, false, "uninstall etcd (only applicable to github.com/storageos/etcd-cluster-operator etcd cluster)")
	cmd.Flags().String(installer.EtcdNamespaceFlag, consts.EtcdOperatorNamespace, "namespace of etcd operator and cluster to be uninstalled")
	cmd.Flags().String(installer.StosOperatorNSFlag, consts.NewOperatorNamespace, "namespace of storageos operator to be uninstalled")
	cmd.Flags().String(installer.StosConfigPathFlag, "", "path to look for kubectl-storageos-config.yaml")
	cmd.Flags().String(installer.StosOperatorYamlFlag, "", "storageos-operator.yaml path or url")
	cmd.Flags().String(installer.StosClusterYamlFlag, "", "storageos-cluster.yaml path or url")
	cmd.Flags().String(installer.StosPortalConfigYamlFlag, "", "storageos-portal-manager-configmap.yaml path or url")
	cmd.Flags().String(installer.StosPortalClientSecretYamlFlag, "", "storageos-portal-manager-client-secret.yaml path or url")
	cmd.Flags().String(installer.EtcdClusterYamlFlag, "", "etcd-cluster.yaml path or url")
	cmd.Flags().String(installer.EtcdOperatorYamlFlag, "", "etcd-operator.yaml path or url")
	cmd.Flags().String(installer.ResourceQuotaYamlFlag, "", "resource-quota.yaml path or url")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func uninstallCmd(config *apiv1.KubectlStorageOSConfig, skipNamespaceDeletionHasSet bool) error {
	// if skip namespace delete was not passed via flag or config, prompt user to enter manually
	if !config.Spec.SkipNamespaceDeletion && !skipNamespaceDeletionHasSet {
		var err error
		config.Spec.SkipNamespaceDeletion, err = skipNamespaceDeletionPrompt()
		if err != nil {
			return err
		}
	}

	operatorVersion, err := pluginversion.GetExistingOperatorVersion(config.Spec.Uninstall.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}
	fmt.Printf("Discovered StorageOS cluster and operator version %s...\n", operatorVersion)
	version.SetOperatorLatestSupportedVersion(operatorVersion)

	if config.Spec.IncludeEtcd {
		etcdOperatorVersion, err := pluginversion.GetExistingEtcdOperatorVersion(config.Spec.Uninstall.EtcdNamespace)
		if err != nil {
			return err
		}

		version.SetEtcdOperatorLatestSupportedVersion(etcdOperatorVersion)
	}

	if err = setVersionSpecificValues(config, operatorVersion); err != nil {
		return err
	}

	cliInstaller, err := installer.NewUninstaller(config)
	if err != nil {
		return err
	}

	err = cliInstaller.Uninstall(false, operatorVersion)

	return err
}

func setUninstallValues(cmd *cobra.Command, config *apiv1.KubectlStorageOSConfig) error {
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
		config.Spec.SkipNamespaceDeletion, err = cmd.Flags().GetBool(installer.SkipNamespaceDeletionFlag)
		if err != nil {
			return err
		}
		config.Spec.SkipExistingWorkloadCheck, err = cmd.Flags().GetBool(installer.SkipExistingWorkloadCheckFlag)
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
		config.Spec.Uninstall.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
		config.Spec.Uninstall.EtcdNamespace = cmd.Flags().Lookup(installer.EtcdNamespaceFlag).Value.String()
		config.Spec.Uninstall.StorageOSOperatorYaml = cmd.Flags().Lookup(installer.StosOperatorYamlFlag).Value.String()
		config.Spec.Uninstall.StorageOSClusterYaml = cmd.Flags().Lookup(installer.StosClusterYamlFlag).Value.String()
		config.Spec.Uninstall.StorageOSPortalConfigYaml = cmd.Flags().Lookup(installer.StosPortalConfigYamlFlag).Value.String()
		config.Spec.Uninstall.StorageOSPortalClientSecretYaml = cmd.Flags().Lookup(installer.StosPortalClientSecretYamlFlag).Value.String()
		config.Spec.Uninstall.EtcdOperatorYaml = cmd.Flags().Lookup(installer.EtcdOperatorYamlFlag).Value.String()
		config.Spec.Uninstall.EtcdClusterYaml = cmd.Flags().Lookup(installer.EtcdClusterYamlFlag).Value.String()
		config.Spec.Uninstall.ResourceQuotaYaml = cmd.Flags().Lookup(installer.ResourceQuotaYamlFlag).Value.String()

		return nil
	}
	// config file read without error, set fields in new config object
	config.Spec.StackTrace = viper.GetBool(installer.StackTraceConfig)
	config.Spec.SkipNamespaceDeletion = viper.GetBool(installer.SkipNamespaceDeletionConfig)
	config.Spec.SkipExistingWorkloadCheck = viper.GetBool(installer.SkipExistingWorkloadCheckConfig)
	config.Spec.SkipStorageOSCluster = viper.GetBool(installer.SkipStosClusterConfig)
	config.Spec.IncludeEtcd = viper.GetBool(installer.IncludeEtcdConfig)
	config.Spec.Uninstall.StorageOSOperatorNamespace = viper.GetString(installer.UninstallStosOperatorNSConfig)
	config.Spec.Uninstall.EtcdNamespace = valueOrDefault(viper.GetString(installer.UninstallEtcdNSConfig), consts.EtcdOperatorNamespace)
	config.Spec.Uninstall.StorageOSOperatorYaml = viper.GetString(installer.UninstallStosOperatorYamlConfig)
	config.Spec.Uninstall.StorageOSClusterYaml = viper.GetString(installer.UninstallStosClusterYamlConfig)
	config.Spec.Uninstall.StorageOSPortalConfigYaml = viper.GetString(installer.UninstallStosPortalConfigYamlConfig)
	config.Spec.Uninstall.StorageOSPortalClientSecretYaml = viper.GetString(installer.UninstallStosPortalClientSecretYamlConfig)
	config.Spec.Uninstall.EtcdOperatorYaml = viper.GetString(installer.UninstallEtcdOperatorYamlConfig)
	config.Spec.Uninstall.EtcdClusterYaml = viper.GetString(installer.UninstallEtcdClusterYamlConfig)
	config.Spec.Uninstall.ResourceQuotaYaml = viper.GetString(installer.UninstallResourceQuotaYamlConfig)

	return nil
}

func setVersionSpecificValues(config *apiv1.KubectlStorageOSConfig, version string) (err error) {
	// Don't fetch version specific manifests for develop edition
	if pluginversion.IsDevelop(version) {
		return
	}

	// set additional values to be used by Installer for in memory fs build
	if config.Spec.Uninstall.StorageOSOperatorYaml == "" {
		config.Spec.Uninstall.StorageOSOperatorYaml, err = pluginversion.OperatorImageUrlByVersion(version)
		if err != nil {
			return
		}
	}

	if config.Spec.Uninstall.StorageOSClusterYaml == "" {
		config.Spec.Uninstall.StorageOSClusterYaml, err = pluginversion.ClusterUrlByVersion(version)
		if err != nil {
			return
		}
	}
	if config.Spec.Uninstall.ResourceQuotaYaml == "" {
		config.Spec.Uninstall.ResourceQuotaYaml, err = pluginversion.ResourceQuotaUrlByVersion(version)
		if err != nil {
			return
		}
	}

	config.InstallerMeta.StorageOSSecretYaml, err = pluginversion.SecretUrlByVersion(version)
	if err != nil {
		return
	}

	return
}
