package repo

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type File struct {
	ID        string
	UserID    string // empty = legacy file without owner
	Name      string
	SizeBytes int64
	SHA256Hex string
	CID       string
	LocalPath string
	CreatedAt time.Time
}

type FileShare struct {
	FileID    string
	Token     string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

