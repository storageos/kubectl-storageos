package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	apiclient "github.com/storageos/kubectl-storageos/api_client"
	"github.com/storageos/kubectl-storageos/pkg/utils"
)

func TestAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "testapi",
		Short:              "Does a GetVolumes api request to test",
		Args:               cobra.MinimumNArgs(0),
		SilenceUsage:       true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		DisableFlagParsing: true,

		RunE: func(cmd *cobra.Command, args []string) error {
			// Example of usage of the new api client

			cfg, err := utils.NewClientConfig()
			if err != nil {
				return err
			}
			// 1 - get client
			client, err := apiclient.GetAPIClient(cfg)
			if err != nil {
				return err
			}

			// 2 - authenticate
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
			defer cancel()

			username, password, err := utils.GetAPICredentialsFromSecret(cfg)
			if err != nil {
				return err
			}
			_, _, err = client.Authenticate(ctx, username, password)
			if err != nil {
				return err
			}

			// 3 - profit
			vols, err := client.GetAllVolumes(ctx)
			if err != nil {
				return err
			}

			fmt.Printf("Got %d volumes back\n", len(vols))

			vols, err = client.GetAllVolumes(ctx)
			if err != nil {
				return err
			}

			fmt.Printf("Got %d volumes back\n", len(vols))

			// testing the use of the cached token
			_, _, err = client.Authenticate(ctx, username, password)
			if err != nil {
				return err
			}

			vols, err = client.GetAllVolumes(ctx)
			if err != nil {
				return err
			}

			fmt.Printf("Got %d volumes back\n", len(vols))
			return nil
		},
	}

	return cmd
}
