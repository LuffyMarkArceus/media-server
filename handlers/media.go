package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"media-server/config"

	"github.com/gin-gonic/gin"
)

func ListMedia(c *gin.Context) {
	// mediaDir := "./media"
	// mediaDir := config.MediaRoot
	subPath := c.Query("path")
	if strings.Contains(subPath, ".."){
		c.JSON(http.StatusBadRequest, gin.H{"error" : "Invalid Path, Not Allowed"})
	}
	subPath = filepath.Clean(subPath)
	if subPath == "." || subPath == string(filepath.Separator) {
		subPath = ""
	} 
	
	baseDir := filepath.Join(config.MediaRoot, subPath)
	entries, err := os.ReadDir(baseDir)
	// fmt.Println(entries)
	// log.Println("error : %w", err)
	if err != nil{
		log.Printf("Error reading directory %s: %v", baseDir, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read directory"})
		return
	}
	var folders []string
	var files []gin.H

	for _, entry := range entries {
		
		if entry.Name() == "thumbnails" || entry.Name() == "subtitles" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			folders = append(folders, entry.Name())
		} else {
			info, err := entry.Info()
			if err != nil {
				log.Printf("Error getting file info for %s: %v", entry.Name(), err)
				continue
				// return nil
			}

			if strings.HasSuffix(entry.Name(), ".ini"){
				continue
			}
			fullRelativePath := filepath.ToSlash(filepath.Join(subPath, entry.Name()))

			files = append(files, gin.H{
				"name": entry.Name(),
				"size": info.Size(),
				"path": fullRelativePath, // Use the full relative path for media operations
				"type": filepath.Ext(entry.Name()),
				"created_at"  : info.ModTime(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"folders" : folders,
		"files" : files,
	})

}

func ServeMedia(c *gin.Context) {
	rawPath := c.Query("path")
	if rawPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing path"})
		return
	}

	// Normalize slashes from frontend input (which might use backslashes)
	// and clean the path for security.
	relPath := filepath.FromSlash(rawPath) // Converts / to \ on Windows if necessary
	relPath = filepath.Clean(relPath)      // Resolves ".." component
	
	// Security check: ensure path doesn't escape the media root after cleaning
	// This is important if `Clean` resolves `..` to something that could point outside
	if strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, string(filepath.Separator) + "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path for security reasons"})
		return
	}

	fullPath := filepath.Join(config.MediaRoot, relPath)
	// Debug log
	log.Printf("Attempting to serve file : %s", fullPath)
	
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		} else {
			log.Printf("Stat error for %s: %v", fullPath, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not access file"})
		}
		return
	}
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is a directory"})
		return
	}

	c.File(fullPath)
}
