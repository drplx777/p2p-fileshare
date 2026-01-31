package repo

import "time"

type File struct {
	ID        string
	Name      string
	SizeBytes int64
	SHA256Hex string
	CID       string
	LocalPath string
	CreatedAt time.Time
}

