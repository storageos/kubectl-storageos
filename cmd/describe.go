package cmd

import (
	"github.com/spf13/cobra"

	"github.com/storageos/kubectl-storageos/pkg/forwarder"
	"github.com/storageos/kubectl-storageos/pkg/logger"
)

func DescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "describe",
		Short:              "Fetch extended details for resources",
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
