package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type result struct {
	duration time.Duration
	status   int
	cached   bool
}

type summary struct {
	label       string
	p50         time.Duration
	p95         time.Duration
	p99         time.Duration
	avg         time.Duration
	min         time.Duration
	max         time.Duration
	totalReqs   int
	errors      int
	cacheHits   int
}

var client = &http.Client{Timeout: 10 * time.Second}

var queries = []string{
	"/api/profiles?page=1&limit=10",
	"/api/profiles?gender=male&page=1&limit=10",
	"/api/profiles?country_id=NG&page=1&limit=10",
	"/api/profiles?gender=male&country_id=NG&page=1&limit=10",
	"/api/profiles?age_group=adult&page=1&limit=10",
	"/api/profiles?gender=female&sort_by=age&order=desc&page=1&limit=10",
	"/api/profiles?gender=male&min_age=20&max_age=40&page=1&limit=10",
	"/api/profiles?page=1&limit=50",
	"/api/profiles?gender=male&country_id=NG&age_group=adult&page=1&limit=10",
	"/api/profiles?sort_by=created_at&order=asc&page=2&limit=20",
}

func makeRequest(baseURL, path, token string) result {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return result{status: 0}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-API-Version", "1")

	start := time.Now()
	resp, err := client.Do(req)
	dur := time.Since(start)

	if err != nil {
		return result{duration: dur, status: 0}
	}
	defer resp.Body.Close()

	cached := resp.Header.Get("X-Cache") == "HIT"
	return result{duration: dur, status: resp.StatusCode, cached: cached}
}

func runBenchmark(label, baseURL, token string, concurrency, requests int) summary {
	results := make([]result, 0, requests)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	fmt.Printf("\nRunning %s — %d requests, %d concurrent...\n", label, requests, concurrency)

	for i := 0; i < requests; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			path := queries[i%len(queries)]
			r := makeRequest(baseURL, path, token)
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	return computeSummary(label, results)
}

func computeSummary(label string, results []result) summary {
	durations := make([]time.Duration, 0, len(results))
	errors := 0
	cacheHits := 0
	var total time.Duration

	for _, r := range results {
		if r.status == 0 || r.status >= 500 {
			errors++
			continue
		}
		durations = append(durations, r.duration)
		total += r.duration
		if r.cached {
			cacheHits++
		}
	}

	if len(durations) == 0 {
		return summary{label: label, errors: errors}
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	return summary{
		label:     label,
		p50:       durations[len(durations)*50/100],
		p95:       durations[len(durations)*95/100],
		p99:       durations[len(durations)*99/100],
		avg:       total / time.Duration(len(durations)),
		min:       durations[0],
		max:       durations[len(durations)-1],
		totalReqs: len(results),
		errors:    errors,
		cacheHits: cacheHits,
	}
}

func printSummary(s summary) {
	fmt.Printf("\n%-12s %8s %8s %8s %8s %8s %8s %6s %6s\n",
		"Label", "P50", "P95", "P99", "Avg", "Min", "Max", "Errors", "Cache%")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────")
	cachePercent := 0.0
	if s.totalReqs > 0 {
		cachePercent = float64(s.cacheHits) / float64(s.totalReqs) * 100
	}
	fmt.Printf("%-12s %8s %8s %8s %8s %8s %8s %6d %5.1f%%\n",
		s.label,
		s.p50.Round(time.Millisecond),
		s.p95.Round(time.Millisecond),
		s.p99.Round(time.Millisecond),
		s.avg.Round(time.Millisecond),
		s.min.Round(time.Millisecond),
		s.max.Round(time.Millisecond),
		s.errors,
		cachePercent,
	)
}

func printComparison(before, after summary) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║               BEFORE vs AFTER COMPARISON                    ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	printMetric("P50", before.p50, after.p50)
	printMetric("P95", before.p95, after.p95)
	printMetric("P99", before.p99, after.p99)
	printMetric("Avg", before.avg, after.avg)
	printMetric("Max", before.max, after.max)

	fmt.Printf("║  %-10s %8s → %8s    Cache hits: %d/%d (%.1f%%)         ║\n",
		"Cache%",
		"0.0%",
		fmt.Sprintf("%.1f%%", float64(after.cacheHits)/float64(after.totalReqs)*100),
		after.cacheHits, after.totalReqs,
		float64(after.cacheHits)/float64(after.totalReqs)*100,
	)
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}

func printMetric(label string, before, after time.Duration) {
	improvement := float64(before-after) / float64(before) * 100
	direction := "▼"
	if after > before {
		direction = "▲"
		improvement = -improvement
	}
	fmt.Printf("║  %-10s %8s → %8s    %s %.1f%% improvement%s          ║\n",
		label,
		before.Round(time.Millisecond),
		after.Round(time.Millisecond),
		direction,
		improvement,
		spaces(improvement),
	)
}

func spaces(f float64) string {
	if f >= 100 {
		return ""
	}
	if f >= 10 {
		return " "
	}
	return "  "
}

func main() {
	
	if err := godotenv.Load(); err != nil {
			log.Println("Running in production, skipping .env")
	}

	baseURL := os.Getenv("API_URL")
	token := os.Getenv("API_TOKEN")

	if baseURL == "" {
		baseURL = "https://127.0.0.1:8080"
	}
	if token == "" {
		fmt.Println("ERROR: API_TOKEN environment variable required")
		fmt.Println("Run: export API_TOKEN=your_token_here")
		os.Exit(1)
	}

	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  Benchmarking: %s\n", baseURL)
	fmt.Println("═══════════════════════════════════════════")

	// Phase 1 — warm up (discarded)
	fmt.Println("\nWarming up...")
	runBenchmark("warmup", baseURL, token, 5, 20)

	// Phase 2 — BEFORE (first run, no warm cache)
	// To simulate "before" accurately, run this before deploying the cache
	// If cache is already deployed, this measures cold-cache / cache-miss behaviour
	fmt.Println("\n─── PHASE 1: Cold / Before ───")
	before := runBenchmark("before", baseURL, token, 10, 100)
	printSummary(before)

	// Small pause so repeated queries warm the cache
	time.Sleep(500 * time.Millisecond)

	// Phase 3 — AFTER (cache is warm, repeated queries should hit cache)
	fmt.Println("\n─── PHASE 2: Warm / After ───")
	after := runBenchmark("after", baseURL, token, 10, 100)
	printSummary(after)

	// Print comparison table
	printComparison(before, after)

	// Save results to JSON
	output := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"base_url":  baseURL,
		"before": map[string]string{
			"p50": before.p50.String(),
			"p95": before.p95.String(),
			"p99": before.p99.String(),
			"avg": before.avg.String(),
		},
		"after": map[string]string{
			"p50": after.p50.String(),
			"p95": after.p95.String(),
			"p99": after.p99.String(),
			"avg": after.avg.String(),
		},
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile("benchmark_results.json", data, 0644)
	fmt.Println("\nResults saved to benchmark_results.json")
}