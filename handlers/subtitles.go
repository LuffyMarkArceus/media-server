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

func GetSubtitles(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	// Clean the filepath parameter
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")
	relPath = filepath.Clean(relPath)
	if relPath == "" || strings.Contains(relPath, "..") {
		log.Printf("Invalid subtitle path: %s", relPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	log.Printf("Requested subtitle for path: %s", relPath)

	// Construct subtitle path in MediaRoot/subtitles, preserving folder structure
	subtitleDir := filepath.Join(config.MediaRoot, "subtitles", filepath.Dir(relPath))
	subtitleFile := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)) + ".vtt"
	subtitlePath := filepath.Join(subtitleDir, subtitleFile)
	log.Printf("Looking for subtitle at: %s", subtitlePath)

	// Check if subtitle file exists
	if _, err := os.Stat(subtitlePath); err == nil {
		log.Printf("Serving existing subtitle: %s", subtitlePath)
		c.File(subtitlePath)
		return
	}

	// Verify video exists in database
	url := fmt.Sprintf("http://localhost:%v/media_stream?path=%s", config.AppPort, relPath)
	var fileID int64
	err := db.QueryRow("SELECT id FROM files_table WHERE url = ?", url).Scan(&fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Video not found in database for URL: %s", url)
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found in database"})
			return
		}
		log.Printf("Error querying file %s: %v", url, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not access file"})
		return
	}

	// Check if video file exists
	videoPath := filepath.Join(config.MediaRoot, relPath)
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		log.Printf("Source video file not found: %s", videoPath)
		c.JSON(http.StatusNotFound, gin.H{"error": "Source video file not found"})
		return
	}

	// Create subtitles directory if needed
	if err := os.MkdirAll(subtitleDir, os.ModePerm); err != nil {
		log.Printf("Error creating subtitles directory %s: %v", subtitleDir, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subtitles directory"})
		return
	}

	// Attempt to extract subtitles using FFmpeg
	log.Printf("Generating subtitle for: %s to %s", videoPath, subtitlePath)
	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,
		"-map", "0:s:0?",
		"-f", "webvtt",
		subtitlePath,
	)
	cmd.Stderr = os.Stderr // Capture FFmpeg errors
	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error generating subtitle for %s: %v", videoPath, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "No subtitles found in video or failed to extract"})
		return
	}

	// Verify subtitle was created
	if _, err := os.Stat(subtitlePath); os.IsNotExist(err) {
		log.Printf("Subtitle file not created: %s", subtitlePath)
		c.JSON(http.StatusNotFound, gin.H{"error": "Failed to generate subtitle"})
		return
	}

	log.Printf("Serving generated subtitle: %s", subtitlePath)
	c.File(subtitlePath)
}