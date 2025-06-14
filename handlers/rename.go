package handlers

import (
	"fmt"
	"log"
	"media-server/config"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// RenameRequest defines the expected JSON body for the rename request
type RenameRequest struct {
	NewName string `json:"newName" binding:"required"`
}

func RenameFile(c *gin.Context) {
	rawPath  := c.Query("path")
	if rawPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "Missing path query parameter"})
		return
	}

	relPath := filepath.FromSlash(rawPath)             // Convert Slashes
	relPath = filepath.Clean(relPath)                  // Resolve '..'

	if strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, string(filepath.Separator)+"..") {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "Invalid Path for security reasons"})
		return
	}

	fullPath := filepath.Join(config.MediaRoot, relPath)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File Not found"})
		} else {
			log.Printf("Stat error for %s: %v", fullPath, err)
			c.JSON(http.StatusInternalServerError, gin.H{"erro" : "Could not access file"})
		}
		return
	}

	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is a directory"})
		return
	}

	var req RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error" : fmt.Sprintf("Invalid request body: %v", err)})
		return
	}
	newName := strings.TrimSpace(req.NewName)
	if newName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "New Name cannot be empty string"})
		return
	}
	if strings.ContainsAny(newName, "/\\:") {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "New Name contains invalid charecters (/, \\, :)"})
		return
	}

	dir := filepath.Dir(fullPath)
	oldName := strings.Split(filepath.Base(fullPath), ".")[0]
	ext := filepath.Ext(fullPath)
	newBaseName := newName + ext
	newFullPath := filepath.Join(dir, newBaseName)

	// Check if file exist with same name, if so fail, raise http 409
	if _, err := os.Stat(newFullPath); err == nil {
		log.Printf("A file with name:%s already exists, OLDNAME : %s", newFullPath, oldName)
		c.JSON(http.StatusConflict, gin.H{"error" : fmt.Sprintf("A file with name:%s already exists", newName)})
		return
	}
	if err:= os.Rename(fullPath, newFullPath); err != nil {
		log.Printf("Error renaming file from %s to %s: %v", fullPath, newFullPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error" : fmt.Sprintf("Failed to rename '%s' to '%s'", oldName, newName)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status" : "renamed",
		"oldPath" : filepath.ToSlash(relPath),
		"newPath" : filepath.ToSlash(filepath.Join(filepath.Dir(relPath), newBaseName)),
	})
}
