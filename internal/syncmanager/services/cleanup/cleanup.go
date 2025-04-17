package cleanup

import (
	"fmt"
	"jarvist/internal/common/models"
	"jarvist/internal/syncmanager/services/log"
	"jarvist/pkg/logger"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Config holds configuration for the cleanup service
type Config struct {
	// General
	Enabled  bool          // Whether cleanup is enabled
	Interval time.Duration // How often to run cleanup

	// Retention periods (in days)
	LogRetention           int // How many days of logs to keep
	MessageRetention       int // How many days of messages to keep
	ProcessedFileRetention int // How many days of processed files to keep
	SyncedFolderRetention  int // How many days of synced folders to keep if not fully synced

	// Limits
	MaxLogFiles        int // Maximum number of log files to keep
	MaxPendingMessages int // Maximum number of pending messages to keep

	// Paths
	DataDirectory string // Base directory for data files
}

// DefaultConfig returns the default cleanup configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:                true,
		Interval:               24 * time.Hour, // Daily cleanup
		LogRetention:           7,              // 7 days
		MessageRetention:       7,              // 7 days
		ProcessedFileRetention: 7,              // 7 days
		SyncedFolderRetention:  7,              // 7 days
		MaxLogFiles:            10,             // 10 log files
		MaxPendingMessages:     10000,          // 10,000 pending messages
		DataDirectory:          "./data",       // Default data directory
	}
}

// CleanupService handles automatic cleanup of old data
type CleanupService struct {
	db          *gorm.DB
	logger      *logger.Logger
	logService  *log.LogService
	config      *Config
	running     bool
	stopCh      chan struct{}
	lastCleanup time.Time
	mu          sync.Mutex
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(
	db *gorm.DB,
	logger *logger.Logger,
	logService *log.LogService,
	config *Config,
) *CleanupService {
	if config == nil {
		config = DefaultConfig()
	}

	return &CleanupService{
		db:          db,
		logger:      logger,
		logService:  logService,
		config:      config,
		stopCh:      make(chan struct{}),
		lastCleanup: time.Time{},
	}
}

// Start begins the periodic cleanup process
func (s *CleanupService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil // Already running
	}

	if !s.config.Enabled {
		s.logger.Info("cleanup", "Cleanup service is disabled")
		return nil
	}

	s.running = true

	go s.runCleanupLoop()

	s.logger.Info("cleanup", "Cleanup service started (interval: %v)", s.config.Interval)
	return nil
}

// Stop halts the cleanup process
func (s *CleanupService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil // Not running
	}

	close(s.stopCh)
	s.running = false

	s.logger.Info("cleanup", "Cleanup service stopped")
	return nil
}

// runCleanupLoop runs the cleanup loop until stopped
func (s *CleanupService) runCleanupLoop() {
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	// Run cleanup immediately on start
	s.runCleanup()

	for {
		select {
		case <-ticker.C:
			s.runCleanup()
		case <-s.stopCh:
			return
		}
	}
}

// runCleanup executes all cleanup operations
func (s *CleanupService) runCleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := time.Now()
	s.logger.Info("cleanup", "Starting data cleanup process")

	// Track statistics
	stats := map[string]interface{}{
		"start_time": start.Format(time.RFC3339),
	}

	// Cleanup logs in database and files
	logCount, err := s.cleanupLogs()
	if err != nil {
		s.logger.Error("cleanup", "Error cleaning up logs: %v", err)
	}
	stats["logs_deleted"] = logCount

	// Cleanup pending messages
	msgCount, err := s.cleanupMessages()
	if err != nil {
		s.logger.Error("cleanup", "Error cleaning up messages: %v", err)
	}
	stats["messages_deleted"] = msgCount

	// Cleanup processed files
	fileCount, filesRemoved, err := s.cleanupProcessedFiles()
	if err != nil {
		s.logger.Error("cleanup", "Error cleaning up processed files: %v", err)
	}
	stats["processed_files_deleted"] = fileCount
	stats["physical_files_removed"] = filesRemoved

	// Cleanup synced folders
	folderCount, err := s.cleanupSyncedFolders()
	if err != nil {
		s.logger.Error("cleanup", "Error cleaning up synced folders: %v", err)
	}
	stats["folders_deleted"] = folderCount

	// Record completion and duration
	duration := time.Since(start)
	stats["duration"] = duration.String()
	stats["duration_ms"] = duration.Milliseconds()

	s.lastCleanup = time.Now()
	stats["completed_at"] = s.lastCleanup.Format(time.RFC3339)

	s.logger.Info("cleanup", "Data cleanup completed in %v", duration)

	// Log cleanup summary to database
	if s.logService != nil {
		summary := fmt.Sprintf(
			"Cleanup summary: removed %d logs, %d messages, %d processed files (%d actual files), %d folders",
			logCount, msgCount, fileCount, filesRemoved, folderCount,
		)
		s.logService.LogMessage("INFO", "cleanup", summary)
	}
}

// cleanupLogs deletes old logs in database and files
func (s *CleanupService) cleanupLogs() (int64, error) {
	if s.config.LogRetention <= 0 {
		return 0, nil // Log retention disabled
	}

	cutoffTime := time.Now().AddDate(0, 0, -s.config.LogRetention)
	s.logger.Info("cleanup", "Cleaning up logs older than %s", cutoffTime.Format("2006-01-02"))

	// Use the log service if available, otherwise do a direct DB delete
	if s.logService != nil {
		// Also cleans up log files if configured
		return s.logService.DeleteOldLogs(s.config.LogRetention)
	}

	// Direct DB delete
	result := s.db.Where("timestamp < ?", cutoffTime).Delete(&models.LogEntry{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// cleanupMessages deletes old messages
func (s *CleanupService) cleanupMessages() (int64, error) {
	if s.config.MessageRetention <= 0 {
		return 0, nil // Message retention disabled
	}

	cutoffTime := time.Now().AddDate(0, 0, -s.config.MessageRetention)
	s.logger.Info("cleanup", "Cleaning up messages older than %s", cutoffTime.Format("2006-01-02"))

	sentResult := s.db.Where("sent = ? AND timestamp < ?", true, cutoffTime).Delete(&models.PendingMessage{})
	if sentResult.Error != nil {
		return 0, fmt.Errorf("failed to delete sent messages: %w", sentResult.Error)
	}

	// Second, check if we need to limit the number of pending messages
	if s.config.MaxPendingMessages > 0 {
		var pendingCount int64
		if err := s.db.Model(&models.PendingMessage{}).Where("sent = ?", false).Count(&pendingCount).Error; err != nil {
			s.logger.Error("cleanup", "Error counting pending messages: %v", err)
		} else if pendingCount > int64(s.config.MaxPendingMessages) {
			// We have too many pending messages, clean up the oldest ones
			excessCount := pendingCount - int64(s.config.MaxPendingMessages)
			s.logger.Info("cleanup", "Found %d pending messages, removing oldest %d", pendingCount, excessCount)

			// Get IDs of the oldest pending messages to delete
			var oldestIDs []uint
			if err := s.db.Model(&models.PendingMessage{}).
				Where("sent = ?", false).
				Order("timestamp ASC").
				Limit(int(excessCount)).
				Pluck("id", &oldestIDs).Error; err != nil {
				s.logger.Error("cleanup", "Error identifying oldest messages: %v", err)
			} else if len(oldestIDs) > 0 {
				// Delete the oldest messages
				if err := s.db.Delete(&models.PendingMessage{}, oldestIDs).Error; err != nil {
					s.logger.Error("cleanup", "Error deleting oldest messages: %v", err)
				} else {
					s.logger.Info("cleanup", "Deleted %d oldest pending messages", len(oldestIDs))
					sentResult.RowsAffected += int64(len(oldestIDs))
				}
			}
		}
	}

	return sentResult.RowsAffected, nil
}

// cleanupProcessedFiles deletes old processed files and their physical files
func (s *CleanupService) cleanupProcessedFiles() (int64, int, error) {
	if s.config.ProcessedFileRetention <= 0 {
		return 0, 0, nil // Processed file retention disabled
	}

	cutoffTime := time.Now().AddDate(0, 0, -s.config.ProcessedFileRetention)
	s.logger.Info("cleanup", "Cleaning up processed files older than %s", cutoffTime.Format("2006-01-02"))

	// First, get the list of files to delete so we can remove the physical files
	var filesToDelete []models.ProcessedFile
	if err := s.db.Where("processed_at < ?", cutoffTime).Find(&filesToDelete).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to query processed files: %w", err)
	}

	// Delete the database records
	result := s.db.Where("processed_at < ?", cutoffTime).Delete(&models.ProcessedFile{})
	if result.Error != nil {
		return 0, 0, fmt.Errorf("failed to delete processed files: %w", result.Error)
	}

	// Delete the physical files
	filesRemoved := 0
	for _, file := range filesToDelete {
		// Build the file path
		filePath := filepath.Join(s.config.DataDirectory, file.Filename)

		// Check if the file exists
		if _, err := os.Stat(filePath); err == nil {
			// File exists, delete it
			if err := os.Remove(filePath); err != nil {
				s.logger.Error("cleanup", "Failed to delete file %s: %v", filePath, err)
			} else {
				filesRemoved++
			}
		}
	}

	s.logger.Info("cleanup", "Deleted %d database entries and %d physical files", result.RowsAffected, filesRemoved)
	return result.RowsAffected, filesRemoved, nil
}

// cleanupSyncedFolders deletes old synced folders that are not fully synced
func (s *CleanupService) cleanupSyncedFolders() (int64, error) {
	if s.config.SyncedFolderRetention <= 0 {
		return 0, nil // Synced folder retention disabled
	}

	cutoffTime := time.Now().AddDate(0, 0, -s.config.SyncedFolderRetention)
	s.logger.Info("cleanup", "Cleaning up not-fully-synced folders older than %s", cutoffTime.Format("2006-01-02"))

	// Only delete folders that are not fully synced and haven't been checked recently
	result := s.db.Where("fully_synced = ? AND last_checked < ?", false, cutoffTime).Delete(&models.SyncedFolder{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete synced folders: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// GetStatus returns the current status of the cleanup service
func (s *CleanupService) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := map[string]interface{}{
		"enabled":  s.config.Enabled,
		"running":  s.running,
		"interval": s.config.Interval.String(),
		"retention": map[string]interface{}{
			"logs":            s.config.LogRetention,
			"messages":        s.config.MessageRetention,
			"processed_files": s.config.ProcessedFileRetention,
			"synced_folders":  s.config.SyncedFolderRetention,
		},
		"limits": map[string]interface{}{
			"max_log_files":        s.config.MaxLogFiles,
			"max_pending_messages": s.config.MaxPendingMessages,
		},
	}

	if !s.lastCleanup.IsZero() {
		status["last_cleanup"] = s.lastCleanup.Format(time.RFC3339)
		status["next_cleanup"] = s.lastCleanup.Add(s.config.Interval).Format(time.RFC3339)
	}

	return status
}

// ForceCleanup triggers an immediate cleanup
func (s *CleanupService) ForceCleanup() error {
	s.logger.Info("cleanup", "Forced cleanup triggered manually")
	go s.runCleanup()
	return nil
}

// UpdateConfig updates the cleanup configuration
func (s *CleanupService) UpdateConfig(newConfig *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if newConfig.Interval < time.Minute {
		return fmt.Errorf("cleanup interval must be at least 1 minute")
	}

	s.logger.Info("cleanup", "Updating cleanup configuration")

	// Store the old interval to check if we need to restart the ticker
	oldInterval := s.config.Interval
	oldEnabled := s.config.Enabled

	// Update the configuration
	s.config = newConfig

	// If the service was running and the interval changed, restart it
	if s.running && (oldInterval != newConfig.Interval || oldEnabled != newConfig.Enabled) {
		s.Stop()
		if newConfig.Enabled {
			s.Start()
		}
	}

	// If the service was not running and is now enabled, start it
	if !s.running && newConfig.Enabled {
		s.Start()
	}

	return nil
}
