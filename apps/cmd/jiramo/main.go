package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/conflict-driven-devs/headquarter/internal/config"
	"github.com/conflict-driven-devs/headquarter/internal/db"
	"github.com/conflict-driven-devs/headquarter/internal/handler"
	"github.com/conflict-driven-devs/headquarter/internal/middleware"
	"github.com/conflict-driven-devs/headquarter/internal/models"
	"github.com/conflict-driven-devs/headquarter/internal/routes"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func connectDatabaseWithRetry() *gorm.DB {
	dbConfig, err := config.LoadDBConfig()
	if err != nil {
		log.Fatalf("error loading DB config: %v", err)
	}

	var DB *gorm.DB
	for attempt := 1; attempt <= 10; attempt++ {
		log.Printf("Connecting to database (attempt %d/10)...", attempt)
		DB, err = db.Connect(
			dbConfig.User,
			dbConfig.Host,
			dbConfig.Password,
			dbConfig.Name,
			dbConfig.Port,
		)
		if err == nil && DB != nil {
			return DB
		}
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("unable to connect to database with default configuration")
	return nil
}

func main() {
	DB := connectDatabaseWithRetry()
	models.AppState = models.NoAdmin

	setupHandler := handler.NewSetupHandler(DB)
	authHandlers := handler.NewAuthHandler(DB)
	instanceHandler := handler.NewInstanceHandler(DB)
	projectHandlers := handler.NewProjectHandler(DB)
	userHandler := handler.NewUserHandler(DB)
	webHandler := handler.NewWebHandler()
	profileHandlers := handler.NewProfileHandler(DB)
	analyticsHandlers := handler.NewAnalyticsHandler(DB)
	apiKeyHandler := handler.NewAPIKeyHandler(DB)

	setupHandler.SetHandlerRegistry(&handler.HandlerRegistry{
		Auth:     authHandlers,
		Instance: instanceHandler,
		Project:  projectHandlers,
		User:     userHandler,
		Profile:  profileHandlers,
	})

	router := mux.NewRouter()

	router.Use(middleware.Recover)
	router.Use(middleware.Logging)
	router.Use(middleware.AppState)

	routes.SetupRoutes(router, authHandlers, instanceHandler, projectHandlers, webHandler, userHandler, setupHandler, profileHandlers, analyticsHandlers, apiKeyHandler, DB)

	fmt.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
