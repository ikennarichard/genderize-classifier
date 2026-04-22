package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var httpClient = &http.Client{Timeout: 8 * time.Second}

type GenderizeResponse struct {
	Name        string  `json:"name"`
	Gender      *string `json:"gender"`
	Probability float64 `json:"probability"`
	Count       int     `json:"count"`
}

type AgifyResponse struct {
	Name  string `json:"name"`
	Age   *int   `json:"age"`
	Count int    `json:"count"`
}

type NationalizeResponse struct {
	Name    string    `json:"name"`
	Country []Country `json:"country"`
}

type Country struct {
	CountryID   string  `json:"country_id"`
	Probability float64 `json:"probability"`
}

func fetchJSON(apiURL string, target any) error {
	resp, err := httpClient.Get(apiURL)
	if err != nil {
		fmt.Println("error:", err)
		return err
	}
	// fmt.Println("data:", resp)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 status: %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func buildURL(base, name string) string {
	params := url.Values{}
	params.Set("name", name)
	return base + "?" + params.Encode()
}

func FetchGenderize(name string) (*GenderizeResponse, error) {
	var r GenderizeResponse
	return &r, fetchJSON(buildURL("https://api.genderize.io/", name), &r)
}

func FetchAgify(name string) (*AgifyResponse, error) {
	var r AgifyResponse
	return &r, fetchJSON(buildURL("https://api.agify.io/", name), &r)
}

func FetchNationalize(name string) (*NationalizeResponse, error) {
	var r NationalizeResponse
	return &r, fetchJSON(buildURL("https://api.nationalize.io/", name), &r)
}