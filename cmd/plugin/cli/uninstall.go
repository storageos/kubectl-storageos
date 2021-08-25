package cli

import (
	"fmt"

	"github.com/replicatedhq/troubleshoot/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/storageos/kubectl-storageos/pkg/installer"
)

func UninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "uninstall",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Uninstall StorageOS",
		Long:         `Uninstall StorageOS and/or ETCD`,
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			viper.BindPFlag(installer.StosOperatorNSFlag, cmd.Flags().Lookup(installer.StosOperatorNSFlag))
			viper.BindPFlag(installer.StosClusterNSFlag, cmd.Flags().Lookup(installer.StosClusterNSFlag))
			viper.BindPFlag(installer.EtcdNamespaceFlag, cmd.Flags().Lookup(installer.EtcdNamespaceFlag))
			viper.BindPFlag(installer.SkipEtcdFlag, cmd.Flags().Lookup(installer.SkipEtcdFlag))

			v := viper.GetViper()
			version, err := installer.GetExistingOperatorVersion(v.GetString(installer.StosOperatorNSFlag))
			if err != nil {
				return err
			}
			if version != "" {
				fmt.Printf("Uninstalling StorageOS cluster and operator version %s...\n", version)
				operatorUrl, err := installer.OperatorUrlByVersion(version)
				if err != nil {
					return err
				}

				clusterUrl, err := installer.ClusterUrlByVersion(version)
				if err != nil {
					return err
				}

				secretUrl, err := installer.SecretUrlByVersion(version)
				if err != nil {
					return err
				}

				viper.Set(installer.StosOperatorYamlFlag, operatorUrl)
				viper.Set(installer.StosClusterYamlFlag, clusterUrl)
				viper.Set(installer.StosSecretYamlFlag, secretUrl)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			logger.SetQuiet(v.GetBool("quiet"))
			cliInstaller, err := installer.NewInstaller()
			if err != nil {
				return err
			}

			err = cliInstaller.Uninstall()
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().Bool(installer.SkipEtcdFlag, false, "uninstall storageos only, leaving ETCD untouched")
	cmd.Flags().String(installer.EtcdNamespaceFlag, "", "namespace of etcd operator and cluster")
	cmd.Flags().String(installer.StosOperatorNSFlag, "", "namespace of storageos operator")
	cmd.Flags().String(installer.StosClusterNSFlag, "", "namespace of storageos cluster")

	viper.BindPFlags(cmd.Flags())

	return cmd
}
