package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Profile struct {
    ID                 uuid.UUID
    Name               string
    Gender             string
    GenderProbability  float64
    SampleSize         int
    Age                int
    AgeGroup           string
    CountryID          string
    CountryName        string
    CountryProbability float64
    CreatedAt          time.Time
}

type ProfileFilters struct {
    Gender   string
    CountryID string
    AgeGroup  string
    Limit     int
    Offset    int
}

type ProfileRepository interface {
	Create(ctx context.Context, profile *Profile) error
	GetByID(ctx context.Context, id string) (*Profile, error)
	GetByName(ctx context.Context, name string) (*Profile, error)
	Update(ctx context.Context, profile *Profile) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters ProfileFilters) ([]*Profile, error)
}