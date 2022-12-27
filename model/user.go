package model

import "time"

// Resource encapsulates a StorageOS user API resource as data type.
type User struct {
	ID       string `json:"id"`
	Username string `json:"name"`

	IsAdmin bool     `json:"isAdmin"`
	Groups  []string `json:"groups"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   string    `json:"version"`
}
