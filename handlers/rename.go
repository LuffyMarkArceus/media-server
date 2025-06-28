// handlers/rename.go (R2 version)
package handlers

import (
	"database/sql"
	"log"
	"media-server/config"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type RenameRequest struct {
	NewName string `json:"newName" binding:"required"`
}

func RenameFile(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	relPath := filepath.ToSlash(filepath.Clean(c.Query("path")))
	if relPath == "" || strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	// Full current URL in R2
	oldURL := config.CloudflarePublicDevURL + "/" + relPath

	// Get file entry
	var fileID int64
	var ext string
	err := db.QueryRow("SELECT id, type FROM files_table WHERE url = $1", oldURL).Scan(&fileID, &ext)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found in database"})
		} else {
			log.Printf("DB query error for %s: %v", oldURL, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query database"})
		}
		return
	}

	// Validate request body
	var req RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.NewName) == "" || strings.ContainsAny(req.NewName, "/\\:") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new name"})
		return
	}

	dir := filepath.Dir(relPath)
	newBase := req.NewName + ext
	newRelPath := filepath.ToSlash(filepath.Join(dir, newBase))
	newURL := config.CloudflarePublicDevURL + "/" + newRelPath

	// Check conflicts
	var conflictID int64
	err = db.QueryRow("SELECT id FROM files_table WHERE url = $1", newURL).Scan(&conflictID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A file with the new name already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for existing file"})
		return
	}

	// Update DB
	_, err = db.Exec("UPDATE files_table SET name = $1, url = $2 WHERE id = $3", newBase, newURL, fileID)
	if err != nil {
		log.Printf("Failed to update DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update database"})
		return
	}

	// Note: If needed, move the object in R2 bucket via `CopyObject` + `DeleteObject`

	c.JSON(http.StatusOK, gin.H{
		"status":  "renamed",
		"oldPath": relPath,
		"newPath": newRelPath,
	})
}
