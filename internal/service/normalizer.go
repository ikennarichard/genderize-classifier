package service

import (
	"fmt"
	"strings"

	"github.com/ikennarichard/insighta/internal/domain"
)

// NormalizeFilters converts a ProfileFilters into a canonical form.
// Two filters that express the same intent must produce identical output.
// Rules:
//   - All string fields lowercased and trimmed
//   - Gender: "women"/"females" → "female", "men"/"males" → "male"
//   - AgeGroup: normalised to exact DB values (child/teenager/adult/senior)
//   - CountryID: uppercased (ISO codes are uppercase)
//   - Order: "ascending"/"asc" → "asc", everything else → "desc"
//   - SortBy: whitelist — invalid values cleared
//   - Age ranges derived from AgeGroup take precedence over explicit min/max
//     only when explicit values are absent

func NormalizeFilters(f domain.ProfileFilters) domain.ProfileFilters {
	n := f

	n.Gender = normalizeGender(f.Gender)
	n.CountryID = strings.ToUpper(strings.TrimSpace(f.CountryID))
	n.AgeGroup = normalizeAgeGroup(f.AgeGroup)

	// Age range derived from AgeGroup when explicit values absent
	if n.AgeGroup != "" && n.MinAge == nil && n.MaxAge == nil {
		min, max := ageRangeFromGroup(n.AgeGroup)
		if min > 0 {
			n.MinAge = &min
		}
		if max > 0 {
			n.MaxAge = &max
		}
	}

	allowed := map[string]bool{
		"age": true, "created_at": true,
		"gender_probability": true, "name": true,
	}
	sortBy := strings.ToLower(strings.TrimSpace(f.SortBy))
	if allowed[sortBy] {
		n.SortBy = sortBy
	} else {
		n.SortBy = "created_at"
	}
	order := strings.ToLower(strings.TrimSpace(f.Order))
	if order == "asc" || order == "ascending" {
		n.Order = "asc"
	} else {
		n.Order = "desc"
	}

	return n
}

func normalizeGender(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "male", "man", "men", "males":
		return "male"
	case "female", "woman", "women", "females":
		return "female"
	default:
		return ""
	}
}

func normalizeAgeGroup(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "child", "children", "kid", "kids":
		return "child"
	case "teen", "teenager", "teenagers", "youth", "young", "adolescent":
		return "teenager"
	case "adult", "adults", "grown", "grownup":
		return "adult"
	case "senior", "seniors", "elderly", "old", "older":
		return "senior"
	default:
		return ""
	}
}

func ageRangeFromGroup(group string) (int, int) {
	switch group {
	case "child":
		return 0, 12
	case "teenager":
		return 13, 19
	case "adult":
		return 20, 59
	case "senior":
		return 60, 0
	default:
		return 0, 0
	}
}

func NormalizedCacheKey(f domain.ProfileFilters, page, limit int) string {
	n := NormalizeFilters(f)
	var parts []string

	add := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+v)
		}
	}

	add("gender", n.Gender)
	add("country", n.CountryID)
	add("age_group", n.AgeGroup)
	add("sort", n.SortBy)
	add("order", n.Order)

	if n.MinAge != nil {
		add("min_age", fmt.Sprintf("%d", *n.MinAge))
	}
	if n.MaxAge != nil {
		add("max_age", fmt.Sprintf("%d", *n.MaxAge))
	}
	if n.MinGenderProb != nil {
		add("min_gp", fmt.Sprintf("%.2f", *n.MinGenderProb))
	}

	add("page", fmt.Sprintf("%d", page))
	add("limit", fmt.Sprintf("%d", limit))

	return "profiles:list:" + strings.Join(parts, "&")
}