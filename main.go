package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/helioschat/sync/internal/config"
	"github.com/helioschat/sync/internal/database"
	"github.com/helioschat/sync/internal/handlers"
	"github.com/helioschat/sync/internal/middleware"
	"github.com/helioschat/sync/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.NewRedisClient(cfg.RedisURL, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer db.Close()

	// Initialize services
	authService := services.NewAuthService(cfg.JWTSecret, db) // Added db argument
	syncService := services.NewSyncService(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	syncHandler := handlers.NewSyncHandler(syncService, authService)

	// Setup router
	router := setupRouter(cfg, authHandler, syncHandler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRouter(cfg *config.Config, authHandler *handlers.AuthHandler, syncHandler *handlers.SyncHandler) *gin.Engine {
	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(cfg.CORSOrigins))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API versioning
	v1 := router.Group("/api/v1")
	{
		// Authentication endpoints
		auth := v1.Group("/auth")
		{
			auth.POST("/generate-wallet", authHandler.GenerateWallet)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected sync endpoints
		sync := v1.Group("/sync")
		sync.Use(middleware.RequireAuth(authHandler.AuthService))
		{
			// Thread endpoints
			sync.GET("/threads", syncHandler.GetThreads)
			sync.PUT("/threads/:id", syncHandler.UpsertThread)
			sync.DELETE("/threads/:id", syncHandler.DeleteThread)

			// Message endpoints
			sync.GET("/messages", syncHandler.GetMessages)
			sync.POST("/messages", syncHandler.CreateMessage)
			sync.PUT("/messages/:id", syncHandler.UpdateMessage)
			sync.DELETE("/messages/:id", syncHandler.DeleteMessage)

			// User settings endpoints
			sync.GET("/provider-instances", syncHandler.GetProviderInstances)
			sync.PUT("/provider-instances", syncHandler.UpdateProviderInstances)

			sync.GET("/disabled-models", syncHandler.GetDisabledModels)
			sync.PUT("/disabled-models", syncHandler.UpdateDisabledModels)

			sync.GET("/advanced-settings", syncHandler.GetAdvancedSettings)
			sync.PUT("/advanced-settings", syncHandler.UpdateAdvancedSettings)

			sync.GET("/changes-since/:timestamp", syncHandler.GetChangesSince)
		}
	}

	return router
}
