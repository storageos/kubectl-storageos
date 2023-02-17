package apiclient

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"

	"github.com/storageos/kubectl-storageos/api_client/openapi"
	"github.com/storageos/kubectl-storageos/model"
)

// GetVolumeByName requests the volume resource which has name in namespace.
//
// The resource model for the API is build around using unique identifiers,
// so this operation is inherently more expensive than the corresponding
// GetVolume() operation.
//
// Retrieving a volume resource by name involves requesting a list of all
// volumes in the namespace from the StorageOS API and returning the first one
// where the name matches.
func (c *client) GetVolumeByName(ctx context.Context, namespaceID string, name string) (*model.Volume, error) {
	volumes, err := c.ListVolumes(ctx, namespaceID)
	if err != nil {
		return nil, err
	}

	for _, v := range volumes {
		if v.Name == name {
			return v, nil
		}
	}

	return nil, openapi.NewVolumeNameNotFoundError(name)
}

// GetNamespaceVolumesByUID requests basic information for each volume resource in
// namespace from the StorageOS API.
//
// The returned list is filtered using uids so that it contains only those
// resources which have a matching ID. Omitting uids will skip the filtering.
func (c *client) GetNamespaceVolumesByUID(ctx context.Context, namespaceID string, uids ...string) ([]*model.Volume, error) {
	volumes, err := c.ListVolumes(ctx, namespaceID)
	if err != nil {
		return nil, err
	}

	return filterVolumesForUIDs(volumes, uids...)
}

// GetNamespaceVolumesByName requests basic information for each volume resource in
// namespace from the StorageOS API.
//
// The returned list is filtered using names so that it contains only those
// resources which have a matching name. Omitting names will skip the filtering.
func (c *client) GetNamespaceVolumesByName(ctx context.Context, namespaceID string, names ...string) ([]*model.Volume, error) {
	volumes, err := c.ListVolumes(ctx, namespaceID)
	if err != nil {
		return nil, err
	}

	return filterVolumesForNames(volumes, names...)
}

// GetAllVolumes requests basic information for each volume resource in every
// namespace exposed by the StorageOS API to the authenticated user.
func (c *client) GetAllVolumes(ctx context.Context) ([]*model.Volume, error) {
	return c.fetchAllVolumesParallel(ctx)
}

// fetchAllVolumesParallel requests the list of all namespaces from the
// StorageOS API, then requests the list of volumes within each namespace,
// calling all of them in parallel, returning an aggregate list of the volumes
// returned.
//
// If access is not granted when listing volumes for a retrieved namespace it
// is noted but will not return an error. Only if access is denied for all
// attempts will this return a permissions error.
//
// If any of the call returns an error:
//   - the context is canceled so all pending requests are cut
//   - this method returns an error
func (c *client) fetchAllVolumesParallel(ctx context.Context) ([]*model.Volume, error) {
	namespaces, err := c.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	// The derived Context is canceled the first time a function passed to Go
	// returns a non-nil error or the first time Wait returns, whichever occurs
	// first.
	group, ctx := errgroup.WithContext(ctx)

	results := make(chan []*model.Volume, len(namespaces))

	for _, ns := range namespaces {
		ns := ns

		// Go calls the given function in a new goroutine.
		//
		// The first call to return a non-nil error cancels the group; its error
		// will be returned by Wait.
		group.Go(func() error {

			nsvols, err := c.ListVolumes(ctx, ns.ID)
			switch {
			case err == nil, errors.As(err, &openapi.UnauthorisedError{}):
				// For an unauthorised error, ignore - its not fatal to the operation.
			default:
				return err
			}

			results <- nsvols
			return nil
		})
	}

	// blocks until all function calls from the Go method have returned
	if err := group.Wait(); err != nil {
		return nil, err
	}

	close(results)

	// merge the results
	volumes := []*model.Volume{}
	for r := range results {
		volumes = append(volumes, r...)
	}

	return volumes, nil
}

// filterVolumesForUIDs will return a subset of volumes containing resources
// which have one of the provided uids. If uids is not provided, volumes is
// returned as is.
//
// If there is no resource for a given uid then an error is returned, thus
// this is a strict helper.
func filterVolumesForUIDs(volumes []*model.Volume, uids ...string) ([]*model.Volume, error) {
	if len(uids) == 0 {
		return volumes, nil
	}

	retrieved := map[string]*model.Volume{}

	for _, v := range volumes {
		retrieved[v.ID] = v
	}

	filtered := make([]*model.Volume, 0, len(uids))

	for _, idVar := range uids {
		v, ok := retrieved[idVar]
		if !ok {
			return nil, openapi.NewVolumeIDNotFoundError(idVar)
		}
		filtered = append(filtered, v)
	}

	return filtered, nil
}

// filterVolumesForNames will return a subset of volumes containing resources
// which have one of the provided names. If names is not provided, volumes is
// returned as is.
//
// If there is no resource for a given name then an error is returned, thus
// this is a strict helper.
func filterVolumesForNames(volumes []*model.Volume, names ...string) ([]*model.Volume, error) {
	if len(names) == 0 {
		return volumes, nil
	}

	retrieved := map[string]*model.Volume{}

	for _, v := range volumes {
		retrieved[v.Name] = v
	}

	filtered := make([]*model.Volume, 0, len(names))

	for _, name := range names {
		v, ok := retrieved[name]
		if !ok {
			return nil, openapi.NewVolumeNameNotFoundError(name)
		}
		filtered = append(filtered, v)
	}

	return filtered, nil
}

// SetReplicas sends a new replica count number for updating the selected volume.
//
// The behaviour of the operation is dictated by params:
//
//	 Version constraints:
//		- If params is nil or params.CASVersion is empty then the detach request is
//		unconditional
//		- If params.CASVersion is set, the request is conditional upon it matching
//		the volume entity's version as seen by the server.
func (c *client) SetReplicas(ctx context.Context, nsID string, volID string, numReplicas uint64, params *openapi.SetReplicasRequestParams) error {
	if params == nil || params.CASVersion == "" {
		v, err := c.GetVolume(ctx, nsID, volID)
		if err != nil {
			return err
		}

		params = &openapi.SetReplicasRequestParams{CASVersion: v.Version}
	}

	return c.SetReplicas(ctx, nsID, volID, numReplicas, params)
}

// UpdateVolumeDescription sends a new description for updating the selected volume.
//
// The behaviour of the operation is dictated by params:
//
//	 Version constraints:
//		- If params.CASVersion is set, the request is conditional upon it matching
//		the volume entity's version as seen by the server.
//
//
//	 Asynchrony:
//	 - If params is nil or params.AsyncMax is empty/zero valued then the create
//	 request is performed synchronously.
//	 - If params.AsyncMax is set, the request is performed asynchronously using
//	 the duration given as the maximum amount of time allowed for the request
//	 before it times out.
func (c *client) UpdateVolumeDescription(ctx context.Context, nsID string, volID string, description string, params *openapi.UpdateVolumeRequestParams) (*model.Volume, error) {
	v, err := c.GetVolume(ctx, nsID, volID)
	if err != nil {
		return nil, err
	}

	newParams := &openapi.UpdateVolumeRequestParams{}

	if params != nil {
		if params.CASVersion != "" {
			newParams.CASVersion = params.CASVersion
		}
		if params.AsyncMax != 0 {
			newParams.AsyncMax = params.AsyncMax
		}
	}

	if newParams.CASVersion == "" {
		newParams.CASVersion = v.Version
	}

	return c.UpdateVolume(ctx, nsID, volID, description, v.Labels, params)
}

// UpdateVolumeLabels sends a new set of labels for updating the selected volume
// labels.
//
// The behaviour of the operation is dictated by params:
//
//	Version constraints:
//	- If params.CASVersion is set, the request is conditional upon it matching
//	the volume entity's version as seen by the server.
//
//
//	Asynchrony:
//	- If params is nil or params.AsyncMax is empty/zero valued then the create
//	request is performed synchronously.
//	- If params.AsyncMax is set, the request is performed asynchronously using
//	the duration given as the maximum amount of time allowed for the request
//	before it times out.
func (c *client) UpdateVolumeLabels(ctx context.Context, nsID string, volID string, labels map[string]string, params *openapi.UpdateVolumeRequestParams) (*model.Volume, error) {
	v, err := c.GetVolume(ctx, nsID, volID)
	if err != nil {
		return nil, err
	}

	newParams := &openapi.UpdateVolumeRequestParams{}

	if params != nil {
		if params.CASVersion != "" {
			newParams.CASVersion = params.CASVersion
		}
		if params.AsyncMax != 0 {
			newParams.AsyncMax = params.AsyncMax
		}
	}

	if newParams.CASVersion == "" {
		newParams.CASVersion = v.Version
	}

	return c.UpdateVolume(ctx, nsID, volID, v.Description, labels, params)
}

// ResizeVolume sends a new volume size for updating the selected volume.
//
// The behaviour of the operation is dictated by params:
//
//	 Version constraints:
//		- If params is nil or params.CASVersion is empty then the resize request is
//		unconditional
//		- If params.CASVersion is set, the request is conditional upon it matching
//		the volume entity's version as seen by the server.
func (c *client) ResizeVolume(ctx context.Context, nsID string, volID string, sizeBytes uint64, params *openapi.ResizeVolumeRequestParams) (*model.Volume, error) {

	newParams := &openapi.ResizeVolumeRequestParams{}

	if params == nil || params.CASVersion == "" {
		v, err := c.GetVolume(ctx, nsID, volID)
		if err != nil {
			return nil, err
		}
		newParams.CASVersion = v.Version
	} else {
		newParams.CASVersion = params.CASVersion
	}

	if params != nil && params.AsyncMax != 0 {
		newParams.AsyncMax = params.AsyncMax
	}

	return c.ResizeVolume(ctx, nsID, volID, sizeBytes, newParams)
}

// AttachNFSVolume sends a new attach volume request for NFS.
//
// The behaviour of the operation is dictated by params:
//
//	 Version constraints:
//		- If params is nil or params.CASVersion is empty then the attach request is
//		unconditional
//		- If params.CASVersion is set, the request is conditional upon it matching
//		the volume entity's version as seen by the server.
func (c *client) AttachNFSVolume(ctx context.Context, namespaceID string, volumeID string, params *openapi.AttachNFSVolumeRequestParams) error {

	newParams := &openapi.AttachNFSVolumeRequestParams{}

	if params == nil || params.CASVersion == "" {
		v, err := c.GetVolume(ctx, namespaceID, volumeID)
		if err != nil {
			return err
		}
		newParams.CASVersion = v.Version
	} else {
		newParams.CASVersion = params.CASVersion
	}

	if params != nil && params.AsyncMax != 0 {
		newParams.AsyncMax = params.AsyncMax
	}

	return c.AttachNFSVolume(ctx, namespaceID, volumeID, newParams)
}

// UpdateNFSVolumeExports sends a new NFS volume export update request.
//
// The behaviour of the operation is dictated by params:
//
//	Version constraints:
//	- If params is nil or params.CASVersion is empty then the attach request is
//	  unconditional
//	- If params.CASVersion is set, the request is conditional upon it matching
//	  the volume entity's version as seen by the server.
func (c *client) UpdateNFSVolumeExports(
	ctx context.Context,
	namespaceID string,
	volumeID string,
	exports []model.NFSExportConfig,
	params *openapi.UpdateNFSVolumeExportsRequestParams,
) error {

	newParams := &openapi.UpdateNFSVolumeExportsRequestParams{}

	if params == nil || params.CASVersion == "" {
		v, err := c.GetVolume(ctx, namespaceID, volumeID)
		if err != nil {
			return err
		}
		newParams.CASVersion = v.Version
	} else {
		newParams.CASVersion = params.CASVersion
	}

	if params != nil && params.AsyncMax != 0 {
		newParams.AsyncMax = params.AsyncMax
	}

	return c.UpdateNFSVolumeExports(ctx, namespaceID, volumeID, exports, newParams)
}

// UpdateNFSVolumeMountEndpoint sends a new NFS volume mount endpoint update
// request.
//
// The behaviour of the operation is dictated by params:
//
//	Version constraints:
//	- If params is nil or params.CASVersion is empty then the attach request is
//	  unconditional
//	- If params.CASVersion is set, the request is conditional upon it matching
//	  the volume entity's version as seen by the server.
func (c *client) UpdateNFSVolumeMountEndpoint(
	ctx context.Context,
	namespaceID string,
	volumeID string,
	endpoint string,
	params *openapi.UpdateNFSVolumeMountEndpointRequestParams,
) error {

	newParams := &openapi.UpdateNFSVolumeMountEndpointRequestParams{}

	if params == nil || params.CASVersion == "" {
		v, err := c.GetVolume(ctx, namespaceID, volumeID)
		if err != nil {
			return err
		}
		newParams.CASVersion = v.Version
	} else {
		newParams.CASVersion = params.CASVersion
	}

	if params != nil && params.AsyncMax != 0 {
		newParams.AsyncMax = params.AsyncMax
	}

	return c.UpdateNFSVolumeMountEndpoint(ctx, namespaceID, volumeID, endpoint, newParams)
}

// SetFailureModeIntent attempts to set the intent based failure mode for the
// target volume to the behaviour with the provided name.
//
// The behaviour of the operation is dictated by params:
//
//	Version constraints:
//	- If params is nil or params.CASVersion is empty then the attach request is
//	  unconditional
//	- If params.CASVersion is set, the request is conditional upon it matching
//	  the volume entity's version as seen by the server.
func (c *client) SetFailureModeIntent(
	ctx context.Context,
	namespaceID string,
	volumeID string,
	intent string,
	params *openapi.SetFailureModeRequestParams,
) (*model.Volume, error) {
	newParams := &openapi.SetFailureModeRequestParams{}

	if params == nil || params.CASVersion == "" {
		v, err := c.GetVolume(ctx, namespaceID, volumeID)
		if err != nil {
			return nil, err
		}
		newParams.CASVersion = v.Version
	} else {
		newParams.CASVersion = params.CASVersion
	}

	return c.SetFailureModeIntent(ctx, namespaceID, volumeID, intent, newParams)
}

// SetFailureThreshold attempts to set the failure mode for the target volume
// to the numerical threshold given.
//
// The behaviour of the operation is dictated by params:
//
//	Version constraints:
//	- If params is nil or params.CASVersion is empty then the attach request is
//	  unconditional
//	- If params.CASVersion is set, the request is conditional upon it matching
//	  the volume entity's version as seen by the server.
func (c *client) SetFailureThreshold(
	ctx context.Context,
	namespaceID string,
	volumeID string,
	threshold uint64,
	params *openapi.SetFailureModeRequestParams,
) (*model.Volume, error) {
	newParams := &openapi.SetFailureModeRequestParams{}

	if params == nil || params.CASVersion == "" {
		v, err := c.GetVolume(ctx, namespaceID, volumeID)
		if err != nil {
			return nil, err
		}
		newParams.CASVersion = v.Version
	} else {
		newParams.CASVersion = params.CASVersion
	}

	return c.SetFailureThreshold(ctx, namespaceID, volumeID, threshold, newParams)
}
