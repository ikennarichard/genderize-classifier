package service

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/ikennarichard/genderize-classifier/internal/domain"
)

func ParseNaturalLanguage(query string) (domain.ProfileFilters, error) {
	q := strings.ToLower(query)
	filters := domain.ProfileFilters{}
	interpreted := false

	// gnder mapping
	if strings.Contains(q, "male") && !strings.Contains(q, "female") {
		filters.Gender = "male"
		interpreted = true
	} else if strings.Contains(q, "female") {
		filters.Gender = "female"
		interpreted = true
	}

	// age mapping
	if strings.Contains(q, "young") {
		min := 16
		max := 24
		filters.MinAge = &min
		filters.MaxAge = &max
		interpreted = true
	}

	aboveRegex := regexp.MustCompile(`above (\d+)`)
	if match := aboveRegex.FindStringSubmatch(q); len(match) > 1 {
		age, _ := strconv.Atoi(match[1])
		filters.MinAge = &age
		interpreted = true
	}

	countries := map[string]string{
		"nigeria": "NG",
		"angola":  "AO",
		"kenya":   "KE",
		"benin":   "BJ",
		"tanzania": "TZ",
	}
	for name, code := range countries {
		if strings.Contains(q, name) {
			filters.CountryID = code
			interpreted = true
		}
	}

	ageGroups := []string{"child", "teenager", "adult", "senior"}
	for _, group := range ageGroups {
		if strings.Contains(q, group) {
			filters.AgeGroup = group
			interpreted = true
		}
	}

	if !interpreted {
		return filters, errors.New("unable to interpret query")
	}

	return filters, nil
}