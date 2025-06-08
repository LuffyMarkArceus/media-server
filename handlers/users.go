package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetUser(c *gin.Context) {
	// user := c.Param("name")
	// value, ok := storage.DB[user]
	// if ok {
	// 	c.JSON(http.StatusOK, gin.H{"user": user, "value": value})
	// } else {
	// 	c.JSON(http.StatusOK, gin.H{"user": user, "status": "no value"})
	// }
	c.JSON(http.StatusOK, gin.H{"status" : "Yet to make this route useful"})
}
