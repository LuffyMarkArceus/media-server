package handlers

import (
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

	if info, err := os.Stat(thumbPath); err == nil && !info.IsDir() {
		c.File(thumbPath)
		return
	}

	err := os.MkdirAll(filepath.Dir(thumbPath), os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create thumbnails directory"})
		return
	}
	// Check if video file exists before trying to generate thumbnail
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source video file not found"})
		return
	}

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
