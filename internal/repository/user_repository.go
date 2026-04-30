package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ikennarichard/genderize-classifier/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
    query := `
        SELECT id, github_id, username, email, avatar_url, role, is_active
        FROM users
        WHERE username = $1`

    var u domain.User
    err := r.pool.QueryRow(ctx, query, username).Scan(
        &u.ID, &u.GitHubID, &u.Username, &u.Email,
        &u.AvatarURL, &u.Role, &u.IsActive,
    )
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    return &u, nil
}


// Used by the AuthenticateJWT middleware on every request.
func (r *PostgresUserRepository) FindByID(id string) (*domain.User, error) {
	query := `
		SELECT id, github_id, username, email, avatar_url, role, is_active, created_at 
		FROM users 
		WHERE id = $1`

	var user domain.User
	err := r.pool.QueryRow(context.Background(), query, id).Scan(
		&user.ID,
		&user.GitHubID,
		&user.Username,
		&user.Email,
		&user.AvatarURL,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// used during the oauth callback to check if the user exists.
func (r *PostgresUserRepository) FindByGitHubID(githubID string) (*domain.User, error) {
	query := `
		SELECT id, github_id, username, email, avatar_url, role, is_active, created_at 
		FROM users 
		WHERE github_id = $1`

	var user domain.User
	err := r.pool.QueryRow(context.Background(), query, githubID).Scan(
		&user.ID,
		&user.GitHubID,
		&user.Username,
		&user.Email,
		&user.AvatarURL,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (r *PostgresUserRepository) Upsert(ctx context.Context, user *domain.User) error {
    query := `
        INSERT INTO users (id, github_id, username, email, avatar_url, role, is_active, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (github_id) DO UPDATE SET
            username = EXCLUDED.username,
            email = EXCLUDED.email,
            avatar_url = EXCLUDED.avatar_url,
            last_login_at = CURRENT_TIMESTAMP
        RETURNING id, role, is_active`

    if user.ID == "" {
        user.ID = uuid.New().String()
    }

    err := r.pool.QueryRow(ctx, query,
        user.ID,
        user.GitHubID,
        user.Username,
        user.Email,
        user.AvatarURL,
        user.Role,
        user.IsActive,
        time.Now(),
    ).Scan(&user.ID, &user.Role, &user.IsActive)

    return err
}