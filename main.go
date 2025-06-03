// package main

// import (
// 	"fmt"
// 	"net/http"
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// 	"strings"

// 	"github.com/gin-gonic/gin"
// )
// var db = make(map[string]string)
// func init_db() {
// 	db["shrey"] = "1234"
// }

// func setupRouter() *gin.Engine {
// 	// gin.DisableConsoleColor()
// 	r := gin.Default()

// 	// Ping test
// 	r.GET("/ping", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, gin.H{"messsage": "pong"})
// 	})

// 	// Get user value
// 	r.GET("/user/:name", func(c *gin.Context) {
// 		user := c.Params.ByName("name")
// 		value, ok := db[user]
// 		if ok {
// 			c.JSON(http.StatusOK, gin.H{"user": user, "value": value})
// 		} else {
// 			c.JSON(http.StatusOK, gin.H{"user": user, "status": "no value"})
// 		}
// 	})

// 	// Basic Redirection example
// 	r.GET("/", func(c *gin.Context) {
// 		c.Redirect(http.StatusMovedPermanently, "/ping")
// 	})

// 	// List all Media files in a DIR
// 	r.GET("/media", func(c *gin.Context) {
// 		mediaDir := "./media"
// 		var fileList []gin.H

// 		err := filepath.WalkDir(mediaDir, func(path string, d os.DirEntry, err error) error {
// 			if err != nil {
// 				// Skip problematic entries
// 				return nil
// 			}
// 			if d.IsDir() {
// 				return nil // Skip directories
// 			}
// 			info, err := d.Info()
// 			if err != nil {
// 				return nil
// 			}

// 			relPath, err := filepath.Rel(mediaDir, path)
// 			if err != nil {
// 				relPath = path // fallback
// 			}

// 			fileList = append(fileList, gin.H{
// 				"name":  d.Name(),
// 				"size":  info.Size(),
// 				"path":  relPath,  // relative path from /media
// 				"type":  filepath.Ext(d.Name()),
// 			})
// 			return nil
// 		})

// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan media directory"})
// 			return
// 		}

// 		c.JSON(http.StatusOK, fileList)
// 	})

// 	// Upload files
// 	r.POST("/upload", func(c *gin.Context) {
// 		form, err := c.MultipartForm()
// 		if err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Multipart Form"})
// 			return
// 		}
// 		files := form.File["files"]
// 		if len(files) == 0 {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "No files to upload"})
// 			return
// 		}
// 		uploadDir := "./media"
// 		os.MkdirAll(uploadDir, os.ModePerm)

// 		var uploaded []string
// 		for _, file := range files {
// 			dst := filepath.Join(uploadDir, file.Filename)
// 			if err := c.SaveUploadedFile(file, dst); err != nil {
// 				c.JSON(http.StatusInternalServerError, gin.H{
// 					"error" : fmt.Sprintf("Failed to upload %s", file.Filename),
// 				})
// 				return
// 			}
// 			uploaded = append(uploaded, file.Filename)
// 		}
// 		c.JSON(http.StatusOK, gin.H{
// 			"uploaded": uploaded,
// 		})

// 	})

// 	r.GET("/thumbnail/*filepath", func(c *gin.Context) {
// 		relPath := c.Param("filepath") // includes leading slash
// 		relPath = strings.TrimPrefix(relPath, "/")

// 		// Security check: prevent directory traversal
// 		if strings.Contains(relPath, "..") {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
// 			return
// 		}

// 		thumbPath := filepath.Join("./media/thumbnails", relPath)
// 		thumbPath = strings.TrimSuffix(thumbPath, filepath.Ext(thumbPath)) + ".jpg"
// 		videoPath := filepath.Join("./media", relPath)

// 		// Check if thumbnail already exists
// 		info, err := os.Stat(thumbPath)
// 		if err == nil && !info.IsDir() {
// 			c.File(thumbPath)
// 			return
// 		}

// 		// Create thumbnails dir if needed
// 		err = os.MkdirAll(filepath.Dir(thumbPath), os.ModePerm)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create thumbnails directory"})
// 			return
// 		}

// 		// Generate thumbnail using ffmpeg
// 		err = exec.Command(
// 			"ffmpeg",
// 			"-i", videoPath,
// 			"-ss", "00:00:01",
// 			"-vframes", "1",
// 			thumbPath,
// 		).Run()

// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create thumbnail for %s", relPath)})
// 			return
// 		}

// 		c.File(thumbPath)
// 	})

// 	r.GET("/media/*filepath", func(c *gin.Context) {
// 		relPath := c.Param("filepath") // includes leading slash
// 		relPath = strings.TrimPrefix(relPath, "/")

// 		// Security check: no directory traversal
// 		if strings.Contains(relPath, "..") {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
// 			return
// 		}

// 		fullPath := filepath.Join("./media", relPath)
// 		info, err := os.Stat(fullPath)
// 		if os.IsNotExist(err) || info.IsDir() {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
// 			return
// 		}
// 		c.File(fullPath)
// 	})

// 	return r
// }

// func main() {
// 	init_db()
// 	r := setupRouter()
// 	r.Run("localhost:8080")
// }

package main

import (
	"media-server/storage"
)

func main() {
	storage.InitDB()
	r := setupRouter()
	r.Run("localhost:8080")
}
