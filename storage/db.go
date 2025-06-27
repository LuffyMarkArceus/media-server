package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"media-server/config"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// InitDB remains the same, your updated version is correct for PostgreSQL.
func InitDB(dataSourceName string) (*sql.DB, error) {
    // ... your existing InitDB code is fine ...
    // Make sure it returns after creating tables and indexes
	db, err := sql.Open("postgres", dataSourceName)
     if err != nil {
         return nil, fmt.Errorf("failed to open database: %w", err)
     }

     if err = db.Ping(); err != nil {
         return nil, fmt.Errorf("failed to connect to database: %w", err)
     }

     // Create tables
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


// SyncFilesWithR2 replaces the old SyncFiles function.
// It lists all objects in the R2 bucket and syncs them to the database.
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

			// Skip ignored files/folders
			if shouldSkip(objectKey) {
				continue
			}
			
			// The full path is the object key itself
			relPath := objectKey

			if processedPaths[relPath] {
				continue
			}

            // In R2/S3, folders are just zero-byte objects ending in "/" or implicit.
            // We will manage folders based on file paths instead of explicit folder objects.
			_, err = insertFileFromR2(db, relPath, obj, rootFolderID)
			if err != nil {
				log.Printf("Could not insert file %s: %v", relPath, err)
                continue // Continue with the next file
			}
			processedPaths[relPath] = true
		}
	}

	log.Println("Finished syncing files from R2.")
	return nil
}

// A helper to decide if a path should be ignored.
func shouldSkip(path string) bool {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") || part == "thumbnails" || part == "subtitles" {
			return true
		}
	}
	// Add other file extension checks if needed
	if strings.HasSuffix(path, ".ini") || strings.HasSuffix(path, ".dat") {
		return true
	}
	return false
}


// insertFileFromR2 is the new version of insertFile.
// It takes an S3 object instead of os.FileInfo.
func insertFileFromR2(db *sql.DB, relPath string, obj types.Object, rootFolderID int64) (int64, error) {
	// Construct the public URL for the file
	url := fmt.Sprintf("%s/%s", config.CloudflarePublicDevURL, relPath)

	var fileID int64
	err := db.QueryRow("SELECT id FROM files_table WHERE url = $1", url).Scan(&fileID)
	if err == nil {
		return fileID, nil // File already exists
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking file %s: %w", relPath, err)
	}

	parentPath := filepath.ToSlash(filepath.Dir(relPath))
	if parentPath == "." {
		parentPath = ""
	}
	parentID, err := insertFolder(db, parentPath, rootFolderID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert parent folder %s: %w", parentPath, err)
	}

	fileName := filepath.Base(relPath)
	fileSize := obj.Size
	fileType := filepath.Ext(fileName)
	modTime := aws.ToTime(obj.LastModified)

	err = db.QueryRow(
		`INSERT INTO files_table (ownerId, name, size, url, type, parent, created_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         RETURNING id`,
		"default_user", fileName, fileSize, url, fileType, parentID, modTime,
	).Scan(&fileID)

	if err != nil {
		return 0, fmt.Errorf("failed to insert file %s: %w", relPath, err)
	}
	log.Printf("Synced file: %s", relPath)
	return fileID, nil
}

func ensureRootFolder(db *sql.DB) (int64, error) {
	var rootID int64
	err := db.QueryRow("SELECT id FROM folders_table WHERE path = '' AND name = ''").Scan(&rootID)
	if err == nil {
		return rootID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking root folder: %w", err)
	}

	err = db.QueryRow(
		`INSERT INTO folders_table (ownerId, name, path, parent, created_at)
		 VALUES ($1, '', '', NULL, $2)
		 RETURNING id`,
		"default_user", time.Now(),
	).Scan(&rootID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert root folder: %w", err)
	}
	return rootID, nil
}

func insertFolder(db *sql.DB, relPath string, rootFolderID int64) (int64, error) {
	var folderID int64
	normalizedPath := filepath.ToSlash(relPath)
	err := db.QueryRow("SELECT id FROM folders_table WHERE path = $1", normalizedPath).Scan(&folderID)
	if err == nil {
		return folderID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking folder %s: %w", relPath, err)
	}

	name := filepath.Base(relPath)
	parentPath := filepath.ToSlash(filepath.Dir(relPath))
	if parentPath == "." || parentPath == "/" {
		parentPath = ""
	}

	var parentId int64
	if parentPath == "" {
		parentId = rootFolderID
	} else {
		parentId, err = insertFolder(db, parentPath, rootFolderID)
		if err != nil {
			return 0, fmt.Errorf("failed to insert parent folder %s: %w", parentPath, err)
		}
	}

	err = db.QueryRow(
		`INSERT INTO folders_table (ownerId, name, path, parent, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		"default_user", name, relPath, parentId, time.Now(),
	).Scan(&folderID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert folder %s: %w", name, err)
	}
	return folderID, nil
}
