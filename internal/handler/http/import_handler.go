package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ikennarichard/insighta/internal/cache"
	"github.com/ikennarichard/insighta/internal/domain"
	"github.com/ikennarichard/insighta/internal/utils"
)

const (
	maxFileSize  = 100 << 20 // 100MB
	chunkSize    = 500        // rows per batch insert
	maxWorkers   = 4          // concurrent chunk workers
)

type ImportResult struct {
	Status    string         `json:"status"`
	TotalRows int            `json:"total_rows"`
	Inserted  int            `json:"inserted"`
	Skipped   int            `json:"skipped"`
	Reasons   map[string]int `json:"reasons"`
}

type ImportHandler struct {
	repo  domain.ProfileRepository
	cache *cache.Cache
}

func NewImportHandler(repo domain.ProfileRepository, c *cache.Cache) *ImportHandler {
	return &ImportHandler{repo: repo, cache: c}
}

func (h *ImportHandler) ImportProfiles(w http.ResponseWriter, r *http.Request) {
	// Limit memory used for multipart parsing — stream the rest to disk
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Missing file field")
		return
	}
	defer file.Close()

	if header.Size > maxFileSize {
		utils.RespondError(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("File exceeds maximum size of %dMB", maxFileSize>>20))
		return
	}

	slog.Info("csv import started",
		"filename", header.Filename,
		"size_bytes", header.Size,
	)

	result := h.processCSV(r.Context(), file)
	result.Status = "success"

	// Invalidate list cache after bulk insert
	if h.cache != nil && result.Inserted > 0 {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			h.cache.InvalidateListCache(ctx)
		}()
	}

	slog.Info("csv import complete",
		"total", result.TotalRows,
		"inserted", result.Inserted,
		"skipped", result.Skipped,
	)

	utils.Respond(w, http.StatusOK, result)
}

func (h *ImportHandler) processCSV(ctx context.Context, r io.Reader) ImportResult {
	result := ImportResult{
		Reasons: make(map[string]int),
	}

	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		result.Reasons["malformed_file"]++
		return result
	}

	colIndex, missingCols := parseHeader(header)
	if len(missingCols) > 0 {
		result.Reasons["missing_required_columns"]++
		return result
	}

	type chunk struct {
		profiles []domain.Profile
	}

	chunkCh := make(chan chunk, maxWorkers*2)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ch := range chunkCh {
				inserted, skipReasons := h.bulkInsert(ctx, ch.profiles)
				mu.Lock()
				result.Inserted += inserted
				for reason, count := range skipReasons {
					result.Reasons[reason] += count
					result.Skipped += count
				}
				mu.Unlock()
			}
		}()
	}

	batch := make([]domain.Profile, 0, chunkSize)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			mu.Lock()
			result.TotalRows++
			result.Skipped++
			result.Reasons["malformed_row"]++
			mu.Unlock()
			continue
		}

		mu.Lock()
		result.TotalRows++
		mu.Unlock()

		profile, reason := validateRow(row, colIndex)
		if reason != "" {
			mu.Lock()
			result.Skipped++
			result.Reasons[reason]++
			mu.Unlock()
			continue
		}

		batch = append(batch, *profile)

		if len(batch) >= chunkSize {
			toSend := make([]domain.Profile, len(batch))
			copy(toSend, batch)
			chunkCh <- chunk{profiles: toSend}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		chunkCh <- chunk{profiles: batch}
	}

	close(chunkCh)
	wg.Wait()

	return result
}

func (h *ImportHandler) bulkInsert(ctx context.Context, profiles []domain.Profile) (int, map[string]int) {
	if len(profiles) == 0 {
		return 0, nil
	}

	err := h.repo.BulkCreate(ctx, profiles)
	if err == nil {
		return len(profiles), nil
	}

	slog.Warn("bulk insert failed, falling back to row-by-row", "error", err)

	inserted := 0
	reasons := make(map[string]int)

	for _, p := range profiles {
		if err := h.repo.Create(ctx, &p); err != nil {
			errStr := err.Error()
			switch {
			case strings.Contains(errStr, "unique") || strings.Contains(errStr, "duplicate"):
				reasons["duplicate_name"]++
			default:
				reasons["insert_error"]++
			}
		} else {
			inserted++
		}
	}

	return inserted, reasons
}

func parseHeader(header []string) (map[string]int, []string) {
	required := []string{"name"}
	optional := []string{"gender", "gender_probability", "age", "age_group",
		"country_id", "country_name", "country_probability", "sample_size"}

	index := make(map[string]int)
	for i, col := range header {
		index[strings.ToLower(strings.TrimSpace(col))] = i
	}

	var missing []string
	for _, col := range required {
		if _, ok := index[col]; !ok {
			missing = append(missing, col)
		}
	}
	_ = optional
	return index, missing
}

func validateRow(row []string, cols map[string]int) (*domain.Profile, string) {
	get := func(key string) string {
		i, ok := cols[key]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	name := get("name")
	if name == "" {
		return nil, "missing_fields"
	}

	gender := strings.ToLower(get("gender"))
	if gender != "" && gender != "male" && gender != "female" {
		return nil, "invalid_gender"
	}

	var age int
	if ageStr := get("age"); ageStr != "" {
		a, err := strconv.Atoi(ageStr)
		if err != nil || a < 0 || a > 150 {
			return nil, "invalid_age"
		}
		age = a
	}

	parseProb := func(key string) (float64, bool) {
		s := get(key)
		if s == "" {
			return 0, true
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v < 0 || v > 1 {
			return 0, false
		}
		return v, true
	}

	genderProb, ok := parseProb("gender_probability")
	if !ok {
		return nil, "invalid_gender_probability"
	}
	countryProb, ok := parseProb("country_probability")
	if !ok {
		return nil, "invalid_country_probability"
	}

	sampleSize := 0
	if s := get("sample_size"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return nil, "invalid_sample_size"
		}
		sampleSize = n
	}

	id, _ := uuid.NewV7()
	return &domain.Profile{
		ID:                 id,
		Name:               name,
		Gender:             gender,
		GenderProbability:  genderProb,
		Age:                age,
		AgeGroup:           get("age_group"),
		CountryID:          strings.ToUpper(get("country_id")),
		CountryName:        get("country_name"),
		CountryProbability: countryProb,
		SampleSize:         sampleSize,
		CreatedAt:          time.Now().UTC(),
	}, ""
}