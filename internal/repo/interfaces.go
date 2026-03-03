package repo

import (
	"context"
	"time"
)

type UserRepository interface {
	CreateUser(ctx context.Context, u User) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
}

type FileRepository interface {
	CreateFile(ctx context.Context, f File) (File, error)
	GetFileByID(ctx context.Context, id string) (File, error)
	GetFileByCID(ctx context.Context, cid string) (File, error)
	ListFiles(ctx context.Context, userID string, limit int) ([]File, error)
}

type FileShareRepository interface {
	CreateOrUpdateShare(ctx context.Context, fileID, token string, expiresAt *time.Time) (FileShare, error)
	GetShareByToken(ctx context.Context, token string) (FileShare, error)
	GetShareByFileID(ctx context.Context, fileID string) (FileShare, error)
	DeleteShare(ctx context.Context, fileID string) error
}

