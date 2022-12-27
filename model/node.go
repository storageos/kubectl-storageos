package model

import (
	"time"

	"github.com/storageos/kubectl-storageos/pkg/health"
)

// Stats struct for CapacityStats
type Stats struct {
	// Total bytes in the filesystem
	Total uint64 `json:"total,omitempty"`
	// Free bytes in the filesystem available to root user
	Free uint64 `json:"free,omitempty"`
}

// Resource encapsulates a StorageOS node API resource as a data type.
type Node struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Health   health.NodeState `json:"health"`
	Capacity Stats            `json:"capacity,omitempty"`

	IOAddr         string `json:"ioAddress"`
	SupervisorAddr string `json:"supervisorAddress"`
	GossipAddr     string `json:"gossipAddress"`
	ClusteringAddr string `json:"clusteringAddress"`

	Cordoned   bool      `json:"cordoned"`
	CordonedAt time.Time `json:"cordonedAt"`

	Labels map[string]string `json:"labels"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   string    `json:"version"`
}
