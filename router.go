package main

import (
	"media-server/handlers"
	"media-server/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	middleware.InitJWKS()

	r := gin.Default()

	// Enable CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // or "*" for all origins,      // http://localhost:3000
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Public Routes
	r.GET("/health", handlers.Health)
	r.GET("/proxy_subtitle/*filepath", handlers.ProxySubtitle)


	// Protected routes
	authorized := r.Group("/")
	authorized.Use(middleware.JWTAuthMiddleware())
	{
		authorized.GET("/media", handlers.ListMedia)
		authorized.GET("/media_stream", handlers.ServeMedia) // This will now be a redirect handler
		authorized.GET("/thumbnail/*filepath", handlers.GetThumbnail)
		authorized.GET("/proxy_thumbnail/*filepath", handlers.ProxyThumbnail)

		authorized.GET("/subtitle/*filepath", handlers.GetSubtitles)
		// authorized.
		
		
		authorized.GET("/user/:name", handlers.GetUser)

		authorized.POST("/upload", handlers.UploadFiles)
		authorized.PUT("/rename", handlers.RenameFile)
	}

	return r
}
