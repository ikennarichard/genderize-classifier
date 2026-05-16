package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ikennarichard/insighta/internal/client"
	"github.com/ikennarichard/insighta/internal/domain"
)


func newApiError(apiName string) error {
    return fmt.Errorf("%s returned an invalid response", apiName)
}

func ageGroup(age int) string {
	switch {
	case age <= 12:
		return "child"
	case age <= 19:
		return "teenager"
	case age <= 59:
		return "adult"
	default:
		return "senior"
	}
}

func topCountry(countries []client.Country) client.Country {
	top := countries[0]
	for _, c := range countries[1:] {
		if c.Probability > top.Probability {
			top = c
		}
	}
	return top
}

type results struct {
	gender      *client.GenderizeResponse
	age         *client.AgifyResponse
	nationality *client.NationalizeResponse
	genderErr   error
	ageErr      error
	natErr      error
}

func fetchAll(name string) results {
	var (
		wg  sync.WaitGroup
		res results
	)
	wg.Add(3)

	go func() { defer wg.Done(); res.gender, res.genderErr = client.FetchGenderize(name) }()
	go func() { defer wg.Done(); res.age, res.ageErr = client.FetchAgify(name) }()
	go func() { defer wg.Done(); res.nationality, res.natErr = client.FetchNationalize(name) }()

	wg.Wait()
	return res
}


func BuildProfile(name string) (*domain.Profile, error) {
	res := fetchAll(name)

	if res.genderErr != nil || res.gender.Gender == nil || res.gender.Count == 0 {
		return nil, newApiError("Genderize")
	}
	if res.ageErr != nil || res.age.Age == nil {
		return nil, newApiError("Agify")
	}
	if res.natErr != nil || len(res.nationality.Country) == 0 {
		return nil, newApiError("Nationalize")
	}

	country := topCountry(res.nationality.Country)
	id, _ := uuid.NewV7()

	return &domain.Profile{
		ID:                 id,
		Name:               name,
		Gender:             *res.gender.Gender,
		GenderProbability:  res.gender.Probability,
		SampleSize:         res.gender.Count,
		Age:                *res.age.Age,
		AgeGroup:           ageGroup(*res.age.Age),
		CountryID:          country.CountryID,
		CountryName:        resolveCountryName(country.CountryID),
		CountryProbability: country.Probability,
		CreatedAt:          time.Now().UTC(),
	}, nil
}


func resolveCountryName(code string) string {
	countries := map[string]string{
		"AF": "Afghanistan",
		"AL": "Albania",
		"DZ": "Algeria",
		"AR": "Argentina",
		"AU": "Australia",
		"AT": "Austria",
		"BD": "Bangladesh",
		"BE": "Belgium",
		"BR": "Brazil",
		"BG": "Bulgaria",
		"CA": "Canada",
		"CL": "Chile",
		"CN": "China",
		"CO": "Colombia",
		"HR": "Croatia",
		"CZ": "Czech Republic",
		"DK": "Denmark",
		"EG": "Egypt",
		"ET": "Ethiopia",
		"FI": "Finland",
		"FR": "France",
		"DE": "Germany",
		"GH": "Ghana",
		"GR": "Greece",
		"HU": "Hungary",
		"IN": "India",
		"ID": "Indonesia",
		"IQ": "Iraq",
		"IE": "Ireland",
		"IL": "Israel",
		"IT": "Italy",
		"JP": "Japan",
		"KE": "Kenya",
		"MY": "Malaysia",
		"MX": "Mexico",
		"MA": "Morocco",
		"NL": "Netherlands",
		"NZ": "New Zealand",
		"NG": "Nigeria",
		"NO": "Norway",
		"PK": "Pakistan",
		"PE": "Peru",
		"PH": "Philippines",
		"PL": "Poland",
		"PT": "Portugal",
		"RO": "Romania",
		"RU": "Russia",
		"SA": "Saudi Arabia",
		"ZA": "South Africa",
		"ES": "Spain",
		"SE": "Sweden",
		"CH": "Switzerland",
		"TZ": "Tanzania",
		"TH": "Thailand",
		"TN": "Tunisia",
		"TR": "Turkey",
		"UG": "Uganda",
		"UA": "Ukraine",
		"AE": "United Arab Emirates",
		"GB": "United Kingdom",
		"US": "United States",
		"VN": "Vietnam",
		"ZW": "Zimbabwe",
	}

	if name, ok := countries[code]; ok {
		return name
	}
	return code
}