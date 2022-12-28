package apiclient

import (
	"context"
	"errors"
	"io"

	"github.com/storageos/kubectl-storageos/model"
)

// TransportWithReauth wraps calls to an inner transport implementation with a
// re-authenticate and retry mechanism when an authentication error is
// encountered.
type TransportWithReauth struct {
	inner Transport

	username, password string
}

// Authenticate is passed through and does not try to reauth. An
// authentication error here cannot be due to a session timeout.
func (tr *TransportWithReauth) Authenticate(ctx context.Context, username, password string) (AuthSession, error) {
	return tr.inner.Authenticate(ctx, username, password)
}

// UseAuthSession is passed through and does not try to reauth.
func (tr *TransportWithReauth) UseAuthSession(ctx context.Context, session AuthSession) error {
	return tr.inner.UseAuthSession(ctx, session)
}

// GetUser wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) GetUser(ctx context.Context, uid string) (*model.User, error) {

	var resource *model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.GetUser(ctx, uid)

		return err
	})

	return resource, err
}

// GetCluster wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) GetCluster(ctx context.Context) (*model.Cluster, error) {

	var resource *model.Cluster
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.GetCluster(ctx)

		return err
	})

	return resource, err
}

// GetLicence wraps the inner transport'call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) GetLicence(ctx context.Context) (*model.License, error) {

	var resource *model.License
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.GetLicence(ctx)

		return err
	})

	return resource, err
}

// GetNode wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) GetNode(ctx context.Context, nodeID string) (*model.Node, error) {

	var resource *model.Node
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.GetNode(ctx, nodeID)

		return err
	})

	return resource, err
}

// GetVolume wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) GetVolume(ctx context.Context, namespaceID string, volumeID string) (*model.Volume, error) {

	var resource *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.GetVolume(ctx, namespaceID, volumeID)

		return err
	})

	return resource, err
}

// GetNamespace wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) GetNamespace(ctx context.Context, namespaceID string) (*model.Namespace, error) {
	var resource *model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.GetNamespace(ctx, namespaceID)

		return err
	})

	return resource, err
}

// GetDiagnostics wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) GetDiagnostics(ctx context.Context) (io.ReadCloser, string, error) {
	var diagnosticsReadCloser io.ReadCloser
	var diagnosticsName string
	err := tr.doWithReauth(ctx, func() error {
		var err error
		diagnosticsReadCloser, diagnosticsName, err = tr.inner.GetDiagnostics(ctx)

		return err
	})

	return diagnosticsReadCloser, diagnosticsName, err
}

// GetSingleNodeDiagnostics wraps the inner transport's call with a
// reauthenticate and retry upon encountering an authentication error.
func (tr *TransportWithReauth) GetSingleNodeDiagnostics(ctx context.Context, nodeID string) (io.ReadCloser, string, error) {
	var diagnosticsReadCloser io.ReadCloser
	var diagnosticsName string
	err := tr.doWithReauth(ctx, func() error {
		var err error
		diagnosticsReadCloser, diagnosticsName, err = tr.inner.GetSingleNodeDiagnostics(ctx, nodeID)

		return err
	})

	return diagnosticsReadCloser, diagnosticsName, err
}

// GetPolicyGroup wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) GetPolicyGroup(ctx context.Context, uid string) (*model.PolicyGroup, error) {

	var resource *model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.GetPolicyGroup(ctx, uid)

		return err
	})

	return resource, err
}

// ListNodes wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) ListNodes(ctx context.Context) ([]*model.Node, error) {

	var resources []*model.Node
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.inner.ListNodes(ctx)

		return err
	})

	return resources, err
}

// ListVolumes wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) ListVolumes(ctx context.Context, namespaceID string) ([]*model.Volume, error) {

	var resources []*model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.inner.ListVolumes(ctx, namespaceID)

		return err
	})

	return resources, err
}

// ListNamespaces wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) ListNamespaces(ctx context.Context) ([]*model.Namespace, error) {

	var resources []*model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.inner.ListNamespaces(ctx)

		return err
	})

	return resources, err
}

// ListPolicyGroups wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) ListPolicyGroups(ctx context.Context) ([]*model.PolicyGroup, error) {

	var resources []*model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.inner.ListPolicyGroups(ctx)

		return err
	})

	return resources, err
}

// ListUsers wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) ListUsers(ctx context.Context) ([]*model.User, error) {

	var resources []*model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.inner.ListUsers(ctx)

		return err
	})

	return resources, err
}

// CreateUser wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) CreateUser(ctx context.Context, username, password string, withAdmin bool, groups ...string) (*model.User, error) {

	var resource *model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.CreateUser(ctx, username, password, withAdmin, groups...)

		return err
	})

	return resource, err
}

// CreateVolume wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) CreateVolume(ctx context.Context, namespaceID string, name, description string, fs model.FsType, sizeBytes uint64, labels map[string]string, params *CreateVolumeRequestParams) (*model.Volume, error) {

	var resource *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.CreateVolume(
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

// CreatePolicyGroup wraps the inner transport's call with a reauthenticate
// and retry upon encountering an authentication error.
func (tr *TransportWithReauth) CreatePolicyGroup(ctx context.Context, name string, specs []*model.PolicyGroupSpec) (*model.PolicyGroup, error) {

	var resource *model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.CreatePolicyGroup(ctx, name, specs)

		return err
	})

	return resource, err
}

// CreateNamespace wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) CreateNamespace(ctx context.Context, name string, labels map[string]string) (*model.Namespace, error) {

	var resource *model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.inner.CreateNamespace(ctx, name, labels)

		return err
	})

	return resource, err
}

// UpdateCluster wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) UpdateCluster(ctx context.Context, c *model.Cluster, params *UpdateClusterRequestParams) (*model.Cluster, error) {

	var updated *model.Cluster
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.inner.UpdateCluster(ctx, c, params)

		return err
	})

	return updated, err
}

// UpdateLicence wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) UpdateLicence(ctx context.Context, lic []byte, params *UpdateLicenceRequestParams) (*model.License, error) {

	var updated *model.License
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.inner.UpdateLicence(ctx, lic, params)

		return err
	})

	return updated, err
}

// DeleteNode wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) DeleteNode(ctx context.Context, nodeID string, params *DeleteNodeRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.DeleteNode(ctx, nodeID, params)
	})

	return err
}

// SetReplicas wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) SetReplicas(ctx context.Context, nsID string, volID string, numReplicas uint64, params *SetReplicasRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.SetReplicas(ctx, nsID, volID, numReplicas, params)
	})

	return err
}

// UpdateVolume wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) UpdateVolume(ctx context.Context, nsID string, volID string, description string, labels map[string]string, params *UpdateVolumeRequestParams) (*model.Volume, error) {

	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.inner.UpdateVolume(ctx, nsID, volID, description, labels, params)
		return err
	})

	return updated, err
}

// ResizeVolume wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) ResizeVolume(ctx context.Context, nsID string, volID string, sizeBytes uint64, params *ResizeVolumeRequestParams) (*model.Volume, error) {

	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.inner.ResizeVolume(ctx, nsID, volID, sizeBytes, params)
		return err
	})

	return updated, err
}

// UpdateNFSVolumeExports wraps the inner transport's call with a reauthenticate
// and retry upon encountering an authentication error.
func (tr *TransportWithReauth) UpdateNFSVolumeExports(ctx context.Context, namespaceID string, volumeID string, exports []model.NFSExportConfig, params *UpdateNFSVolumeExportsRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.UpdateNFSVolumeExports(ctx, namespaceID, volumeID, exports, params)
	})

	return err
}

// UpdateNFSVolumeMountEndpoint wraps the inner transport's call with a
// reauthenticate and retry upon encountering an authentication error.
func (tr *TransportWithReauth) UpdateNFSVolumeMountEndpoint(ctx context.Context, namespaceID string, volumeID string, endpoint string, params *UpdateNFSVolumeMountEndpointRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.UpdateNFSVolumeMountEndpoint(ctx, namespaceID, volumeID, endpoint, params)
	})

	return err
}

// SetFailureModeIntent wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) SetFailureModeIntent(ctx context.Context, nsID string, volID string, intent string, params *SetFailureModeRequestParams) (*model.Volume, error) {

	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.inner.SetFailureModeIntent(ctx, nsID, volID, intent, params)
		return err
	})

	return updated, err
}

// SetFailureThreshold wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) SetFailureThreshold(ctx context.Context, nsID string, volID string, intent uint64, params *SetFailureModeRequestParams) (*model.Volume, error) {

	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.inner.SetFailureThreshold(ctx, nsID, volID, intent, params)
		return err
	})

	return updated, err
}

// DeleteVolume wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) DeleteVolume(ctx context.Context, namespaceID string, volumeID string, params *DeleteVolumeRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.DeleteVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

// DeleteNamespace wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) DeleteNamespace(ctx context.Context, uid string, params *DeleteNamespaceRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.DeleteNamespace(ctx, uid, params)
	})

	return err
}

// DeleteUser wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) DeleteUser(ctx context.Context, uid string, params *DeleteUserRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.DeleteUser(ctx, uid, params)
	})

	return err
}

// DeletePolicyGroup wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) DeletePolicyGroup(ctx context.Context, uid string, params *DeletePolicyGroupRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.DeletePolicyGroup(ctx, uid, params)
	})

	return err
}

// AttachVolume wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) AttachVolume(ctx context.Context, namespaceID string, volumeID string, nodeID string) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.AttachVolume(ctx, namespaceID, volumeID, nodeID)
	})

	return err
}

// AttachNFSVolume wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) AttachNFSVolume(ctx context.Context, namespaceID string, volumeID string, params *AttachNFSVolumeRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.AttachNFSVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

// SetCordoned wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) SetCordoned(ctx context.Context, nodeID string, params *SetCordonedRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.SetCordoned(ctx, nodeID, params)
	})

	return err
}

func (tr *TransportWithReauth) EvictReplica(ctx context.Context, namespaceID string, id string, deploymentID string) error {
	return tr.doWithReauth(ctx, func() error {
		return tr.inner.EvictReplica(ctx, namespaceID, id, deploymentID)
	})
}

func (tr *TransportWithReauth) AttemptPromotion(ctx context.Context, namespaceID string, id string, deploymentID string) error {
	return tr.doWithReauth(ctx, func() error {
		return tr.inner.AttemptPromotion(ctx, namespaceID, id, deploymentID)
	})
}

// DetachVolume wraps the inner transport's call with a reauthenticate and retry
// upon encountering an authentication error.
func (tr *TransportWithReauth) DetachVolume(ctx context.Context, namespaceID string, volumeID string, params *DetachVolumeRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.DetachVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

// AddDeploymentOnNode wraps the inner transport's call with a reauthenticate and
// retry upon encountering an authentication error.
func (tr *TransportWithReauth) AddDeploymentOnNode(ctx context.Context, namespaceID string, volumeID string, params *AddDeploymentOnNodeRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.AddDeploymentOnNode(ctx, namespaceID, volumeID, params)
	})

	return err
}

// SetPreferredEvictionCandidates wraps the inner transport's call with a
// reauthenticate and retry upon encountering an authentication error.
func (tr *TransportWithReauth) SetPreferredEvictionCandidates(ctx context.Context, namespaceID string, volID string, params *SetPreferredEvictionCandidatesRequestParams) error {

	err := tr.doWithReauth(ctx, func() error {
		return tr.inner.SetPreferredEvictionCandidates(ctx, namespaceID, volID, params)
	})

	return err
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
		inner:    transport,
		username: username,
		password: password,
	}
}
