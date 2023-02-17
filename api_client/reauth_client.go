package apiclient

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/storageos/kubectl-storageos/api_client/openapi"
	"github.com/storageos/kubectl-storageos/model"
)

// ClientWithReauth wraps calls to the internal api client with a
// re-authenticate and retry mechanism when an authentication error is
// encountered.
type ClientWithReauth struct {
	client

	username, password string
}

// doWithReauth invokes fn, checking the resultant error.
//
//   - If the error is an *AuthenticationError then tr's credentials are
//     used to reauthenticate before returning the result from re-invoking fn.
//     If any errors occur during reauthentication, they are returned.
//   - Otherwise, the original error is returned to the caller.
func (tr *ClientWithReauth) doWithReauth(ctx context.Context, fn func() error) error {
	originalErr := fn()

	// If the returned error from fn indicates authentication failure then
	// fetch credentials from the provider, reauthenticate and try the request
	// one more time.
	//
	// This will reliably catch a cached auth session expiring.
	if errors.As(originalErr, &openapi.AuthenticationError{}) {
		// Attempt to reauth with credentials from the provider.
		_, _, err := tr.Authenticate(ctx, tr.username, tr.password)
		if err != nil {
			return err
		}

		return fn()
	}

	return originalErr
}

// NewClientWithReauth wraps calls to transport with a retry on authentication
// failure, sourcing username and password from credentials.
func NewClientWithReauth(client client, username, password string) *ClientWithReauth {
	return &ClientWithReauth{
		client:   client,
		username: username,
		password: password,
	}
}

// Authenticate is passed through and does not try to reauth. An authentication
// error here cannot be due to a session timeout.
func (tr *ClientWithReauth) Authenticate(ctx context.Context, username, password string) (string, time.Time, error) {
	return tr.client.Authenticate(ctx, username, password)
}

func (tr *ClientWithReauth) UseAuthSession(ctx context.Context, token string) error {
	return tr.client.UseAuthSession(ctx, token)
}

func (tr *ClientWithReauth) GetUser(ctx context.Context, uid string) (*model.User, error) {
	var resource *model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetUser(ctx, uid)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) GetCluster(ctx context.Context) (*model.Cluster, error) {
	var resource *model.Cluster
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetCluster(ctx)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) GetLicence(ctx context.Context) (*model.License, error) {
	var resource *model.License
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetLicence(ctx)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) GetNode(ctx context.Context, nodeID string) (*model.Node, error) {
	var resource *model.Node
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetNode(ctx, nodeID)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) GetVolume(ctx context.Context, namespaceID string, volumeID string) (*model.Volume, error) {
	var resource *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetVolume(ctx, namespaceID, volumeID)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) GetNamespace(ctx context.Context, namespaceID string) (*model.Namespace, error) {
	var resource *model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetNamespace(ctx, namespaceID)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) GetDiagnostics(ctx context.Context) (io.ReadCloser, string, error) {
	var diagnosticsReadCloser io.ReadCloser
	var diagnosticsName string
	err := tr.doWithReauth(ctx, func() error {
		var err error
		diagnosticsReadCloser, diagnosticsName, err = tr.client.GetDiagnostics(ctx)

		return err
	})

	return diagnosticsReadCloser, diagnosticsName, err
}

func (tr *ClientWithReauth) GetSingleNodeDiagnostics(ctx context.Context, nodeID string) (io.ReadCloser, string, error) {
	var diagnosticsReadCloser io.ReadCloser
	var diagnosticsName string
	err := tr.doWithReauth(ctx, func() error {
		var err error
		diagnosticsReadCloser, diagnosticsName, err = tr.client.GetSingleNodeDiagnostics(ctx, nodeID)

		return err
	})

	return diagnosticsReadCloser, diagnosticsName, err
}

func (tr *ClientWithReauth) GetPolicyGroup(ctx context.Context, uid string) (*model.PolicyGroup, error) {
	var resource *model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.GetPolicyGroup(ctx, uid)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) ListNodes(ctx context.Context) ([]*model.Node, error) {
	var resources []*model.Node
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListNodes(ctx)

		return err
	})

	return resources, err
}

func (tr *ClientWithReauth) ListVolumes(ctx context.Context, namespaceID string) ([]*model.Volume, error) {
	var resources []*model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListVolumes(ctx, namespaceID)

		return err
	})

	return resources, err
}

func (tr *ClientWithReauth) ListNamespaces(ctx context.Context) ([]*model.Namespace, error) {
	var resources []*model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListNamespaces(ctx)

		return err
	})

	return resources, err
}

func (tr *ClientWithReauth) ListPolicyGroups(ctx context.Context) ([]*model.PolicyGroup, error) {
	var resources []*model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListPolicyGroups(ctx)

		return err
	})

	return resources, err
}

func (tr *ClientWithReauth) ListUsers(ctx context.Context) ([]*model.User, error) {
	var resources []*model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resources, err = tr.client.ListUsers(ctx)

		return err
	})

	return resources, err
}

func (tr *ClientWithReauth) CreateUser(ctx context.Context, username, password string, withAdmin bool, groups ...string) (*model.User, error) {
	var resource *model.User
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.CreateUser(ctx, username, password, withAdmin, groups...)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) CreateVolume(ctx context.Context, namespaceID string, name, description string, fs model.FsType, sizeBytes uint64, labels map[string]string, params *openapi.CreateVolumeRequestParams) (*model.Volume, error) {
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

func (tr *ClientWithReauth) CreatePolicyGroup(ctx context.Context, name string, specs []*model.PolicyGroupSpec) (*model.PolicyGroup, error) {
	var resource *model.PolicyGroup
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.CreatePolicyGroup(ctx, name, specs)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) CreateNamespace(ctx context.Context, name string, labels map[string]string) (*model.Namespace, error) {
	var resource *model.Namespace
	err := tr.doWithReauth(ctx, func() error {
		var err error
		resource, err = tr.client.CreateNamespace(ctx, name, labels)

		return err
	})

	return resource, err
}

func (tr *ClientWithReauth) UpdateCluster(ctx context.Context, c *model.Cluster, params *openapi.UpdateClusterRequestParams) (*model.Cluster, error) {
	var updated *model.Cluster
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.UpdateCluster(ctx, c, params)

		return err
	})

	return updated, err
}

func (tr *ClientWithReauth) UpdateLicence(ctx context.Context, lic []byte, params *openapi.UpdateLicenceRequestParams) (*model.License, error) {
	var updated *model.License
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.UpdateLicence(ctx, lic, params)

		return err
	})

	return updated, err
}

func (tr *ClientWithReauth) DeleteNode(ctx context.Context, nodeID string, params *openapi.DeleteNodeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeleteNode(ctx, nodeID, params)
	})

	return err
}

func (tr *ClientWithReauth) SetReplicas(ctx context.Context, nsID string, volID string, numReplicas uint64, params *openapi.SetReplicasRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.SetReplicas(ctx, nsID, volID, numReplicas, params)
	})

	return err
}

func (tr *ClientWithReauth) UpdateVolume(ctx context.Context, nsID string, volID string, description string, labels map[string]string, params *openapi.UpdateVolumeRequestParams) (*model.Volume, error) {
	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.UpdateVolume(ctx, nsID, volID, description, labels, params)
		return err
	})

	return updated, err
}

func (tr *ClientWithReauth) ResizeVolume(ctx context.Context, nsID string, volID string, sizeBytes uint64, params *openapi.ResizeVolumeRequestParams) (*model.Volume, error) {
	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.ResizeVolume(ctx, nsID, volID, sizeBytes, params)
		return err
	})

	return updated, err
}

func (tr *ClientWithReauth) UpdateNFSVolumeExports(ctx context.Context, namespaceID string, volumeID string, exports []model.NFSExportConfig, params *openapi.UpdateNFSVolumeExportsRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.UpdateNFSVolumeExports(ctx, namespaceID, volumeID, exports, params)
	})

	return err
}

func (tr *ClientWithReauth) UpdateNFSVolumeMountEndpoint(ctx context.Context, namespaceID string, volumeID string, endpoint string, params *openapi.UpdateNFSVolumeMountEndpointRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.UpdateNFSVolumeMountEndpoint(ctx, namespaceID, volumeID, endpoint, params)
	})

	return err
}

func (tr *ClientWithReauth) SetFailureModeIntent(ctx context.Context, nsID string, volID string, intent string, params *openapi.SetFailureModeRequestParams) (*model.Volume, error) {
	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.SetFailureModeIntent(ctx, nsID, volID, intent, params)
		return err
	})

	return updated, err
}

func (tr *ClientWithReauth) SetFailureThreshold(ctx context.Context, nsID string, volID string, intent uint64, params *openapi.SetFailureModeRequestParams) (*model.Volume, error) {
	var updated *model.Volume
	err := tr.doWithReauth(ctx, func() error {
		var err error
		updated, err = tr.client.SetFailureThreshold(ctx, nsID, volID, intent, params)
		return err
	})

	return updated, err
}

func (tr *ClientWithReauth) DeleteVolume(ctx context.Context, namespaceID string, volumeID string, params *openapi.DeleteVolumeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeleteVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

func (tr *ClientWithReauth) DeleteNamespace(ctx context.Context, uid string, params *openapi.DeleteNamespaceRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeleteNamespace(ctx, uid, params)
	})

	return err
}

func (tr *ClientWithReauth) DeleteUser(ctx context.Context, uid string, params *openapi.DeleteUserRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeleteUser(ctx, uid, params)
	})

	return err
}

func (tr *ClientWithReauth) DeletePolicyGroup(ctx context.Context, uid string, params *openapi.DeletePolicyGroupRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DeletePolicyGroup(ctx, uid, params)
	})

	return err
}

func (tr *ClientWithReauth) AttachVolume(ctx context.Context, namespaceID string, volumeID string, nodeID string) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.AttachVolume(ctx, namespaceID, volumeID, nodeID)
	})

	return err
}

func (tr *ClientWithReauth) AttachNFSVolume(ctx context.Context, namespaceID string, volumeID string, params *openapi.AttachNFSVolumeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.AttachNFSVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

func (tr *ClientWithReauth) SetCordoned(ctx context.Context, nodeID string, params *openapi.SetCordonedRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.SetCordoned(ctx, nodeID, params)
	})

	return err
}

func (tr *ClientWithReauth) EvictReplica(ctx context.Context, namespaceID string, id string, deploymentID string) error {
	return tr.doWithReauth(ctx, func() error {
		return tr.client.EvictReplica(ctx, namespaceID, id, deploymentID)
	})
}

func (tr *ClientWithReauth) AttemptPromotion(ctx context.Context, namespaceID string, id string, deploymentID string) error {
	return tr.doWithReauth(ctx, func() error {
		return tr.client.AttemptPromotion(ctx, namespaceID, id, deploymentID)
	})
}

func (tr *ClientWithReauth) DetachVolume(ctx context.Context, namespaceID string, volumeID string, params *openapi.DetachVolumeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.DetachVolume(ctx, namespaceID, volumeID, params)
	})

	return err
}

func (tr *ClientWithReauth) AddDeploymentOnNode(ctx context.Context, namespaceID string, volumeID string, params *openapi.AddDeploymentOnNodeRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.AddDeploymentOnNode(ctx, namespaceID, volumeID, params)
	})

	return err
}

func (tr *ClientWithReauth) SetPreferredEvictionCandidates(ctx context.Context, namespaceID string, volID string, params *openapi.SetPreferredEvictionCandidatesRequestParams) error {
	err := tr.doWithReauth(ctx, func() error {
		return tr.client.SetPreferredEvictionCandidates(ctx, namespaceID, volID, params)
	})

	return err
}
