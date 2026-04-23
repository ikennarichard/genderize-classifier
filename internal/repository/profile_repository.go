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

type PostgresProfileRepository struct {
	db *pgxpool.Pool
}

func NewPostgresProfileRepository(db *pgxpool.Pool) *PostgresProfileRepository {
	return &PostgresProfileRepository{db: db}
}

const profileColumns = `id, name, gender, gender_probability, sample_size, age, age_group, country_id, country_name, country_probability, created_at`

func (r *PostgresProfileRepository) Create(ctx context.Context, profile *domain.Profile) error {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO profiles (` + profileColumns + `) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.Exec(ctx, query,
		profile.ID, profile.Name, profile.Gender, profile.GenderProbability,
		profile.SampleSize, profile.Age, profile.AgeGroup, profile.CountryID,
		profile.CountryName, profile.CountryProbability, profile.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}
	return nil
}

func (r *PostgresProfileRepository) GetByID(ctx context.Context, id string) (*domain.Profile, error) {
	query := `SELECT ` + profileColumns + ` FROM profiles WHERE id = $1`
	return r.scanProfile(r.db.QueryRow(ctx, query, id))
}

func (r *PostgresProfileRepository) GetByName(ctx context.Context, name string) (*domain.Profile, error) {
	query := `SELECT ` + profileColumns + ` FROM profiles WHERE name = $1`
	return r.scanProfile(r.db.QueryRow(ctx, query, name))
}

func (r *PostgresProfileRepository) Update(ctx context.Context, p *domain.Profile) error {
	query := `
		UPDATE profiles SET
			name = $2, gender = $3, gender_probability = $4, sample_size = $5,
			age = $6, age_group = $7, country_id = $8, country_name = $9,
			country_probability = $10
		WHERE id = $1`

	res, err := r.db.Exec(ctx, query,
		p.ID, p.Name, p.Gender, p.GenderProbability, p.SampleSize,
		p.Age, p.AgeGroup, p.CountryID, p.CountryName, p.CountryProbability,
	)

	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresProfileRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM profiles WHERE id = $1`
	res, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresProfileRepository) List(ctx context.Context, f domain.ProfileFilters) ([]*domain.Profile, error) {
    query := `SELECT ` + profileColumns + ` FROM profiles WHERE 1=1`
    var args []any
    argCount := 1

    if f.Gender != "" {
        query += fmt.Sprintf(" AND LOWER(gender) = LOWER($%d)", argCount)
        args = append(args, f.Gender)
        argCount++
    }
    if f.CountryID != "" {
        query += fmt.Sprintf(" AND LOWER(country_id) = LOWER($%d)", argCount)
        args = append(args, f.CountryID)
        argCount++
    }
    if f.AgeGroup != "" {
        query += fmt.Sprintf(" AND LOWER(age_group) = LOWER($%d)", argCount)
        args = append(args, f.AgeGroup)
        argCount++
    }

    query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
    args = append(args, f.Limit, f.Offset)

    rows, err := r.db.Query(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to list profiles: %w", err)
    }
    defer rows.Close()

    var profiles []*domain.Profile
    for rows.Next() {
        p, err := r.scanProfile(rows)
        if err != nil {
            return nil, err
        }
        profiles = append(profiles, p)
    }
    return profiles, rows.Err()
}

func (r *PostgresProfileRepository) scanProfile(row pgx.Row) (*domain.Profile, error) {
	var p domain.Profile
	err := row.Scan(
		&p.ID, &p.Name, &p.Gender, &p.GenderProbability, &p.SampleSize,
		&p.Age, &p.AgeGroup, &p.CountryID, &p.CountryName,
		&p.CountryProbability, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}
	return &p, nil
}