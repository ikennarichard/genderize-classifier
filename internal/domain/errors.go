package domain

import (
	"errors"
	"strings"
)

var (
    ErrNotFound            = errors.New("resource not found")
    ErrAlreadyExists       = errors.New("resource already exists")
    ErrUnauthorized        = errors.New("unauthorized")
    ErrInvalidResponse     = errors.New("invalid response")
    ErrMissingName         = errors.New("missing or empty name")
    ErrUnprocessableEntity = errors.New("invalid type")
)

type ErrorResponse struct {
    Status  string `json:"status"`
    Message string `json:"message"`
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