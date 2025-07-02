package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"media-server/config"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

type RenameRequest struct {
	NewName string `json:"newName" binding:"required"`
}

func RenameFile(c *gin.Context) {
	if db == nil || r2Client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not initialized"})
		return
	}

	// Get and sanitize path
	relPath := filepath.ToSlash(filepath.Clean(c.Query("path")))
	if relPath == "" || strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	oldKey := relPath
	oldURL := config.CloudflarePublicDevURL + "/" + oldKey

	// Get file from DB
	var fileID int64
	var ext string
	err := db.QueryRow("SELECT id, type FROM files_table WHERE url = $1", oldURL).Scan(&fileID, &ext)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found in DB"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB query error"})
		}
		return
	}

	// Parse new name
	var req RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.NewName) == "" || strings.ContainsAny(req.NewName, "/\\:") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new name"})
		return
	}

	newBase := req.NewName + ext
	dir := filepath.Dir(oldKey)
	newKey := filepath.ToSlash(filepath.Join(dir, newBase))
	newURL := config.CloudflarePublicDevURL + "/" + newKey

	// Check if file with same name exists
	var conflictID int64
	err = db.QueryRow("SELECT id FROM files_table WHERE url = $1", newURL).Scan(&conflictID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A file with that name already exists"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Conflict check failed"})
		return
	}

	// R2 rename via Copy + Delete
	_, err = r2Client.CopyObject(c, &s3.CopyObjectInput{
		Bucket:     aws.String(config.CloudflareR2BucketName),
		CopySource: aws.String(fmt.Sprintf("%s/%s", config.CloudflareR2BucketName, oldKey)),
		Key:        aws.String(newKey),
	})
	if err != nil {
		log.Printf("R2 copy failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "R2 copy failed"})
		return
	}

	_, err = r2Client.DeleteObject(c, &s3.DeleteObjectInput{
		Bucket: aws.String(config.CloudflareR2BucketName),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		log.Printf("R2 delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "R2 delete failed"})
		return
	}

	// DB update
	_, err = db.Exec("UPDATE files_table SET name = $1, url = $2 WHERE id = $3", newBase, newURL, fileID)
	if err != nil {
		log.Printf("DB update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "renamed",
		"oldPath": oldKey,
		"newPath": newKey,
	})
}
