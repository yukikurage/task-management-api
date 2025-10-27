package main

import (
	"log"

	"github.com/gin-contrib/sessions"
	redisStore "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/config"
	"github.com/yukikurage/task-management-api/internal/database"
	"github.com/yukikurage/task-management-api/internal/handlers"
	"github.com/yukikurage/task-management-api/internal/middleware"
	"github.com/yukikurage/task-management-api/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Gin router
	r := gin.Default()

	// Setup session middleware with Redis
	redisAddr := cfg.RedisHost + ":" + cfg.RedisPort
	store, err := redisStore.NewStore(
		10,        // Redis pool size
		"tcp",     // network type
		redisAddr, // Redis address from config
		"",        // username (empty for default user)
		"",        // password (empty = no password)
		[]byte(cfg.SessionSecret), // authentication key
	)
	if err != nil {
		log.Fatalf("Failed to create Redis store: %v", err)
	}
	// Configure session options based on environment
	isProduction := cfg.GinMode == "release"
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   isProduction,       // true in production (HTTPS), false in development
		SameSite: 2,                  // SameSite=Lax (1=Strict, 2=Lax, 3=None)
	})
	r.Use(sessions.Sessions("task_session", store))

	// Initialize AI service
	var aiService *services.AIService
	if cfg.OpenAIAPIKey != "" {
		aiService = services.NewAIService(cfg.OpenAIAPIKey)
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler()
	taskHandler := handlers.NewTaskHandler(aiService)
	orgHandler := handlers.NewOrganizationHandler()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Task Management API is running",
		})
	})

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/signup", authHandler.Signup)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", middleware.RequireAuth(), authHandler.GetCurrentUser)
		}

		// Organization routes (protected)
		orgs := api.Group("/organizations")
		orgs.Use(middleware.RequireAuth())
		{
			orgs.POST("", orgHandler.CreateOrganization)
			orgs.GET("", orgHandler.ListOrganizations)
			orgs.POST("/join", orgHandler.JoinOrganization)
			orgs.GET("/:id", middleware.RequireOrganizationAccess(), orgHandler.GetOrganization)
			orgs.PUT("/:id", middleware.RequireOrganizationAccess(), middleware.RequireOrganizationOwner(), orgHandler.UpdateOrganization)
			orgs.DELETE("/:id", middleware.RequireOrganizationAccess(), middleware.RequireOrganizationOwner(), orgHandler.DeleteOrganization)
			orgs.POST("/:id/regenerate-code", middleware.RequireOrganizationAccess(), middleware.RequireOrganizationOwner(), orgHandler.RegenerateInviteCode)
			orgs.DELETE("/:id/members/:user_id", middleware.RequireOrganizationAccess(), middleware.RequireOrganizationOwner(), orgHandler.RemoveMember)
		}

		// Task routes (protected)
		tasks := api.Group("/tasks")
		tasks.Use(middleware.RequireAuth())
		{
			tasks.GET("", taskHandler.ListTasks)
			tasks.POST("", taskHandler.CreateTask)
			tasks.POST("/generate", taskHandler.GenerateTasks)
			tasks.GET("/:id", middleware.RequireTaskAccess(), taskHandler.GetTask)
			tasks.PATCH("/:id", middleware.RequireTaskAccess(), taskHandler.UpdateTask)
			tasks.DELETE("/:id", middleware.RequireTaskAccess(), taskHandler.DeleteTask)
			tasks.POST("/:id/assign", middleware.RequireTaskAccess(), taskHandler.AssignTask)
			tasks.POST("/:id/unassign", middleware.RequireTaskAccess(), taskHandler.UnassignTask)
		}
	}

	// Start server
	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
