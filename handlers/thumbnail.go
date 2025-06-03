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

func GetThumbnail(c *gin.Context) {
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")

	if strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	thumbPath := filepath.Join("./media/thumbnails", relPath)
	thumbPath = strings.TrimSuffix(thumbPath, filepath.Ext(thumbPath)) + ".jpg"
	videoPath := filepath.Join("./media", relPath)

	if info, err := os.Stat(thumbPath); err == nil && !info.IsDir() {
		c.File(thumbPath)
		return
	}

	err := os.MkdirAll(filepath.Dir(thumbPath), os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create thumbnails directory"})
		return
	}

	err = exec.Command("ffmpeg", "-i", videoPath, "-ss", "00:00:01", "-vframes", "1", thumbPath).Run()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create thumbnail for %s", relPath)})
		return
	}

	c.File(thumbPath)
}
