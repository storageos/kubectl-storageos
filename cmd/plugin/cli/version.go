package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "version",
		Args:         cobra.MinimumNArgs(0),
		Short:        "Show kubectl storageos version",
		Long:         `Show kubectl storageos version`,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s\n", Version)
		},
	}

	return cmd
}
