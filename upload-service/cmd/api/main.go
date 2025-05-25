package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"upload-service/pkg/logger"
	"upload-service/pkg/middleware"
)

var (
	minioClient *minio.Client
	db          *sql.DB
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

	// Initialize MinIO client
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	var err error
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.Fatal("Failed to initialize MinIO client", err, map[string]interface{}{
			"endpoint": endpoint,
		})
	}

	// Create bucket if it doesn't exist
	bucketName := os.Getenv("MINIO_BUCKET")
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		logger.Fatal("Failed to check if bucket exists", err, map[string]interface{}{
			"bucket": bucketName,
		})
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			logger.Fatal("Failed to create bucket", err, map[string]interface{}{
				"bucket": bucketName,
			})
		}
		logger.Info("Bucket created successfully", map[string]interface{}{
			"bucket": bucketName,
		})
	}

	logger.Info("MinIO client initialized", map[string]interface{}{
		"endpoint": endpoint,
		"bucket":   bucketName,
		"ssl":      useSSL,
	})

	// Initialize PostgreSQL
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

	// Upload to MinIO
	bucketName := os.Getenv("MINIO_BUCKET")
	objectName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), header.Filename)
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx := context.Background()
	info, err := minioClient.PutObject(ctx, bucketName, objectName, file, header.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		log.Error().Err(err).
			Str("correlation_id", correlationID).
			Str("component", "upload_handler").
			Str("filename", header.Filename).
			Msg("MinIO upload failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Construct the MinIO URL
	minioURL := fmt.Sprintf("minio://%s/%s", bucketName, objectName) // Store as reference

	log.Info().
		Str("correlation_id", correlationID).
		Str("component", "upload_handler").
		Str("filename", header.Filename).
		Str("object_name", objectName).
		Str("url", minioURL).
		Int64("size", info.Size).
		Msg("Video uploaded to MinIO")

	// Store metadata
	timestamp := time.Now().Unix()
	_, err = db.Exec(
		"INSERT INTO videos (name, url, timestamp) VALUES ($1, $2, $3)",
		header.Filename,
		minioURL,
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

		// Generate presigned URL for external access
		if strings.HasPrefix(v.URL, "minio://") {
			objectName := strings.TrimPrefix(v.URL, fmt.Sprintf("minio://%s/", os.Getenv("MINIO_BUCKET")))
			presignedURL, err := minioClient.PresignedGetObject(context.Background(), os.Getenv("MINIO_BUCKET"), objectName, time.Hour, nil)
			if err == nil {
				v.URL = presignedURL.String()
			}
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

	// Parse the MinIO URL to extract the object name
	u, err := url.Parse(v.URL)
	if err != nil {
		log.Error().Err(err).Str("video_id", id).Str("url", v.URL).Msg("Failed to parse MinIO URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate video URL"})
		return
	}

	// Extract the object name from the path (remove bucket name)
	pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(pathParts) < 2 {
		log.Error().Str("video_id", id).Str("url", v.URL).Msg("Invalid MinIO URL format")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid video URL"})
		return
	}
	objectName := strings.Join(pathParts[1:], "/")

	// Generate a presigned URL that expires in 1 hour
	ctx := context.Background()
	presignedURL, err := minioClient.PresignedGetObject(ctx, os.Getenv("MINIO_BUCKET"), objectName, time.Hour, nil)
	if err != nil {
		log.Error().Err(err).Str("video_id", id).Msg("Failed to generate presigned URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate video URL"})
		return
	}

	// Update the URL in the response with the presigned URL
	v.URL = presignedURL.String()

	log.Info().Str("video_id", id).Str("name", v.Name).Msg("Video retrieved successfully with presigned URL")
	c.JSON(http.StatusOK, v)
}
