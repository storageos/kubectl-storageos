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
	client := apiclient.New()

	// TODO get version of plugin
	userAgent := strings.Join([]string{"ondat plugin", "version.String()"}, "/")

	// disabledAuthCache, err := config.AuthCacheDisabled()
	// if err != nil {
	// 	return err
	// }
	disabledAuthCache := true

	transport, err := func() (apiclient.Transport, error) {
		// TODO get from env vars
		transport, err := openapi.NewOpenAPI([]string{"http://localhost:5705"}, userAgent)
		if err != nil {
			return nil, err
		}

		if disabledAuthCache {
			return transport, nil
		}

		// Ensure that the cache dir exists
		// TODO get from env vars?
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return transport, nil
		}

		err = os.Mkdir(cacheDir, 0700)
		switch {
		case err == nil, os.IsExist(err):
			// This is ok
		default:
			return nil, err
		}

		return apiclient.NewAuthCachedTransport(transport), nil
	}()
	if err != nil {
		return nil, err
	}

	// TODO get from somewhere
	username, password := "storageos", "storageos"

	// Wrap the transport implementation in a re-authenticate layer.
	return client, client.ConfigureTransport(
		apiclient.NewTransportWithReauth(transport, username, password),
	)
}
