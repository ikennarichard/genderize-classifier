package service

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/ikennarichard/genderize-classifier/internal/domain"
)

type ExportService struct{}

func (s *ExportService) StreamProfilesToCSV(w io.Writer, profiles []domain.Profile) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"ID", "Name", "Gender", "Gender_Probability", 
		"Age", "Age_Group", "Country_ID", "Country_Name", 
		"Created_At",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, p := range profiles {
		row := []string{
			p.ID.String(),
			p.Name,
			p.Gender,
			fmt.Sprintf("%.2f", p.GenderProbability),
			fmt.Sprintf("%d", p.Age),
			p.AgeGroup,
			p.CountryID,
			p.CountryName,
			p.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}