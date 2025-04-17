package log

import (
	"fmt"
	"jarvist/internal/common/config"
	"jarvist/internal/common/models"
	"jarvist/pkg/logger"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type LogService struct {
	cfg         *config.Config
	db          *gorm.DB
	logger      *logger.Logger
	logDir      string // Add log directory
	maxLogFiles int    // Add maximum number of log files to keep
}

// NewLogService creates a new LogService
func NewLogService(db *gorm.DB, cfg *config.Config, logger *logger.Logger, logDir string, maxLogFiles int) *LogService {
	if maxLogFiles <= 0 {
		maxLogFiles = 10 // Default to keeping 10 log files
	}

	return &LogService{
		db:          db,
		logDir:      logDir,
		maxLogFiles: maxLogFiles,
		logger:      logger,
		cfg:         cfg,
	}
}

// Existing methods...

func (s *LogService) LogMessage(level, component, message string) error {
	maxLength := 4000
	if len(message) > maxLength {
		message = message[:maxLength-3] + "..."
	}

	logEntry := models.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: component,
		Message:   message,
	}

	result := s.db.Create(&logEntry)
	if result.Error != nil {
		return fmt.Errorf("failed to log message: %w", result.Error)
	}

	return nil
}

func (s *LogService) GetLogs(level, component string, limit, offset int, startTime, endTime string) ([]models.LogEntry, error) {
	var logs []models.LogEntry
	query := s.db.Model(&models.LogEntry{})

	if level != "" {
		query = query.Where("level = ?", level)
	}

	if component != "" {
		query = query.Where("component = ?", component)
	}

	if startTime != "" {
		query = query.Where("timestamp >= ?", startTime)
	}

	if endTime != "" {
		query = query.Where("timestamp <= ?", endTime)
	}

	result := query.Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query logs: %w", result.Error)
	}

	return logs, nil
}

func (s *LogService) GetStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	for _, level := range levels {
		var count int64
		if err := s.db.Model(&models.LogEntry{}).Where("level = ?", level).Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to count %s logs: %w", level, err)
		}
		stats[fmt.Sprintf("%s_logs", level)] = count
	}
	var totalCount int64
	if err := s.db.Model(&models.LogEntry{}).Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count total logs: %w", err)
	}
	stats["total_logs"] = totalCount

	return stats, nil
}

// DeleteOldLogs deletes logs older than the specified number of days
func (s *LogService) DeleteOldLogs(daysToKeep int) (int64, error) {
	// Calculate the cutoff date
	cutoffTime := time.Now().AddDate(0, 0, -daysToKeep)

	// Delete logs older than the cutoff
	result := s.db.Where("timestamp < ?", cutoffTime).Delete(&models.LogEntry{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", result.Error)
	}

	// Also clean up log files if a log directory is specified
	if s.logDir != "" {
		if err := s.cleanupLogFiles(daysToKeep); err != nil {
			return result.RowsAffected, fmt.Errorf("deleted database logs but failed to clean log files: %w", err)
		}
	}

	return result.RowsAffected, nil
}

// cleanupLogFiles removes old log files beyond maxLogFiles or older than daysToKeep
func (s *LogService) cleanupLogFiles(daysToKeep int) error {
	if s.logDir == "" {
		return nil // No log directory specified
	}

	// Get all log files
	files, err := s.findLogFiles()
	if err != nil {
		return err
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	// Cutoff time for old files
	cutoffTime := time.Now().AddDate(0, 0, -daysToKeep)

	// Keep track of how many files to delete
	var filesToDelete []string

	// First identify files to delete based on age
	for _, file := range files {
		if file.ModTime.Before(cutoffTime) {
			filesToDelete = append(filesToDelete, file.Path)
		}
	}

	// Then identify files to delete based on count limit
	// (but only if we have more than the limit)
	if len(files) > s.maxLogFiles {
		for i := s.maxLogFiles; i < len(files); i++ {
			// Check if this file isn't already marked for deletion
			alreadyMarked := false
			for _, path := range filesToDelete {
				if path == files[i].Path {
					alreadyMarked = true
					break
				}
			}

			if !alreadyMarked {
				filesToDelete = append(filesToDelete, files[i].Path)
			}
		}
	}

	// Delete the files
	for _, path := range filesToDelete {
		if err := os.Remove(path); err != nil {
			// Log but continue with other files
			fmt.Printf("Error deleting log file %s: %v\n", path, err)
		}
	}

	return nil
}

// LogFileInfo represents information about a log file
type LogFileInfo struct {
	Path    string
	ModTime time.Time
	Size    int64
}

// findLogFiles finds all log files in the log directory
func (s *LogService) findLogFiles() ([]LogFileInfo, error) {
	var files []LogFileInfo

	// Common log file extensions
	logExtensions := map[string]bool{
		".log":   true,
		".txt":   true,
		".error": true,
		".out":   true,
	}

	// Walk the directory
	err := filepath.Walk(s.logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if this is a log file
		ext := strings.ToLower(filepath.Ext(path))
		if logExtensions[ext] || strings.Contains(strings.ToLower(info.Name()), "log") {
			files = append(files, LogFileInfo{
				Path:    path,
				ModTime: info.ModTime(),
				Size:    info.Size(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking log directory: %w", err)
	}

	return files, nil
}

// GetLogFileStats returns statistics about log files
func (s *LogService) GetLogFileStats() (map[string]interface{}, error) {
	files, err := s.findLogFiles()
	if err != nil {
		return nil, err
	}

	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}

	stats := map[string]interface{}{
		"file_count":       len(files),
		"total_size_bytes": totalSize,
		"total_size_mb":    float64(totalSize) / (1024 * 1024), // Convert to MB
		"log_directory":    s.logDir,
		"max_log_files":    s.maxLogFiles,
	}

	// Include the 5 newest files for reference
	if len(files) > 0 {
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime.After(files[j].ModTime)
		})

		newestFiles := []map[string]interface{}{}
		for i := 0; i < 5 && i < len(files); i++ {
			newestFiles = append(newestFiles, map[string]interface{}{
				"name":       filepath.Base(files[i].Path),
				"mod_time":   files[i].ModTime.Format(time.RFC3339),
				"size_bytes": files[i].Size,
				"size_kb":    float64(files[i].Size) / 1024,
			})
		}
		stats["newest_files"] = newestFiles
	}

	return stats, nil
}

// SetMaxLogFiles changes the maximum number of log files to keep
func (s *LogService) SetMaxLogFiles(maxFiles int) {
	if maxFiles <= 0 {
		maxFiles = 10 // Minimum value
	}
	s.maxLogFiles = maxFiles
}

// GetMaxLogFiles returns the current maximum number of log files setting
func (s *LogService) GetMaxLogFiles() int {
	return s.maxLogFiles
}
