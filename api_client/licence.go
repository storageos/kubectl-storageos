package apiclient

import (
	"context"

	"github.com/storageos/kubectl-storageos/model"
)

// UpdateLicenceRequestParams contains optional request parameters for a update
// licence operation.
type UpdateLicenceRequestParams struct {
	CASVersion string
}

// UpdateLicence sends a new version of the licence to apply to the current
// cluster. It returns the new licence resource if correctly applied. It doesn't
// require a version but overwrite the licence using the last available version
// from the current licence.
func (c *Client) UpdateLicence(ctx context.Context, licence []byte, params *UpdateLicenceRequestParams) (*model.License, error) {
	if params == nil || params.CASVersion == "" {
		l, err := c.Transport.GetLicence(ctx)
		if err != nil {
			return nil, err
		}
		params = &UpdateLicenceRequestParams{
			CASVersion: l.Version,
		}
	}

	return c.Transport.UpdateLicence(ctx, licence, params)
}
