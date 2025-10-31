package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"postgresus-backend/internal/config"
	"postgresus-backend/internal/downdetect"
	"postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/disk"
	healthcheck_attempt "postgresus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "postgresus-backend/internal/features/healthcheck/config"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/features/restores"
	"postgresus-backend/internal/features/storages"
	system_healthcheck "postgresus-backend/internal/features/system/healthcheck"
	"postgresus-backend/internal/features/users"
	env_utils "postgresus-backend/internal/util/env"
	files_utils "postgresus-backend/internal/util/files"
	"postgresus-backend/internal/util/logger"
	_ "postgresus-backend/swagger" // swagger docs

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Postgresus Backend API
// @version 1.0
// @description API for Postgresus
// @termsOfService http://swagger.io/terms/

// @host localhost:4005
// @BasePath /api/v1
// @schemes http
func main() {
	log := logger.GetLogger()

	runMigrations(log)

	// create directories that used for backups and restore
	err := files_utils.EnsureDirectories([]string{
		config.GetEnv().TempFolder,
		config.GetEnv().DataFolder,
	})

	if err != nil {
		log.Error("Failed to ensure directories", "error", err)
		os.Exit(1)
	}

	// Handle password reset if flag is provided
	newPassword := flag.String("new-password", "", "Set a new password for the user")
	flag.Parse()
	if *newPassword != "" {
		resetPassword(*newPassword, log)
	}

	go generateSwaggerDocs(log)

	gin.SetMode(gin.ReleaseMode)
	ginApp := gin.Default()

	// Add GZIP compression middleware
	ginApp.Use(gzip.Gzip(
		gzip.DefaultCompression,
		// Don't compress already compressed files
		gzip.WithExcludedExtensions(
			[]string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".svg", ".pdf", ".mp4"},
		),
	))

	enableCors(ginApp)
	setUpRoutes(ginApp)
	setUpDependencies()
	runBackgroundTasks(log)
	mountFrontend(ginApp)

	startServerWithGracefulShutdown(log, ginApp)
}

func resetPassword(newPassword string, log *slog.Logger) {
	log.Info("Resetting password...")

	userService := users.GetUserService()
	err := userService.ChangePassword(newPassword)
	if err != nil {
		log.Error("Failed to reset password", "error", err)
		os.Exit(1)
	}

	log.Info("Password reset successfully")
	os.Exit(0)
}

func startServerWithGracefulShutdown(log *slog.Logger, app *gin.Engine) {
	host := ""
	if config.GetEnv().EnvMode == env_utils.EnvModeDevelopment {
		// for dev we use localhost to avoid firewall
		// requests on each run for Windows
		host = "127.0.0.1"
	}

	srv := &http.Server{
		Addr:    host + ":4005",
		Handler: app,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen:", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("Shutdown signal received")

	// The context is used to inform the server it has 10 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown:", "error", err)
	}

	log.Info("Server gracefully stopped")
}

func setUpRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	// Mount Swagger UI
	v1.GET("/docs/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	downdetectContoller := downdetect.GetDowndetectController()
	userController := users.GetUserController()
	notifierController := notifiers.GetNotifierController()
	storageController := storages.GetStorageController()
	databaseController := databases.GetDatabaseController()
	backupController := backups.GetBackupController()
	restoreController := restores.GetRestoreController()
	healthcheckController := system_healthcheck.GetHealthcheckController()
	healthcheckConfigController := healthcheck_config.GetHealthcheckConfigController()
	healthcheckAttemptController := healthcheck_attempt.GetHealthcheckAttemptController()
	diskController := disk.GetDiskController()
	backupConfigController := backups_config.GetBackupConfigController()

	downdetectContoller.RegisterRoutes(v1)
	userController.RegisterRoutes(v1)
	notifierController.RegisterRoutes(v1)
	storageController.RegisterRoutes(v1)
	databaseController.RegisterRoutes(v1)
	backupController.RegisterRoutes(v1)
	restoreController.RegisterRoutes(v1)
	healthcheckController.RegisterRoutes(v1)
	diskController.RegisterRoutes(v1)
	healthcheckConfigController.RegisterRoutes(v1)
	healthcheckAttemptController.RegisterRoutes(v1)
	backupConfigController.RegisterRoutes(v1)
}

func setUpDependencies() {
	backups.SetupDependencies()
	restores.SetupDependencies()
	healthcheck_config.SetupDependencies()
}

func runBackgroundTasks(log *slog.Logger) {
	log.Info("Preparing to run background tasks...")

	err := files_utils.CleanFolder(config.GetEnv().TempFolder)
	if err != nil {
		log.Error("Failed to clean temp folder", "error", err)
	}

	go runWithPanicLogging(log, "backup background service", func() {
		backups.GetBackupBackgroundService().Run()
	})

	go runWithPanicLogging(log, "restore background service", func() {
		restores.GetRestoreBackgroundService().Run()
	})

	go runWithPanicLogging(log, "healthcheck attempt background service", func() {
		healthcheck_attempt.GetHealthcheckAttemptBackgroundService().Run()
	})
}

func runWithPanicLogging(log *slog.Logger, serviceName string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic in "+serviceName, "error", r)
		}
	}()
	fn()
}

// Keep in mind: docs appear after second launch, because Swagger
// is generated into Go files. So if we changed files, we generate
// new docs, but still need to restart the server to see them.
func generateSwaggerDocs(log *slog.Logger) {
	if config.GetEnv().EnvMode == env_utils.EnvModeProduction {
		return
	}

	// Run swag from the current directory instead of parent
	// Use the current directory as the base for swag init
	// This ensures swag can find the files regardless of where the command is run from
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error("Failed to get current directory", "error", err)
		return
	}

	cmd := exec.Command("swag", "init", "-d", currentDir, "-g", "cmd/main.go", "-o", "swagger")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Failed to generate Swagger docs", "error", err, "output", string(output))
		return
	}

	log.Info("Swagger documentation generated successfully")
}

func runMigrations(log *slog.Logger) {
	log.Info("Running database migrations...")

	cmd := exec.Command("goose", "up")
	cmd.Env = append(
		os.Environ(),
		"GOOSE_DRIVER=postgres",
		"GOOSE_DBSTRING="+config.GetEnv().DatabaseDsn,
	)

	// Set the working directory to where migrations are located
	cmd.Dir = "./migrations"

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Failed to run migrations", "error", err, "output", string(output))
		os.Exit(1)
	}

	log.Info("Database migrations completed successfully", "output", string(output))
}

func enableCors(ginApp *gin.Engine) {
	if config.GetEnv().EnvMode == env_utils.EnvModeDevelopment {
		// Setup CORS
		ginApp.Use(cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders: []string{
				"Origin",
				"Content-Length",
				"Content-Type",
				"Authorization",
				"Accept",
				"Accept-Language",
				"Accept-Encoding",
				"Access-Control-Request-Method",
				"Access-Control-Request-Headers",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
				"Access-Control-Allow-Origin",
			},
			AllowCredentials: true,
		}))
	}
}

func mountFrontend(ginApp *gin.Engine) {
	staticDir := "./ui/build"
	ginApp.NoRoute(func(c *gin.Context) {
		path := filepath.Join(staticDir, c.Request.URL.Path)

		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			c.File(path)
			return
		}

		c.File(filepath.Join(staticDir, "index.html"))
	})
}
