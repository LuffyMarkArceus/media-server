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

func GetThumbnail(c *gin.Context) {
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

	log.Printf("Requested thumbnail for: %s", relPath)

	thumbnailName := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)) + ".jpg"
	thumbnailKey := filepath.ToSlash(filepath.Join("thumbnails", filepath.Dir(relPath), thumbnailName))

	// Check if thumbnail exists
	_, err := r2Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(config.CloudflareR2BucketName),
		Key:    aws.String(thumbnailKey),
	})
	if err == nil {
		c.Redirect(http.StatusFound, "/proxy_thumbnail/"+relPath)
		return
	}

	// Try finding matching video with known extensions
	base := strings.TrimSuffix(relPath, ".jpg")
	extensions := []string{".mp4", ".mkv", ".avi", ".mov", ".webm"}
	var videoRelPath string
	var found bool

	for _, ext := range extensions {
		candidate := base + ext
		candidateURL := fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, candidate)
		var fileID int64
		err := db.QueryRowContext(context.TODO(), "SELECT id FROM files_table WHERE url = $1", candidateURL).Scan(&fileID)
		if err == nil {
			videoRelPath = candidate
			found = true
			break
		} else if err != sql.ErrNoRows {
			log.Printf("DB error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
			return
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	// Presign download URL for video
	presignClient := s3.NewPresignClient(r2Client)
	presigned, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(config.CloudflareR2BucketName),
		Key:    aws.String(videoRelPath),
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		log.Printf("Presign error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Presign failed"})
		return
	}

	var thumbnailBuf bytes.Buffer
	var stderr bytes.Buffer

	// -ss after -i allows accurate seeking with streaming URLs
	cmd := exec.Command("ffmpeg",
		"-i", presigned.URL,
		"-ss", "5",
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-f", "image2pipe",
		"pipe:1",
	)
	cmd.Stdout = &thumbnailBuf
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error: %v, stderr: %s", err, stderr.String())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Thumbnail generation failed"})
		return
	}

	// Upload thumbnail to R2
	uploader := manager.NewUploader(r2Client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(config.CloudflareR2BucketName),
		Key:         aws.String(thumbnailKey),
		Body:        bytes.NewReader(thumbnailBuf.Bytes()),
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		log.Printf("Upload error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, thumbnailKey))
}

func ProxyThumbnail(c *gin.Context) {
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")
	relPath = filepath.ToSlash(filepath.Clean(relPath))

	key := filepath.ToSlash(filepath.Join("thumbnails", relPath))
	log.Printf("Proxying thumbnail from R2: %s", key)

	resp, err := r2Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(config.CloudflareR2BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Failed to fetch thumbnail from R2: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "thumbnail not found"})
		return
	}
	defer resp.Body.Close()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}