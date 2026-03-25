package model

import "time"

type AdminSettings struct {
	PasswordHash  string    `json:"password_hash"`
	InitializedAt time.Time `json:"initialized_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AdminSession struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type NodeDisplayName struct {
	NodeID      string    `json:"node_id"`
	DisplayName string    `json:"display_name"`
	UpdatedAt   time.Time `json:"updated_at"`
}
