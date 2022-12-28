package cmd

import (
	"context"
	"fmt"
	"os"
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
			client, err := Test()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
			defer cancel()
			_, err = client.Authenticate(ctx, "storageos", "storageos")
			if err != nil {
				return err
			}

			vols, err := client.GetAllVolumes(ctx)
			if err != nil {
				return err
			}

			fmt.Printf("Got %d volumes back\n", len(vols))
			return nil
		},
	}

	return cmd
}

func Test() (*apiclient.Client, error) {
	userAgent := strings.Join([]string{"ondat kubectl plugin", version.PluginVersion}, "/")

	// TODO get from somewhere
	username, password := "storageos", "storageos"

	// disabledAuthCache, err := config.AuthCacheDisabled()
	// if err != nil {
	// 	return err
	// }
	disabledAuthCache := true

	// TODO make endpoints configurable
	// get the OpenAPI client
	var client apiclient.Transport
	client, err := openapi.NewOpenAPI([]string{consts.DefaultStorageOSEndpoint}, userAgent)
	if err != nil {
		return nil, err
	}

	clientWithReauth := apiclient.NewTransportWithReauth(client, username, password)

	// internal client wrapper
	internalClient := &apiclient.Client{
		Transport: clientWithReauth,
	}

	if disabledAuthCache {
		return internalClient, nil
	}

	// TODO make dir configurable
	// Ensure that the cache dir exists
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return internalClient, nil
	}

	err = os.Mkdir(cacheDir, 0700)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	// wrap transport with authentication chaching capacilities
	clientWithReauthAndCaching := apiclient.NewAuthCachedTransport(client)
	internalClient.Transport = clientWithReauthAndCaching

	return internalClient, nil
}
