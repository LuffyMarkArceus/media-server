package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"media-server/config"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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


func ServeHLS(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error" : "DB not Initialized"})
		return
	}

	path := c.Query("path")
	if path == "" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	// Decode the path to handle double encoding
	decodedPath, err := url.QueryUnescape(path)
	if err != nil {
		log.Printf("Failed to decode path %s: %v", path, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path encoding"})
		return
	}

	// Verfiy file in DB:
	url := fmt.Sprintf("http://localhost:%v/media_stream?path=%s", config.AppPort, decodedPath)
	var fileID int64
	err = db.QueryRow("SELECT id FROM files_table WHERE url = ?", url).Scan(&fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("File not found for path: %s", decodedPath)
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found in database"})
		} else {
			log.Printf("Error querying file %s: %v", decodedPath, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query file"})
		}
		return
	}

	absPath := filepath.Join(config.MediaRoot, decodedPath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("Source video file not found: %s", absPath)
		c.JSON(http.StatusNotFound, gin.H{"error": "Source video file not found"})
		return
	}

	// Temp HLS Dir.
	tempHlsDir := filepath.Join(config.MediaRoot, "temp_hls", filepath.Base(decodedPath))
	if err := os.MkdirAll(tempHlsDir, os.ModePerm); err != nil {
		log.Printf("Failed to create temp HLS directory %s: %v", tempHlsDir, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}

	masterPlaylist := filepath.Join(tempHlsDir, "playlist.m3u8")

	if info, err := os.Stat(masterPlaylist); err == nil && time.Since(
		info.ModTime()) < 24*60*time.Minute {
			c.File(masterPlaylist)
			log.Printf("Served cached HLS Playlist for: %s", path)
			return
	}
	
	// Generate HLS segments on the go
	segmentPattern := filepath.Join(tempHlsDir, "%03d.ts")
	cmd := exec.Command(
		"ffmpeg",
		"-i", absPath,
		"-map", "v:0", "-map", "a:0",
		"-f", "hls",
		"-hls_time", "10",
		"-hls_list_size", "0",
		"-hls_segment_filename", segmentPattern,
		"-master_pl_name", "playlist.m3u8",
		masterPlaylist,
	)

	// PIPE OUTPUT to handle streaming
	cmd.Stdout = c.Writer
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("FFmpeg error starting HLS for %s: %v", absPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start HLS transcoding"})
		return
	}

	// Wait for FFMpeg in a goroutine to avoid blocking the HTTP response
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("FFmpeg error completing HLS for %s: %v", absPath, err)
		}
		// Clean old segments (e.g. older than 1 day)
		cleanUpOldSegments(tempHlsDir, 24*60*time.Minute)
	}()
	
	c.Header("Content-Type", "application/x-mpegURL")
	c.Status(http.StatusOK)
}

func cleanUpOldSegments(dir string, maxAge time.Duration){
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && time.Since(info.ModTime()) > maxAge {
			if err := os.Remove(path); err != nil {
				log.Printf("Failed to remove old segment %s: %v", path, err)
			} else {
				log.Printf("Removed old segment: %s", path)
			}
		}
		return nil
	}); err != nil {
		log.Printf("Error cleaning up segments in %s: %v", dir, err)
	}
}
