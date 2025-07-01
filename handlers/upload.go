package handlers

import (
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
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

const multipartChunkSize = 5 * 1024 * 1024 // 5 MB chunks

func UploadFiles(c *gin.Context) {
	log.Println("UploadFiles handler hit")
	if db == nil || r2Client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not initialized"})
		return
	}

	uploadPath := filepath.ToSlash(filepath.Clean(c.Query("path")))
	if strings.Contains(uploadPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	if uploadPath == "." || uploadPath == "/" {
		uploadPath = ""
	}

	// 2. Instantiate the S3 Uploader
	// This manager will handle the multipart upload automatically based on PartSize.
	uploader := manager.NewUploader(r2Client, func(u *manager.Uploader) {
		u.PartSize = multipartChunkSize
		u.Concurrency = 3 // Adjust the number of parallel workers as needed
	})

	ctx := c.Request.Context()
	var uploadedFiles []gin.H
	
	// 3. Get a streaming multipart reader from the request
	mpReader, err := c.Request.MultipartReader()
	if err != nil {
		log.Printf("Error getting multipart reader: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart request"})
		return
	}

	// 4. Ensure the target folder exists in the database
	// We get the root folder ID once to avoid looking it up in every loop.
	rootFolderID, err := dbstore.EnsureRootFolder(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not ensure root folder exists"})
		return
	}
	parentID, err := dbstore.InsertFolder(db, uploadPath, rootFolderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create destination folder in DB"})
		return
	}
	
	// 5. Process each part of the stream
	for {
		part, err := mpReader.NextPart()
		if err == io.EOF {
			break // Finished reading all parts
		}
		if err != nil {
			log.Printf("Error reading multipart part: %v", err)
			continue
		}

		// Skip parts that are not files
		if part.FileName() == "" {
			part.Close()
			continue
		}

		fileName := part.FileName()
		key := filepath.ToSlash(filepath.Join(uploadPath, fileName))
		publicURL := fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, key)

		// The part object is an io.Reader, which we can pass directly to the uploader.
		// The manager will read from this stream.
		result, err := uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(config.CloudflareR2BucketName),
			Key:    aws.String(key),
			Body:   part, // Stream the part directly
		})

		// Must close the part after processing
		part.Close()

		if err != nil {
			log.Printf("Failed to upload file %s: %v", fileName, err)
			// Decide if you want to stop or continue with other files
			continue
		}

		log.Printf("Successfully uploaded %s to %s", fileName, result.Location)

		// To get the file size, we need to query R2 after the upload,
		// as we can't know the size from a stream beforehand.
		head, err := r2Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(config.CloudflareR2BucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("Failed to get metadata for %s after upload: %v", key, err)
			continue
		}
		fileSize := head.ContentLength

		// 6. Insert file metadata into the database
		fileExt := filepath.Ext(fileName)
		_, err = db.Exec(
			`INSERT INTO files_table (ownerId, name, size, url, type, parent, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			"default_user", fileName, fileSize, publicURL, fileExt, parentID, time.Now(),
		)
		if err != nil {
			log.Printf("DB insert failed for %s: %v", fileName, err)
			// You might want to delete the uploaded R2 object here for consistency
			continue
		}

		uploadedFiles = append(uploadedFiles, gin.H{
			"name": fileName,
			"size": fileSize,
			"url":  publicURL,
		})
	}

	if len(uploadedFiles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files were successfully uploaded"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Files uploaded successfully",
		"uploaded": uploadedFiles,
	})
}
