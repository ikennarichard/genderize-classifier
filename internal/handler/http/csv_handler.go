package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/ikennarichard/genderize-classifier/internal/utils"
)

func (h *ProfileHandler) ExportProfiles(w http.ResponseWriter, r *http.Request) {
    filters := h.parseFilters(r)

    profiles, err := h.repo.GetAllFiltered(r.Context(), filters)
    if err != nil {
        utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch profiles for export")
        return
    }

    filename := fmt.Sprintf("profiles_%s.csv", time.Now().Format("20060102_150405"))

    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

    writer := csv.NewWriter(w)
    defer writer.Flush()

    writer.Write([]string{
        "id", "name", "gender", "gender_probability",
        "age", "age_group", "country_id", "country_name",
        "country_probability", "created_at",
    })

    for _, p := range profiles {
        writer.Write([]string{
            p.ID.String(),
            p.Name,
            p.Gender,
            fmt.Sprintf("%.4f", p.GenderProbability),
            fmt.Sprintf("%d", p.Age),
            p.AgeGroup,
            p.CountryID,
            p.CountryName,
            fmt.Sprintf("%.4f", p.CountryProbability),
            p.CreatedAt.Format(time.RFC3339),
        })
    }
}