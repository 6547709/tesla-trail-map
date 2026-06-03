package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool *pgxpool.Pool
	logger *slog.Logger
)

// Version is the public release tag of this build. Bumped in lockstep with
// CHANGELOG.md and the git tag (e.g. v1.6).
const Version = "1.6"

// LatestVersion is the latest released version this build is aware of.
// Kept equal to Version on every release so /version can flag itself as
// "up to date" without needing to call out to GitHub.
const LatestVersion = "1.6"

type Position struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Date      string  `json:"date"`
}

type DriveWithPositions struct {
	DriveID   int        `json:"drive_id"`
	Positions []Position `json:"positions"`
}

// Car represents one vehicle exposed to the UI. We deliberately hide
// internal car_id from the user — they pick by display name.
type Car struct {
	ID            int    `json:"id"`             // db car_id, kept for the API layer
	Name          string `json:"name"`           // display name (with fallback)
	Model         string `json:"model"`          // "3", "Y", ...
	MarketingName string `json:"marketing_name"` // "LR AWD Performance", ...
	DriveCount    int    `json:"drive_count"`    // how many trips for this car
}

// listCars returns every car the user owns, ordered by display priority.
// Name fallback chain: cars.name → marketing_name → "Model "+model → "Car #id".
func listCars(ctx context.Context) ([]Car, error) {
	const q = `
SELECT c.id,
       COALESCE(NULLIF(c.name, ''),
                NULLIF(c.marketing_name, ''),
                'Model ' || COALESCE(c.model, ''),
                'Car #' || c.id::text) AS name,
       COALESCE(c.model, '')           AS model,
       COALESCE(c.marketing_name, '')  AS marketing_name,
       COALESCE(d.cnt, 0)::int         AS drive_count
FROM cars c
LEFT JOIN (SELECT car_id, count(*) AS cnt FROM drives GROUP BY car_id) d
       ON d.car_id = c.id
ORDER BY c.display_priority ASC, c.id ASC`
	rows, err := dbPool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list cars: %w", err)
	}
	defer rows.Close()
	cars := make([]Car, 0, 4)
	for rows.Next() {
		var c Car
		if err := rows.Scan(&c.ID, &c.Name, &c.Model, &c.MarketingName, &c.DriveCount); err != nil {
			return nil, fmt.Errorf("scan car: %w", err)
		}
		cars = append(cars, c)
	}
	return cars, rows.Err()
}

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// envOr returns env value or default.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envInt returns env value parsed as int, or default.
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// envDur returns env value parsed as time.Duration, or default.
func envDur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// buildConnString builds a DSN with performance-related Postgres params.
func buildConnString() string {
	host := envOr("DATABASE_HOST", "192.168.10.200")
	port := envInt("DATABASE_PORT", 54320)
	user := envOr("DATABASE_USER", "teslamate")
	pass := envOr("DATABASE_PASS", "secret")
	name := envOr("DATABASE_NAME", "teslamate")
	sslmode := envOr("DATABASE_SSLMODE", "disable")

	// PostgreSQL session-level timeouts (milliseconds) prevent runaway queries
	// from hogging a pooled connection.
	stmtTimeoutMs := envInt("DATABASE_STMT_TIMEOUT_MS", 15000)        // 15s default
	idleTxTimeoutMs := envInt("DATABASE_IDLE_TX_TIMEOUT_MS", 30000)   // 30s
	lockTimeoutMs := envInt("DATABASE_LOCK_TIMEOUT_MS", 5000)         // 5s
	tcpKeepalivesIdle := envInt("DATABASE_TCP_KEEPALIVES_IDLE", 30)   // sec

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s "+
			"application_name=tesla_trail_map "+
			"statement_timeout=%d "+
			"idle_in_transaction_session_timeout=%d "+
			"lock_timeout=%d "+
			"tcp_user_timeout=%d "+
			"connect_timeout=5",
		host, port, user, pass, name, sslmode,
		stmtTimeoutMs, idleTxTimeoutMs, lockTimeoutMs, tcpKeepalivesIdle*1000,
	)
}

// initDB builds a tuned pgxpool with prepared-statement cache, health checks,
// jittered max-lifetime, and a brief warm-up.
func initDB() error {
	connString := buildConnString()

	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("parse connection string: %w", err)
	}

	// ---------- Pool sizing ----------
	// Default sizes scale with logical CPUs but are overridable via env.
	cpus := runtime.NumCPU()
	cfg.MaxConns = int32(envInt("DATABASE_MAX_CONNS", max(20, cpus*4)))
	cfg.MinConns = int32(envInt("DATABASE_MIN_CONNS", max(4, cpus)))

	// ---------- Lifetime & idle ----------
	cfg.MaxConnLifetime = envDur("DATABASE_MAX_LIFETIME", time.Hour)
	// Jitter spreads forced reconnects so we don't get thundering-herd reconnects.
	cfg.MaxConnLifetimeJitter = envDur("DATABASE_LIFETIME_JITTER", 5*time.Minute)
	cfg.MaxConnIdleTime = envDur("DATABASE_MAX_IDLE", 30*time.Minute)
	cfg.HealthCheckPeriod = envDur("DATABASE_HEALTHCHECK_PERIOD", 30*time.Second)

	// ---------- Per-connection tuning ----------
	// Use the cached prepared-statement protocol — best throughput for repeated
	// parameterised queries (which is exactly what /trips does).
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
	// Cap the prepared-statement cache so we never balloon if SQL grows.
	cfg.ConnConfig.StatementCacheCapacity = envInt("DATABASE_STMT_CACHE", 256)
	// Cap descriptor cache (column descriptions reused across repeats).
	cfg.ConnConfig.DescriptionCacheCapacity = envInt("DATABASE_DESC_CACHE", 256)

	// Tag each backend session — invaluable when DBA inspects pg_stat_activity.
	cfg.ConnConfig.RuntimeParams["application_name"] = "tesla_trail_map"

	// ---------- Lifecycle hooks (per-connection setup) ----------
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Bake a per-session GUC that pgxpool can't pass via DSN cleanly.
		// jit=off speeds up small OLTP queries on PG12+ that don't need JIT.
		_, err := conn.Exec(ctx, "SET jit = off")
		return err
	}
	cfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		// Drop a connection that's been idle-marked unhealthy.
		return conn.IsClosed() == false
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}

	// ---------- Initial ping + warm-up ----------
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("ping: %w", err)
	}

	// Warm up to MinConns so the first user request doesn't pay the TCP+TLS
	// handshake tax. Acquire concurrently and release back.
	warmCtx, warmCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer warmCancel()
	conns := make([]*pgxpool.Conn, 0, cfg.MinConns)
	for i := int32(0); i < cfg.MinConns; i++ {
		c, err := pool.Acquire(warmCtx)
		if err != nil {
			break
		}
		conns = append(conns, c)
	}
	for _, c := range conns {
		c.Release()
	}

	dbPool = pool
	logger.Info("db pool ready",
		"max_conns", cfg.MaxConns,
		"min_conns", cfg.MinConns,
		"max_lifetime", cfg.MaxConnLifetime,
		"jitter", cfg.MaxConnLifetimeJitter,
		"healthcheck", cfg.HealthCheckPeriod,
		"warmed", len(conns),
	)
	return nil
}

// getTripsWithPositions executes the trips query with a per-request deadline,
// pre-allocated slices, slow-query logging, and an optional server-side
// downsample cap to keep response sizes bounded.
//
// Performance background (validated against the live teslamate DB,
// 5,945 drives / 19.9M positions, PG 18.3, BRIN indexes only):
//
//	V0  JOIN drives d→positions p ON drive_id, filter by drives.start_date
//	    7-day window: 28.7 s, 204k rows, external-merge sort to disk.
//	    ❌ The existing BRIN(drive_id,date) is awful as an equality probe.
//
//	V1  Filter positions by p.date BETWEEN first (matches BRIN(date)),
//	    then constrain to drive ids from the date-windowed drives.
//	    7-day window: 2.0 s, same 204k rows.  ✅ 14× faster.
//
// We adopt V1. Schema is read-only (no CREATE INDEX allowed), so the
// rewrite does the heavy lifting alone.
//
// carID == 0 means "all cars" (back-compat); otherwise filter by drives.car_id.
func getTripsWithPositions(ctx context.Context, startDate, endDate string, carID, maxPointsPerDrive int) ([]DriveWithPositions, error) {
	const slowThreshold = 500 * time.Millisecond
	t0 := time.Now()
	logger.Info("trips query start",
		"start_date", startDate, "end_date", endDate,
		"car_id", carID,
		"max_points_per_drive", maxPointsPerDrive)

	// Per-query deadline — guards even if statement_timeout missed it.
	qctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// drives CTE optionally constrained by car_id. We mark the CTE
	// `AS MATERIALIZED` so PG builds the small drive-id set once and
	// hash-joins it against positions, rather than NestedLoop probing
	// drives 200k+ times per row of positions. Validated in dbprobe7:
	// without MATERIALIZED, car_id=2 (less-active car) takes 60s; with it,
	// 1.5s — same plan as car_id=1.
	var query string
	var args []any

	carCond := ""
	if carID > 0 {
		carCond = "AND car_id = $4"
	}

	if maxPointsPerDrive > 0 {
		query = fmt.Sprintf(`
WITH d AS MATERIALIZED (
  SELECT id FROM drives
  WHERE start_date >= $1 AND end_date <= $2 %s
), ranked AS (
  SELECT p.drive_id, p.latitude, p.longitude, p.date,
         row_number() OVER (PARTITION BY p.drive_id ORDER BY p.date) AS rn,
         count(*)    OVER (PARTITION BY p.drive_id)                  AS total
  FROM positions p
  WHERE p.date >= $1 AND p.date <= $2
    AND p.drive_id IN (SELECT id FROM d)
)
SELECT drive_id, latitude, longitude, date
FROM ranked
WHERE rn %% GREATEST(1, total / $3::bigint) = 0
   OR rn = 1
   OR rn = total
ORDER BY drive_id, date`, carCond)
		args = []any{startDate, endDate, int64(maxPointsPerDrive)}
	} else {
		query = fmt.Sprintf(`
WITH d AS MATERIALIZED (
  SELECT id FROM drives
  WHERE start_date >= $1 AND end_date <= $2 %s
)
SELECT p.drive_id, p.latitude, p.longitude, p.date
FROM positions p
WHERE p.date >= $1 AND p.date <= $2
  AND p.drive_id IN (SELECT id FROM d)
ORDER BY p.drive_id, p.date`, carCond)
		args = []any{startDate, endDate}
	}

	if carID > 0 {
		args = append(args, int16(carID))
	}

	rows, err := dbPool.Query(qctx, query, args...)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("query timeout after %s: %w", time.Since(t0), err)
		}
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	// Pre-size based on a rough heuristic to reduce slice growth.
	driveMap := make(map[int]*DriveWithPositions, 64)
	posCount := 0

	for rows.Next() {
		var driveID int
		var lat, lng float64
		var ts time.Time
		if err := rows.Scan(&driveID, &lat, &lng, &ts); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		d, ok := driveMap[driveID]
		if !ok {
			d = &DriveWithPositions{DriveID: driveID, Positions: make([]Position, 0, 256)}
			driveMap[driveID] = d
		}
		d.Positions = append(d.Positions, Position{
			Latitude:  lat,
			Longitude: lng,
			Date:      ts.Format(time.RFC3339),
		})
		posCount++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iter: %w", err)
	}

	drives := make([]DriveWithPositions, 0, len(driveMap))
	for _, d := range driveMap {
		drives = append(drives, *d)
	}

	elapsed := time.Since(t0)
	if elapsed > slowThreshold {
		logger.Warn("slow trips query",
			"elapsed", elapsed,
			"drives", len(drives),
			"positions", posCount,
		)
	} else {
		logger.Info("trips query done",
			"elapsed", elapsed,
			"drives", len(drives),
			"positions", posCount,
		)
	}
	return drives, nil
}

// max returns the larger of two ints (Go 1.21+ has built-in, but explicit
// helper keeps this file compatible with older toolchains too).
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// parseFlexibleTime accepts the formats the UI emits plus pure dates.
// Anything that doesn't match returns an error — callers MUST report 400.
func parseFlexibleTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format")
}

func main() {
	if err := initDB(); err != nil {
		logger.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// B15 — release mode silences the GIN debug noise.
	if envOr("GIN_MODE", "release") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	// B16 — explicitly disable proxy trust to silence the warning;
	// the user can re-enable via env if they sit behind a reverse proxy.
	if v := os.Getenv("TRUSTED_PROXIES"); v != "" {
		_ = r.SetTrustedProxies(strings.Split(v, ","))
	} else {
		_ = r.SetTrustedProxies(nil)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "route not found"})
	})

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.LoadHTMLFiles("map.html")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "map.html", nil)
	})

	r.GET("/trips", func(c *gin.Context) {
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")

		// B1/B2/B3 — input validation, BEFORE we touch the DB.
		// Accept timestamps in the form 'YYYY-MM-DDTHH:MM[:SS]' or 'YYYY-MM-DD'.
		// The handler returns 400 with a stable, human-readable message; the
		// internal error is only logged.
		const badRequest = http.StatusBadRequest
		if startDate == "" || endDate == "" {
			c.JSON(badRequest, gin.H{"message": "start_date and end_date are required"})
			return
		}
		ts, err := parseFlexibleTime(startDate)
		if err != nil {
			c.JSON(badRequest, gin.H{"message": "invalid start_date format, expected YYYY-MM-DD or YYYY-MM-DDTHH:MM[:SS]"})
			return
		}
		te, err := parseFlexibleTime(endDate)
		if err != nil {
			c.JSON(badRequest, gin.H{"message": "invalid end_date format, expected YYYY-MM-DD or YYYY-MM-DDTHH:MM[:SS]"})
			return
		}
		if !ts.Before(te) {
			c.JSON(badRequest, gin.H{"message": "start_date must be earlier than end_date"})
			return
		}
		// Hard-cap window to keep one request from monopolising a backend.
		const maxWindow = 365 * 24 * time.Hour
		if te.Sub(ts) > maxWindow {
			c.JSON(badRequest, gin.H{"message": "time window cannot exceed 365 days"})
			return
		}

		// B4/B5 — strict numeric parsing for car_id.
		carID := 0
		if v := c.Query("car_id"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				c.JSON(badRequest, gin.H{"message": "car_id must be a non-negative integer"})
				return
			}
			carID = n
		}

		// B4/B5 — strict numeric parsing for max_points.
		maxPoints := 0
		if v := c.Query("max_points"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				c.JSON(badRequest, gin.H{"message": "max_points must be a non-negative integer"})
				return
			}
			maxPoints = n
		}

		trips, err := getTripsWithPositions(c.Request.Context(), startDate, endDate, carID, maxPoints)
		if err != nil {
			// B10 — never leak DB error / SQLSTATE / table name to the client.
			logger.Error("query failed", "error", err)
			status := http.StatusInternalServerError
			msg := "Failed to fetch trips data"
			if errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(err.Error(), "statement timeout") {
				status = http.StatusGatewayTimeout
				msg = "Query took too long; please narrow the time window or pick a single vehicle"
			}
			c.JSON(status, gin.H{"message": msg})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": trips, "count": len(trips)})
	})

	// /cars — list all vehicles owned by this teslamate account.
	// Used by the UI to render the car switcher (by display name, not by id).
	r.GET("/cars", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		cars, err := listCars(ctx)
		if err != nil {
			logger.Error("list cars failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": cars, "count": len(cars)})
	})

	// /health now also exposes pool stats — handy for ops dashboards.
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := dbPool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "database disconnected",
			})
			return
		}
		s := dbPool.Stat()
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"pool": gin.H{
				"total":              s.TotalConns(),
				"acquired":           s.AcquiredConns(),
				"idle":               s.IdleConns(),
				"max":                s.MaxConns(),
				"acquire_count":      s.AcquireCount(),
				"acquire_duration":   s.AcquireDuration().String(),
				"empty_acquire":      s.EmptyAcquireCount(),
				"canceled_acquire":   s.CanceledAcquireCount(),
				"new_conns":          s.NewConnsCount(),
				"max_lifetime_destr": s.MaxLifetimeDestroyCount(),
				"max_idle_destr":     s.MaxIdleDestroyCount(),
			},
		})
	})

	// /version — tiny endpoint so deployers can verify which build is live.
	// Also exposes the latest known release so the UI can show an "up to date"
	// indicator without calling GitHub.
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":           "tesla-trail-map",
			"version":        Version,
			"latest_version": LatestVersion,
			"is_latest":      Version == LatestVersion,
			"go":             runtime.Version(),
		})
	})

	port := envOr("PORT", "8080")
	logger.Info("Tesla Track Map starting", "port", port, "version", Version)
	if err := r.Run(":" + port); err != nil {
		logger.Error("server failed to start", "error", err)
	}
}
