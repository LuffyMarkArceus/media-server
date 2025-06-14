package handlers

import (
	"database/sql"
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
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	rawPath  := c.Query("path")
	if rawPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "Missing path query parameter"})
		return
	}

	// Normalize & Clean path
	relPath := filepath.FromSlash(rawPath)             // Convert Slashes
	relPath = filepath.Clean(relPath)                  // Resolve '..'

	// Security check, prevent path traversal
	if strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, string(filepath.Separator)+"..") {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "Invalid Path for security reasons"})
		return
	}

	// Check file in DB
	url := fmt.Sprintf("http://localhost:%v/media_stream?path=%s", config.AppPort, rawPath)
	var fileID int64
	var oldName, ext string
	err := db.QueryRow("SELECT id, name, type FROM files_table WHERE url = ?", url).Scan(&fileID, &oldName, &ext)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found in database"})
		} else {
			log.Printf("Error querying file %s: %v", url, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not access file"})
		}
		return
	}

	// Check file on FileSystem
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

	// Parse Request Body
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

	// Construct new path & URL
	dir := filepath.Dir(fullPath)
	newBaseName := newName + ext
	newRelPath := filepath.Join(dir, newBaseName)
	newFullPath := filepath.Join(config.MediaRoot, newRelPath)

	newURL := fmt.Sprintf("http://localhost:%c/media_stream?path=%s", config.AppPort, newRelPath)

	// Check if file exist with same name, if so fail, raise http 409 
	// In FilesSystem
	if _, err := os.Stat(newFullPath); err == nil {
		log.Printf("A file with name:%s already exists in FS", newFullPath)
		c.JSON(http.StatusConflict, gin.H{"error" : fmt.Sprintf("A file with name:%s already exists", newBaseName)})
		return
	}
	// In DB
	if _, err := db.Query("SELECT id FROM files_table WHERE url = ?", newURL); err == nil {
		log.Printf("A file with URL %s already exists in database", newURL)
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("A file with name '%s' already exists", newBaseName)})
		return
	}

	// Begin Transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update DB
	_, err = tx.Exec(
		"UPDATE files_table SET name = ?, url = ?, WHERE id = ?",
		newBaseName, newURL, fileID,
	)
	if err != nil {
		log.Printf("Error updating file %d in database: %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update file in database"})
		return
	}

	// Rename file on FileSystem
	if err := os.Rename(fullPath, newFullPath); err != nil {
		log.Printf("Error renaming file from %s to %s: %v", fullPath, newFullPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error" : fmt.Sprintf("Failed to rename file to '%s'", newBaseName)})
		return
	}

	// Commit Transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction for file : %d, error : %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	} 

	// Return Success
	c.JSON(http.StatusOK, gin.H{
		"status" : "renamed",
		"oldPath" : filepath.ToSlash(relPath),
		"newPath" : filepath.ToSlash(newRelPath),
	})
}
