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
	"time"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

// SetDB sets the database connection for handlers.
func SetDB(database *sql.DB){
	db = database
}

func ListMedia(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error" : "DB not initialized"})
		return
	}
	
	subPath := c.Query("path")
	if strings.Contains(subPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Path, Not Allowed"})
		return
	}

	subPath = filepath.ToSlash(filepath.Clean(subPath))
	if subPath == "." || subPath == string(filepath.Separator){
		subPath = ""
	}

	var folderID int64
	err := db.QueryRow("SELECT id FROM folders_table WHERE path = ?", subPath).Scan(&folderID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Folder not found %s: %v", subPath, err)
			c.JSON(http.StatusNotFound, gin.H{"error" : "Folder not Found"})
		} else {
			log.Printf("Error querying folder %s: %v", subPath, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query folder"})
		}
		return
	}

	// Query subfolders
	folders := []string{}
	rows, err := db.Query("SELECT name FROM folders_table WHERE parent = ? AND name != ''", folderID)
	if err != nil {
		log.Printf("Error querying subfolders for folder %d: %v", folderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query subfolders"})
		return
	}
	defer rows.Close()
	for rows.Next(){
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Printf("Error scanning folder name: %v", err)
			continue
		}
		folders = append(folders, name)
	}

	//Query Files
	files := []gin.H{}
	rows, err = db.Query("SELECT name, size, url, type, created_At FROM files_table WHERE parent = ?", folderID)
	if err != nil {
		log.Printf("Error querying files for folder %d: %v", folderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query files"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name, url, typ string
		var size int
		var createdAt time.Time
		if err := rows.Scan(&name, &size, &url, &typ, &createdAt); err != nil {
			log.Printf("Error Scanning file %s : %v", name, err)
		}
		relPath := strings.TrimPrefix(url, fmt.Sprintf("http://localhost:%v/media_stream?path=", config.AppPort))
		files = append(files, gin.H{
			"name":       name,
			"size":       size,
			"path":       relPath, // Keep "path" for frontend compatibility
			"type":       typ,
			"created_at": createdAt,
		})
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating files: %v", err)
	}
	log.Printf("Returning %d folders and %d files for path %s", len(folders), len(files), subPath)

	c.JSON(http.StatusOK, gin.H{
		"folders" : folders,
		"files" : files,
	})
}

func ServeMedia(c *gin.Context) {
	if db == nil {
		log.Println("Database is nil")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	path := c.Query("path")
	if path == "" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	log.Printf("Serving media for path: %s", path)

	// Verify File Exists in DB
	url := fmt.Sprintf("http://localhost:%v/media_stream?path=%s", config.AppPort, path)

	var filePath string
	var fileSize int64
	err := db.QueryRow("SELECT name, size FROM files_table WHERE url = ?", url).Scan(&filePath, &fileSize)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("File not found for URL: %s", url)
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found in database"})
		} else {
			log.Printf("Error querying file %s: %v", url, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to Query file"})
		}
		return
	}

	// Server file from filesystem
	absPath := filepath.Join(config.MediaRoot, path)
	log.Printf("Resolved filesystem path: %s", absPath)

	info, err := os.Stat(absPath)
	if err != nil {
		log.Printf("Error accessing file %s: %v", absPath, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on filesystem"})
		return
	}

	// Open the file
	file, err := os.Open(absPath)
	if err != nil {
		log.Printf("Error opening file %s: %v", absPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// Set headers
	c.Header("Content-Type", "video/mp4") // Adjust based on file type
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size()))
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", info.Name()))


	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is a directory"})
		return
	}

	// Stream the file
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
	log.Printf("Served media file: %s", absPath)
}
