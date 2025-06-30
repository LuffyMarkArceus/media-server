package main

import (
	"media-server/handlers"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	r := gin.Default()

	// Enable CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // or "*" for all origins,      // http://localhost:3000
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// r.GET("/", func(c *gin.Context) {
	// 	c.Redirect(301, "/ping")
	// })

	// r.GET("/ping", handlers.Ping)
	r.GET("/media", handlers.ListMedia)
	r.GET("/media_stream", handlers.ServeMedia) // This will now be a redirect handler
	r.GET("/thumbnail/*filepath", handlers.GetThumbnail)

	r.GET("/subtitle/*filepath", handlers.GetSubtitles)
	r.GET("/proxy_subtitle/*filepath", handlers.ProxySubtitle)
	
	r.GET("/user/:name", handlers.GetUser)

	r.GET("/health", handlers.Health)

	r.POST("/upload", handlers.UploadFiles)
	// r.PUT("/rename", handlers.RenameFile)

	return r
}
