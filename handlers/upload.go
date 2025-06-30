package handlers

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"media-server/config"
	dbstore "media-server/storage"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

func UploadFiles(c *gin.Context) {
	if db == nil || r2Client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not initialized"})
		return
	}

	// Optional path parameter (folder)
	uploadPath := filepath.ToSlash(filepath.Clean(c.Query("path")))
	if strings.Contains(uploadPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	if uploadPath == "." || uploadPath == "/" {
		uploadPath = ""
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form"})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files to upload"})
		return
	}

	var uploaded []string
	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}
		defer src.Close()

		// Buffer the file content
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, src); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file content"})
			return
		}

		key := filepath.ToSlash(filepath.Join(uploadPath, file.Filename))
		publicURL := fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, key)

		// Upload to R2
		_, err = r2Client.PutObject(c, &s3.PutObjectInput{
			Bucket: aws.String(config.CloudflareR2BucketName),
			Key:    aws.String(key),
			Body:   bytes.NewReader(buf.Bytes()),
		})
		if err != nil {
			log.Printf("R2 upload failed for %s: %v", key, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to cloud"})
			return
		}

		// Insert into DB
		parentID, err := dbstore.InsertFolder(db, uploadPath, 1) // assuming 1 is root ID (or call ensureRootFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder in DB"})
			return
		}
		ext := filepath.Ext(file.Filename)

		_, err = db.Exec(
			`INSERT INTO files_table (ownerId, name, size, url, type, parent, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			"default_user", file.Filename, file.Size, publicURL, ext, parentID, time.Now(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB insert failed"})
			return
		}

		uploaded = append(uploaded, file.Filename)
	}

	c.JSON(http.StatusOK, gin.H{"uploaded": uploaded})
}
