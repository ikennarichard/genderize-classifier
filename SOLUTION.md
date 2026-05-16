# SOLUTION.md

## Part 1 — Query Performance

### Approach

Two changes applied in order of risk — indexes first, caching second.

**Indexes**

Added compound indexes on the columns that appear most often in WHERE and
ORDER BY clauses: `(gender, country_id)`, `(age_group)`, `(age)`,
`(created_at DESC)`, `(name)`. All created with `CONCURRENTLY` to avoid
locking the table during migration.

Also added `(expires_at)` on the sessions table so the daily cleanup job
runs as a range delete rather than a full scan.

**Caching**

Added a Redis cache layer in front of the database. On every list or search
request the handler derives a cache key, checks Redis, and returns the
cached result if present. On cache miss it queries the database, stores the
result with a 5-minute TTL, and returns it. Any write (create, delete,
import) invalidates all list cache keys.

Redis failure is treated as a cache miss — the system degrades gracefully
to database-only rather than returning errors to clients. Cache writes are
done in a goroutine so they never delay the response.

### Before / After

| Metric | Before (cold) | After (warm cache) | Improvement |
|--------|--------------|-------------------|-------------|
| P50    | ~129ms       | ~50ms             | ~61%        |
| P95    | ~1400ms      | ~83ms             | ~89%        |
| P99    | ~1500ms      | ~140ms            | ~94.2%        |
| Avg    | ~272ms       | ~51ms             | ~81.3%        |
| Cache hit rate | 0%   | ~87%              | —           |

The before figures reflect uncached database queries over a remote
connection. The after figures reflect a warm cache where ~87% of requests
are served without a database call.

### Decisions and trade-offs

- **5-minute TTL**: profile data changes in batches, not in real time.
  Five minutes of staleness is acceptable for analytics. Shorter reduces
  hit rate; longer risks stale results after a write.
- **Invalidate on write, not per row**: batch imports invalidate the cache
  once at the end of the job. Per-row invalidation would cause cache
  thrashing and defeat the purpose of caching.
- **No in-process cache**: a `sync.Map` would be simpler but does not
  survive restarts and does not share state across instances.

---

## Part 2 — Query Normalization

### Approach

Before checking the cache or executing a query, every filter object passes
through `NormalizeFilters`. This converts semantically equivalent inputs
into a single canonical form:

- `"males"` / `"men"` → `"male"`. `"women"` / `"females"` → `"female"`
- `"ng"` / `"NG"` → `"NG"` (ISO codes are always uppercase)
- `"youth"` / `"teenager"` / `"young"` → `"teenager"`
- `"ascending"` / `"asc"` → `"asc"`
- Invalid sort columns fall back to `"created_at"`
- AgeGroup implies a min/max age range when explicit values are absent

The cache key is then built from the normalised object with fields appended
in a fixed order. Two queries expressing the same intent always produce the
same key.

### Why no sorting of fields

Fields are appended in a fixed declared order rather than sorted at
runtime. This is simpler, faster, and equally deterministic — there is no
ambiguity about field order because it never changes.

### What it does not do

It does not attempt semantic interpretation beyond the documented patterns.
"People who are not young" is not handled — the parser returns an error and
the normalizer is never reached. This is intentional: incorrect
interpretation is worse than no result.

---

## Part 3 — CSV Ingestion

### Approach

**Streaming**: the CSV is read row by row using `encoding/csv`. The entire
file is never held in memory. A sliding batch of 500 rows is accumulated
and sent to a worker as a chunk.

**Worker pool**: four goroutines consume chunks from a channel and bulk-
insert them concurrently. This keeps CPU and database connection usage
bounded regardless of file size.

**Bulk insert**: each chunk is inserted as a single multi-row
`INSERT ... ON CONFLICT (name) DO NOTHING`. This is orders of magnitude
faster than 500 individual inserts and lets the database handle duplicate
detection atomically.

**Fallback**: if a bulk insert fails (e.g. a constraint violation that
`DO NOTHING` does not cover), the chunk falls back to row-by-row insertion
so as many rows as possible are saved and skip reasons are reported
accurately.

### Failure handling

- A malformed row (wrong column count, broken encoding) is skipped and
  counted under `malformed_row`. Processing continues.
- Missing required fields, invalid age, unrecognised gender, invalid
  probabilities — each skipped and counted under its own reason key.
- Duplicate names are handled by `ON CONFLICT DO NOTHING` at the database
  layer — no pre-check query needed.
- If processing fails midway (context cancelled, database down), rows
  already inserted remain. There is no rollback. This matches the spec:
  partial inserts are kept.
- A single bad row never fails the upload.

### Decisions and trade-offs

- **Chunk size 500**: small enough to keep per-insert latency low, large
  enough to amortise round-trip overhead. At 500,000 rows this is 1000
  database round trips instead of 500,000.
- **4 workers**: matches typical database connection pool size. More
  workers would saturate the pool and cause queueing.
- **No pre-existence check**: checking `GetByName` before every row would
  double the database calls. `ON CONFLICT DO NOTHING` is a single atomic
  operation.
- **Cache invalidated once at job end**: not per chunk. Invalidating per
  chunk would thrash the cache during a large import.
- **100MB file size limit**: prevents a single upload from exhausting
  server memory. 500,000 rows at ~200 bytes each is ~100MB.
- **No message queue**: the spec says no unnecessary infrastructure.
  Goroutine workers with a buffered channel achieve the same concurrency
  without the operational overhead of a queue.

### Edge cases handled

| Case | Handling |
|------|---------|
| Missing `name` column in header | Abort entire file, return error |
| Row with wrong column count | Skip row, count as `malformed_row` |
| Negative age | Skip row, count as `invalid_age` |
| Age > 150 | Skip row, count as `invalid_age` |
| Probability outside 0–1 | Skip row, count as `invalid_*_probability` |
| Duplicate name | `ON CONFLICT DO NOTHING`, count as `duplicate_name` |
| File > 100MB | Reject before processing, return 413 |
| Context cancelled mid-import | Inserted rows kept, workers drain cleanly |
| Empty file | Returns zero counts, no error |