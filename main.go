package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	host     = os.Getenv("DATABASE_HOST")
	port, _  = strconv.Atoi(os.Getenv("DATABASE_PORT"))
	user     = os.Getenv("DATABASE_USER")
	password = os.Getenv("DATABASE_PASS")
	dbname   = os.Getenv("DATABASE_NAME")
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

var dbPool *pgxpool.Pool

func init() {
	if host == "" {
		host = "192.168.10.200"
	}
	if port == 0 {
		port = 54320
	}
	if user == "" {
		user = "teslamate"
	}
	if password == "" {
		password = "secret"
	}
	if dbname == "" {
		dbname = "teslamate"
	}
}

func initDB() error {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	dbPool, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 测试连接
	if err := dbPool.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func getTripsWithPositions(startDate, endDate string) ([]DriveWithPositions, error) {
	// 调试输出
	fmt.Printf("执行查询 - 开始时间: %s, 结束时间: %s\n", startDate, endDate)

	query := `
	SELECT d.id as drive_id, p.latitude, p.longitude, p.date
	FROM drives d
	JOIN positions p ON d.id = p.drive_id
	WHERE d.start_date >= $1 AND d.end_date <= $2
	ORDER BY d.start_date ASC, p.date ASC;
	`

	// 先测试是否有数据
	countQuery := `
	SELECT COUNT(*) as drive_count, 
	       (SELECT COUNT(*) FROM positions p JOIN drives d ON d.id = p.drive_id 
	        WHERE d.start_date >= $1 AND d.end_date <= $2) as position_count
	FROM drives d
	WHERE d.start_date >= $1 AND d.end_date <= $2;
	`

	var driveCount, positionCount int
	err := dbPool.QueryRow(context.Background(), countQuery, startDate, endDate).Scan(&driveCount, &positionCount)
	if err != nil {
		fmt.Printf("统计查询失败: %v\n", err)
	} else {
		fmt.Printf("数据库中符合条件的行程数: %d, 位置点数: %d\n", driveCount, positionCount)
	}

	rows, err := dbPool.Query(context.Background(), query, startDate, endDate)
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

		// 先扫描到正确的类型
		if err := rows.Scan(&driveID, &latitude, &longitude, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// 创建Position结构体，将时间转换为字符串
		pos := Position{
			Latitude:  latitude,
			Longitude: longitude,
			Date:      timestamp.Format(time.RFC3339), // 转换为ISO8601格式
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

	fmt.Printf("实际返回的行程数: %d, 位置点总数: %d\n", len(drives), actualPositionCount)
	return drives, nil
}

func main() {
	// 初始化数据库连接池
	if err := initDB(); err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	r := gin.Default()

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/trips", func(c *gin.Context) {
		startDate := c.DefaultQuery("start_date", "2000-01-01")
		endDate := c.DefaultQuery("end_date", "3000-01-01")

		// 日志记录查询参数
		fmt.Printf("查询参数 - 开始时间: %s, 结束时间: %s\n", startDate, endDate)

		trips, err := getTripsWithPositions(startDate, endDate)
		if err != nil {
			fmt.Printf("查询失败: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   err.Error(),
				"message": "Failed to fetch trips data",
			})
			return
		}

		fmt.Printf("查询到 %d 个行程\n", len(trips))
		c.JSON(http.StatusOK, gin.H{
			"data":  trips,
			"count": len(trips),
		})
	})

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "map.html", nil)
	})

	r.LoadHTMLFiles("map.html")
	r.GET("/map", func(c *gin.Context) {
		c.HTML(http.StatusOK, "map.html", nil)
	})

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		if err := dbPool.Ping(context.Background()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unhealthy",
				"database": "disconnected",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

	fmt.Println("Tesla Track Map 2.0 starting on :8080")
	r.Run(":8080")
}
