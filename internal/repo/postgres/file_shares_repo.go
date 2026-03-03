package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileSharesRepo struct {
	pool *pgxpool.Pool
}

func NewFileSharesRepo(pool *pgxpool.Pool) *FileSharesRepo {
	return &FileSharesRepo{pool: pool}
}

func (r *FileSharesRepo) CreateOrUpdateShare(ctx context.Context, fileID, token string, expiresAt *time.Time) (repo.FileShare, error) {
	const q = `
INSERT INTO file_shares (file_id, token, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (file_id) DO UPDATE SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at
RETURNING file_id, token, created_at, expires_at
`
	var share repo.FileShare
	err := r.pool.QueryRow(ctx, q, fileID, token, expiresAt).Scan(&share.FileID, &share.Token, &share.CreatedAt, &share.ExpiresAt)
	if err != nil {
		return repo.FileShare{}, fmt.Errorf("upsert file share: %w", err)
	}
	return share, nil
}

func (r *FileSharesRepo) GetShareByToken(ctx context.Context, token string) (repo.FileShare, error) {
	const q = `
SELECT file_id, token, created_at, expires_at
FROM file_shares
WHERE token = $1 AND (expires_at IS NULL OR expires_at > now())
`
	var share repo.FileShare
	err := r.pool.QueryRow(ctx, q, token).Scan(&share.FileID, &share.Token, &share.CreatedAt, &share.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.FileShare{}, repo.ErrNotFound
		}
		return repo.FileShare{}, fmt.Errorf("get share by token: %w", err)
	}
	return share, nil
}

func (r *FileSharesRepo) GetShareByFileID(ctx context.Context, fileID string) (repo.FileShare, error) {
	const q = `
SELECT file_id, token, created_at, expires_at
FROM file_shares
WHERE file_id = $1 AND (expires_at IS NULL OR expires_at > now())
`
	var share repo.FileShare
	err := r.pool.QueryRow(ctx, q, fileID).Scan(&share.FileID, &share.Token, &share.CreatedAt, &share.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.FileShare{}, repo.ErrNotFound
		}
		return repo.FileShare{}, fmt.Errorf("get share by file id: %w", err)
	}
	return share, nil
}

func (r *FileSharesRepo) DeleteShare(ctx context.Context, fileID string) error {
	const q = `DELETE FROM file_shares WHERE file_id = $1`
	_, err := r.pool.Exec(ctx, q, fileID)
	if err != nil {
		return fmt.Errorf("delete share: %w", err)
	}
	return nil
}
