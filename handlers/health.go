package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func Health(c *gin.Context) {
	// CPU Usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Printf("Error getting CPU usage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get CPU usage"})
		return
	}

	// RAM Usage
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting RAM usage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get RAM usage"})
		return
	}

	// Basic server status
	dbStatus := "OK"
	if err := db.Ping(); err != nil {
		dbStatus = "Down"
	}

	response := gin.H{
		"status":        "healthy",
		"cpu_usage":     fmt.Sprintf("%.2f%%", cpuPercent[0]),
		"ram_total":     fmt.Sprintf("%.2f GB", float64(memInfo.Total)/1024/1024/1024),
		"ram_used":      fmt.Sprintf("%.2f GB", float64(memInfo.Used)/1024/1024/1024),
		"ram_available": fmt.Sprintf("%.2f GB", float64(memInfo.Available)/1024/1024/1024),
		"db_status":     dbStatus,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}