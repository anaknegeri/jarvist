package logmanager

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"jarvist/internal/common/config"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Component string `json:"component"`
	Message   string `json:"message"`
	Details   string `json:"details"`
}

// LogService handles log-related operations
type LogService struct {
	serviceLogDir string
	appLogDir     string
}

// New creates a new LogService
func New(cfg *config.Config) *LogService {
	return &LogService{
		serviceLogDir: filepath.Join(cfg.BinDir, "services", "logs"),
		appLogDir:     filepath.Join(cfg.BinDir, "logs"),
	}
}

// ReadLogs reads log files from all log directories
func (s *LogService) ReadLogs() ([]string, error) {
	var allLogs []string

	// Read service logs
	serviceLogs, err := s.readLogsFromDirectory(s.serviceLogDir)
	if err != nil {
		return nil, fmt.Errorf("error reading service logs: %v", err)
	}
	allLogs = append(allLogs, serviceLogs...)

	// Read application logs
	appLogs, err := s.readLogsFromDirectory(s.appLogDir)
	if err != nil {
		return nil, fmt.Errorf("error reading application logs: %v", err)
	}
	allLogs = append(allLogs, appLogs...)

	return allLogs, nil
}

// readLogsFromDirectory reads log files from the specified directory
func (s *LogService) readLogsFromDirectory(directory string) ([]string, error) {
	// Ensure directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(directory, 0755); err != nil {
			return nil, fmt.Errorf("error creating log directory %s: %v", directory, err)
		}
		return []string{}, nil // Return empty slice for new directory
	}

	// Find all .logs files in the directory
	files, err := filepath.Glob(filepath.Join(directory, "*.logs"))
	if err != nil {
		return nil, fmt.Errorf("error finding log files in %s: %v", directory, err)
	}

	// Also find .log files (without s)
	logFiles, err := filepath.Glob(filepath.Join(directory, "*.log"))
	if err != nil {
		return nil, fmt.Errorf("error finding .log files in %s: %v", directory, err)
	}

	// Combine both file types
	files = append(files, logFiles...)

	// If no log files found, return empty slice
	if len(files) == 0 {
		return []string{}, nil
	}

	// Sort files by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, err1 := os.Stat(files[i])
		infoJ, err2 := os.Stat(files[j])
		if err1 != nil || err2 != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Read the most recent log file
	mostRecentLogFile := files[0]
	return readLogFile(mostRecentLogFile)
}

// GetLogFiles returns a list of available log files
func (s *LogService) GetLogFiles() ([]string, error) {
	var allFiles []string

	// Get service log files
	serviceFiles, err := s.getLogFilesFromDirectory(s.serviceLogDir)
	if err != nil {
		return nil, err
	}
	allFiles = append(allFiles, serviceFiles...)

	// Get application log files
	appFiles, err := s.getLogFilesFromDirectory(s.appLogDir)
	if err != nil {
		return nil, err
	}
	allFiles = append(allFiles, appFiles...)

	return allFiles, nil
}

// getLogFilesFromDirectory returns log files from a specific directory
// getLogFilesFromDirectory returns log files from a specific directory
func (s *LogService) getLogFilesFromDirectory(directory string) ([]string, error) {
	// Ensure directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(directory, 0755); err != nil {
			return nil, fmt.Errorf("error creating log directory %s: %v", directory, err)
		}
		return []string{}, nil // Return empty slice for new directory
	}

	// Find all log files in the directory (.logs and .log)
	logsFiles, err := filepath.Glob(filepath.Join(directory, "*.logs"))
	if err != nil {
		return nil, fmt.Errorf("error finding log files in %s: %v", directory, err)
	}

	logFiles, err := filepath.Glob(filepath.Join(directory, "*.log"))
	if err != nil {
		return nil, fmt.Errorf("error finding log files in %s: %v", directory, err)
	}

	// Combine both file types
	files := append(logsFiles, logFiles...)

	// Extract just the filenames without the full path
	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, filepath.Base(file))
	}

	return fileNames, nil
}

// readLogFile reads a single log file and returns its lines
func readLogFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening log file %s: %v", filePath, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Optional: Add any preprocessing of log lines here
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading log file %s: %v", filePath, err)
	}

	return lines, nil
}

// DownloadLogs copies log files from all directories to a specified destination
func (s *LogService) DownloadLogs(destinationDir string) ([]string, error) {
	// Ensure destination directory exists
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %v", err)
	}

	var copiedFiles []string

	// Download service logs
	serviceDestDir := filepath.Join(destinationDir, "service_logs")
	if err := os.MkdirAll(serviceDestDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create service logs destination directory: %v", err)
	}

	serviceFiles, err := s.downloadLogsFromDirectory(s.serviceLogDir, serviceDestDir)
	if err != nil {
		return nil, err
	}
	copiedFiles = append(copiedFiles, serviceFiles...)

	// Download application logs
	appDestDir := filepath.Join(destinationDir, "app_logs")
	if err := os.MkdirAll(appDestDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app logs destination directory: %v", err)
	}

	appFiles, err := s.downloadLogsFromDirectory(s.appLogDir, appDestDir)
	if err != nil {
		return nil, err
	}
	copiedFiles = append(copiedFiles, appFiles...)

	return copiedFiles, nil
}

// downloadLogsFromDirectory copies log files from the specified directory
func (s *LogService) downloadLogsFromDirectory(sourceDir, destDir string) ([]string, error) {
	// Ensure source directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return []string{}, nil // Skip if source doesn't exist
	}

	// Find all log files in the directory (.logs and .log)
	logsFiles, err := filepath.Glob(filepath.Join(sourceDir, "*.logs"))
	if err != nil {
		return nil, fmt.Errorf("error finding log files: %v", err)
	}

	logFiles, err := filepath.Glob(filepath.Join(sourceDir, "*.log"))
	if err != nil {
		return nil, fmt.Errorf("error finding log files: %v", err)
	}

	// Combine both file types
	files := append(logsFiles, logFiles...)

	var copiedFiles []string
	for _, srcFile := range files {
		// Generate destination filename
		filename := filepath.Base(srcFile)
		destFile := filepath.Join(destDir, filename)

		// Copy the file
		if err := copyFile(srcFile, destFile); err != nil {
			return nil, fmt.Errorf("failed to copy log file %s: %v", filename, err)
		}

		copiedFiles = append(copiedFiles, destFile)
	}

	return copiedFiles, nil
}

// copyFile copies a single file from source to destination
func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file contents
	_, err = io.Copy(destFile, sourceFile)
	return err
}

// ParseLogLine parses a single log line into a LogEntry
func ParseLogLine(line string) (*LogEntry, error) {
	// Log format: 2025-01-23 12:56:26 - INFO -  Berhasil Menambahkan Data ke DB Pada 2025-01-23 12:56:26
	parts := strings.SplitN(line, " - ", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid log line format: %s", line)
	}

	timestamp := strings.TrimSpace(parts[0])
	level := strings.TrimSpace(parts[1])
	message := strings.TrimSpace(parts[2])

	return &LogEntry{
		Timestamp: timestamp,
		Level:     strings.ToLower(level),
		Component: "Scheduler",       // Default component
		Message:   "Database Update", // Default message
		Details:   message,
	}, nil
}

// FilterLogs allows filtering logs based on various criteria
func (s *LogService) FilterLogs(
	level string,
	startDate, endDate time.Time,
	searchTerm string,
) ([]LogEntry, error) {
	// Read all log files
	allLogs, err := s.ReadLogs()
	if err != nil {
		return nil, err
	}

	var filteredLogs []LogEntry
	for _, line := range allLogs {
		logEntry, err := ParseLogLine(line)
		if err != nil {
			// Skip lines that can't be parsed
			continue
		}

		// Level filter
		if level != "" && level != "all" && logEntry.Level != level {
			continue
		}

		// Date filtering
		logTime, err := time.Parse("2006-01-02 15:04:05", logEntry.Timestamp)
		if err != nil {
			continue
		}

		if !startDate.IsZero() && logTime.Before(startDate) {
			continue
		}

		if !endDate.IsZero() && logTime.After(endDate) {
			continue
		}

		// Search term filter
		if searchTerm != "" {
			searchTermLower := strings.ToLower(searchTerm)
			if !strings.Contains(strings.ToLower(logEntry.Message), searchTermLower) &&
				!strings.Contains(strings.ToLower(logEntry.Details), searchTermLower) {
				continue
			}
		}

		filteredLogs = append(filteredLogs, *logEntry)
	}

	return filteredLogs, nil
}

// ExportLogsToCSV exports filtered logs to a CSV file
func (s *LogService) ExportLogsToCSV(
	outputPath string,
	level string,
	startDate, endDate time.Time,
	searchTerm string,
) error {
	// Filter logs
	filteredLogs, err := s.FilterLogs(level, startDate, endDate, searchTerm)
	if err != nil {
		return err
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Timestamp",
		"Level",
		"Component",
		"Message",
		"Details",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write log entries
	for _, log := range filteredLogs {
		record := []string{
			log.Timestamp,
			log.Level,
			log.Component,
			log.Message,
			log.Details,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// SetupLogRotation manages log file rotation
func (s *LogService) SetupLogRotation(maxFiles int, maxSizeBytes int64) error {
	// Setup rotation for service logs
	if err := s.setupLogRotationForDirectory(s.serviceLogDir, maxFiles, maxSizeBytes); err != nil {
		return err
	}

	// Setup rotation for app logs
	if err := s.setupLogRotationForDirectory(s.appLogDir, maxFiles, maxSizeBytes); err != nil {
		return err
	}

	return nil
}

// setupLogRotationForDirectory handles log rotation for a specific directory
func (s *LogService) setupLogRotationForDirectory(directory string, maxFiles int, maxSizeBytes int64) error {
	// Ensure directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return nil // Skip if directory doesn't exist
	}

	// Find all log files
	logsFiles, err := filepath.Glob(filepath.Join(directory, "*.logs"))
	if err != nil {
		return err
	}

	logFiles, err := filepath.Glob(filepath.Join(directory, "*.log"))
	if err != nil {
		return err
	}

	// Combine both file types
	files := append(logsFiles, logFiles...)

	if len(files) == 0 {
		return nil // No files to rotate
	}

	// Sort files by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, err1 := os.Stat(files[i])
		infoJ, err2 := os.Stat(files[j])
		if err1 != nil || err2 != nil {
			return false
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// Remove old log files if exceeding max number
	if len(files) > maxFiles {
		for _, file := range files[:len(files)-maxFiles] {
			os.Remove(file)
		}
	}

	// Check current log file size
	currentLogFile := files[len(files)-1]
	fileInfo, err := os.Stat(currentLogFile)
	if err != nil {
		return err
	}

	// Rotate log if size exceeds max
	if fileInfo.Size() > maxSizeBytes {
		// Generate new log filename with timestamp
		extension := filepath.Ext(currentLogFile)
		baseName := strings.TrimSuffix(filepath.Base(currentLogFile), extension)
		newLogFilename := fmt.Sprintf("%s_%s%s", baseName, time.Now().Format("20060102_150405"), extension)
		newLogPath := filepath.Join(directory, newLogFilename)

		// Rename current log file
		if err := os.Rename(currentLogFile, newLogPath); err != nil {
			return err
		}
	}

	return nil
}
