package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func ListMedia(c *gin.Context) {
	mediaDir := "./media"
	var fileList []gin.H

	err := filepath.WalkDir(mediaDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(mediaDir, path)
		if err != nil {
			relPath = path
		}

		fileList = append(fileList, gin.H{
			"name": d.Name(),
			"size": info.Size(),
			"path": relPath,
			"type": filepath.Ext(d.Name()),
		})
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan media directory"})
		return
	}

	c.JSON(http.StatusOK, fileList)
}

func ServeMedia(c *gin.Context) {
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")

	if strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	fullPath := filepath.Join("./media", relPath)
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	c.File(fullPath)
}
