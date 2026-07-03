package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"inkflow-go/config"
)

func main() {
	cfg := config.Load()
	r := gin.Default()
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
	})
	addr := fmt.Sprintf(":%d", cfg.Port)
	r.Run(addr)
}