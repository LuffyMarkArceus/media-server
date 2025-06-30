package storage

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"media-server/config"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	_ "github.com/lib/pq"
)

// InitDB initializes the database and creates tables if not present
func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	_, err = db.Exec(CreateFoldersTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create folders_table: %w", err)
	}
	log.Println("Created/Verified Table: folders_table")

	_, err = db.Exec(CreateFilesTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create files_table: %w", err)
	}
	log.Println("Created/Verified Table: files_table")

	return db, nil
}

// StartSyncAndAssetGeneration runs both file sync and asset generation
func StartSyncAndAssetGeneration(db *sql.DB, r2Client *s3.Client, bucket string) error {
	if err := SyncFilesWithR2(db, r2Client, bucket); err != nil {
		return err
	}
	return GenerateMissingAssetsForExistingFiles(db, r2Client, bucket)
}

// SyncFilesWithR2 pulls files from R2 and inserts new ones into DB
func SyncFilesWithR2(db *sql.DB, r2Client *s3.Client, bucketName string) error {
	rootFolderID, err := ensureRootFolder(db)
	if err != nil {
		return fmt.Errorf("failed to ensure root folder: %w", err)
	}

	log.Println("Starting file sync from R2 bucket:", bucketName)
	processedPaths := make(map[string]bool)

	paginator := s3.NewListObjectsV2Paginator(r2Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list objects from R2: %w", err)
		}

		for _, obj := range page.Contents {
			objectKey := aws.ToString(obj.Key)

			if shouldSkip(objectKey) {
				continue
			}

			if processedPaths[objectKey] {
				continue
			}

			_, err := insertFileFromR2(db, objectKey, obj, rootFolderID, r2Client, bucketName)
			if err != nil {
				log.Printf("Could not insert file %s: %v", objectKey, err)
				continue
			}
			processedPaths[objectKey] = true
		}
	}

	log.Println("Finished syncing files from R2.")
	return nil
}

func shouldSkip(path string) bool {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") || part == "thumbnails" || part == "subtitles" {
			return true
		}
	}
	if strings.HasSuffix(path, ".ini") || strings.HasSuffix(path, ".dat") {
		return true
	}
	return false
}

func insertFileFromR2(db *sql.DB, relPath string, obj types.Object, rootFolderID int64, r2Client *s3.Client, bucket string) (int64, error) {
	url := fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, relPath)

	var fileID int64
	err := db.QueryRow("SELECT id FROM files_table WHERE url = $1", url).Scan(&fileID)
	if err == nil {
		return fileID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking file %s: %w", relPath, err)
	}

	parentPath := filepath.ToSlash(filepath.Dir(relPath))
	if parentPath == "." {
		parentPath = ""
	}
	parentID, err := InsertFolder(db, parentPath, rootFolderID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert parent folder %s: %w", parentPath, err)
	}

	fileName := filepath.Base(relPath)
	fileSize := obj.Size
	fileType := filepath.Ext(fileName)
	modTime := aws.ToTime(obj.LastModified)

	var thumbnailURL, subtitleURL *string
	if isVideoFile(fileType) {
		thumbnailURL, _ = generateThumbnailAndUpload(r2Client, bucket, relPath)
		subtitleURL, _ = generateSubtitleAndUpload(r2Client, bucket, relPath)
	}

	err = db.QueryRow(
		`INSERT INTO files_table 
		(ownerId, name, size, url, type, parent, created_at, thumbnail_url, subtitle_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		"default_user", fileName, fileSize, url, fileType, parentID, modTime, thumbnailURL, subtitleURL,
	).Scan(&fileID)

	if err != nil {
		return 0, fmt.Errorf("failed to insert file %s: %w", relPath, err)
	}

	log.Printf("Synced file: %s", relPath)
	return fileID, nil
}

func isVideoFile(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".mp4" || ext == ".mkv" || ext == ".avi" || ext == ".mov" || ext == ".webm"
}

func generateThumbnailAndUpload(r2Client *s3.Client, bucket, objectKey string) (*string, error) {
	presignClient := s3.NewPresignClient(r2Client)
	presigned, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return nil, err
	}

	thumbnailKey := "thumbnails/" + strings.TrimSuffix(objectKey, filepath.Ext(objectKey)) + ".jpg"
	var buf bytes.Buffer

	cmd := exec.Command("ffmpeg", "-ss", "00:00:05", "-i", presigned.URL, "-vframes", "1", "-q:v", "2", "-f", "image2", "pipe:1")
	cmd.Stdout = &buf
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		log.Printf("Thumbnail error: %v", err)
		return nil, nil
	}

	uploader := manager.NewUploader(r2Client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(thumbnailKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		return nil, nil
	}

	url := fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, thumbnailKey)
	return &url, nil
}

func generateSubtitleAndUpload(r2Client *s3.Client, bucket, objectKey string) (*string, error) {
	presignClient := s3.NewPresignClient(r2Client)
	presigned, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return nil, err
	}

	subtitleKey := "subtitles/" + strings.TrimSuffix(objectKey, filepath.Ext(objectKey)) + ".vtt"
	var buf bytes.Buffer

	cmd := exec.Command("ffmpeg", "-i", presigned.URL, "-map", "0:s:0?", "-f", "webvtt", "pipe:1")
	cmd.Stdout = &buf
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		log.Printf("Subtitle error for %s: %v", objectKey, err)
		return nil, nil
	}

	uploader := manager.NewUploader(r2Client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(subtitleKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("text/vtt"),
	})
	if err != nil {
		return nil, nil
	}

	url := fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, subtitleKey)
	log.Printf("Generated subtitle for %s", subtitleKey)
	return &url, nil
}

func ensureRootFolder(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM folders_table WHERE path = '' AND name = ''").Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	err = db.QueryRow(`
		INSERT INTO folders_table (ownerId, name, path, parent, created_at)
		VALUES ('default_user', '', '', NULL, $1)
		RETURNING id
	`, time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func InsertFolder(db *sql.DB, relPath string, rootID int64) (int64, error) {
	var id int64
	relPath = filepath.ToSlash(relPath)
	err := db.QueryRow("SELECT id FROM folders_table WHERE path = $1", relPath).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	name := filepath.Base(relPath)
	parentPath := filepath.ToSlash(filepath.Dir(relPath))
	if parentPath == "." || parentPath == "/" {
		parentPath = ""
	}

	var parentID int64
	if parentPath == "" {
		parentID = rootID
	} else {
		parentID, err = InsertFolder(db, parentPath, rootID)
		if err != nil {
			return 0, err
		}
	}

	err = db.QueryRow(`
		INSERT INTO folders_table (ownerId, name, path, parent, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, "default_user", name, relPath, parentID, time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GenerateMissingAssetsForExistingFiles updates thumbnail/subtitle URLs for DB entries with NULLs.
func GenerateMissingAssetsForExistingFiles(db *sql.DB, r2Client *s3.Client, bucket string) error {
	rows, err := db.Query(`
		SELECT id, url, type, thumbnail_url, subtitle_url
		FROM files_table
		WHERE thumbnail_url IS NULL OR subtitle_url IS NULL
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var url, fileType string
		var tURL, sURL *string

		if err := rows.Scan(&id, &url, &fileType, &tURL, &sURL); err != nil {
			log.Printf("Scan failed: %v", err)
			continue
		}

		if !isVideoFile(fileType) {
			continue
		}

		objectKey := strings.TrimPrefix(url, config.CloudflarePublicDevURL+"/")

		if tURL == nil {
			tURL, _ = generateThumbnailAndUpload(r2Client, bucket, objectKey)
		}
		if sURL == nil {
			sURL, _ = generateSubtitleAndUpload(r2Client, bucket, objectKey)
		}

		if tURL != nil || sURL != nil {
			_, err := db.Exec(`
				UPDATE files_table
				SET thumbnail_url = COALESCE($1, thumbnail_url),
				    subtitle_url = COALESCE($2, subtitle_url)
				WHERE id = $3
			`, tURL, sURL, id)

			if err != nil {
				log.Printf("Failed to update file %d: %v", id, err)
			} else {
				log.Printf("Updated file ID %d with missing assets", id)
			}
		}
	}
	return nil
}
