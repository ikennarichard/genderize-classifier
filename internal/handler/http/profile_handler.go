package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ikennarichard/genderize-classifier/internal/domain"
	"github.com/ikennarichard/genderize-classifier/internal/service"
	"github.com/ikennarichard/genderize-classifier/internal/utils"
)

type ProfileHandler struct {
	repo domain.ProfileRepository 
}

func New(repo domain.ProfileRepository) *ProfileHandler {
	return &ProfileHandler{repo: repo}
}

func (h *ProfileHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    if err := h.repo.Delete(r.Context(), id); err != nil {
        utils.RespondError(w, http.StatusNotFound, "Profile not found")
        return
    }
    w.WriteHeader(http.StatusNoContent)
}


func (h *ProfileHandler) SearchProfiles(w http.ResponseWriter, r *http.Request) {
    queryStr := strings.TrimSpace(r.URL.Query().Get("q"))
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
    if page < 1 {
        page = 1
    }
    if limit <= 0 || limit > 50 {
        limit = 10
    }

    if err := filters.Validate(); err != nil {
        utils.RespondError(w, http.StatusBadRequest, "Invalid query parameters")
        return
    }

    profiles, total, err := h.repo.GetFiltered(r.Context(), filters, page, limit)
    if err != nil {
        utils.RespondError(w, http.StatusInternalServerError, "Database error")
        return
    }

    data := mapToDTOs(profiles)

    resp := utils.BuildPaginatedResponse(
        data,
        page,
        limit,
        total,
        "/api/profiles/search",
        r.URL.Query(),
    )

    utils.Respond(w, http.StatusOK, resp)
}

func (h *ProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("failed to create profile", 
            "error", err, 
            "user_id", r.Context().Value("user_id"),
        )
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	createdRes := fromDomain(profile)
	utils.Respond(w, http.StatusCreated, ProfileResponse{Status: "success", Data: &createdRes})
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
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

func (h *ProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
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

    filters := h.parseFilters(r)

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

		profiles, total, err := h.repo.GetFiltered(r.Context(), filters, page, limit)
    if err != nil {
			fmt.Println("GetFiltered Error:", err.Error())
        utils.RespondError(w, 500, "Database failure")
        return
    }

    data := mapToDTOs(profiles)


     resp := utils.BuildPaginatedResponse(
        data, 
        page, 
        limit, 
        total, 
        "/api/v1/profiles", 
        r.URL.Query(),
    )
		utils.Respond(w, http.StatusOK, resp)
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

func (h *ProfileHandler) parseFilters(r *http.Request) domain.ProfileFilters {
	q := r.URL.Query()

	filters := domain.ProfileFilters{
		Gender:    strings.TrimSpace(q.Get("gender")),
		CountryID: strings.TrimSpace(q.Get("country_id")),
		AgeGroup:  strings.TrimSpace(q.Get("age_group")),
		SortBy:    strings.TrimSpace(strings.ToLower(q.Get("sort_by"))),
		Order:     strings.TrimSpace(strings.ToLower(q.Get("order"))),
	}

	if minAge, err := strconv.Atoi(q.Get("min_age")); err == nil {
		filters.MinAge = &minAge
	}
	if maxAge, err := strconv.Atoi(q.Get("max_age")); err == nil {
		filters.MaxAge = &maxAge
	}

	if minProb, err := strconv.ParseFloat(q.Get("min_gender_probability"), 64); err == nil {
		filters.MinGenderProb = &minProb
	}

	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}


	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	return filters
}

