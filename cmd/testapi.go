package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	apiclient "github.com/storageos/kubectl-storageos/api_client"
	"github.com/storageos/kubectl-storageos/api_client/openapi"
	"github.com/storageos/kubectl-storageos/pkg/consts"
	"github.com/storageos/kubectl-storageos/pkg/version"
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
			client, err := GetAPIClient()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
			defer cancel()
			_, _, err = client.Authenticate(ctx, "storageos", "storageos")
			if err != nil {
				return err
			}

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

			_, _, err = client.Authenticate(ctx, "storageos", "storageos")
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

func GetAPIClient() (*apiclient.ClientWithReauth, error) {
	userAgent := strings.Join([]string{"ondat kubectl plugin", version.PluginVersion}, "/")

	// TODO retrieve credentials like before
	username, password := "storageos", "storageos"

	// TODO make endpoints configurable
	basicOpenAPIClient, err := openapi.NewOpenAPI([]string{consts.DefaultStorageOSEndpoint}, userAgent)
	if err != nil {
		return nil, err
	}

	// client wrapper adding some extra useful methods on top
	internalClient := apiclient.NewClient(basicOpenAPIClient)
	clientWithReauth := apiclient.NewClientWithReauth(internalClient, username, password)

	return clientWithReauth, nil
}
