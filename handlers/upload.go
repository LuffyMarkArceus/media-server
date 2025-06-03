package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func UploadFiles(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Multipart Form"})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files to upload"})
		return
	}

	uploadDir := "./media"
	os.MkdirAll(uploadDir, os.ModePerm)

	var uploaded []string
	for _, file := range files {
		dst := filepath.Join(uploadDir, file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to upload %s", file.Filename),
			})
			return
		}
		uploaded = append(uploaded, file.Filename)
	}
	c.JSON(http.StatusOK, gin.H{"uploaded": uploaded})
}
