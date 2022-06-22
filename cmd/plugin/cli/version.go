package cli

import (
	"github.com/spf13/cobra"
	"github.com/storageos/kubectl-storageos/pkg/logger"
	"github.com/storageos/kubectl-storageos/pkg/version"
)

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "version",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Show kubectl storageos version",
		Long:         `Show kubectl storageos version`,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			logger.NewLogger().Infof("%s", version.PluginVersion)
		},
	}

	return cmd
}
