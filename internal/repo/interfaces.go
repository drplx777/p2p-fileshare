package repo

import "context"

type FileRepository interface {
	CreateFile(ctx context.Context, f File) (File, error)
	GetFileByID(ctx context.Context, id string) (File, error)
	GetFileByCID(ctx context.Context, cid string) (File, error)
	ListFiles(ctx context.Context, limit int) ([]File, error)
}

