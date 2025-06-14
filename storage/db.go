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

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)


func InitDB(dataSourceName string) (* sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
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
	log.Println("Created/Already Exists Table: folders_table")

	_, err = db.Exec(CreateFilesTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create files_table: %w", err)
	}
	log.Println("Created/Already Exists Table: files_table")


	// Create indexes
	_, err = db.Exec(CreateFilesParentIndexSQL)
	if err != nil {
		log.Printf("Warning: failed to create files_parent_index: %v", err)
	}
	_, err = db.Exec(CreateFilesOwnerIDIndexSQL)
	if err != nil {
		log.Printf("Warning: failed to create files_ownerId_index: %v", err)
	}
	_, err = db.Exec(CreateFoldersParentIndexSQL)
	if err != nil {
		log.Printf("Warning: failed to create folders_parent_index: %v", err)
	}
	_, err = db.Exec(CreateFoldersOwnerIDIndexSQL)
	if err != nil {
		log.Printf("Warning: failed to create folders_ownerId_index: %v", err)
	}

	return db, nil
}

func SyncFiles(db *sql.DB) error {
	rootFolderID, err := ensureRootFolder(db)
	if err != nil {
		return fmt.Errorf("failed to ensure root folder : %w", err)
	}

	err = filepath.Walk(config.MediaRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error Accessing path %s : %v", path, err)
			return nil
		}
		if path == config.MediaRoot{
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") ||
			info.Name() == "thumbnails" ||
			info.Name() == "subtitles"  ||
			strings.HasSuffix(info.Name(), ".ini") ||
			strings.HasSuffix(info.Name(), ".dat") ||
			strings.HasSuffix(info.Name(), ".tmp") {
				return nil
		}
		relPath, err := filepath.Rel(config.MediaRoot, path)
		if err != nil {
			log.Printf("Error getting relative path for %s: %v", path, err)
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		if info.IsDir(){
			_, err := insertFolder(db, relPath, rootFolderID)
			return err
		}
		_, err = insertFile(db, relPath, info, rootFolderID)
		return err

	})
	if err != nil {
		return fmt.Errorf("error walking filesystem: %w", err)
	}
	log.Printf("Walked System, ran SyncFiles successfully..")
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

	result, err := db.Exec("INSERT INTO folders_table (ownerId, name, path, parent, created_at) VALUES (?, '', '', NULL, ?)", "default_user", time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to insert root folder: %w", err)
	}
	return result.LastInsertId()
}

func insertFolder(db *sql.DB, relPath string, rootFolderID int64) (int64, error){
	var folderID int64
	err := db.QueryRow("SELECT id FROM folders_table WHERE path = ?", relPath).Scan(&folderID)
	if err == nil {
		return folderID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking folder %s: %w", relPath, err)
	}

	name := filepath.Base(relPath)
	parentPath := filepath.Dir(relPath)

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

	result, err := db.Exec(
		"INSERT INTO folders_table (ownerId, name, path, parent, created_at) VALUES (?, ?, ?, ?, ?)",
		"default_user", name, relPath, parentId, time.Now(),
	)
	return result.LastInsertId()
} 

func insertFile(db *sql.DB, relPath string, info os.FileInfo, rootFolderID int64) (int64, error) {
	var fileID int64
	url := fmt.Sprintf("http://localhost:%v/media_stream?path=%s", config.AppPort, relPath)
	err := db.QueryRow("SELECT id FROM files_table WHERE url = ?", url).Scan(&fileID)
	if err == nil {
		return fileID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking file %s: %w", relPath, err)
	}

	parentPath := filepath.Dir(relPath)
	if parentPath == "." || parentPath == "/" {
		parentPath = ""
	}
	parentID, err := insertFolder(db, parentPath, rootFolderID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert parent folder %s: %w", parentPath, err)
	}

	result, err := db.Exec(
		"INSERT INTO files_table (ownerId, name, size, url, type, parent, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"default_user", info.Name(), info.Size(), url, filepath.Ext(info.Name()), parentID, info.ModTime(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert %s: %w", relPath, err)
	}
	return result.LastInsertId()
}
