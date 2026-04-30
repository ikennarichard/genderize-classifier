package service

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/ikennarichard/genderize-classifier/internal/domain"
)


func ParseNaturalLanguage(query string) (domain.ProfileFilters, error) {
    if strings.TrimSpace(query) == "" {
        return domain.ProfileFilters{}, errors.New("empty query")
    }

    q := strings.ToLower(strings.TrimSpace(query))
    filters := domain.ProfileFilters{}
    parsedAnything := false

    hasMale := containsWord(q, "male") || containsWord(q, "males")
    hasFemale := containsWord(q, "female") || containsWord(q, "females")

    if hasMale && !hasFemale {
        filters.Gender = "male"
        parsedAnything = true
    } else if hasFemale && !hasMale {
        filters.Gender = "female"
        parsedAnything = true
    }

    if containsWord(q, "teenager") || containsWord(q, "teenagers") {
        filters.AgeGroup = "teenager"
        min, max := 13, 19
        filters.MinAge = &min
        filters.MaxAge = &max
        parsedAnything = true
    }

    if containsWord(q, "adult") || containsWord(q, "adults") {
        filters.AgeGroup = "adult"
        min := 18
        filters.MinAge = &min
        parsedAnything = true
    }

    if containsWord(q, "senior") || containsWord(q, "seniors") {
        filters.AgeGroup = "senior"
        min := 60
        filters.MinAge = &min
        parsedAnything = true
    }

    // "young" is special per spec (not an age group)
    if containsWord(q, "young") {
        min, max := 16, 24
        filters.MinAge = &min
        filters.MaxAge = &max
        parsedAnything = true
    }


    if match := regexp.MustCompile(`\b(?:above|over)\s+(\d{1,3})\b`).FindStringSubmatch(q); len(match) == 2 {
        if age, err := strconv.Atoi(match[1]); err == nil && age > 0 {
            filters.MinAge = &age
            parsedAnything = true
        }
    }

    // Under / Below
    if match := regexp.MustCompile(`\b(?:under|below)\s+(\d{1,3})\b`).FindStringSubmatch(q); len(match) == 2 {
        if age, err := strconv.Atoi(match[1]); err == nil && age > 0 {
            filters.MaxAge = &age
            parsedAnything = true
        }
    }

    // X+
    if match := regexp.MustCompile(`\b(\d{1,3})\+\b`).FindStringSubmatch(q); len(match) == 2 {
        if age, err := strconv.Atoi(match[1]); err == nil {
            filters.MinAge = &age
            parsedAnything = true
        }
    }

    // Between X and Y
    if match := regexp.MustCompile(`\bbetween\s+(\d{1,3})\s+and\s+(\d{1,3})\b`).FindStringSubmatch(q); len(match) == 3 {
        min, err1 := strconv.Atoi(match[1])
        max, err2 := strconv.Atoi(match[2])
        if err1 == nil && err2 == nil && min <= max {
            filters.MinAge = &min
            filters.MaxAge = &max
            parsedAnything = true
        }
    }

    // X to Y / X-Y
    if match := regexp.MustCompile(`\b(\d{1,3})\s*(?:to|-)\s*(\d{1,3})\b`).FindStringSubmatch(q); len(match) == 3 {
        min, err1 := strconv.Atoi(match[1])
        max, err2 := strconv.Atoi(match[2])
        if err1 == nil && err2 == nil && min <= max {
            filters.MinAge = &min
            filters.MaxAge = &max
            parsedAnything = true
        }
    }

    countries := map[string]string{
        "nigeria":  "NG",
        "angola":   "AO",
        "kenya":    "KE",
        "benin":    "BJ",
        "tanzania": "TZ",
        "united states": "US",
    }

    // "from X" or "X people"
    fromRegex := regexp.MustCompile(`\bfrom\s+([a-z]+)\b`)
    if match := fromRegex.FindStringSubmatch(q); len(match) == 2 {
        if code, ok := countries[match[1]]; ok {
            filters.CountryID = code
            parsedAnything = true
        }
    }

    // Direct country name match
    for name, code := range countries {
        if containsWord(q, name) {
            filters.CountryID = code
            parsedAnything = true
            break
        }
    }


    if !parsedAnything {
        return domain.ProfileFilters{}, errors.New("Unable to interpret query")
    }

    return filters, nil
}

func containsWord(text, word string) bool {
    pattern := `\b` + regexp.QuoteMeta(word) + `\b`
    matched, _ := regexp.MatchString(pattern, text)
    return matched
}