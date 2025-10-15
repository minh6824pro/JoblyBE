package main

import (
	"Jobly/api/handler/route"
	"Jobly/internal/config"
	"Jobly/internal/database"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

func main() {

	// Load env var
	config.LoadEnv()

	// Connect & create DB
	database.ConnectDatabase()
	database.AutoMigrate()

	db := database.DB

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")

	job := InitJobModule(db)
	auth := InitAuthModule(db)
	route.RegisterJobRoutes(api, job)
	route.RegisterAuthRoutes(api, auth)
	err := r.Run(":8080")
	if err != nil {
		log.Println(err)
	}
}
