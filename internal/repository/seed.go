package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "embed"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed seed_profiles.json
var seedJSON []byte

type SeedData struct {
	Profiles []ProfileSeedDTO `json:"profiles"`
}

type ProfileSeedDTO struct {
	Name               string  `json:"name"`
	Gender             string  `json:"gender"`
	GenderProbability  float64 `json:"gender_probability"`
	Age                int     `json:"age"`
	AgeGroup           string  `json:"age_group"`
	CountryID          string  `json:"country_id"`
	CountryName        string  `json:"country_name"`
	CountryProbability float64 `json:"country_probability"`
}

func InitSchema(db *pgxpool.Pool, ctx context.Context) error {
	schema := `
DROP TABLE IF EXISTS users CASCADE;
	
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    github_id BIGINT UNIQUE NOT NULL,
    username TEXT NOT NULL,
    email TEXT,
    avatar_url TEXT,
    role TEXT DEFAULT 'analyst',
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

	CREATE TABLE IF NOT EXISTS sessions (
		id UUID PRIMARY KEY,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		token TEXT UNIQUE NOT NULL,
		is_revoked BOOLEAN DEFAULT false,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS profiles (
		id UUID PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		gender TEXT,
		gender_probability DOUBLE PRECISION,
		sample_size INTEGER,
		age INTEGER,
		age_group TEXT,
		country_id TEXT,
		country_name TEXT,
		country_probability DOUBLE PRECISION,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	fmt.Println("Database schema initialized successfully")
	return nil
}

func SeedFromJSON(db *pgxpool.Pool, ctx context.Context, filePath string) error {
	if err := InitSchema(db, ctx); err != nil {
		return err
	}

	// 2. Open and read the seed file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open seed file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	// Navigate to "profiles" array in the JSON
	for {
		t, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("failed to read json tokens: %w", err)
		}
		if t == "profiles" {
			break
		}
	}

	if _, err := decoder.Token(); err != nil {
		return err
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO profiles (
			id, name, gender, gender_probability, sample_size, 
			age, age_group, country_id, country_name, country_probability, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (name) DO NOTHING`

	now := time.Now().UTC()
	count := 0

	for decoder.More() {
		var s ProfileSeedDTO
		if err := decoder.Decode(&s); err != nil {
			return fmt.Errorf("failed to decode profile at index %d: %w", count, err)
		}

		batch.Queue(query,
			uuid.New(), s.Name, s.Gender, s.GenderProbability, 0,
			s.Age, s.AgeGroup, s.CountryID, s.CountryName, s.CountryProbability, now,
		)
		count++
	}

	if count == 0 {
		return nil
	}

	fmt.Printf("Executing streaming batch seed: %d records\n", count)
	results := db.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < count; i++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch error at index %d: %w", i, err)
		}
	}

	return nil
}

func SeedTestUsers(pool *pgxpool.Pool, ctx context.Context) error {
    query := `
        INSERT INTO users (id, github_id, username, email, avatar_url, role, is_active, created_at)
        VALUES 
            ($1, $2, 'insighta_admin', 'admin@insighta.com', '', 'admin', true, NOW()),
            ($3, $4, 'insighta_analyst', 'analyst@insighta.com', '', 'analyst', true, NOW())
        ON CONFLICT (github_id) DO NOTHING`

    _, err := pool.Exec(ctx, query,
        uuid.New().String(), "100000001", // admin
        uuid.New().String(), "100000002", // analyst
    )
    if err != nil {
        return fmt.Errorf("failed to seed test users: %w", err)
    }
    return nil
}