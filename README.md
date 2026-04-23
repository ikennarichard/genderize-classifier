# Genderize Classifier

A high-performance demographic intelligence API that transforms raw data into searchable insights.

## Features

- **Advanced Filtering:** Multi-condition queries (gender, age ranges, country, probability thresholds).
- **Intelligent Sorting & Pagination:** Optimized for large datasets using PostgreSQL window functions.
- **Natural Language Query (NLQ):** Plain English search interpretation without LLM overhead.

## Natural Language Query (NLQ) Engine

The engine uses a **Rule-Based Tokenizer** to translate plain English strings into structured database filters. This allows for low-latency, deterministic queries.

### Supported Keywords & Mapping

| Keyword | Filter Mapping | Logic |
|---|---|---|
| `male` / `female` | `gender` | Strict string match (case-insensitive) |
| `young` | `min_age: 16`, `max_age: 24` | Hardcoded age range |
| `above {N}` | `min_age: N` | Regex extraction of digits after "above" |
| `from {country}` | `country_id` | Dictionary lookup (e.g., `"nigeria"` → `"NG"`) |
| `adult`, `teenager`, etc. | `age_group` | Matches stored classification groups |

### How the Logic Works

1. **Normalization:** The query is lowercased and stripped of special characters.
2. **Tokenization:** The system scans for specific "Trigger Keywords" (e.g., `"young"`).
3. **Regex Extraction:** For dynamic values like `"above 30"`, a regular expression `above (\d+)` identifies the numeric value.
4. **Filter Aggregation:** All identified intents are merged into a single `ProfileFilters` struct and passed to the PostgreSQL engine.

## Limitations & Edge Cases

While powerful, the current rule-based parser has specific constraints:

### 1. Linguistic Limitations

- **No Negation:** Queries like `"not from Nigeria"` or `"not male"` are not supported. The parser looks for positive presence of keywords.
- **Conjunction Ambiguity:** Using `"and/or"` (e.g., `"males and females"`) will result in the last identified gender overriding the first, as the system currently supports strict single-value filtering per field.
- **Strict "Above" Logic:** We currently only support `"above"`. `"Below"`, `"under"`, or `"older than"` are not yet implemented in the regex layer.

### 2. Edge Cases

- **Unknown Countries:** If a country is mentioned that isn't in the internal ISO mapping dictionary, it will be ignored.
- **Conflicting Age Logic:** If a user searches `"young adult"`, the system will apply both the `"young"` range (16–24) and the `"adult"` group filter, which may return zero results if they don't overlap in the data.
- **Non-English Queries:** The parser is strictly tuned for English keywords.

## API Endpoints

### 1. Advanced Search (Standard Filters)

```
GET /api/profiles
```

**Query Parameters:**

| Param | Description |
|---|---|
| `gender` | Filter by gender |
| `age_group` | Filter by age group |
| `country_id` | Filter by country ISO code |
| `min_age` | Minimum age |
| `max_age` | Maximum age |
| `min_gender_probability` | Minimum gender probability threshold |
| `sort_by` | Field to sort results by |
| `page` | Page number for pagination |
| `limit` | Results per page |

### 2. Natural Language Search

```
GET /api/profiles/search?q=young+males+from+nigeria
```

## Local Development

### Seed Database

The system automatically seeds **2026 unique profiles** on the first run using `go:embed` to prevent duplicates.

```bash
go run cmd/seed/main.go
go run main.go
```

## Deployment

Live on Vercel: [https://genderize-plum.vercel.app](https://genderize-plum.vercel.app)