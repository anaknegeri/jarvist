package api

import (
	"fmt"
	"jarvist/internal/common/models"
	"jarvist/internal/syncmanager/config"
	"jarvist/internal/syncmanager/mqtt"
	"jarvist/internal/syncmanager/services/cleanup"
	"jarvist/internal/syncmanager/services/log"
	"jarvist/internal/syncmanager/services/message"
	"jarvist/internal/syncmanager/services/stats"
	"jarvist/internal/syncmanager/sync"
	"jarvist/pkg/logger"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLog "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type Server struct {
	app            *fiber.App
	cfg            *config.Config
	logger         *logger.Logger
	synchronizer   *sync.Synchronizer
	mqttSender     *mqtt.Sender
	messageService *message.MessageService
	statsService   *stats.StatsService
	logService     *log.LogService
	cleanupService *cleanup.CleanupService
}

type LogRequest struct {
	Level     string `json:"level"`
	Component string `json:"component"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp,omitempty"`
	AppID     string `json:"app_id,omitempty"`
	Source    string `json:"source,omitempty"`
}

type BatchLogRequest struct {
	Logs []LogRequest `json:"logs"`
}

type LogResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	LogID   int64  `json:"log_id,omitempty"`
	Count   int    `json:"count,omitempty"`
}

// NewServer creates a new API server instance
func NewServer(
	cfg *config.Config,
	logger *logger.Logger,
	synchronizer *sync.Synchronizer,
	mqttSender *mqtt.Sender,
	messageService *message.MessageService,
	statsService *stats.StatsService,
	logService *log.LogService,
	cleanupService *cleanup.CleanupService,
) *Server {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			c.Set("Content-Type", "application/json")
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(fiberLog.New(fiberLog.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))

	if cfg.API.Username != "" && cfg.API.Password != "" {
		app.Use(basicauth.New(basicauth.Config{
			Users: map[string]string{
				cfg.API.Username: cfg.API.Password,
			},
		}))
	}

	server := &Server{
		app:            app,
		cfg:            cfg,
		logger:         logger,
		synchronizer:   synchronizer,
		mqttSender:     mqttSender,
		messageService: messageService,
		statsService:   statsService,
		logService:     logService,
	}

	server.registerRoutes()

	return server
}

// Start starts the API server
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	s.logger.Info("API", "Starting API server on %s", addr)

	if s.cfg.API.EnableTLS && s.cfg.API.CertFile != "" && s.cfg.API.KeyFile != "" {
		return s.app.ListenTLS(addr, s.cfg.API.CertFile, s.cfg.API.KeyFile)
	}

	return s.app.Listen(addr)
}

// Stop stops the API server
func (s *Server) Stop() error {
	s.logger.Info("API", "Stopping API server")

	done := make(chan struct{})

	var shutdownErr error
	go func() {
		shutdownErr = s.app.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		if shutdownErr != nil {
			s.logger.Error("API", "Error during API server shutdown: %v", shutdownErr)
			return shutdownErr
		}
		s.logger.Info("API", "API server shutdown completed gracefully")
	case <-time.After(15 * time.Second):
		s.logger.Warning("API", "API server shutdown timed out after 15 seconds, server may not have fully terminated")
		return fmt.Errorf("API server shutdown timed out")
	}

	return shutdownErr
}

// registerRoutes sets up all API routes
func (s *Server) registerRoutes() {
	// API version group
	api := s.app.Group("/api")

	// Status endpoints
	api.Get("/status", s.getStatus)
	api.Get("/health", s.getHealth)

	// Synchronizer endpoints
	sync := api.Group("/sync")
	sync.Get("/status", s.getSyncStatus)
	sync.Post("/start", s.startSync)
	sync.Get("/folders", s.getSyncFolders)
	sync.Post("/folders/:folder/resync", s.resyncFolder)
	sync.Post("/summary", s.sendSyncSummary)
	sync.Get("/files/:folder/:filename", s.getFileStatus) // Added endpoint for file status

	// MQTT endpoints
	mqtt := api.Group("/mqtt")
	mqtt.Get("/status", s.getMQTTStatus)
	mqtt.Post("/test", s.sendTestMessage)
	mqtt.Get("/queue", s.getQueueStatus)
	mqtt.Post("/queue/drain", s.drainQueue)
	mqtt.Get("/stats", s.getMQTTStats)
	mqtt.Post("/refresh", s.refreshMQTT)

	// Message endpoints
	messages := api.Group("/messages")
	messages.Get("/pending", s.getPendingMessages)
	messages.Get("/count", s.getMessageCount)
	messages.Post("/:id/resend", s.resendMessage)

	logs := api.Group("/logs")
	logs.Get("/", s.getLogs)
	logs.Post("/", s.createLog)
	logs.Post("/batch", s.createBatchLogs)
	logs.Get("/stats", s.getLogStats)

	cleanupGroup := api.Group("/cleanup")
	cleanupGroup.Get("/status", s.getCleanupStatus)
	cleanupGroup.Post("/run", s.runCleanup)
	cleanupGroup.Put("/config", s.updateCleanupConfig)
}

// getStatus returns the overall system status
func (s *Server) getStatus(c *fiber.Ctx) error {
	status := map[string]interface{}{
		"service": "running",
		"time":    time.Now().Format(time.RFC3339),
		"uptime":  time.Since(time.Now()), // This should be replaced with actual service start time
		"mqtt":    s.mqttSender.GetStatus(),
	}

	return c.JSON(status)
}

// getHealth returns a simple health check response
func (s *Server) getHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// getSyncStatus returns the synchronizer status
func (s *Server) getSyncStatus(c *fiber.Ctx) error {
	status := s.synchronizer.GetStatus()
	return c.JSON(status)
}

// startSync triggers a manual synchronization
func (s *Server) startSync(c *fiber.Ctx) error {
	go func() {
		s.synchronizer.SyncData() // Assuming this method is exported from the synchronizer
	}()

	return c.JSON(fiber.Map{
		"status":  "sync_started",
		"message": "Synchronization process started",
	})
}

// getSyncFolders returns all synchronized folders
func (s *Server) getSyncFolders(c *fiber.Ctx) error {
	detailed := c.QueryBool("detailed", false)

	if detailed {
		folders, err := s.synchronizer.GetSyncedFoldersDetails()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get synced folders: "+err.Error())
		}
		return c.JSON(fiber.Map{
			"folders": folders,
		})
	} else {
		folders, err := s.synchronizer.GetSyncedFoldersList()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get synced folders: "+err.Error())
		}
		return c.JSON(fiber.Map{
			"folders": folders,
		})
	}
}

// resyncFolder forces a resync of a specific folder
func (s *Server) resyncFolder(c *fiber.Ctx) error {
	folder := c.Params("folder")
	if folder == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Folder parameter is required")
	}

	err := s.synchronizer.ResyncFolder(folder)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to resync folder: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "resync_started",
		"folder":  folder,
		"message": "Folder resync started",
	})
}

// getFileStatus returns processing status for a specific file
func (s *Server) getFileStatus(c *fiber.Ctx) error {
	folder := c.Params("folder")
	filename := c.Params("filename")

	if folder == "" || filename == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Folder and filename parameters are required")
	}

	status, err := s.synchronizer.GetFileProcessingStatus(filename, folder)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get file status: "+err.Error())
	}

	return c.JSON(status)
}

// sendSyncSummary sends a sync summary via MQTT
func (s *Server) sendSyncSummary(c *fiber.Ctx) error {
	err := s.synchronizer.SendSyncFolderSummary() // Assuming this method is implemented
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to send sync summary: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "summary_sent",
		"message": "Sync summary sent to MQTT",
	})
}

// getMQTTStatus returns the MQTT connection status
func (s *Server) getMQTTStatus(c *fiber.Ctx) error {
	return c.JSON(s.mqttSender.GetStatus())
}

// sendTestMessage sends a test message via MQTT
func (s *Server) sendTestMessage(c *fiber.Ctx) error {
	var request struct {
		Topic   string      `json:"topic"`
		Payload interface{} `json:"payload"`
	}

	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	// If no payload provided, create a test payload
	if request.Payload == nil {
		request.Payload = map[string]interface{}{
			"type":      "test",
			"timestamp": time.Now().Format(time.RFC3339),
			"random_id": fmt.Sprintf("test_%d", time.Now().UnixNano()%10000),
		}
	}

	// If no topic provided, use default with test suffix
	if request.Topic == "" {
		request.Topic = s.cfg.MQTT.Topic + "/test"
	}

	messageID, err := s.mqttSender.SendData(request.Topic, request.Payload)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to send test message: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":     "message_sent",
		"message_id": messageID,
		"topic":      request.Topic,
	})
}

// getQueueStatus returns the MQTT message queue status
func (s *Server) getQueueStatus(c *fiber.Ctx) error {
	// Assuming a GetQueueStatus method exists in the sender
	status := map[string]interface{}{
		"time": time.Now().Format(time.RFC3339),
	}

	// Add queue stats from the message service
	pendingCount, err := s.messageService.CountPendingMessages()
	if err != nil {
		s.logger.Error("API", "Failed to get pending message count: %v", err)
	} else {
		status["pending_messages"] = pendingCount
	}

	// Get additional status from the MQTT sender
	mqttStatus := s.mqttSender.GetStatus()
	for k, v := range mqttStatus {
		if k == "channel_queue_len" || k == "backing_queue_len" || k == "total_queued" {
			status[k] = v
		}
	}

	return c.JSON(status)
}

// drainQueue manually drains the message queue
func (s *Server) drainQueue(c *fiber.Ctx) error {
	// Assuming a DrainQueues method exists
	// s.mqttSender.DrainQueues()

	return c.JSON(fiber.Map{
		"status":  "queue_drain_started",
		"message": "Queue drain process started",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// getMQTTStats returns MQTT statistics
func (s *Server) getMQTTStats(c *fiber.Ctx) error {
	stats, err := s.statsService.GetDatabaseStats()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get statistics: "+err.Error())
	}

	// Add timestamps and enhance with more data
	enhancedStats := map[string]interface{}{
		"time":           time.Now().Format(time.RFC3339),
		"database_stats": stats,
	}

	// Get message processing stats from MQTT sender
	mqttStatus := s.mqttSender.GetStatus()
	for k, v := range mqttStatus {
		if k == "messages_processed" || k == "processing_rate" || k == "avg_processing_time" {
			enhancedStats[k] = v
		}
	}

	return c.JSON(enhancedStats)
}

// refreshMQTT forces a reconnection of the MQTT client
func (s *Server) refreshMQTT(c *fiber.Ctx) error {
	err := s.mqttSender.Refresh()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to refresh MQTT connection: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "mqtt_refreshed",
		"message": "MQTT connection refreshed",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// getPendingMessages gets pending messages from the queue
func (s *Server) getPendingMessages(c *fiber.Ctx) error {
	limitStr := c.Query("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	messages, err := s.messageService.GetPendingMessages(limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get pending messages: "+err.Error())
	}

	// Create a simplified view of the messages
	simplifiedMessages := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		simplifiedMessages = append(simplifiedMessages, map[string]interface{}{
			"id":          msg.ID,
			"topic":       msg.Topic,
			"timestamp":   msg.Timestamp.Format(time.RFC3339),
			"connected":   msg.ConnectionState,
			"retry_count": msg.RetryCount,
		})
	}

	return c.JSON(fiber.Map{
		"count":    len(messages),
		"messages": simplifiedMessages,
	})
}

// getMessageCount gets the count of messages in different states
func (s *Server) getMessageCount(c *fiber.Ctx) error {
	pendingCount, err := s.messageService.CountPendingMessages()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to count pending messages: "+err.Error())
	}

	// You could add more counts here (sent messages, failed messages, etc.)

	return c.JSON(fiber.Map{
		"pending": pendingCount,
	})
}

// resendMessage forces a resend of a specific message
func (s *Server) resendMessage(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Message ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid message ID: "+err.Error())
	}

	// Get the message from database
	messages, err := s.messageService.GetPendingMessages(1) // This is not ideal - we should have a GetMessageByID method
	if err != nil || len(messages) == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Message not found")
	}

	// Verify we found the right message
	var message models.PendingMessage
	for _, msg := range messages {
		if msg.ID == uint(id) {
			message = msg
			break
		}
	}

	if message.ID == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Message not found")
	}

	// Resend the message
	newMessageID, err := s.mqttSender.SendData(message.Topic, message.Payload)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to resend message: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":         "message_resent",
		"original_id":    id,
		"new_message_id": newMessageID,
		"topic":          message.Topic,
	})
}

func (s *Server) getLogs(c *fiber.Ctx) error {
	level := c.Query("level", "")
	component := c.Query("component", "")
	limitStr := c.Query("limit", "100")
	offsetStr := c.Query("offset", "0")
	startTime := c.Query("start_time", "")
	endTime := c.Query("end_time", "")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	logs, err := s.logService.GetLogs(level, component, limit, offset, startTime, endTime)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get logs: "+err.Error())
	}

	formattedLogs := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		formattedLogs[i] = map[string]interface{}{
			"id":        log.ID,
			"timestamp": log.Timestamp.Format(time.RFC3339),
			"level":     log.Level,
			"component": log.Component,
			"message":   log.Message,
		}
	}

	return c.JSON(fiber.Map{
		"logs":  formattedLogs,
		"count": len(logs),
		"meta": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (s *Server) createLog(c *fiber.Ctx) error {
	var request LogRequest
	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	if request.Level == "" || request.Component == "" || request.Message == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Level, component, and message are required")
	}

	request.Level = strings.ToUpper(request.Level)
	validLevels := map[string]bool{"INFO": true, "WARN": true, "ERROR": true, "DEBUG": true}
	if !validLevels[request.Level] {
		request.Level = "INFO"
	}

	err := s.logService.LogMessage(request.Level, request.Component, request.Message)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to log message: "+err.Error())
	}

	return c.JSON(LogResponse{
		Status:  "success",
		Message: "Log entry created",
	})
}

func (s *Server) createBatchLogs(c *fiber.Ctx) error {
	var request BatchLogRequest
	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	if len(request.Logs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "No logs provided")
	}

	// Validate and normalize levels
	validLevels := map[string]bool{"INFO": true, "WARN": true, "ERROR": true, "DEBUG": true}

	successCount := 0
	for _, logEntry := range request.Logs {
		if logEntry.Level == "" || logEntry.Component == "" || logEntry.Message == "" {
			continue
		}

		// Normalize level
		logEntry.Level = strings.ToUpper(logEntry.Level)
		if !validLevels[logEntry.Level] {
			logEntry.Level = "INFO"
		}

		// Log the message
		err := s.logService.LogMessage(logEntry.Level, logEntry.Component, logEntry.Message)
		if err == nil {
			successCount++
		} else {
			s.logger.Error("API", "Failed to log message in batch: %v", err)
		}
	}

	return c.JSON(LogResponse{
		Status:  "success",
		Message: fmt.Sprintf("%d of %d log entries created", successCount, len(request.Logs)),
		Count:   successCount,
	})
}

func (s *Server) getLogStats(c *fiber.Ctx) error {
	stats, err := s.logService.GetStats()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get log statistics: "+err.Error())
	}

	return c.JSON(stats)
}

// getCleanupStatus returns the current status of the cleanup service
func (s *Server) getCleanupStatus(c *fiber.Ctx) error {
	if s.cleanupService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Cleanup service not available")
	}

	status := s.cleanupService.GetStatus()
	return c.JSON(status)
}

// runCleanup triggers an immediate cleanup
func (s *Server) runCleanup(c *fiber.Ctx) error {
	if s.cleanupService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Cleanup service not available")
	}

	if err := s.cleanupService.ForceCleanup(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to run cleanup: %v", err))
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Cleanup process started",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// updateCleanupConfig updates the cleanup configuration
func (s *Server) updateCleanupConfig(c *fiber.Ctx) error {
	if s.cleanupService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Cleanup service not available")
	}

	// Get current config
	currentConfig := s.cleanupService.GetStatus()

	// Parse request
	var request struct {
		Enabled                *bool   `json:"enabled"`
		IntervalHours          *int    `json:"interval_hours"`
		LogRetention           *int    `json:"log_retention_days"`
		MessageRetention       *int    `json:"message_retention_days"`
		ProcessedFileRetention *int    `json:"processed_file_retention_days"`
		SyncedFolderRetention  *int    `json:"synced_folder_retention_days"`
		MaxLogFiles            *int    `json:"max_log_files"`
		MaxPendingMessages     *int    `json:"max_pending_messages"`
		DataDirectory          *string `json:"data_directory"`
	}

	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
	}

	// Create new config based on current values
	newConfig := &cleanup.Config{
		Enabled:                currentConfig["enabled"].(bool),
		Interval:               parseDurationString(currentConfig["interval"].(string)),
		LogRetention:           currentConfig["retention"].(map[string]interface{})["logs"].(int),
		MessageRetention:       currentConfig["retention"].(map[string]interface{})["messages"].(int),
		ProcessedFileRetention: currentConfig["retention"].(map[string]interface{})["processed_files"].(int),
		SyncedFolderRetention:  currentConfig["retention"].(map[string]interface{})["synced_folders"].(int),
		MaxLogFiles:            currentConfig["limits"].(map[string]interface{})["max_log_files"].(int),
		MaxPendingMessages:     currentConfig["limits"].(map[string]interface{})["max_pending_messages"].(int),
		DataDirectory:          "./data", // Default
	}

	// Apply changes from request
	if request.Enabled != nil {
		newConfig.Enabled = *request.Enabled
	}

	if request.IntervalHours != nil {
		hours := *request.IntervalHours
		if hours < 1 {
			hours = 1 // Minimum 1 hour
		}
		newConfig.Interval = time.Duration(hours) * time.Hour
	}

	if request.LogRetention != nil {
		newConfig.LogRetention = *request.LogRetention
	}

	if request.MessageRetention != nil {
		newConfig.MessageRetention = *request.MessageRetention
	}

	if request.ProcessedFileRetention != nil {
		newConfig.ProcessedFileRetention = *request.ProcessedFileRetention
	}

	if request.SyncedFolderRetention != nil {
		newConfig.SyncedFolderRetention = *request.SyncedFolderRetention
	}

	if request.MaxLogFiles != nil {
		newConfig.MaxLogFiles = *request.MaxLogFiles
	}

	if request.MaxPendingMessages != nil {
		newConfig.MaxPendingMessages = *request.MaxPendingMessages
	}

	if request.DataDirectory != nil {
		newConfig.DataDirectory = *request.DataDirectory
	}

	// Update the configuration
	if err := s.cleanupService.UpdateConfig(newConfig); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Failed to update config: %v", err))
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Cleanup configuration updated",
		"config":  s.cleanupService.GetStatus(),
	})
}

// Helper function to parse a duration string
func parseDurationString(durationStr string) time.Duration {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// Default to 24 hours if parsing fails
		return 24 * time.Hour
	}
	return duration
}
