package model

import "time"

// Entry is a single credential record stored in the vault.
type Entry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"password,omitempty"`
	URL       string    `json:"url,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
