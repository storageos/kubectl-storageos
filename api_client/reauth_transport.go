package apiclient

import (
	"context"
	"errors"
	"io"

	"github.com/storageos/kubectl-storageos/model"
)

// TransportWithReauth wraps calls to the internal api client with a
// re-authenticate and retry mechanism when an authentication error is
// encountered.
type TransportWithReauth struct {
	client Transport

	username, password string
}

// doWithReauth invokes fn, checking the resultant error.
//
//   - If the error is an *AuthenticationError then tr's credentials are
//     used to reauthenticate before returning the result from re-invoking fn.
//     If any errors occur during reauthentication, they are returned.
//   - Otherwise, the original error is returned to the caller.
func (tr *TransportWithReauth) doWithReauth(ctx context.Context, fn func() error) error {
	originalErr := fn()

	// If the returned error from fn indicates authentication failure then
	// fetch credentials from the provider, reauthenticate and try the request
	// one more time.
	//
	// This will reliably catch a cached auth session expiring.
	if errors.As(originalErr, &AuthenticationError{}) {
		// Attempt to reauth with credentials from the provider.
		_, err := tr.Authenticate(ctx, tr.username, tr.password)
		if err != nil {
			return err
		}

		return fn()
	}

	return originalErr
}

// NewTransportWithReauth wraps calls to transport with a retry on
// authentication failure, sourcing username and password from credentials.
func NewTransportWithReauth(transport Transport, username, password string) *TransportWithReauth {
	return &TransportWithReauth{
		client:   transport,
		username: username,
		password: password,
	}
}

// Authenticate is passed through and does not try to reauth. An
// authentication error here cannot be due to a session timeout.
func (tr *TransportWithReauth) Authenticate(ctx context.Context, username, password string) (AuthSession, error) {
	return tr.client.Authenticate(ctx, username, password)
}

func (tr *TransportWithReauth) UseAuthSession(ctx context.Context, session AuthSession) error {
	return tr.client.UseAuthSession(ctx, session)
}

func (tr *TransportWithReauth) GetUser(ctx context.Context, uid string) (*model.User, error) {
	var resource *model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetUser(ctx, uid)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) GetCluster(ctx context.Context) (*model.Cluster, error) {
	var resource *model.Cluster
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetCluster(ctx)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) GetLicence(ctx context.Context) (*model.License, error) {
	var resource *model.License
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetLicence(ctx)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) GetNode(ctx context.Context, nodeID string) (*model.Node, error) {
	var resource *model.Node
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetNode(ctx, nodeID)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) GetVolume(ctx context.Context, namespaceID string, volumeID string) (*model.Volume, error) {
	var resource *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetVolume(ctx, namespaceID, volumeID)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) GetNamespace(ctx context.Context, namespaceID string) (*model.Namespace, error) {
	var resource *model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetNamespace(ctx, namespaceID)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) GetDiagnostics(ctx context.Context) (io.ReadCloser, string, error) {
	var diagnosticsReadCloser io.ReadCloser
	var diagnosticsName string
	err := tr.doWithReauth(ctx, func() error {
		var err error
		diagnosticsReadCloser, diagnosticsName, err = tr.client.GetDiagnostics(ctx)

		return err
	})

	return diagnosticsReadCloser, diagnosticsName, err
}

func (tr *TransportWithReauth) GetSingleNodeDiagnostics(ctx context.Context, nodeID string) (io.ReadCloser, string, error) {
	var diagnosticsReadCloser io.ReadCloser
	var diagnosticsName string
	err := tr.doWithReauth(ctx, func() error {
		var err error
		diagnosticsReadCloser, diagnosticsName, err = tr.client.GetSingleNodeDiagnostics(ctx, nodeID)

		return err
	})

	return diagnosticsReadCloser, diagnosticsName, err
}

func (tr *TransportWithReauth) GetPolicyGroup(ctx context.Context, uid string) (*model.PolicyGroup, error) {
	var resource *model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetPolicyGroup(ctx, uid)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) ListNodes(ctx context.Context) ([]*model.Node, error) {
	var resources []*model.Node
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListNodes(ctx)

		return err
	})

	return resources, err
}

func (tr *TransportWithReauth) ListVolumes(ctx context.Context, namespaceID string) ([]*model.Volume, error) {
	var resources []*model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListVolumes(ctx, namespaceID)

		return err
	})

	return resources, err
}

func (tr *TransportWithReauth) ListNamespaces(ctx context.Context) ([]*model.Namespace, error) {
	var resources []*model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListNamespaces(ctx)

		return err
	})

	return resources, err
}

func (tr *TransportWithReauth) ListPolicyGroups(ctx context.Context) ([]*model.PolicyGroup, error) {
	var resources []*model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListPolicyGroups(ctx)

		return err
	})

	return resources, err
}

func (tr *TransportWithReauth) ListUsers(ctx context.Context) ([]*model.User, error) {
	var resources []*model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListUsers(ctx)

		return err
	})

	return resources, err
}

func (tr *TransportWithReauth) CreateUser(ctx context.Context, username, password string, withAdmin bool, groups ...string) (*model.User, error) {
	var resource *model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.CreateUser(ctx, username, password, withAdmin, groups...)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) CreateVolume(ctx context.Context, namespaceID string, name, description string, fs model.FsType, sizeBytes uint64, labels map[string]string, params *CreateVolumeRequestParams) (*model.Volume, error) {
	var resource *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.CreateVolume(
			ctx,
			namespaceID,
			name,
			description,
			fs,
			sizeBytes,
			labels,
			params,
		)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) CreatePolicyGroup(ctx context.Context, name string, specs []*model.PolicyGroupSpec) (*model.PolicyGroup, error) {
	var resource *model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.CreatePolicyGroup(ctx, name, specs)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) CreateNamespace(ctx context.Context, name string, labels map[string]string) (*model.Namespace, error) {
	var resource *model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.CreateNamespace(ctx, name, labels)

		return err
	})

	return resource, err
}

func (tr *TransportWithReauth) UpdateCluster(ctx context.Context, c *model.Cluster, params *UpdateClusterRequestParams) (*model.Cluster, error) {
	var updated *model.Cluster
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.UpdateCluster(ctx, c, params)

		return err
	})

	return updated, err
}

func (tr *TransportWithReauth) UpdateLicence(ctx context.Context, lic []byte, params *UpdateLicenceRequestParams) (*model.License, error) {
	var updated *model.License
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.UpdateLicence(ctx, lic, params)

		return err
	})

	return updated, err
}

func (tr *TransportWithReauth) DeleteNode(ctx context.Context, nodeID string, params *DeleteNodeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeleteNode(ctx, nodeID, params)
	})

	return err
}

func (tr *TransportWithReauth) SetReplicas(ctx context.Context, nsID string, volID string, numReplicas uint64, params *SetReplicasRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.SetReplicas(ctx, nsID, volID, numReplicas, params)
	})

	return err
}

func (tr *TransportWithReauth) UpdateVolume(ctx context.Context, nsID string, volID string, description string, labels map[string]string, params *UpdateVolumeRequestParams) (*model.Volume, error) {
	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.UpdateVolume(ctx, nsID, volID, description, labels, params)
		return err
	})

	return updated, err
}

func (tr *TransportWithReauth) ResizeVolume(ctx context.Context, nsID string, volID string, sizeBytes uint64, params *ResizeVolumeRequestParams) (*model.Volume, error) {
	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.ResizeVolume(ctx, nsID, volID, sizeBytes, params)
		return err
	})

	return updated, err
}

func (tr *TransportWithReauth) UpdateNFSVolumeExports(ctx context.Context, namespaceID string, volumeID string, exports []model.NFSExportConfig, params *UpdateNFSVolumeExportsRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.UpdateNFSVolumeExports(ctx, namespaceID, volumeID, exports, params)
	})

	return err
}

func (tr *TransportWithReauth) UpdateNFSVolumeMountEndpoint(ctx context.Context, namespaceID string, volumeID string, endpoint string, params *UpdateNFSVolumeMountEndpointRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.UpdateNFSVolumeMountEndpoint(ctx, namespaceID, volumeID, endpoint, params)
	})

	return err
}

func (tr *TransportWithReauth) SetFailureModeIntent(ctx context.Context, nsID string, volID string, intent string, params *SetFailureModeRequestParams) (*model.Volume, error) {
	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.SetFailureModeIntent(ctx, nsID, volID, intent, params)
		return err
	})

	return updated, err
}

func (tr *TransportWithReauth) SetFailureThreshold(ctx context.Context, nsID string, volID string, intent uint64, params *SetFailureModeRequestParams) (*model.Volume, error) {
	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.SetFailureThreshold(ctx, nsID, volID, intent, params)
		return err
	})

	return updated, err
}

func (tr *TransportWithReauth) DeleteVolume(ctx context.Context, namespaceID string, volumeID string, params *DeleteVolumeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeleteVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

func (tr *TransportWithReauth) DeleteNamespace(ctx context.Context, uid string, params *DeleteNamespaceRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeleteNamespace(ctx, uid, params)
	})

	return err
}

func (tr *TransportWithReauth) DeleteUser(ctx context.Context, uid string, params *DeleteUserRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeleteUser(ctx, uid, params)
	})

	return err
}

func (tr *TransportWithReauth) DeletePolicyGroup(ctx context.Context, uid string, params *DeletePolicyGroupRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeletePolicyGroup(ctx, uid, params)
	})

	return err
}

func (tr *TransportWithReauth) AttachVolume(ctx context.Context, namespaceID string, volumeID string, nodeID string) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.AttachVolume(ctx, namespaceID, volumeID, nodeID)
	})

	return err
}

func (tr *TransportWithReauth) AttachNFSVolume(ctx context.Context, namespaceID string, volumeID string, params *AttachNFSVolumeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.AttachNFSVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

func (tr *TransportWithReauth) SetCordoned(ctx context.Context, nodeID string, params *SetCordonedRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.SetCordoned(ctx, nodeID, params)
	})

	return err
}

func (tr *TransportWithReauth) EvictReplica(ctx context.Context, namespaceID string, id string, deploymentID string) error {
	return tr.doWithReauth(ctx, func() error {
		return tr.client.EvictReplica(ctx, namespaceID, id, deploymentID)
	})
}

func (tr *TransportWithReauth) AttemptPromotion(ctx context.Context, namespaceID string, id string, deploymentID string) error {
	return tr.doWithReauth(ctx, func() error {
		return tr.client.AttemptPromotion(ctx, namespaceID, id, deploymentID)
	})
}

func (tr *TransportWithReauth) DetachVolume(ctx context.Context, namespaceID string, volumeID string, params *DetachVolumeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DetachVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

func (tr *TransportWithReauth) AddDeploymentOnNode(ctx context.Context, namespaceID string, volumeID string, params *AddDeploymentOnNodeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.AddDeploymentOnNode(ctx, namespaceID, volumeID, params)
	})

	return err
}

func (tr *TransportWithReauth) SetPreferredEvictionCandidates(ctx context.Context, namespaceID string, volID string, params *SetPreferredEvictionCandidatesRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.SetPreferredEvictionCandidates(ctx, namespaceID, volID, params)
	})

	return err
}
