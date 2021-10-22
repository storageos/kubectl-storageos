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

			err = uninstallCmd(config)
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

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func uninstallCmd(config *apiv1.KubectlStorageOSConfig) error {
	// if skip namespace delete was not passed via flag or config, prompt user to enter manually
	if !config.Spec.SkipNamespaceDeletion && isatty.IsTerminal(os.Stdout.Fd()) {
		var err error
		config.Spec.SkipNamespaceDeletion, err = skipNamespaceDeletionPrompt()
		if err != nil {
			return err
		}
	}

	version, err := pluginversion.GetExistingOperatorVersion(config.Spec.Uninstall.StorageOSOperatorNamespace)
	if err != nil {
		return err
	}
	fmt.Printf("Discovered StorageOS cluster and operator version %s...\n", version)

	if err = setVersionSpecificValues(config, version); err != nil {
		return err
	}

	cliInstaller, err := installer.NewInstaller(config, false, false)
	if err != nil {
		return err
	}

	err = cliInstaller.Uninstall(false, version)

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
		config.Spec.StackTrace, err = strconv.ParseBool(cmd.Flags().Lookup(installer.StackTraceFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.SkipNamespaceDeletion, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipNamespaceDeletionFlag).Value.String())
		if err != nil {
			return err
		}
		config.Spec.SkipExistingWorkloadCheck, err = strconv.ParseBool(cmd.Flags().Lookup(installer.SkipExistingWorkloadCheckFlag).Value.String())
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
		config.Spec.Uninstall.StorageOSOperatorNamespace = cmd.Flags().Lookup(installer.StosOperatorNSFlag).Value.String()
		config.Spec.Uninstall.EtcdNamespace = cmd.Flags().Lookup(installer.EtcdNamespaceFlag).Value.String()
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
	return nil
}

func setVersionSpecificValues(config *apiv1.KubectlStorageOSConfig, version string) (err error) {
	// set additional values to be used by Installer for in memory fs build
	config.Spec.Install.StorageOSOperatorYaml, err = pluginversion.OperatorUrlByVersion(version)
	if err != nil {
		return
	}

	config.InstallerMeta.StorageOSSecretYaml, err = pluginversion.SecretUrlByVersion(version)
	if err != nil {
		return
	}

	// Don't override on dev versions
	if pluginutils.IsDevelop(version) {
		return
	}

	config.Spec.Install.StorageOSClusterYaml, err = pluginversion.ClusterUrlByVersion(version)
	if err != nil {
		return
	}

	config.Spec.Install.ResourceQuotaYaml, err = pluginversion.ResourceQuotaUrlByVersion(version)
	if err != nil {
		return
	}

	return
}
