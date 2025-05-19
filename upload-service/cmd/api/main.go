package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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
	// Initialize S3 session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	}))
	s3Uploader = s3manager.NewUploader(sess)

	// Initialize PostgreSQL
	var err error
	db, err = sql.Open("postgres", fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
		log.Fatal(err)
	}

	// Set up routes with Gin
	router := gin.Default()
	router.POST("/upload", uploadHandler)
	router.GET("/videos", listVideosHandler)
	router.GET("/video/:id", getVideoHandler)

	log.Println("Server starting on :3000")
	log.Fatal(router.Run(":3000"))
}

func uploadHandler(c *gin.Context) {
	// Get file from form
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	// Upload to S3
	result, err := s3Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(fmt.Sprintf("%d-%s", time.Now().UnixNano(), header.Filename)),
		Body:   file,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Store metadata
	_, err = db.Exec(
		"INSERT INTO videos (name, url, timestamp) VALUES ($1, $2, $3)",
		header.Filename,
		result.Location,
		time.Now().Unix(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func listVideosHandler(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, url, timestamp FROM videos")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		var v Video
		err := rows.Scan(&v.ID, &v.Name, &v.URL, &v.Timestamp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		videos = append(videos, v)
	}

	c.JSON(http.StatusOK, videos)
}

func getVideoHandler(c *gin.Context) {
	id := c.Param("id")
	var v Video

	err := db.QueryRow(
		"SELECT id, name, url, timestamp FROM videos WHERE id = $1",
		id,
	).Scan(&v.ID, &v.Name, &v.URL, &v.Timestamp)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	c.JSON(http.StatusOK, v)
}
