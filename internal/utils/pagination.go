package utils

import (
	"fmt"
	"math"
	"net/url"
)

type PaginationLinks struct {
	Self string  `json:"self"`
	Next *string `json:"next"`
	Prev *string `json:"prev"`
}

type PaginatedResponse struct {
	Status     string          `json:"status"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	Total      int             `json:"total"`
	TotalPages int             `json:"total_pages"`
	Links      PaginationLinks `json:"links"`
	Data       any             `json:"data"`
}

func BuildPaginatedResponse(data any, page, limit, total int, baseUrl string, queryParams url.Values) PaginatedResponse {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	genUrl := func(p int) string {
		q := queryParams
		q.Set("page", fmt.Sprintf("%d", p))
		q.Set("limit", fmt.Sprintf("%d", limit))
		return fmt.Sprintf("%s?%s", baseUrl, q.Encode())
	}

	links := PaginationLinks{
		Self: genUrl(page),
	}

	if page < totalPages {
		next := genUrl(page + 1)
		links.Next = &next
	}

	if page > 1 {
		prev := genUrl(page - 1)
		links.Prev = &prev
	}

	return PaginatedResponse{
		Status:     "success",
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		Links:      links,
		Data:       data,
	}
}