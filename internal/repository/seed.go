package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "embed"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *PostgresProfileRepository) SeedFromJSON(ctx context.Context, filePath string) error {
    reader := strings.NewReader(string(seedJSON))
    decoder := json.NewDecoder(reader)

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

	// Stream individual profile objects
	for decoder.More() {
		var s ProfileSeedDTO
		if err := decoder.Decode(&s); err != nil {
			return fmt.Errorf("failed to decode profile at index %d: %w", count, err)
		}

		batch.Queue(query,
			uuid.New(),
			s.Name,
			s.Gender,
			s.GenderProbability,
			0,
			s.Age,
			s.AgeGroup,
			s.CountryID,
			s.CountryName,
			s.CountryProbability,
			now,
		)
		count++
	}

	if count == 0 {
		return nil
	}

	fmt.Println("executing streaming batch seed", "count", count)
	results := r.db.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < count; i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("batch error at index %d: %w", i, err)
		}
	}

	return nil
}