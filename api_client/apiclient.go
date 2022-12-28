// Package apiclient provides a type which implements an abstraction layer
// for consuming the storageos API programmatically.
package apiclient

import (
	"context"
	"errors"
	"io"

	"github.com/storageos/kubectl-storageos/model"
)

var (
	// ErrNoTransportConfigured indicates that the API client has not be
	// configured with an underlying transport implementation, which is required
	// for operation.
	ErrNoTransportConfigured = errors.New("the client has not been configured with a transport")
	// ErrTransportAlreadyConfigured indicates an attempt was made to configure
	// an API client with a new transport implementation when the client already
	// has one.
	//
	// To use a new transport implementation a consumer of the package must
	// instantiate a new Client.
	ErrTransportAlreadyConfigured = errors.New("the client's transport has already been configured")
)

// Transport describes the set of methods required by an API client to use a
// given transport implementation provider.
//
// A Transport implementation only needs to provide a direct mapping to the
// StorageOS API - it is the responsibility of the client to compose
// functionality for complex operations.
type Transport interface {
	// Authenticate presents the username and password to the StorageOS API,
	// requesting a new login session.
	Authenticate(ctx context.Context, username, password string) (AuthSession, error)
	// UseAuthSession configures the transport to re-use the provided auth
	// session for its future requests.
	UseAuthSession(ctx context.Context, session AuthSession) error

	// GetUser requests the details of the StorageOS user account with uid and
	// returns it to the caller.
	GetUser(ctx context.Context, uid string) (*model.User, error)
	// GetCluster returns the API resource representing the cluster.
	GetCluster(ctx context.Context) (*model.Cluster, error)
	// GetLicence returns the API resource representing the licence currently
	// applied to the cluster.
	GetLicence(ctx context.Context) (*model.License, error)
	// GetNode requests the node resource which corresponds to uid from the
	// StorageOS API.
	GetNode(ctx context.Context, nodeID string) (*model.Node, error)
	// GetVolume requests the volume resource with volumeID in the namespace
	// with namespaceID from the StorageOS API.
	GetVolume(ctx context.Context, namespaceID string, volumeID string) (*model.Volume, error)
	// GetNamespace requests the namespace resource which corresponds to uid
	// from the StorageOS API.
	GetNamespace(ctx context.Context, namespaceID string) (*model.Namespace, error)
	// GetDiagnostics requests a new diagnostics bundle for the cluster
	// from the StorageOS API.
	GetDiagnostics(ctx context.Context) (io.ReadCloser, string, error)
	// GetSingleNodeDiagnostics requests a new single node diagnostics bundle for
	// the cluster from the StorageOS API.
	GetSingleNodeDiagnostics(ctx context.Context, nodeID string) (io.ReadCloser, string, error)
	// GetPolicyGroup requests a new policy group resource which corresponds to
	// uid from the StorageOS API.
	GetPolicyGroup(ctx context.Context, uid string) (*model.PolicyGroup, error)
	// ListNodes returns all the node resources in the cluster.
	ListNodes(ctx context.Context) ([]*model.Node, error)
	// ListVolumes returns all the user resources in the cluster.
	ListVolumes(ctx context.Context, namespaceID string) ([]*model.Volume, error)
	// ListNamespaces returns all the namespace resources in the cluster.
	ListNamespaces(ctx context.Context) ([]*model.Namespace, error)
	// ListPolicyGroups returns all the policy group resources in the cluster.
	ListPolicyGroups(ctx context.Context) ([]*model.PolicyGroup, error)
	// ListUsers returns the list of all StorageOS user accounts
	ListUsers(ctx context.Context) ([]*model.User, error)

	// CreateUser requests the creation of a new StorageOS user account from the
	// provided fields. If successful the created resource for the user account
	// is returned to the caller.
	CreateUser(ctx context.Context, username, password string, withAdmin bool, groups ...string) (*model.User, error)
	// CreateVolume requests the creation of a new StorageOS volume in namespace
	// from the provided fields. If successful the created resource for the volume
	// is returned to the caller.
	//
	// The behaviour of the operation is dictated by params:
	//
	//   Asynchrony:
	//   - If params is nil or params.AsyncMax is empty/zero valued then the
	//     create request is performed synchronously.
	//   - If params.AsyncMax is set, the request is performed asynchronously
	//     using the duration given as the maximum amount of time allowed for
	//     the request before it times out.
	CreateVolume(ctx context.Context, namespaceID string, name, description string, fs model.FsType, sizeBytes uint64, labels map[string]string, params *CreateVolumeRequestParams) (*model.Volume, error)
	// CreateNamespace requests the creation of a new StorageOS namespace from the
	// provided fields. If successful the created resource for the namespace is
	// returned to the caller.
	CreateNamespace(ctx context.Context, name string, labels map[string]string) (*model.Namespace, error)
	// Create requests the creation of a new StorageOS policy group from the
	// provided fields. If successful the created resource for the policy group
	// is returned to the caller.
	CreatePolicyGroup(ctx context.Context, name string, specs []*model.PolicyGroupSpec) (*model.PolicyGroup, error)

	// UpdateCluster attempts to perform an update of the cluster configuration
	// through the StorageOS API using resource as the update value.
	UpdateCluster(ctx context.Context, resource *model.Cluster, params *UpdateClusterRequestParams) (*model.Cluster, error)
	// UpdateLicence attempts to perform an update of the licence currently
	// applied to the cluster through the StorageOS API.
	// In case of success, it returns all the info of the licence just applied.
	UpdateLicence(ctx context.Context, licence []byte, params *UpdateLicenceRequestParams) (*model.License, error)
	// SetReplicas attempts to perform a change in the number of replicas
	// of a specified volume. If everything is correct, job is accepted.
	SetReplicas(ctx context.Context, nsID string, volID string, numReplicas uint64, params *SetReplicasRequestParams) error
	// UpdateVolume attempts to perform an update of a specified  volume
	// changing descriptions and labels.
	UpdateVolume(ctx context.Context, nsID string, volID string, description string, labels map[string]string, params *UpdateVolumeRequestParams) (*model.Volume, error)
	// ResizeVolume attempts to perform a change in the size of a specified
	// volume. If everything is correct, job is accepted.
	ResizeVolume(ctx context.Context, nsID string, volID string, sizeBytes uint64, params *ResizeVolumeRequestParams) (*model.Volume, error)
	// UpdateNFSVolumeExports attempts to perform an update of the NFS export
	// configuration for the specific volume.
	UpdateNFSVolumeExports(ctx context.Context, namespaceID string, volumeID string, exports []model.NFSExportConfig, params *UpdateNFSVolumeExportsRequestParams) error
	// UpdateNFSVolumeMountEndpoint attempts to perform an update of the NFS
	// mount endpoint for the specific volume.
	UpdateNFSVolumeMountEndpoint(ctx context.Context, namespaceID string, volumeID string, endpoint string, params *UpdateNFSVolumeMountEndpointRequestParams) error
	// SetFailureModeIntent attempts to perform an update of the failure mode
	// for the target volume to the provided intent-based behaviour.
	SetFailureModeIntent(ctx context.Context, namespaceID string, volumeID string, intent string, params *SetFailureModeRequestParams) (*model.Volume, error)
	// SetFailureThreshold attempts to perform an update of the failure mode
	// for the target volume to the provided numerical threshold.
	SetFailureThreshold(ctx context.Context, namespaceID string, volumeID string, threshold uint64, params *SetFailureModeRequestParams) (*model.Volume, error)

	// DeleteVolume makes a delete request for volumeID in namespaceID.
	//
	// The behaviour of the operation is dictated by params:
	//
	//   Version constraints:
	//   - If params is nil or params.CASVersion is empty then the delete
	//     request is unconditional
	//   - If params.CASVersion is set, the request is conditional upon it
	//     matching the volume entity's version as seen by the server.
	//
	//   Asynchrony:
	//   - If params is nil or params.AsyncMax is empty/zero valued then the
	//     delete request is performed synchronously.
	//   - If params.AsyncMax is set, the request is performed asynchronously
	//     using the duration given as the maximum amount of time allowed for
	//     the request before it times out.
	DeleteVolume(ctx context.Context, namespaceID string, volumeID string, params *DeleteVolumeRequestParams) error
	// DeleteNamespace makes a delete request for a namespace given its id.
	//
	// The behaviour of the operation is dictated by params:
	//
	//   Version constraints:
	//   - If params is nil or params.CASVersion is empty then the delete
	//     request is unconditional
	//   - If params.CASVersion is set, the request is conditional upon it
	//     matching the volume entity's version as seen by the server.
	DeleteNamespace(ctx context.Context, uid string, params *DeleteNamespaceRequestParams) error
	// DeleteNode makes a delete request for nodeID.
	//
	// The behaviour of the operation is dictated by params:
	//
	//   Version constraints:
	//   - If params is nil or params.CASVersion is empty then the delete
	//   request is unconditional
	//   - If params.CASVersion is set, the request is conditional upon it
	//   matching the node entity's version as seen by the server.
	//
	//   Asynchrony:
	//   - If params is nil or params.AsyncMax is empty/zero valued then the
	//   delete request is performed synchronously.
	//   - If params.AsyncMax is set, the request is performed asynchronously
	//   using the duration given as the maximum amount of time allowed for the
	//   request before it times out.
	DeleteNode(ctx context.Context, nodeID string, params *DeleteNodeRequestParams) error
	// DeleteUser makes a delete request for a user given its id.
	// The behaviour of the operation is dictated by params:
	//
	// Version constraints:
	//  - If params is nil or params.CASVersion is empty then the delete request is
	//    unconditional
	//  - If params.CASVersion is set, the request is conditional upon it matching
	//    the volume entity's version as seen by the server.
	DeleteUser(ctx context.Context, uid string, params *DeleteUserRequestParams) error
	// DeletePolicyGroup makes a delete request for a policy group given its id.
	//
	// The behaviour of the operation is dictated by params:
	//
	//  Version constraints:
	//  - If params is nil or params.CASVersion is empty then the delete request is
	//    unconditional
	//  - If params.CASVersion is set, the request is conditional upon it matching
	//    the volume entity's version as seen by the server.
	DeletePolicyGroup(ctx context.Context, uid string, params *DeletePolicyGroupRequestParams) error

	// AttachVolume requests volumeID in namespaceID is attached to nodeID.
	AttachVolume(ctx context.Context, namespaceID string, volumeID string, nodeID string) error
	// AttachNFSVolume requests volumeID in namespaceID is attached with the
	// already defined mount endpoints and export configurations.
	AttachNFSVolume(ctx context.Context, namespaceID string, volumeID string, params *AttachNFSVolumeRequestParams) error
	// DetachVolume makes a detach request for volumeID in namespaceID.
	//
	// The behaviour of the operation is dictated by params:
	//
	//   Version constraints:
	//   - If params is nil or params.CASVersion is empty then the detach
	//     request is unconditional
	//   - If params.CASVersion is set, the request is conditional upon it
	//     matching the volume entity's version as seen by the server.
	DetachVolume(ctx context.Context, namespaceID string, volumeID string, params *DetachVolumeRequestParams) error
	// SetCordoned sets a nodes cordoned state
	// The behaviour of the operation is dictated by params:
	//
	//   Version constraints:
	//   - If params is nil or params.CASVersion is empty then the cordon
	//   request is unconditional
	//   - If params.CASVersion is set, the request is conditional upon it
	//   matching the node entity's version as seen by the server.
	SetCordoned(ctx context.Context, nodeID string, params *SetCordonedRequestParams) error

	EvictReplica(ctx context.Context, namespaceID string, id string, deploymentID string) error

	AttemptPromotion(ctx context.Context, namespaceID string, id string, deploymentID string) error
	AddDeploymentOnNode(ctx context.Context, namespaceID string, volID string, params *AddDeploymentOnNodeRequestParams) error
	SetPreferredEvictionCandidates(ctx context.Context, namespaceID string, volID string, params *SetPreferredEvictionCandidatesRequestParams) error
}

// Client provides a collection of methods for consumers to interact with the
// StorageOS API.
type Client struct {
	Transport
}
