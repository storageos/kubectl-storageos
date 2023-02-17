package model

import "time"

// Resource encapsulates a StorageOS policy group API resource as a data type.
type PolicyGroup struct {
	ID    string               `json:"id"`
	Name  string               `json:"name"`
	Users []*PolicyGroupMember `json:"users"`
	Specs []*PolicyGroupSpec   `json:"specs"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   string    `json:"version"`
}

// PolicyGroupSpec encapsulates a policy specification API resource belonging to a policy
// group as a data type.
type PolicyGroupSpec struct {
	NamespaceID  string `json:"namespaceID"`
	ResourceType string `json:"resourceType"`
	ReadOnly     bool   `json:"readOnly"`
}

// PolicyGroupMember represents the details of a user that is a member of a policy group.
type PolicyGroupMember struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}
