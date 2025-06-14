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
	r.GET("/media_stream", handlers.ServeMedia)
	r.GET("/subtitle/*filepath", handlers.GetSubtitles)
	r.GET("/thumbnail/*filepath", handlers.GetThumbnail)
	r.GET("/user/:name", handlers.GetUser)

	r.POST("/upload", handlers.UploadFiles)
	r.PUT("/rename", handlers.RenameFile)

	return r
}
