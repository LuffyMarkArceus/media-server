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

	relPath := strings.TrimPrefix(c.Param("filepath"), "/")
	relPath = filepath.Clean(relPath)

	if strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	// Construct thumbnail path in a dedicated thumbnails folder within the media root
	thumbnailDir := filepath.Join(config.MediaRoot, "thumbnails")
	thumbPath := filepath.Join(thumbnailDir, strings.TrimSuffix(relPath, filepath.Ext(relPath))+".jpg")

	// Full path to the original video
	videoPath := filepath.Join(config.MediaRoot, relPath)

	// Check if thumbnail exists
	if info, err := os.Stat(thumbPath); err == nil && !info.IsDir() {
		c.File(thumbPath)
		return
	}

	// Verify video in DB
	url := fmt.Sprintf("http://localhost:%v/media_stream?path=%s", config.AppPort, relPath)
	var fileID int64
	err := db.QueryRow("SELECT id FROM files_table WHERE url = ?", url).Scan(&fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found in database"})
		} else {
			log.Printf("Error querying file %s: %v", url, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not access file"})
		}
		return
	}

	// create thumbnails directory
	err = os.MkdirAll(filepath.Dir(thumbPath), os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create thumbnails directory"})
		return
	}

	// Check if video file exists before trying to generate thumbnail
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source video file not found"})
		return
	}

	// Generate thumbnail using ffmpeg
	cmd := exec.Command(
		"ffmpeg", 
		"-i", videoPath, 
		"-ss", "00:00:01", 
		"-vframes", "1", 
		"-vf", "scale=320:-1", 
		thumbPath,
	)
	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error generating thumbnail for %s: %v", videoPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create thumbnail for %s", relPath)})
		return
	}

	c.File(thumbPath)
}
