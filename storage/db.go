package storage

import (
	"database/sql"
	"fmt"
	"log"
	"media-server/config"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create tables
	if _, err = db.Exec(CreateFoldersTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create folders_table: %w", err)
	}
	log.Println("Created/Verified Table: folders_table")

	if _, err = db.Exec(CreateFilesTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create files_table: %w", err)
	}
	log.Println("Created/Verified Table: files_table")

	// Create indexes
	if _, err = db.Exec(CreateFilesParentIndexSQL); err != nil {
		log.Printf("Warning: failed to create files_parent_index: %v", err)
	}
	if _, err = db.Exec(CreateFilesOwnerIDIndexSQL); err != nil {
		log.Printf("Warning: failed to create files_ownerId_index: %v", err)
	}
	if _, err = db.Exec(CreateFoldersParentIndexSQL); err != nil {
		log.Printf("Warning: failed to create folders_parent_index: %v", err)
	}
	if _, err = db.Exec(CreateFoldersOwnerIDIndexSQL); err != nil {
		log.Printf("Warning: failed to create folders_ownerId_index: %v", err)
	}

	return db, nil
}

func SyncFiles(db *sql.DB) error {
	rootFolderID, err := ensureRootFolder(db)
	if err != nil {
		return fmt.Errorf("failed to ensure root folder: %w", err)
	}
	log.Printf("Started OS Walk")
	processedPaths := make(map[string]bool)

	err = filepath.Walk(config.MediaRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil
		}
		if path == config.MediaRoot {
			return nil
		}
		relPath, err := filepath.Rel(config.MediaRoot, path)
		if err != nil {
			log.Printf("Error getting relative path for %s: %v", path, err)
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		if strings.HasPrefix(info.Name(), ".") ||
			strings.Contains(relPath, "thumbnails") ||
			strings.Contains(relPath, "subtitles") ||
			strings.HasSuffix(info.Name(), ".ini") ||
			strings.HasSuffix(info.Name(), ".dat") ||
			strings.HasSuffix(info.Name(), ".tmp") {
			return nil
		}

		if processedPaths[relPath] {
			return nil
		}

		if info.IsDir() {
			_, err := insertFolder(db, relPath, rootFolderID)
			if err != nil {
				return err
			}
		} else {
			_, err = insertFile(db, relPath, info, rootFolderID)
			if err != nil {
				return err
			}
		}

		processedPaths[relPath] = true
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking filesystem: %w", err)
	}
	log.Printf("SyncFiles completed successfully")
	return nil
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

	result, err := db.Exec(`INSERT INTO folders_table (ownerId, name, path, parent, created_at)
		VALUES ($1, '', '', NULL, $2)`, "default_user", time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to insert root folder: %w", err)
	}
	return result.LastInsertId()
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

	var parentID int64
	if parentPath == "" {
		parentID = rootFolderID
	} else {
		parentID, err = insertFolder(db, parentPath, rootFolderID)
		if err != nil {
			return 0, fmt.Errorf("failed to insert parent folder %s: %w", parentPath, err)
		}
	}

	result, err := db.Exec(`
		INSERT INTO folders_table (ownerId, name, path, parent, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "default_user", name, relPath, parentID, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to insert folder %s: %w", name, err)
	}
	return result.LastInsertId()
}

func insertFile(db *sql.DB, relPath string, info os.FileInfo, rootFolderID int64) (int64, error) {
	var fileID int64
	url := fmt.Sprintf("http://localhost:%v/media_stream?path=%s", config.AppPort, filepath.ToSlash(relPath))

	err := db.QueryRow("SELECT id FROM files_table WHERE url = $1", url).Scan(&fileID)
	if err == nil {
		return fileID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking file %s: %w", relPath, err)
	}

	parentPath := filepath.ToSlash(filepath.Dir(relPath))
	if parentPath == "." || parentPath == "/" {
		parentPath = ""
	}
	parentID, err := insertFolder(db, parentPath, rootFolderID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert parent folder %s: %w", parentPath, err)
	}

	result, err := db.Exec(`
		INSERT INTO files_table (ownerId, name, size, url, type, parent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "default_user", info.Name(), info.Size(), url, filepath.Ext(info.Name()), parentID, info.ModTime())
	if err != nil {
		return 0, fmt.Errorf("failed to insert file %s: %w", relPath, err)
	}
	return result.LastInsertId()
}
