package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Session struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
	IsRevoked bool
}

type SessionRepository interface {
	CreateSession(ctx context.Context, userID string, token string, expiresAt time.Time) error
	FindSession(ctx context.Context, token string) (*Session, error)
	RevokeSession(ctx context.Context, token string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
}

type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}


func (r *PostgresSessionRepository) CreateSession(ctx context.Context, userID string, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO sessions (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)`

	id := uuid.New().String()
	_, err := r.pool.Exec(ctx, query, id, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) FindSession(ctx context.Context, token string) (*Session, error) {
	query := `
		SELECT user_id, token, expires_at, is_revoked
		FROM sessions
		WHERE token = $1`

	var s Session
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&s.UserID, &s.Token, &s.ExpiresAt, &s.IsRevoked,
	)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return &s, nil
}

func (r *PostgresSessionRepository) RevokeSession(ctx context.Context, token string) error {
	query := `UPDATE sessions SET is_revoked = true WHERE token = $1`

	result, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found for token")
	}
	return nil
}

func (r *PostgresSessionRepository) RevokeAllUserSessions(ctx context.Context, userID string) error {
	query := `UPDATE sessions SET is_revoked = true WHERE user_id = $1 AND is_revoked = false`

	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all sessions for user %s: %w", userID, err)
	}
	return nil
}