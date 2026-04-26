package model

import "time"

// SecurityQuestion holds a website security question and its secret answer.
// Answers are stored encrypted alongside the entry — treat them as passwords.
type SecurityQuestion struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Entry is a single credential record stored in the vault.
type Entry struct {
	ID                string             `json:"id"`
	Title             string             `json:"title"`
	Username          string             `json:"username,omitempty"`
	Password          string             `json:"password,omitempty"`
	URL               string             `json:"url,omitempty"`
	Notes             string             `json:"notes,omitempty"`
	Tags              []string           `json:"tags,omitempty"`
	SecurityQuestions []SecurityQuestion `json:"security_questions,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}
