package handlers

import (
	"database/sql"
	"log"
	"media-server/config"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

var db *sql.DB
var r2Client *s3.Client // Add r2Client

// SetDB sets the database connection for handlers.
func SetDB(database *sql.DB) {
	db = database
}

// SetR2Client sets the R2 client for handlers
func SetR2Client(client *s3.Client) {
	r2Client = client
}


// ListMedia is updated to use PG placeholders and the correct URL prefix trim.
func ListMedia(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB not initialized"})
		return
	}

	subPath := c.Query("path")
	// ... (path cleaning logic is the same)
	if strings.Contains(subPath, "..") {
         c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Path, Not Allowed"})
         return
     }
     subPath = filepath.ToSlash(filepath.Clean(subPath))
     if subPath == "." || subPath == string(filepath.Separator){
         subPath = ""
     }

	var folderID int64
    // Use $1 for PostgreSQL
	err := db.QueryRow("SELECT id FROM folders_table WHERE path = $1", subPath).Scan(&folderID)
	if err != nil {
		// ... (error handling is the same)
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not Found"})
		return
	}

	// Query subfolders (use $1)
	folders := []string{}
	rows, err := db.Query("SELECT name FROM folders_table WHERE parent = $1 AND name != ''", folderID)
    // ... (rest of folder logic is the same)
	if err != nil {
		log.Printf("Error querying subfolders for folder %d: %v", folderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query subfolders"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Printf("Error scanning folder name: %v", err)
			continue
		}
		folders = append(folders, name)
	}


	// Query Files (use $1)
	files := []gin.H{}
	rows, err = db.Query(`
        SELECT name, size, url, type, created_at, thumbnail_url, subtitle_url 
        FROM files_table WHERE parent = $1
    `, folderID)
    if err != nil {
        log.Printf("Error querying files for folder %d: %v", folderID, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query files"})
        return
    }
    defer rows.Close()

	for rows.Next() {
		var name, url, typ string
		var size int64 // Use int64 for BIGINT
		var createdAt time.Time
		var thumbnailURL, subtitleURL sql.NullString
		if err := rows.Scan(&name, &size, &url, &typ, &createdAt, &thumbnailURL, &subtitleURL); err != nil {
            log.Printf("Error Scanning file %s : %v", name, err)
            continue
        }
        // The path is now derived by trimming the public R2 URL
		path := strings.TrimPrefix(url, config.CloudflarePublicDevURL+"/")
		files = append(files, gin.H{
			"name":       name,
			"size":       size,
			"path":       path, // This is the object key, used for other API calls
			"type":       typ,
			"url" :       url,
			"created_at": createdAt,
			"thumbnail_url": thumbnailURL.String, // Will be "" if NULL
            "subtitle_url":  subtitleURL.String,  // Will be "" if NULL
		})
	}
	// ... (error checking on rows.Err() is the same)

	c.JSON(http.StatusOK, gin.H{
		"folders": folders,
		"files":   files,
	})
}


// ServeMedia is now a redirector, not a file streamer.
func ServeMedia(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	path := c.Query("path")
	if path == "" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	log.Printf("Request to serve media for path: %s", path)

	// Construct the expected public URL from the path (object key)
	fileURL := config.CloudflarePublicDevURL + "/" + path

	// Verify the file exists in the database by its URL
	var dbURL string
	err := db.QueryRow("SELECT url FROM files_table WHERE url = $1", fileURL).Scan(&dbURL)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("File not found in database for URL: %s", fileURL)
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found in database"})
		} else {
			log.Printf("Error querying file by URL %s: %v", fileURL, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query file"})
		}
		return
	}

	// Redirect the client to the public R2 URL
	log.Printf("Redirecting client to: %s", dbURL)
	c.Redirect(http.StatusFound, dbURL)
}