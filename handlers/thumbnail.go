package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"media-server/config"

	"github.com/gin-gonic/gin"
)
func GetThumbnail(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	// Clean the filepath parameter
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")
	relPath = filepath.ToSlash(filepath.Clean(relPath))

	if strings.Contains(relPath, "..") {
		log.Printf("Invalid thumbnail path: %s", relPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	log.Printf("Requested thumbnail for path: %s", relPath)

	// Construct thumbnail path in MediaRoot/thumbnails, preserving folder structure
	thumbnailDir := filepath.Join(config.MediaRoot, "thumbnails")
	thumbnailPath := filepath.Join(thumbnailDir, relPath)
	thumbnailFile := filepath.Join(filepath.Dir(thumbnailPath), strings.TrimSuffix(filepath.Base(thumbnailPath), filepath.Ext(thumbnailPath))+".jpg")

	// Full path to the original video
	videoPath := filepath.Join(config.MediaRoot, relPath)
	log.Printf("Looking for thumbnail at: %s", thumbnailFile)

	// Check if thumbnail file exists
	if info, err := os.Stat(thumbnailFile); err == nil && !info.IsDir() {
		log.Printf("Serving existing thumbnail: %s", thumbnailFile)
		c.File(thumbnailFile)
		return
	}

	// Verify video exists in DB
	url := fmt.Sprintf("http://localhost:%v/media_stream?path=%s", config.AppPort, relPath)
	var fileID int64
	err := db.QueryRow("SELECT id FROM files_table WHERE url = ?", url).Scan(&fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Video not found in database for URL: %s", url)
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found in database"})
		} else {
			log.Printf("Error querying file %s: %v", url, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not access file"})
		}
		return
	}

	// Create thumbnails directory if needed
	if err := os.MkdirAll(filepath.Dir(thumbnailFile), os.ModePerm); err != nil {
		log.Printf("Error creating thumbnails directory %s: %v", filepath.Dir(thumbnailFile), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create thumbnails directory"})
		return
	}

	// Check if video file exists before trying to generate thumbnail
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		log.Printf("Source video file not found: %s", videoPath)
		c.JSON(http.StatusNotFound, gin.H{"error": "Source video file not found"})
		return
	}

	// Generate thumbnail using ffmpeg
	log.Printf("Generating thumbnail for: %s to %s", videoPath, thumbnailFile)
	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,
		"-ss", "00:00:01",
		"-vframes", "1",
		"-vf", "scale=320:-1",
		thumbnailFile,
	)
	cmd.Stderr = os.Stderr // Capture FFmpeg errors
	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error generating thumbnail for %s: %v", videoPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create thumbnail for %s", relPath)})
		return
	}

	// Verify thumbnail was created
	if _, err := os.Stat(thumbnailFile); os.IsNotExist(err) {
		log.Printf("Thumbnail file not created: %s", thumbnailFile)
		c.JSON(http.StatusNotFound, gin.H{"error": "Failed to generate thumbnail"})
		return
	}

	log.Printf("Serving generated thumbnail: %s", thumbnailFile)
	c.File(thumbnailFile)
}