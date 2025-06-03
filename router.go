package main

import (
	"media-server/handlers"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/ping")
	})

	r.GET("/ping", handlers.Ping)
	r.GET("/user/:name", handlers.GetUser)
	r.POST("/upload", handlers.UploadFiles)
	r.GET("/media", handlers.ListMedia)
	r.GET("/media/*filepath", handlers.ServeMedia)
	r.GET("/thumbnail/*filepath", handlers.GetThumbnail)

	return r
}
