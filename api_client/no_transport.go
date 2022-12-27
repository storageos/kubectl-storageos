package apiclient

import (
	"context"
	"io"

	"github.com/storageos/kubectl-storageos/model"
)

// noTransport implements the Transport interface for the API client, but
// returns a known error from all method invocations. It is a placeholder
// to allow for configuration of the API client to take place after it
// has been constructed.
type noTransport struct{}

var _ Transport = (*noTransport)(nil)

func (t *noTransport) Authenticate(ctx context.Context, username, password string) (AuthSession, error) {
	return AuthSession{}, ErrNoTransportConfigured
}

func (t *noTransport) UseAuthSession(ctx context.Context, session AuthSession) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) GetCluster(ctx context.Context) (*model.Cluster, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) GetLicence(ctx context.Context) (*model.License, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) GetDiagnostics(ctx context.Context) (io.ReadCloser, string, error) {
	return nil, "", ErrNoTransportConfigured
}

func (t *noTransport) GetSingleNodeDiagnostics(ctx context.Context, nodeID string) (io.ReadCloser, string, error) {
	return nil, "", ErrNoTransportConfigured
}

func (t *noTransport) GetUser(ctx context.Context, username string) (*model.User, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) GetNode(ctx context.Context, nodeID string) (*model.Node, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) GetVolume(ctx context.Context, namespaceID string, volumeID string) (*model.Volume, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) GetNamespace(ctx context.Context, namespaceID string) (*model.Namespace, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) GetPolicyGroup(ctx context.Context, uid string) (*model.PolicyGroup, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) ListNodes(ctx context.Context) ([]*model.Node, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) ListVolumes(ctx context.Context, namespaceID string) ([]*model.Volume, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) ListNamespaces(ctx context.Context) ([]*model.Namespace, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) ListPolicyGroups(ctx context.Context) ([]*model.PolicyGroup, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) ListUsers(ctx context.Context) ([]*model.User, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) CreateUser(ctx context.Context, username, password string, withAdmin bool, groups ...string) (*model.User, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) CreateVolume(ctx context.Context, namespaceID string, name, description string, fs model.FsType, sizeBytes uint64, labels map[string]string, params *CreateVolumeRequestParams) (*model.Volume, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) CreateNamespace(ctx context.Context, name string, labels map[string]string) (*model.Namespace, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) CreatePolicyGroup(ctx context.Context, name string, specs []*model.PolicyGroupSpec) (*model.PolicyGroup, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) UpdateCluster(ctx context.Context, resource *model.Cluster, params *UpdateClusterRequestParams) (*model.Cluster, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) UpdateLicence(ctx context.Context, licence []byte, params *UpdateLicenceRequestParams) (*model.License, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) SetReplicas(ctx context.Context, nsID string, volID string, numReplicas uint64, params *SetReplicasRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) UpdateVolume(ctx context.Context, nsID string, volID string, description string, labels map[string]string, params *UpdateVolumeRequestParams) (*model.Volume, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) ResizeVolume(ctx context.Context, nsID string, volID string, sizeBytes uint64, params *ResizeVolumeRequestParams) (*model.Volume, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) UpdateNFSVolumeExports(ctx context.Context, namespaceID string, volumeID string, exports []model.NFSExportConfig, params *UpdateNFSVolumeExportsRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) UpdateNFSVolumeMountEndpoint(ctx context.Context, namespaceID string, volumeID string, endpoint string, params *UpdateNFSVolumeMountEndpointRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) SetFailureModeIntent(ctx context.Context, namespaceID string, volumeID string, intent string, params *SetFailureModeRequestParams) (*model.Volume, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) SetFailureThreshold(ctx context.Context, namespaceID string, volumeID string, threshold uint64, params *SetFailureModeRequestParams) (*model.Volume, error) {
	return nil, ErrNoTransportConfigured
}

func (t *noTransport) DeleteUser(ctx context.Context, uid string, params *DeleteUserRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) DeleteVolume(ctx context.Context, namespaceID string, volumeID string, params *DeleteVolumeRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) DeleteNode(ctx context.Context, volumeID string, params *DeleteNodeRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) DeleteNamespace(ctx context.Context, uid string, params *DeleteNamespaceRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) DeletePolicyGroup(ctx context.Context, uid string, params *DeletePolicyGroupRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) AttachVolume(ctx context.Context, namespaceID string, volumeID string, nodeID string) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) AttachNFSVolume(ctx context.Context, namespaceID string, volumeID string, params *AttachNFSVolumeRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) DetachVolume(ctx context.Context, namespaceID string, volumeID string, params *DetachVolumeRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) SetCordoned(ctx context.Context, nodeID string, params *SetCordonedRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) EvictReplica(ctx context.Context, namespaceID string, id string, deploymentID string) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) AttemptPromotion(ctx context.Context, namespaceID string, id string, deploymentID string) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) AddDeploymentOnNode(ctx context.Context, namespaceID string, volID string, params *AddDeploymentOnNodeRequestParams) error {
	return ErrNoTransportConfigured
}

func (t *noTransport) SetPreferredEvictionCandidates(ctx context.Context, namespaceID string, volID string, params *SetPreferredEvictionCandidatesRequestParams) error {
	return ErrNoTransportConfigured
}
