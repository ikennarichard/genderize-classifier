package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ikennarichard/genderize-classifier/internal/domain"
	"github.com/ikennarichard/genderize-classifier/internal/service"
	"github.com/ikennarichard/genderize-classifier/internal/utils"
)

type Handler struct {
	repo domain.ProfileRepository 
}

func New(repo domain.ProfileRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterProfileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/profiles", h.createProfile)
	mux.HandleFunc("GET /api/profiles/{id}", h.getProfile)
	mux.HandleFunc("GET /api/profiles", h.listProfiles)
	mux.HandleFunc("DELETE /api/profiles/{id}", h.deleteProfile)
	mux.HandleFunc("GET /api/profiles/search", h.searchProfiles)
}


func (h *Handler) searchProfiles(w http.ResponseWriter, r *http.Request) {
	queryStr := r.URL.Query().Get("q")
	if queryStr == "" {
		utils.RespondError(w, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}

	filters, err := service.ParseNaturalLanguage(queryStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Unable to interpret query")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 { page = 1 }
	if limit <= 0 || limit > 50 { limit = 10 }
	
	filters.Page = page
	filters.Limit = limit

    if err := filters.Validate(); err != nil {
        utils.RespondError(w, http.StatusBadRequest, "Invalid query parameters")
        return
    }

	profiles, total, err := h.repo.GetFiltered(r.Context(), filters)
	if err != nil {
		utils.RespondError(w, 500, "Database error")
		return
	}

	utils.Respond(w, http.StatusOK, map[string]any{
		"status": "success",
		"page":   page,
		"limit":  limit,
		"total":  total,
		"data":   mapToDTOs(profiles),
	})
}

func (h *Handler) createProfile(w http.ResponseWriter, r *http.Request) {
	var req CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		utils.RespondError(w, http.StatusBadRequest, "name is required")
		return
	}

	existing, err := h.repo.GetByName(r.Context(), name)
	if err == nil && existing != nil {
        dataResponse := fromDomain(existing)
        utils.Respond(w, http.StatusOK, ProfileResponse{
            Status:  "success",
            Message: "Profile already exists",
            Data:    &dataResponse,
        })
        return
    }

	profile, err := service.BuildProfile(name)
	if err != nil {
		utils.RespondError(w, http.StatusBadGateway, err.Error())
		return
	}

	if err := h.repo.Create(r.Context(), profile); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	createdRes := fromDomain(profile)
	utils.Respond(w, http.StatusCreated, ProfileResponse{Status: "success", Data: &createdRes})
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	profile, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		fmt.Println("GetProfile Error:", err)
		utils.RespondError(w, http.StatusNotFound, "Profile not found")
		return
	}
	profileRes := fromDomain(profile)
	utils.Respond(w, http.StatusOK, ProfileResponse{Status: "success", Data: &profileRes})
}

func (h *Handler) listProfiles(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()

    parseInt := func(key string) (*int, error) {
        val := q.Get(key)
        if val == "" { return nil, nil }
        i, err := strconv.Atoi(val)
        if err != nil { return nil, err }
        return &i, nil
    }

    parseFloat := func(key string) (*float64, error) {
        val := q.Get(key)
        if val == "" { return nil, nil }
        f, err := strconv.ParseFloat(val, 64)
        if err != nil { return nil, err }
        return &f, nil
    }

		page, err := strconv.Atoi(q.Get("page"))
		if err != nil || page < 1 {
        page = 1
    }
    limit, err := strconv.Atoi(q.Get("limit"))
		if err != nil || limit <= 0 {
        limit = 10
    }
    if limit > 50 {
        limit = 50
    }

    filters := domain.ProfileFilters{
        Gender:    strings.TrimSpace(q.Get("gender")),
        AgeGroup:  strings.TrimSpace(q.Get("age_group")),
        CountryID: strings.TrimSpace(q.Get("country_id")),
				SortBy: q.Get("sort_by"),
        Order:  q.Get("order"),
        Page:   page,
        Limit:  limit,
    }

    if filters.MinAge, err = parseInt("min_age"); err != nil {
        utils.RespondError(w, http.StatusUnprocessableEntity, "Invalid min_age parameter")
        return
    }
    if filters.MaxAge, err = parseInt("max_age"); err != nil {
        utils.RespondError(w, http.StatusUnprocessableEntity, "Invalid max_age parameter")
        return
    }
    if filters.MinGenderProb, err = parseFloat("min_gender_probability"); err != nil {
        utils.RespondError(w, http.StatusUnprocessableEntity, "Invalid min_gender_probability parameter")
        return
    }
    if filters.MinCountryProb, err = parseFloat("min_country_probability"); err != nil {
        utils.RespondError(w, http.StatusUnprocessableEntity, "Invalid min_country_probability parameter")
        return
    }

    if err := filters.Validate(); err != nil {
        utils.RespondError(w, http.StatusBadRequest, "Invalid query parameters")
        return
    }

		profiles, total, err := h.repo.GetFiltered(r.Context(), filters)
    if err != nil {
			fmt.Println("GetFiltered Error:", err.Error())
        utils.RespondError(w, 500, "Database failure")
        return
    }

    data := make([]ProfileDTO, len(profiles))
    for i, p := range profiles {
        data[i] = fromDomain(&p)
    }

    utils.Respond(w, http.StatusOK, map[string]any{
        "status": "success",
        "count":  len(data),
				"page":   filters.Page,
        "limit":  filters.Limit,
        "total":  total,
        "data":   data,
    })
}

func (h *Handler) deleteProfile(w http.ResponseWriter, r *http.Request) {
	err := h.repo.Delete(r.Context(), r.PathValue("id"))
	if err != nil {
		utils.RespondError(w, http.StatusNotFound, "Profile not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func fromDomain(p *domain.Profile) ProfileDTO {
	return ProfileDTO{
		ID:                 p.ID.String(),
		Name:               p.Name,
		Gender:             p.Gender,
		GenderProbability:  p.GenderProbability,
		SampleSize:         p.SampleSize,
		Age:                p.Age,
		AgeGroup:           p.AgeGroup,
		CountryID:          p.CountryID,
		CountryName:        p.CountryName,
		CountryProbability: p.CountryProbability,
		CreatedAt:           p.CreatedAt.Format(time.RFC3339),
	}
}

func mapToDTOs(profiles []domain.Profile) []ProfileDTO {
	dtos := make([]ProfileDTO, len(profiles))
	
	for i, p := range profiles {
		dtos[i] = fromDomain(&p)
	}
	
	return dtos
}