package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"jarvist/internal/syncmanager/api"
	"jarvist/internal/syncmanager/interfaces"
	"jarvist/internal/syncmanager/mqtt"
	"jarvist/internal/syncmanager/services/cleanup"
	logService "jarvist/internal/syncmanager/services/log"
	"jarvist/internal/syncmanager/services/message"
	"jarvist/internal/syncmanager/services/stats"
	syncService "jarvist/internal/syncmanager/sync"
	"jarvist/pkg/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"jarvist/internal/common/buildinfo"
	baseConfig "jarvist/internal/common/config"
	"jarvist/internal/common/database"
	"jarvist/internal/syncmanager/config"
)

var (
	buildMode   = "development"
	isInstall   = flag.Bool("install", false, "Install as Windows Service")
	isUninstall = flag.Bool("uninstall", false, "Uninstall Windows Service")
	isService   = flag.Bool("service", false, "Run as Windows Service")
	isStart     = flag.Bool("start", false, "Start Windows Service")
	isStop      = flag.Bool("stop", false, "Stop Windows Service")
	isRestart   = flag.Bool("restart", false, "Restart Windows Service")
	isStatus    = flag.Bool("status", false, "Get Windows Service status")
	isDebug     = flag.Bool("debug", false, "Run with debug logging")
)

const (
	ComponentMain = "main"
	startupWait   = 5 * time.Second
	shutdownWait  = 10 * time.Second
)

func main() {
	flag.Parse()

	buildInfoService := buildinfo.NewBuildInfoService()

	baseConfig, err := baseConfig.LoadConfig(buildMode, buildInfoService)
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	appConfig := config.LoadConfig(buildMode, baseConfig)

	// Create root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup logger with debug level if requested
	logOptions := logger.DefaultOptions()
	logOptions.LogDir = baseConfig.LogDir
	logOptions.EnableMQTT = appConfig.Logger.EnableMQTTLogs
	logOptions.MQTTTopic = appConfig.MQTT.Topic + "/logs"

	if *isDebug {
		logOptions.Level = logger.LevelDebug
		logOptions.MQTTMinLevel = logger.LevelInfo
		logOptions.FileMinLevel = logger.LevelInfo
	} else {
		logOptions.Level = logger.LevelInfo
		logOptions.MQTTMinLevel = logger.LevelWarn
		logOptions.FileMinLevel = logger.LevelWarn
	}

	appLogger := logger.New(logOptions)
	mainLogger := appLogger.WithComponent("syncmanager")

	mainLogger.Info("Starting Jarvist Sync Manager v%s in %s mode",
		buildInfoService.LoadBuildInfo().ProductVersion,
		buildMode)

	// Set service configuration
	SetConfig(appConfig)

	// Handle service commands (install, uninstall, start, stop, etc.)
	if handleServiceCommands(mainLogger) {
		return
	}

	// Create database connection
	mainLogger.Info("Initializing database...")
	err = database.SetupDatabase(baseConfig, appLogger.WithComponent("database"))
	if err != nil {
		mainLogger.Fatal("Failed to create database: %v", err)
	}

	// Run database migrations
	mainLogger.Info("Running database migrations...")
	if err := database.RunMigrations(appLogger.WithComponent("database")); err != nil {
		mainLogger.Fatal("Failed to run migrations: %v", err)
	}

	// Create services and components
	mainLogger.Info("Initializing services...")
	db := database.GetDB()
	messageService := message.NewMessageService(db, appLogger)
	logSvc := logService.NewLogService(db, baseConfig, appLogger, baseConfig.LogDir, 10)
	statsService := stats.NewStatsService(db, logSvc)

	// Create MQTT sender without starting it
	mainLogger.Info("Creating MQTT sender...")
	mqttSender, err := mqtt.NewSender(ctx, appConfig, db, messageService, statsService, appLogger)
	if err != nil {
		mainLogger.Fatal("Failed to create MQTT sender: %v", err)
	}

	// Initialize synchronizer
	mainLogger.Info("Creating synchronizer...")
	synchronizer := syncService.NewSynchronizer(appConfig, appLogger, db, mqttSender)

	// Create cleanup service
	mainLogger.Info("Creating cleanup service...")
	cleanupConfig := cleanup.DefaultConfig()
	cleanupConfig.DataDirectory = baseConfig.ServicesDataDir
	cleanupService := cleanup.NewCleanupService(db, appLogger, logSvc, cleanupConfig)

	// Initialize API server
	mainLogger.Info("Creating API server...")
	apiServer := api.NewServer(
		appConfig,
		appLogger,
		synchronizer,
		mqttSender,
		messageService,
		statsService,
		logSvc,
		cleanupService,
	)

	// Set up signal handling
	setupSignalHandling(ctx, cancel, mainLogger)

	// Prepare service components
	components := []interfaces.ServiceComponent{
		mqttSender,
		synchronizer,
		cleanupService,
	}

	// Run as service or interactively
	if *isService {
		mainLogger.Info("Running as Windows service")
		if err := Run(components, apiServer, appLogger); err != nil {
			mainLogger.Fatal("Service error: %v", err)
		}
		return
	}

	// If we're running interactively, start all components manually
	mainLogger.Info("Running in interactive mode")

	// Start components in sequence with proper error handling
	for i, component := range components {
		componentName := fmt.Sprintf("Component %d", i+1)

		// Try to get a more descriptive name if available
		if named, ok := component.(interface{ Name() string }); ok {
			componentName = named.Name()
		}

		mainLogger.Info("Starting component: %s", componentName)
		if err := component.Start(); err != nil {
			mainLogger.Fatal("Failed to start component %s: %v", componentName, err)
		}
		mainLogger.Info("Component started successfully: %s", componentName)
	}

	// Start API server in its own goroutine
	apiServerStarted := make(chan struct{})
	apiErrorChan := make(chan error, 1)
	var apiWg sync.WaitGroup
	apiWg.Add(1)

	go func() {
		defer apiWg.Done()
		mainLogger.Info("Starting API server on port %d", appConfig.API.Port)

		// Signal that we're trying to start the API server
		close(apiServerStarted)

		if err := apiServer.Start(appConfig.API.Port); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				mainLogger.Error("API server error: %v", err)
				apiErrorChan <- err
			} else {
				mainLogger.Info("API server shutdown gracefully")
			}
		}
	}()

	// Wait briefly for API server to start
	select {
	case <-apiServerStarted:
		mainLogger.Info("API server initialization complete")
	case err := <-apiErrorChan:
		mainLogger.Fatal("API server failed to start: %v", err)
	case <-time.After(startupWait):
		mainLogger.Warning("API server initialization taking longer than expected")
	}

	mainLogger.Info("All components started successfully")
	mainLogger.Info("Application is now running. Press Ctrl+C to exit")

	// Wait for context cancellation when running interactively
	<-ctx.Done()

	// Graceful shutdown
	mainLogger.Info("Shutting down...")

	// Create a context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownWait)
	defer shutdownCancel()

	// Stop API server
	mainLogger.Info("Stopping API server...")
	if err := apiServer.Stop(); err != nil {
		mainLogger.Error("Error stopping API server: %v", err)
	}

	// Wait for API server to stop
	apiWaitChan := make(chan struct{})
	go func() {
		apiWg.Wait()
		close(apiWaitChan)
	}()

	select {
	case <-apiWaitChan:
		mainLogger.Info("API server stopped successfully")
	case <-shutdownCtx.Done():
		mainLogger.Warning("Timeout waiting for API server to stop")
	}

	// Stop all components in reverse order
	for i := len(components) - 1; i >= 0; i-- {
		componentName := fmt.Sprintf("Component %d", i+1)
		if named, ok := components[i].(interface{ Name() string }); ok {
			componentName = named.Name()
		}

		mainLogger.Info("Stopping component: %s", componentName)
		if err := components[i].Stop(); err != nil {
			mainLogger.Error("Error stopping component %s: %v", componentName, err)
		} else {
			mainLogger.Info("Component %s stopped successfully", componentName)
		}
	}

	// Close database connection
	mainLogger.Info("Closing database connection...")
	database.CloseDatabase()

	mainLogger.Info("Shutdown complete")
}

// handleServiceCommands handles the service-related command line flags.
// Returns true if a service command was handled, false otherwise.
func handleServiceCommands(logger *logger.ContextLogger) bool {
	if *isInstall {
		logger.Info("Installing service...")
		if err := Install(); err != nil {
			logger.Fatal("Failed to install service: %v", err)
		}
		logger.Info("Service installed successfully")
		return true
	}

	if *isUninstall {
		logger.Info("Uninstalling service...")
		if err := Uninstall(); err != nil {
			logger.Fatal("Failed to uninstall service: %v", err)
		}
		logger.Info("Service uninstalled successfully")
		return true
	}

	if *isStart {
		logger.Info("Setting up database before starting service...")
		if err := StartService(); err != nil {
			logger.Fatal("Failed to start service: %v", err)
		}
		logger.Info("Service started successfully")
		return true
	}

	if *isStop {
		logger.Info("Stopping service...")
		if err := StopService(); err != nil {
			logger.Fatal("Failed to stop service: %v", err)
		}
		logger.Info("Service stopped successfully")
		return true
	}

	if *isRestart {
		logger.Info("Restarting service...")
		if err := RestartService(); err != nil {
			logger.Fatal("Failed to restart service: %v", err)
		}
		logger.Info("Service restarted successfully")
		return true
	}

	if *isStatus {
		status, err := GetServiceStatus()
		if err != nil {
			logger.Fatal("Failed to get service status: %v", err)
		}
		logger.Info("Service status: %s", status)
		return true
	}

	return false
}

func setupSignalHandling(ctx context.Context, cancel context.CancelFunc, logger *logger.ContextLogger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			logger.Info("Received signal: %v", sig)
			cancel()
		case <-ctx.Done():
		}
	}()
}
