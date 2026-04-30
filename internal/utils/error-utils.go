package utils

import (
	"errors"
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

