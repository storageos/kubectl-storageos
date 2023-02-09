package cmd

import (
	"github.com/spf13/cobra"

	"github.com/storageos/kubectl-storageos/pkg/forwarder"
	"github.com/storageos/kubectl-storageos/pkg/logger"
)

func UncordonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "uncordon",
		Short:              "Uncordons a node",
		Args:               cobra.MinimumNArgs(0),
		SilenceUsage:       true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		DisableFlagParsing: true,

		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			err := forwarder.ForwardToCLIPod(logger, storageosCommand(cmd, args))
			if err != nil {
				logger.Error(err.Error())
			}
		},
	}

	return cmd
}
