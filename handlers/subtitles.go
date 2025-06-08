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

func GetSubtitles(c *gin.Context) {
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")
	relPath = filepath.Clean(relPath)

	if strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	// Construct subtitle path in a dedicated subtitles folder within the media root
	subtitleDir := filepath.Join(config.MediaRoot, "subtitles")
	// The subtitle filename should correspond to the video filename
	subtitlePath := filepath.Join(subtitleDir, strings.TrimSuffix(relPath, filepath.Ext(relPath))+".vtt")

	// Full path to the original video
	videoPath := filepath.Join(config.MediaRoot, relPath)

	// Check if subtitle already exists
	if info, err := os.Stat(subtitlePath); err == nil && !info.IsDir() {
		c.File(subtitlePath)
		return
	}

	// Create subtitles directory if needed
	err := os.MkdirAll(filepath.Dir(subtitlePath), os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subtitles directory"})
		return
	}

	// Check if video file exists before trying to extract subtitles
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source video file not found for subtitle extraction"})
		return
	}

	// Generate subtitle using ffmpeg (assuming the first subtitle track)
	// You might need more sophisticated logic here if there are multiple subtitle tracks
	// or if the video contains no subtitles.
	log.Printf("Generating subtitle for: %s to %s", videoPath, subtitlePath)
	cmd := exec.Command(
		"ffmpeg", 
		"-i", videoPath, 
		"-map", "0:s:0", 
		subtitlePath,
	)

	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error generating subtitle for %s: %v", videoPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create subtitle for %s", relPath)})
		return
	}

	c.File(subtitlePath)
}