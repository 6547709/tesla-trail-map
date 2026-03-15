package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool *pgxpool.Pool
	logger *slog.Logger
)

type Position struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Date      string  `json:"date"`
}

type DriveWithPositions struct {
	DriveID   int        `json:"drive_id"`
	Positions []Position `json:"positions"`
}

func init() {
	// Initialize logger
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func getDBConfig() string {
	host := os.Getenv("DATABASE_HOST")
	if host == "" {
		host = "192.168.10.200"
	}

	portStr := os.Getenv("DATABASE_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil || port == 0 {
		port = 54320
	}

	user := os.Getenv("DATABASE_USER")
	if user == "" {
		user = "teslamate"
	}

	password := os.Getenv("DATABASE_PASS")
	if password == "" {
		password = "secret"
	}

	dbname := os.Getenv("DATABASE_NAME")
	if dbname == "" {
		dbname = "teslamate"
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

func initDB() error {
	connString := getDBConfig()

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Optimize pool settings
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	dbPool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func getTripsWithPositions(ctx context.Context, startDate, endDate string) ([]DriveWithPositions, error) {
	logger.Info("fetching trips", "start_date", startDate, "end_date", endDate)

	query := `
	SELECT d.id as drive_id, p.latitude, p.longitude, p.date
	FROM drives d
	JOIN positions p ON d.id = p.drive_id
	WHERE d.start_date >= $1 AND d.end_date <= $2
	ORDER BY d.start_date ASC, p.date ASC;
	`

	rows, err := dbPool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	driveMap := make(map[int]*DriveWithPositions)
	actualPositionCount := 0

	for rows.Next() {
		var driveID int
		var latitude, longitude float64
		var timestamp time.Time

		if err := rows.Scan(&driveID, &latitude, &longitude, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		pos := Position{
			Latitude:  latitude,
			Longitude: longitude,
			Date:      timestamp.Format(time.RFC3339),
		}

		if _, exists := driveMap[driveID]; !exists {
			driveMap[driveID] = &DriveWithPositions{DriveID: driveID}
		}
		driveMap[driveID].Positions = append(driveMap[driveID].Positions, pos)
		actualPositionCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	var drives []DriveWithPositions
	for _, drive := range driveMap {
		drives = append(drives, *drive)
	}

	logger.Info("trips fetched", "drive_count", len(drives), "position_count", actualPositionCount)
	return drives, nil
}

func main() {
	if err := initDB(); err != nil {
		logger.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	r := gin.New()
	r.Use(gin.Recovery())

	// Simple CORS middleware
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
		startDate := c.DefaultQuery("start_date", "2000-01-01")
		endDate := c.DefaultQuery("end_date", "3000-01-01")

		trips, err := getTripsWithPositions(c.Request.Context(), startDate, endDate)
		if err != nil {
			logger.Error("query failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   err.Error(),
				"message": "Failed to fetch trips data",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  trips,
			"count": len(trips),
		})
	})

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
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Tesla Track Map starting", "port", port)
	if err := r.Run(":" + port); err != nil {
		logger.Error("server failed to start", "error", err)
	}
}
