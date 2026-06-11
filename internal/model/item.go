package model

import "time"

type Item struct {
	ID        string       `json:"id"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	URL       string       `json:"url,omitempty"`
	Files     []StoredFile `json:"files,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
}

type StoredFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}
