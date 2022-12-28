package apiclient

import (
	"context"

	"github.com/storageos/kubectl-storageos/api_client/openapi"
	"github.com/storageos/kubectl-storageos/model"
)

// UpdateLicence sends a new version of the licence to apply to the current
// cluster. It returns the new licence resource if correctly applied. It doesn't
// require a version but overwrite the licence using the last available version
// from the current licence.
func (c *client) UpdateLicence(ctx context.Context, licence []byte, params *openapi.UpdateLicenceRequestParams) (*model.License, error) {
	if params == nil || params.CASVersion == "" {
		l, err := c.GetLicence(ctx)
		if err != nil {
			return nil, err
		}
		params = &openapi.UpdateLicenceRequestParams{
			CASVersion: l.Version,
		}
	}

	return c.UpdateLicence(ctx, licence, params)
}
