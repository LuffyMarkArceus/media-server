package main

import (
	"media-server/handlers"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	r := gin.Default()

	// r.GET("/", func(c *gin.Context) {
	// 	c.Redirect(301, "/ping")
	// })

	// r.GET("/ping", handlers.Ping)
	r.GET("/media", handlers.ListMedia)
	r.GET("/media_stream", handlers.ServeMedia) // This will now be a redirect handler
	r.GET("/thumbnail/*filepath", handlers.GetThumbnail)
	// r.GET("/subtitle/*filepath", handlers.GetSubtitles)
	r.GET("/user/:name", handlers.GetUser)

	r.GET("/health", handlers.Health)

	r.POST("/upload", handlers.UploadFiles)
	// r.PUT("/rename", handlers.RenameFile)

	return r
}
