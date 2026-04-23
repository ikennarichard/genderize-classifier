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
    Limit     int
    Offset    int
	MinGenderProbability *float64
	MinCountryProbability *float64
    Gender         string
    AgeGroup       string
    CountryID      string
    MinAge         *int
    MaxAge         *int
    MinGenderProb  *float64
    MinCountryProb *float64
    SortBy string // age | created_at | gender_probability
    Order  string // asc | desc
    Page   int
}

type ProfileRepository interface {
	Create(ctx context.Context, profile *Profile) error
	GetByID(ctx context.Context, id string) (*Profile, error)
	GetByName(ctx context.Context, name string) (*Profile, error)
	Update(ctx context.Context, profile *Profile) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters ProfileFilters) ([]*Profile, error)
    GetFiltered(ctx context.Context, f ProfileFilters) ([]Profile, int, error)
}