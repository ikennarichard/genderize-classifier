package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ikennarichard/genderize-classifier/internal/utils"
)

func WriteError(w http.ResponseWriter, err error) {
    status := http.StatusInternalServerError
    message := "internal server error"

    switch {
    case errors.Is(err, utils.ErrNotFound):
        status = http.StatusNotFound
        message = err.Error()

    case errors.Is(err, utils.ErrAlreadyExists):
        status = http.StatusConflict
        message = err.Error()

    case errors.Is(err, utils.ErrUnauthorized):
        status = http.StatusUnauthorized
        message = err.Error()

    case errors.Is(err, utils.ErrMissingName):
        status = http.StatusBadRequest
        message = err.Error()

    case errors.Is(err, utils.ErrInvalidResponse):
        status = http.StatusBadRequest
        message = err.Error()
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    _ = json.NewEncoder(w).Encode(utils.ErrorResponse{
        Status:  "error",
        Message: message,
    })
}