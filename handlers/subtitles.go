package handlers

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetSubtitles(c *gin.Context) {
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")

	if strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	subtitlePath := filepath.Join("./media/subtitles", relPath)
	subtitlePath = strings.TrimSuffix(subtitlePath, filepath.Ext(subtitlePath)) + ".vtt"
	videoPath := filepath.Join("./media", relPath)

	if info, err := os.Stat(subtitlePath); err == nil && !info.IsDir() {
		c.File(subtitlePath)
		return
	}

	err := os.MkdirAll(filepath.Dir(subtitlePath), os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subtitles directory"})
		return
	}

	err = exec.Command("ffmpeg", "-i", videoPath, "-map", "0:s:0", subtitlePath).Run()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create subtitle for %s", relPath)})
		return
	}

	c.File(subtitlePath)
}