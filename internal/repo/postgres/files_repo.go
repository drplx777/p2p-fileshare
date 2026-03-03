package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FilesRepo struct {
	pool *pgxpool.Pool
}

func NewFilesRepo(pool *pgxpool.Pool) *FilesRepo {
	return &FilesRepo{pool: pool}
}

func (r *FilesRepo) CreateFile(ctx context.Context, f repo.File) (repo.File, error) {
	const q = `
INSERT INTO files (id, user_id, name, size_bytes, sha256_hex, cid, local_path)
VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7)
RETURNING created_at
`
	err := r.pool.QueryRow(ctx, q, f.ID, f.UserID, f.Name, f.SizeBytes, f.SHA256Hex, f.CID, f.LocalPath).Scan(&f.CreatedAt)
	if err != nil {
		return repo.File{}, fmt.Errorf("insert file: %w", err)
	}
	return f, nil
}

func (r *FilesRepo) GetFileByID(ctx context.Context, id string) (repo.File, error) {
	const q = `
SELECT id, COALESCE(user_id, ''), name, size_bytes, sha256_hex, cid, local_path, created_at
FROM files
WHERE id = $1
`
	var f repo.File
	err := r.pool.QueryRow(ctx, q, id).Scan(&f.ID, &f.UserID, &f.Name, &f.SizeBytes, &f.SHA256Hex, &f.CID, &f.LocalPath, &f.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.File{}, repo.ErrNotFound
		}
		return repo.File{}, fmt.Errorf("get file by id: %w", err)
	}
	return f, nil
}

func (r *FilesRepo) GetFileByCID(ctx context.Context, cid string) (repo.File, error) {
	const q = `
SELECT id, COALESCE(user_id, ''), name, size_bytes, sha256_hex, cid, local_path, created_at
FROM files
WHERE cid = $1
`
	var f repo.File
	err := r.pool.QueryRow(ctx, q, cid).Scan(&f.ID, &f.UserID, &f.Name, &f.SizeBytes, &f.SHA256Hex, &f.CID, &f.LocalPath, &f.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.File{}, repo.ErrNotFound
		}
		return repo.File{}, fmt.Errorf("get file by cid: %w", err)
	}
	return f, nil
}

func (r *FilesRepo) ListFiles(ctx context.Context, userID string, limit int) ([]repo.File, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `
SELECT id, COALESCE(user_id, ''), name, size_bytes, sha256_hex, cid, local_path, created_at
FROM files
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2
`
	rows, err := r.pool.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	out := make([]repo.File, 0, limit)
	for rows.Next() {
		var f repo.File
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.SizeBytes, &f.SHA256Hex, &f.CID, &f.LocalPath, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list files rows: %w", err)
	}
	return out, nil
}

