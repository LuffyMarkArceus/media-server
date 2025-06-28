package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"media-server/config"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

// DB stores full URLs with domain in the `url` column
const dbHasFullURLs = true

func GetSubtitles(c *gin.Context) {
	if db == nil || r2Client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not initialized"})
		return
	}

	relPath := strings.TrimPrefix(c.Param("filepath"), "/")
	relPath = filepath.ToSlash(filepath.Clean(relPath))
	if strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	log.Printf("Requested subtitle for: %s", relPath)

	// Base video path without .vtt suffix
	videoRelPath := strings.TrimSuffix(relPath, ".vtt")

	// List of supported video file extensions to try
	extensions := []string{".mkv", ".mp4", ".avi", ".mov", ".webm"}

	var fileID int64
	// var videoURL string
	found := false

	for _, ext := range extensions {
		candidatePath := videoRelPath + ext
		candidateURL := fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, candidatePath)
		err := db.QueryRow("SELECT id FROM files_table WHERE url = $1", candidateURL).Scan(&fileID)
		if err == nil {
			// videoURL = candidateURL
			videoRelPath = candidatePath
			found = true
			break
		} else if err != sql.ErrNoRows {
			log.Printf("DB error checking file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
			return
		}
	}

	if !found {
		log.Printf("Video not found in DB for any suffix: %s", videoRelPath)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	subtitleKey := filepath.ToSlash(filepath.Join("subtitles", relPath))

	// Check if subtitle already exists on R2
	_, err := r2Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(config.CloudflareR2BucketName),
		Key:    aws.String(subtitleKey),
	})
	if err == nil {
		log.Printf("Subtitle already exists at R2: %s", subtitleKey)
		c.Redirect(http.StatusFound, "/proxy_subtitle/"+relPath)
		return
	}

	// Generate presigned URL for the video file using the discovered videoRelPath
	presignClient := s3.NewPresignClient(r2Client)
	presigned, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(config.CloudflareR2BucketName),
		Key:    aws.String(videoRelPath),
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		log.Printf("Failed to generate presigned URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access video file"})
		return
	}

	var subtitleBuf bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("ffmpeg",
		"-i", presigned.URL,
		"-map", "0:s:0",
		"-f", "webvtt",
		"pipe:1",
	)
	cmd.Stdout = &subtitleBuf
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error: %v, stderr: %s", err, stderr.String())
		c.JSON(http.StatusNotFound, gin.H{"error": "No subtitles found or extraction failed"})
		return
	}

	uploader := manager.NewUploader(r2Client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(config.CloudflareR2BucketName),
		Key:         aws.String(subtitleKey),
		Body:        bytes.NewReader(subtitleBuf.Bytes()),
		ContentType: aws.String("text/vtt"),
	})
	if err != nil {
		log.Printf("Error uploading subtitle to R2: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload subtitle"})
		return
	}

	log.Printf("Subtitle uploaded to R2: %s", subtitleKey)
	c.Redirect(http.StatusFound, "/proxy_subtitle/"+relPath)
}

func ProxySubtitle(c *gin.Context) {
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")
	relPath = filepath.ToSlash(filepath.Clean(relPath))

	key := filepath.ToSlash(filepath.Join("subtitles", relPath))
	log.Printf("Proxying subtitle from R2: %s", key)

	resp, err := r2Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(config.CloudflareR2BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Failed to fetch subtitle from R2: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Subtitle not found"})
		return
	}
	defer resp.Body.Close()

	c.Header("Content-Type", "text/vtt")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}
