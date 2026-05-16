package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ikennarichard/insighta/internal/cache"
	"github.com/ikennarichard/insighta/internal/domain"
	"github.com/ikennarichard/insighta/internal/service"
	"github.com/ikennarichard/insighta/internal/utils"
)

type ProfileHandler struct {
	repo domain.ProfileRepository 
    cache *cache.Cache
}

func New(repo domain.ProfileRepository, cache *cache.Cache) *ProfileHandler {
	return &ProfileHandler{repo: repo, cache: cache}
}

func (h *ProfileHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    if err := h.repo.Delete(r.Context(), id); err != nil {
        utils.RespondError(w, http.StatusNotFound, "Profile not found")
        return
    }
    if h.cache != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			h.cache.InvalidateProfileID(ctx, id)
			h.cache.InvalidateListCache(ctx)
		}()
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

				// Normalize parsed filters — "women aged 20-45 in Nigeria" and
// "Nigerian females between 20 and 45" now produce the same cache key
normalizedFilters := service.NormalizeFilters(filters)
key := service.NormalizedCacheKey(normalizedFilters, page, limit)

	// Check cache first
	if h.cache != nil {
		var cached utils.PaginatedResponse
		if hit, _ := h.cache.Get(r.Context(), key, &cached); hit {
			utils.Respond(w, http.StatusOK, cached)
			return
		}
	}

    profiles, total, err := h.repo.GetFiltered(r.Context(), normalizedFilters, page, limit)
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
    	if h.cache != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			h.cache.InvalidateListCache(ctx)
		}()
	}
	utils.Respond(w, http.StatusCreated, ProfileResponse{Status: "success", Data: &createdRes})
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
    	if h.cache != nil {
		var cached ProfileResponse
		if hit, _ := h.cache.Get(r.Context(), cache.IDKey(id), &cached); hit {
			utils.Respond(w, http.StatusOK, cached)
			return
		}
	}
	profile, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		fmt.Println("GetProfile Error:", err)
		utils.RespondError(w, http.StatusNotFound, "Profile not found")
		return
	}
	profileRes := fromDomain(profile)
   resp := ProfileResponse{Status: "success", Data: &profileRes}

	if h.cache != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			h.cache.Set(ctx, cache.IDKey(id), resp, cache.ProfileByIDTTL)
		}()
	}

	utils.Respond(w, http.StatusOK, resp)
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

    	// Build cache key from all query params
	cacheParams := map[string]string{
		"gender":     filters.Gender,
		"country_id": filters.CountryID,
		"age_group":  filters.AgeGroup,
		"sort_by":    filters.SortBy,
		"order":      filters.Order,
		"page":       fmt.Sprintf("%d", page),
		"limit":      fmt.Sprintf("%d", limit),
	}
	if filters.MinAge != nil {
		cacheParams["min_age"] = fmt.Sprintf("%d", *filters.MinAge)
	}
	if filters.MaxAge != nil {
		cacheParams["max_age"] = fmt.Sprintf("%d", *filters.MaxAge)
	}

normalizedFilters := service.NormalizeFilters(filters)
key := service.NormalizedCacheKey(normalizedFilters, page, limit)

	// Check cache first
	if h.cache != nil {
		var cached utils.PaginatedResponse
		if hit, _ := h.cache.Get(r.Context(), key, &cached); hit {
			utils.Respond(w, http.StatusOK, cached)
			return
		}
	}

		profiles, total, err := h.repo.GetFiltered(r.Context(), normalizedFilters, page, limit)
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

	if h.cache != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			h.cache.Set(ctx, key, resp, cache.ProfileListTTL)
		}()
	}
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

