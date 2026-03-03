package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UsersRepo struct {
	pool *pgxpool.Pool
}

func NewUsersRepo(pool *pgxpool.Pool) *UsersRepo {
	return &UsersRepo{pool: pool}
}

func (r *UsersRepo) CreateUser(ctx context.Context, u repo.User) (repo.User, error) {
	const q = `
INSERT INTO users (id, email, password_hash)
VALUES ($1, $2, $3)
RETURNING created_at
`
	err := r.pool.QueryRow(ctx, q, u.ID, u.Email, u.PasswordHash).Scan(&u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return repo.User{}, repo.ErrDuplicate
		}
		return repo.User{}, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

func (r *UsersRepo) GetUserByID(ctx context.Context, id string) (repo.User, error) {
	const q = `SELECT id, email, password_hash, created_at FROM users WHERE id = $1`
	var u repo.User
	err := r.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.User{}, repo.ErrNotFound
		}
		return repo.User{}, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *UsersRepo) GetUserByEmail(ctx context.Context, email string) (repo.User, error) {
	const q = `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`
	var u repo.User
	err := r.pool.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.User{}, repo.ErrNotFound
		}
		return repo.User{}, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return false
}
