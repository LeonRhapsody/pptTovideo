package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/LeonRhapsody/pptTovideo/internal/api"
	"github.com/LeonRhapsody/pptTovideo/internal/config"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Environment Check
	checkDependency("ffmpeg")
	checkDependency("soffice") // LibreOffice

	// 2. Load Config
	cfg := config.LoadConfig()

	// 3. Setup Router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/api/tasks"},
	}))

	// Load HTML templates
	r.LoadHTMLGlob("templates/*")

	handler := api.NewHandler(cfg)

	// Static files for seeing images/videos
	r.Static("/uploads", "./uploads")

	r.GET("/", handler.RenderIndex)

	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/parse", handler.HandleParse)
		apiGroup.POST("/preview", handler.HandlePreview)
		apiGroup.POST("/render", handler.HandleRender)
		apiGroup.GET("/tasks", handler.HandleGetTasks)
	}

	// 4. Start Server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

func checkDependency(cmdName string) {
	_, err := exec.LookPath(cmdName)
	if err != nil {
		// Try fallback for Mac LibreOffice
		if cmdName == "soffice" {
			if _, err := os.Stat("/Applications/LibreOffice.app/Contents/MacOS/soffice"); err == nil {
				fmt.Printf("Checked %s: OK (Found at default Mac path)\n", cmdName)
				return
			}
		}
		log.Printf("WARNING: %s is not installed or not in PATH. Usage may fail.", cmdName)
	} else {
		fmt.Printf("Checked %s: OK\n", cmdName)
	}
}
