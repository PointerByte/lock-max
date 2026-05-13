package main

import (
	"log"

	serverGin "github.com/PointerByte/GoForge/config/server/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	srv, err := serverGin.CreateApp()
	if err != nil {
		log.Fatal(err)
	}

	api := serverGin.GetRoute("/api/v1")
	api.GET("/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"app":     "iron-auth",
			"message": "hello from GoForge Gin",
		})
	})

	serverGin.Start(srv)
}
