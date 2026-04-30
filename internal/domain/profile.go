package domain

import (
	"context"
	"errors"
	"strings"
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
}

type ProfileRepository interface {
	Create(ctx context.Context, profile *Profile) error
	GetByID(ctx context.Context, id string) (*Profile, error)
	GetByName(ctx context.Context, name string) (*Profile, error)
	Update(ctx context.Context, profile *Profile) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters ProfileFilters) ([]*Profile, error)
  GetFiltered(
    ctx context.Context,
    f ProfileFilters,
    page int,
    limit int,
) ([]Profile, int, error)
    GetAllFiltered(ctx context.Context, f ProfileFilters) ([]Profile, error)
}

func (f *ProfileFilters) Validate() error {
    if f.MinAge != nil && f.MaxAge != nil {
        if *f.MinAge > *f.MaxAge {
            return errors.New("min_age cannot be greater than max_age")
        }
    }

    if f.MinGenderProb != nil && (*f.MinGenderProb < 0 || *f.MinGenderProb > 1) {
        return errors.New("gender probability must be between 0 and 1")
    }
    if f.MinCountryProb != nil && (*f.MinCountryProb < 0 || *f.MinCountryProb > 1) {
        return errors.New("country probability must be between 0 and 1")
    }

    if f.Gender != "" {
        g := strings.ToLower(f.Gender)
        if g != "male" && g != "female" {
            return errors.New("gender must be 'male' or 'female'")
        }
    }

    return nil
}