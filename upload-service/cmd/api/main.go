package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"upload-service/pkg/logger"
	"upload-service/pkg/middleware"
)

var (
	s3Uploader *s3manager.Uploader
	db         *sql.DB
)

type Video struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Timestamp int64  `json:"timestamp"`
}

func main() {
	// Initialize structured logger
	logger.Initialize()

	// Initialize S3 session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	}))
	s3Uploader = s3manager.NewUploader(sess)
	logger.Info("S3 uploader initialized", map[string]interface{}{
		"region": os.Getenv("AWS_REGION"),
		"bucket": os.Getenv("S3_BUCKET"),
	})

	// Initialize PostgreSQL
	var err error
	db, err = sql.Open("postgres", fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	))
	db.SetMaxOpenConns(90)                 // Maximum number of open connections to the database
	db.SetMaxIdleConns(10)                 // Maximum number of idle connections in the pool
	db.SetConnMaxLifetime(time.Minute * 5) // How long a connection can be reused

	if err != nil {
		logger.Fatal("Failed to connect to database", err, map[string]interface{}{
			"host": os.Getenv("DB_HOST"),
			"db":   os.Getenv("DB_NAME"),
		})
	}
	defer db.Close()

	// Check database connection
	if err = db.Ping(); err != nil {
		logger.Fatal("Database ping failed", err, nil)
	}

	logger.Info("Connected to database", map[string]interface{}{
		"host": os.Getenv("DB_HOST"),
		"db":   os.Getenv("DB_NAME"),
	})

	// Create table if not exists
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS videos (
            id SERIAL PRIMARY KEY,
            name TEXT,
            url TEXT,
            timestamp BIGINT
        )
    `)
	if err != nil {
		logger.Fatal("Failed to create videos table", err, nil)
	}
	logger.Info("Videos table initialized", nil)

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") != "" {
		gin.SetMode(os.Getenv("GIN_MODE"))
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Set up routes with Gin
	router := gin.New()
	router.Use(middleware.Logger(), gin.Recovery())

	router.POST("/upload", uploadHandler)
	router.GET("/videos", listVideosHandler)
	router.GET("/video/:id", getVideoHandler)

	// Add health and crash endpoints for Kubernetes testing
	router.GET("/health", healthCheckHandler)
	router.GET("/crash", crashHandler)

	// Start the server
	port := "8080"
	logger.Info("Server starting", map[string]interface{}{
		"port": port,
	})
	logger.Fatal("Server shutdown unexpectedly", router.Run(":"+port), nil)
}

// Health check endpoint for Kubernetes liveness probe
func healthCheckHandler(c *gin.Context) {
	log := middleware.GetLogger(c)
	log.Info().Msg("Health check requested")

	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// Deliberately crash the application to test Kubernetes auto-healing
func crashHandler(c *gin.Context) {
	log := middleware.GetLogger(c)
	correlationID := middleware.GetCorrelationID(c)

	log.Warn().Msg("Crash endpoint called - application will terminate in 2 seconds")

	// Send response before crashing
	c.JSON(http.StatusOK, gin.H{
		"message":        "Application will crash in 2 seconds to simulate failure",
		"pod":            os.Getenv("HOSTNAME"),
		"correlation_id": correlationID,
	})

	// Wait a bit so the HTTP response has time to complete
	go func() {
		time.Sleep(2 * time.Second)
		logger.Fatal("Application crash triggered by /crash endpoint", nil, map[string]interface{}{
			"correlation_id": correlationID,
		})
	}()
}

func uploadHandler(c *gin.Context) {
	log := middleware.GetLogger(c)
	correlationID := middleware.GetCorrelationID(c)

	// Get file from form
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		log.Error().Err(err).
			Str("correlation_id", correlationID).
			Str("component", "upload_handler").
			Msg("Failed to get file from form")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	log.Info().
		Str("correlation_id", correlationID).
		Str("component", "upload_handler").
		Str("filename", header.Filename).
		Int64("size", header.Size).
		Msg("Video upload started")

	// Upload to S3
	s3Key := fmt.Sprintf("%d-%s", time.Now().UnixNano(), header.Filename)
	result, err := s3Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(s3Key),
		Body:   file,
	})
	if err != nil {
		log.Error().Err(err).
			Str("correlation_id", correlationID).
			Str("component", "upload_handler").
			Str("filename", header.Filename).
			Msg("S3 upload failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Str("correlation_id", correlationID).
		Str("component", "upload_handler").
		Str("filename", header.Filename).
		Str("s3_key", s3Key).
		Str("url", result.Location).
		Msg("Video uploaded to S3")

	// Store metadata
	timestamp := time.Now().Unix()
	_, err = db.Exec(
		"INSERT INTO videos (name, url, timestamp) VALUES ($1, $2, $3)",
		header.Filename,
		result.Location,
		timestamp,
	)
	if err != nil {
		log.Error().Err(err).
			Str("correlation_id", correlationID).
			Str("component", "upload_handler").
			Str("filename", header.Filename).
			Msg("Failed to store video metadata")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Str("correlation_id", correlationID).
		Str("component", "upload_handler").
		Str("filename", header.Filename).
		Int64("timestamp", timestamp).
		Msg("Video metadata stored in database")
	c.Status(http.StatusCreated)
}

func listVideosHandler(c *gin.Context) {
	log := middleware.GetLogger(c)
	log.Info().Msg("Listing all videos")

	rows, err := db.Query("SELECT id, name, url, timestamp FROM videos")
	if err != nil {
		log.Error().Err(err).Msg("Failed to query videos")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		var v Video
		err := rows.Scan(&v.ID, &v.Name, &v.URL, &v.Timestamp)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan video row")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		videos = append(videos, v)
	}

	log.Info().Int("count", len(videos)).Msg("Videos retrieved successfully")
	c.JSON(http.StatusOK, videos)
}

func getVideoHandler(c *gin.Context) {
	log := middleware.GetLogger(c)
	id := c.Param("id")

	log.Info().Str("video_id", id).Msg("Getting video by ID")

	var v Video

	err := db.QueryRow(
		"SELECT id, name, url, timestamp FROM videos WHERE id = $1",
		id,
	).Scan(&v.ID, &v.Name, &v.URL, &v.Timestamp)

	if err != nil {
		log.Error().Err(err).Str("video_id", id).Msg("Video not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	// Parse the S3 URL to extract the key
	u, err := url.Parse(v.URL)
	if err != nil {
		log.Error().Err(err).Str("video_id", id).Str("url", v.URL).Msg("Failed to parse S3 URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate video URL"})
		return
	}

	// Extract the key from the path
	key := strings.TrimPrefix(u.Path, "/")

	// Create a request for S3 GetObject
	req, _ := s3Uploader.S3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(key),
	})

	// Generate a presigned URL that expires in 1 hour
	signedURL, err := req.Presign(1 * time.Hour)
	if err != nil {
		log.Error().Err(err).Str("video_id", id).Msg("Failed to generate presigned URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate video URL"})
		return
	}

	// Update the URL in the response with the presigned URL
	v.URL = signedURL

	log.Info().Str("video_id", id).Str("name", v.Name).Msg("Video retrieved successfully with presigned URL")
	c.JSON(http.StatusOK, v)
}
