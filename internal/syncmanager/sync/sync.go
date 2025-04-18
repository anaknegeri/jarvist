package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jarvist/internal/common/models"
	"jarvist/internal/syncmanager/config"
	"jarvist/internal/syncmanager/mqtt"
	"jarvist/pkg/logger"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/mgo.v2/bson"
	"gorm.io/gorm"
)

const (
	ComponentSynchronizer = "synchronizer"
	DateFolderPattern     = "20060102"
)

// Synchronizer handles file synchronization using a file watcher approach
type Synchronizer struct {
	config        *config.Config
	logger        *logger.Logger
	stopCh        chan struct{}
	inSyncProcess bool
	mu            sync.Mutex
	db            *gorm.DB
	mqttSender    *mqtt.Sender

	// Watcher related fields
	watcher      *fsnotify.Watcher
	watchCtx     context.Context
	watchCancel  context.CancelFunc
	watchMutex   sync.Mutex
	watchActive  bool
	pendingFiles chan string
}

type DataEntry struct {
	ID                 string  `bson:"id" json:"id"`
	CCTVID             int     `bson:"cctv_id" json:"cctv_id"`
	DeviceID           string  `bson:"device_id" json:"device_id"`
	DeviceTimestamp    string  `bson:"device_timestamp" json:"device_timestamp"`
	DeviceTimestampUTC float64 `bson:"device_timestamp_utc" json:"device_timestamp_utc"`
	InCount            int     `bson:"in_count" json:"in_count"`
	OutCount           int     `bson:"out_count" json:"out_count"`
	StartTime          string  `bson:"start_time" json:"start_time"`
	SyncStatus         bool    `bson:"sync_status" json:"sync_status"`
}

// NewSynchronizer creates a new synchronizer with file watching capabilities
func NewSynchronizer(config *config.Config, logger *logger.Logger, db *gorm.DB, mqttSender *mqtt.Sender) *Synchronizer {
	// Initialize watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(ComponentSynchronizer, "Failed to create file watcher: %v", err)
	}

	watchCtx, watchCancel := context.WithCancel(context.Background())

	sync := &Synchronizer{
		config:       config,
		logger:       logger,
		stopCh:       make(chan struct{}),
		db:           db,
		mqttSender:   mqttSender,
		watcher:      watcher,
		watchCtx:     watchCtx,
		watchCancel:  watchCancel,
		watchActive:  false,
		pendingFiles: make(chan string, 1000), // Buffer for pending files
	}

	// If watcher creation failed, we'll set up a recovery mechanism
	if watcher == nil {
		go sync.recoverWatcher()
	}

	return sync
}

// recoverWatcher attempts to recreate the file watcher if it failed initially
func (s *Synchronizer) recoverWatcher() {
	// Wait a bit before attempting recovery
	time.Sleep(30 * time.Second)

	s.watchMutex.Lock()
	defer s.watchMutex.Unlock()

	if s.watcher != nil {
		// Watcher was already created elsewhere
		return
	}

	s.logger.Info(ComponentSynchronizer, "Attempting to recover file watcher")

	// Try to create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		s.logger.Error(ComponentSynchronizer, "Failed to recover file watcher: %v", err)
		return
	}

	s.watcher = watcher
	s.logger.Info(ComponentSynchronizer, "File watcher recovered successfully")

	// If synchronizer is already running, start the watcher
	if s.isRunning() {
		go s.startWatching()
	}
}

// isRunning checks if the synchronizer is currently running
func (s *Synchronizer) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if stopCh is closed
	select {
	case <-s.stopCh:
		return false
	default:
		return true
	}
}

// Start begins synchronization and watching
func (s *Synchronizer) Start() error {
	s.logger.Info(ComponentSynchronizer, "Starting synchronizer with file watching")

	// Create a new channel since previous one might have been closed
	s.mu.Lock()
	if s.stopCh == nil || isChanClosed(s.stopCh) {
		s.stopCh = make(chan struct{})
	}
	s.mu.Unlock()

	// Initial sync to catch files created before the watcher was started
	go func() {
		// Create the data directory if it doesn't exist
		if _, err := os.Stat(s.config.BaseConfig.ServicesDataDir); os.IsNotExist(err) {
			if err := os.MkdirAll(s.config.BaseConfig.ServicesDataDir, 0755); err != nil {
				s.logger.Error(ComponentSynchronizer, "Failed to create data directory: %v", err)
			} else {
				s.logger.Info(ComponentSynchronizer, "Created data directory: %s", s.config.BaseConfig.ServicesDataDir)
			}
		}

		s.SyncData()

		// Check if watcher exists
		s.watchMutex.Lock()
		hasWatcher := s.watcher != nil
		s.watchMutex.Unlock()

		if hasWatcher {
			// Setup file watching after initial sync
			s.startWatching()
		} else {
			s.logger.Warning(ComponentSynchronizer, "File watcher not available, will rely on periodic scans")
		}
	}()

	// Start processing workers
	go s.processPendingFiles()

	// Periodic folder scan to catch any missed files
	interval := time.Duration(s.config.Sync.Interval) * time.Second
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.scanForNewFiles()
			case <-s.stopCh:
				s.logger.Info(ComponentSynchronizer, "Synchronizer stopped")
				return
			}
		}
	}()

	return nil
}

// isChanClosed checks if a channel is closed
func isChanClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// Stop halts synchronization and watching
func (s *Synchronizer) Stop() error {
	s.logger.Info(ComponentSynchronizer, "Stopping synchronizer...")

	// Signal all goroutines to stop
	close(s.stopCh)

	// Stop the watcher
	s.stopWatching()

	// Close the watcher
	if s.watcher != nil {
		s.watcher.Close()
	}

	time.Sleep(1 * time.Second)

	s.logger.Info(ComponentSynchronizer, "Synchronizer stopped")
	return nil
}

// startWatching begins watching for file changes
func (s *Synchronizer) startWatching() {
	s.watchMutex.Lock()
	defer s.watchMutex.Unlock()

	if s.watchActive {
		return
	}

	// Make sure the data directory exists
	if _, err := os.Stat(s.config.BaseConfig.ServicesDataDir); os.IsNotExist(err) {
		s.logger.Warning(ComponentSynchronizer, "Services data directory does not exist: %s", s.config.BaseConfig.ServicesDataDir)
		if err := os.MkdirAll(s.config.BaseConfig.ServicesDataDir, 0755); err != nil {
			s.logger.Error(ComponentSynchronizer, "Failed to create data directory: %v", err)
			return
		}
		s.logger.Info(ComponentSynchronizer, "Created data directory: %s", s.config.BaseConfig.ServicesDataDir)
	}

	// Start watching the base directory for new folders
	if err := s.watcher.Add(s.config.BaseConfig.ServicesDataDir); err != nil {
		s.logger.Error(ComponentSynchronizer, "Failed to watch services data directory: %v", err)
		return
	}

	// Find and watch all date folders
	dateFolders, err := s.findDateFolders()
	if err != nil {
		s.logger.Error(ComponentSynchronizer, "Failed to find date folders: %v", err)
	} else {
		s.logger.Info(ComponentSynchronizer, "Found %d existing date folders", len(dateFolders))
		for _, folder := range dateFolders {
			if err := s.watcher.Add(folder); err != nil {
				s.logger.Error(ComponentSynchronizer, "Failed to watch folder %s: %v", folder, err)
			} else {
				s.logger.Info(ComponentSynchronizer, "Now watching folder: %s", folder)
			}
		}
	}

	// Start the event handling goroutine
	go s.watchEvents()

	// Set up a watchdog to periodically verify that our watchers are still active
	go s.watchdogMonitor()

	s.watchActive = true
	s.logger.Info(ComponentSynchronizer, "File watcher started")
}

// watchdogMonitor periodically checks and ensures all required directories are being watched
func (s *Synchronizer) watchdogMonitor() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.watchMutex.Lock()
			if !s.watchActive {
				s.watchMutex.Unlock()
				return
			}
			s.watchMutex.Unlock()

			s.logger.Debug(ComponentSynchronizer, "Watchdog: Checking directory watches")

			// Verify that the base directory is being watched
			baseDir := s.config.BaseConfig.ServicesDataDir
			if _, err := os.Stat(baseDir); os.IsNotExist(err) {
				s.logger.Warning(ComponentSynchronizer, "Watchdog: Data directory has been deleted: %s", baseDir)
				if err := os.MkdirAll(baseDir, 0755); err != nil {
					s.logger.Error(ComponentSynchronizer, "Watchdog: Failed to recreate data directory: %v", err)
				} else {
					s.logger.Info(ComponentSynchronizer, "Watchdog: Recreated data directory: %s", baseDir)
					if err := s.watcher.Add(baseDir); err != nil {
						s.logger.Error(ComponentSynchronizer, "Watchdog: Failed to re-watch data directory: %v", err)
					}
				}
			}

			// Find all date folders and ensure they're being watched
			dateFolders, err := s.findDateFolders()
			if err != nil {
				s.logger.Error(ComponentSynchronizer, "Watchdog: Failed to find date folders: %v", err)
				continue
			}

			for _, folder := range dateFolders {
				if err := s.watcher.Add(folder); err != nil {
					s.logger.Error(ComponentSynchronizer, "Watchdog: Failed to ensure watch on folder %s: %v", folder, err)
				}
			}

		case <-s.watchCtx.Done():
			return
		}
	}
}

// stopWatching stops the file watcher
func (s *Synchronizer) stopWatching() {
	s.watchMutex.Lock()
	defer s.watchMutex.Unlock()

	if !s.watchActive {
		return
	}

	// Cancel the context to stop event handling
	s.watchCancel()

	// Create a new context for future use
	s.watchCtx, s.watchCancel = context.WithCancel(context.Background())

	s.watchActive = false
	s.logger.Info(ComponentSynchronizer, "File watcher stopped")
}

// watchEvents handles file watching events
func (s *Synchronizer) watchEvents() {
	for {
		select {
		case <-s.watchCtx.Done():
			return

		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Handle the event
			s.handleWatchEvent(event)

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			s.logger.Error(ComponentSynchronizer, "Watcher error: %v", err)
		}
	}
}

// handleWatchEvent processes a single file watcher event
func (s *Synchronizer) handleWatchEvent(event fsnotify.Event) {
	s.logger.Debug(ComponentSynchronizer, "Watch event: %s - operation: %s", event.Name, event.Op.String())

	// Handle different event types
	switch {
	case event.Op&fsnotify.Create != 0:
		// Handle creation events
		s.handleCreateEvent(event)

	case event.Op&fsnotify.Remove != 0 || event.Op&fsnotify.Rename != 0:
		// Handle removal or rename events (these operate similarly from the watcher's perspective)
		s.handleRemoveEvent(event)

	case event.Op&fsnotify.Write != 0:
		// Handle write events (file modifications)
		s.handleWriteEvent(event)
	}
}

// handleCreateEvent handles file or directory creation
func (s *Synchronizer) handleCreateEvent(event fsnotify.Event) {
	// Get file info
	info, err := os.Stat(event.Name)
	if err != nil {
		if os.IsNotExist(err) {
			// File might have been quickly created and deleted
			s.logger.Debug(ComponentSynchronizer, "Created file already gone: %s", event.Name)
			return
		}
		s.logger.Error(ComponentSynchronizer, "Failed to get info for %s: %v", event.Name, err)
		return
	}

	if info.IsDir() {
		// If it's a new directory matching our date pattern, watch it
		dirName := filepath.Base(event.Name)
		if len(dirName) == 8 && isNumeric(dirName) {
			_, err := time.Parse(DateFolderPattern, dirName)
			if err == nil {
				s.logger.Info(ComponentSynchronizer, "Adding new date folder to watch: %s", event.Name)
				if err := s.watcher.Add(event.Name); err != nil {
					s.logger.Error(ComponentSynchronizer, "Failed to watch new folder %s: %v", event.Name, err)
				} else {
					// Schedule a scan of the new folder to process any existing files
					go func(folderPath string) {
						s.logger.Info(ComponentSynchronizer, "Scanning new folder: %s", folderPath)
						processedFiles := make(map[string]bool)
						folderName := filepath.Base(folderPath)

						var files []models.ProcessedFile
						if err := s.db.Find(&files).Error; err != nil {
							s.logger.Error(ComponentSynchronizer, "Failed to query processed files: %v", err)
							return
						}

						for _, file := range files {
							processedFiles[file.Filename] = true
						}

						fileCount := s.processFolderFiles(folderPath, folderName, processedFiles)
						s.logger.Info(ComponentSynchronizer, "Processed %d files from new folder %s", fileCount, folderName)
					}(event.Name)
				}
			}
		}
	} else {
		// If it's a file with the right extension, process it
		if strings.HasSuffix(event.Name, ".json.bson") {
			s.logger.Info(ComponentSynchronizer, "New file detected: %s", event.Name)

			// Queue the file for processing
			select {
			case s.pendingFiles <- event.Name:
				// Successfully queued
			default:
				s.logger.Warning(ComponentSynchronizer, "Pending files queue is full, will pick up %s in next scan", event.Name)
			}
		}
	}
}

// handleRemoveEvent handles file or directory removal
func (s *Synchronizer) handleRemoveEvent(event fsnotify.Event) {
	// Extract the path parts
	path := event.Name
	baseDir := s.config.BaseConfig.ServicesDataDir

	// Check if this is a date folder being removed
	if strings.HasPrefix(path, baseDir) {
		relativePath := strings.TrimPrefix(path, baseDir)
		relativePath = strings.TrimPrefix(relativePath, string(filepath.Separator))

		// Check if this is a top-level folder that matches date pattern
		if !strings.Contains(relativePath, string(filepath.Separator)) {
			dirName := relativePath
			if len(dirName) == 8 && isNumeric(dirName) {
				_, err := time.Parse(DateFolderPattern, dirName)
				if err == nil {
					s.logger.Info(ComponentSynchronizer, "Date folder removed: %s", dirName)
				}
			}
		}
	}
}

// handleWriteEvent handles file modifications
func (s *Synchronizer) handleWriteEvent(event fsnotify.Event) {
	if !strings.HasSuffix(event.Name, ".json.bson") {
		return
	}

	info, err := os.Stat(event.Name)
	if err != nil {
		return
	}

	if info.Size() < 100 {
		return
	}

	s.logger.Debug(ComponentSynchronizer, "File modification detected: %s", event.Name)
	select {
	case s.pendingFiles <- event.Name:
	default:
		s.logger.Warning(ComponentSynchronizer, "Pending files queue is full, will pick up %s in next scan", event.Name)
	}
}

// processPendingFiles handles the queue of files to be processed
func (s *Synchronizer) processPendingFiles() {
	for {
		select {
		case <-s.stopCh:
			return

		case filePath := <-s.pendingFiles:
			relPath, err := filepath.Rel(s.config.BaseConfig.ServicesDataDir, filePath)
			if err != nil {
				s.logger.Error(ComponentSynchronizer, "Failed to get relative path for %s: %v", filePath, err)
				continue
			}

			dirPath := filepath.Dir(relPath)
			fileName := filepath.Base(relPath)

			if dirPath == "." {
				continue
			}

			var count int64
			if err := s.db.Model(&models.ProcessedFile{}).
				Where("filename = ? AND date_folder = ?", relPath, dirPath).
				Count(&count).Error; err != nil {
				s.logger.Error(ComponentSynchronizer, "Database error checking file %s: %v", relPath, err)
				continue
			}

			if count > 0 {
				s.logger.Debug(ComponentSynchronizer, "File %s already processed, skipping", relPath)
				continue
			}

			s.logger.Info(ComponentSynchronizer, "Processing new file: %s in folder %s", fileName, dirPath)
			if err := s.processFile(filePath, relPath, dirPath); err != nil {
				s.logger.Error(ComponentSynchronizer, "Error processing file %s: %v", filePath, err)
			}
		}
	}
}

func (s *Synchronizer) SyncData() {
	s.logger.Info(ComponentSynchronizer, "Starting initial data synchronization")

	processedFiles := make(map[string]bool)

	var files []models.ProcessedFile
	if err := s.db.Find(&files).Error; err != nil {
		s.logger.Error(ComponentSynchronizer, "Failed to query processed files: %v", err)
	} else {
		for _, file := range files {
			processedFiles[file.Filename] = true
		}
		s.logger.Info(ComponentSynchronizer, "Found %d already processed files", len(processedFiles))
	}

	dateFolders, err := s.findDateFolders()
	if err != nil {
		s.logger.Error(ComponentSynchronizer, "Failed to find date folders: %v", err)
		return
	}

	s.logger.Info(ComponentSynchronizer, "Found %d date folders", len(dateFolders))

	for _, folder := range dateFolders {
		folderName := filepath.Base(folder)
		s.processFolderFiles(folder, folderName, processedFiles)
	}

	s.logger.Info(ComponentSynchronizer, "Initial synchronization completed")
}

func (s *Synchronizer) scanForNewFiles() {
	s.mu.Lock()
	if s.inSyncProcess {
		s.mu.Unlock()
		s.logger.Debug(ComponentSynchronizer, "Scan already in progress, skipping")
		return
	}

	s.inSyncProcess = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.inSyncProcess = false
		s.mu.Unlock()
	}()

	s.logger.Info(ComponentSynchronizer, "Scanning for new files")

	processedFiles := make(map[string]bool)

	var files []models.ProcessedFile
	if err := s.db.Find(&files).Error; err != nil {
		s.logger.Error(ComponentSynchronizer, "Failed to query processed files: %v", err)
		return
	}

	for _, file := range files {
		processedFiles[file.Filename] = true
	}

	dateFolders, err := s.findDateFolders()
	if err != nil {
		s.logger.Error(ComponentSynchronizer, "Failed to find date folders: %v", err)
		return
	}

	fileCount := 0
	for _, folder := range dateFolders {
		folderName := filepath.Base(folder)
		count := s.processFolderFiles(folder, folderName, processedFiles)
		fileCount += count
	}

	if fileCount > 0 {
		s.logger.Info(ComponentSynchronizer, "Found and processed %d new files during scan", fileCount)
	} else {
		s.logger.Debug(ComponentSynchronizer, "No new files found during scan")
	}
}

// processFolderFiles processes all files in a folder
func (s *Synchronizer) processFolderFiles(folderPath, folderName string, processedFiles map[string]bool) int {
	dataFiles, err := s.getDataFilesInDirectory(folderPath)
	if err != nil {
		s.logger.Error(ComponentSynchronizer, "Failed to read directory %s: %v", folderName, err)
		return 0
	}

	processedCount := 0

	for _, fileName := range dataFiles {
		relPath := filepath.Join(folderName, fileName)

		if processedFiles[relPath] {
			continue
		}

		filePath := filepath.Join(folderPath, fileName)
		if err := s.processFile(filePath, relPath, folderName); err != nil {
			s.logger.Error(ComponentSynchronizer, "Error processing file %s: %v", filePath, err)
			continue
		}

		processedCount++
	}

	return processedCount
}

// getDataFilesInDirectory gets all data files in a directory
func (s *Synchronizer) getDataFilesInDirectory(dirPath string) ([]string, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dataFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), ".json.bson") {
			dataFiles = append(dataFiles, file.Name())
		}
	}

	return dataFiles, nil
}

// findDateFolders finds all folders that match the date pattern
func (s *Synchronizer) findDateFolders() ([]string, error) {
	var dateFolders []string
	entries, err := os.ReadDir(s.config.BaseConfig.ServicesDataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()

		if len(dirName) == 8 && isNumeric(dirName) {
			_, err := time.Parse(DateFolderPattern, dirName)
			if err == nil {
				dateFolders = append(dateFolders, filepath.Join(s.config.BaseConfig.ServicesDataDir, dirName))
			}
		}
	}

	return dateFolders, nil
}

// processFile processes a single file
func (s *Synchronizer) processFile(filePath, filename, folderName string) error {
	var count int64
	if err := s.db.Model(&models.ProcessedFile{}).
		Where("filename = ? AND date_folder = ?", filename, folderName).
		Count(&count).Error; err != nil {
		s.logger.Error(ComponentSynchronizer, "Database error checking file %s: %v", filename, err)
	} else if count > 0 {
		s.logger.Debug(ComponentSynchronizer, "File %s already processed, skipping", filename)
		return nil
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", filePath)
	}

	s.logger.Info(ComponentSynchronizer, "Processing file: %s", filePath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resultCh := make(chan error, 1)

	go func() {
		data, err := decryptAndReadBSON(filePath, s.config.Advanced.FernetKey)
		if err != nil {
			resultCh <- fmt.Errorf("error decrypting and reading file: %w", err)
			return
		}

		if err := s.markFileAsProcessed(filename, folderName, data); err != nil {
			resultCh <- fmt.Errorf("error marking file as processed: %w", err)
			return
		}

		if err := s.sendDecryptedData(filename, folderName, data); err != nil {
			s.logger.Error(ComponentSynchronizer, "Error sending decrypted data for file %s: %v", filename, err)
		}

		resultCh <- nil
	}()

	select {
	case err := <-resultCh:
		if err != nil {
			s.logger.Error(ComponentSynchronizer, "Failed to process file %s: %v", filePath, err)
			return err
		}
		s.logger.Info(ComponentSynchronizer, "Successfully processed file: %s", filePath)
		return nil

	case <-ctx.Done():
		s.logger.Error(ComponentSynchronizer, "Processing timeout for file %s", filePath)
		return fmt.Errorf("processing timeout for file %s", filePath)
	}
}

// sendDecryptedData sends the decrypted data to MQTT
func (s *Synchronizer) sendDecryptedData(filename, folderName string, data map[string]interface{}) error {
	if s.mqttSender == nil {
		return fmt.Errorf("MQTT sender not initialized")
	}

	dataEntry, err := s.mapToDataEntry(data)
	if err != nil {
		return fmt.Errorf("error converting data to DataEntry: %w", err)
	}

	siteId, _ := s.GetSetting("site_id")
	tenantId, _ := s.GetSetting("tenant_id")
	clientId, _ := s.GetSetting("client_id")

	payload := map[string]interface{}{
		"filename":     filename,
		"date_folder":  folderName,
		"tenant_id":    tenantId,
		"client_id":    clientId,
		"site_id":      siteId,
		"processed_at": time.Now().Format(time.RFC3339),
		"data":         dataEntry,
	}

	topic := fmt.Sprintf("%s/data/%s", "jarvist", folderName)

	s.logger.Info(ComponentSynchronizer, "Sending decrypted data from file %s to MQTT topic %s", filename, topic)
	messageID, err := s.mqttSender.SendData(topic, payload)
	if err != nil {
		return fmt.Errorf("failed to send data to MQTT: %w", err)
	}

	s.logger.Info(ComponentSynchronizer, "Successfully queued decrypted data from file %s (Message ID: %d)", filename, messageID)
	return nil
}

// markFileAsProcessed marks a file as processed in the database
func (s *Synchronizer) markFileAsProcessed(filename, dateFolder string, data map[string]interface{}) error {
	dataEntry, err := s.mapToDataEntry(data)
	if err != nil {
		return fmt.Errorf("error converting data to DataEntry: %w", err)
	}

	dataJSON := fmt.Sprintf(`{"id":"%s","cctv_id":%d,"device_id":"%s","in_count":%d,"out_count":%d}`,
		dataEntry.ID, dataEntry.CCTVID, dataEntry.DeviceID, dataEntry.InCount, dataEntry.OutCount)

	processedFile := models.ProcessedFile{
		Filename:    filename,
		DateFolder:  dateFolder,
		DataJSON:    dataJSON,
		ProcessedAt: time.Now(),
	}

	return s.db.Create(&processedFile).Error
}

// mapToDataEntry converts a map to a DataEntry
func (s *Synchronizer) mapToDataEntry(data map[string]interface{}) (DataEntry, error) {
	entry := DataEntry{}

	if v, ok := data["id"]; ok {
		if str, ok := v.(string); ok {
			entry.ID = str
		}
	}

	if v, ok := data["device_id"]; ok {
		if str, ok := v.(string); ok {
			entry.DeviceID = str
		} else if num, ok := v.(int); ok {
			entry.DeviceID = strconv.Itoa(num)
		}
	}

	if v, ok := data["device_timestamp"]; ok {
		if str, ok := v.(string); ok {
			entry.DeviceTimestamp = str
		}
	}

	if v, ok := data["start_time"]; ok {
		if str, ok := v.(string); ok {
			entry.StartTime = str
		}
	}

	if v, ok := data["cctv_id"]; ok {
		switch val := v.(type) {
		case int:
			entry.CCTVID = val
		case float64:
			entry.CCTVID = int(val)
		}
	}

	if v, ok := data["device_timestamp_utc"]; ok {
		if f, ok := v.(float64); ok {
			entry.DeviceTimestampUTC = f
		}
	}

	if v, ok := data["in_count"]; ok {
		switch val := v.(type) {
		case int:
			entry.InCount = val
		case float64:
			entry.InCount = int(val)
		}
	}

	if v, ok := data["out_count"]; ok {
		switch val := v.(type) {
		case int:
			entry.OutCount = val
		case float64:
			entry.OutCount = int(val)
		}
	}

	if v, ok := data["sync_status"]; ok {
		if b, ok := v.(bool); ok {
			entry.SyncStatus = b
		}
	}

	return entry, nil
}

// GetSetting gets a setting from the database
func (s *Synchronizer) GetSetting(key string) (string, error) {
	var setting models.Setting
	result := s.db.Where("key = ?", key).First(&setting)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "", errors.New("setting not found")
	} else if result.Error != nil {
		return "", result.Error
	}

	return setting.Value, nil
}

// GetStatus returns the current status
func (s *Synchronizer) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.watchMutex.Lock()
	watchStatus := s.watchActive
	s.watchMutex.Unlock()

	pendingCount := len(s.pendingFiles)

	return map[string]interface{}{
		"running":          true,
		"in_sync_process":  s.inSyncProcess,
		"watcher_active":   watchStatus,
		"pending_files":    pendingCount,
		"last_status_time": time.Now().Format(time.RFC3339),
	}
}

// GetSyncedFoldersDetails returns details about synced folders
func (s *Synchronizer) GetSyncedFoldersDetails() ([]map[string]interface{}, error) {
	// Query for unique date folders in processed files
	var results []struct {
		DateFolder string
		Count      int64
		LastUpdate time.Time
	}

	err := s.db.Model(&models.ProcessedFile{}).
		Select("date_folder, COUNT(*) as count, MAX(processed_at) as last_update").
		Group("date_folder").
		Find(&results).Error

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	folderDetails := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		folderDetails = append(folderDetails, map[string]interface{}{
			"folder_name":  result.DateFolder,
			"last_checked": result.LastUpdate.Format(time.RFC3339),
			"total_files":  result.Count,
			"fully_synced": true,
		})
	}

	return folderDetails, nil
}

// GetSyncedFoldersList returns a list of synced folders
func (s *Synchronizer) GetSyncedFoldersList() ([]string, error) {
	var folders []string
	err := s.db.Model(&models.ProcessedFile{}).
		Distinct("date_folder").
		Pluck("date_folder", &folders).Error

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return folders, nil
}

// ResyncFolder marks a folder for resyncing
func (s *Synchronizer) ResyncFolder(folderName string) error {
	err := s.db.Where("date_folder = ?", folderName).Delete(&models.ProcessedFile{}).Error
	if err != nil {
		return fmt.Errorf("failed to clear processed files: %w", err)
	}

	s.logger.Info(ComponentSynchronizer, "Folder %s marked for resyncing, removed from processed files", folderName)

	// Trigger a new scan
	go func() {
		processedFiles := make(map[string]bool)
		var files []models.ProcessedFile
		if err := s.db.Find(&files).Error; err != nil {
			s.logger.Error(ComponentSynchronizer, "Failed to query processed files: %v", err)
			return
		}

		for _, file := range files {
			processedFiles[file.Filename] = true
		}

		folderPath := filepath.Join(s.config.BaseConfig.ServicesDataDir, folderName)
		count := s.processFolderFiles(folderPath, folderName, processedFiles)
		s.logger.Info(ComponentSynchronizer, "Resynced %d files from folder %s", count, folderName)
	}()

	return nil
}

// GetFileProcessingStatus checks if a file has been processed
func (s *Synchronizer) GetFileProcessingStatus(filename string, folderName string) (map[string]interface{}, error) {
	var count int64
	if err := s.db.Model(&models.ProcessedFile{}).
		Where("filename = ? AND date_folder = ?", filename, folderName).
		Count(&count).Error; err != nil {
		return nil, err
	}

	if count == 0 {
		return map[string]interface{}{
			"filename":  filename,
			"folder":    folderName,
			"processed": false,
		}, nil
	}

	var processedFile models.ProcessedFile
	if err := s.db.Where("filename = ? AND date_folder = ?", filename, folderName).
		First(&processedFile).Error; err != nil {
		return nil, err
	}

	var dataMap map[string]interface{}
	if err := json.Unmarshal([]byte(processedFile.DataJSON), &dataMap); err != nil {
		dataMap = map[string]interface{}{
			"error": "Failed to parse data JSON",
		}
	}

	return map[string]interface{}{
		"filename":     filename,
		"folder":       folderName,
		"processed":    true,
		"processed_at": processedFile.ProcessedAt.Format(time.RFC3339),
		"data":         dataMap,
	}, nil
}

// SendSyncFolderSummary sends a summary of sync status
func (s *Synchronizer) SendSyncFolderSummary() error {
	if s.mqttSender == nil {
		return fmt.Errorf("MQTT sender not initialized")
	}

	folderDetails, err := s.GetSyncedFoldersDetails()
	if err != nil {
		return fmt.Errorf("failed to get folder details: %w", err)
	}

	summary := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"total_folders":  len(folderDetails),
		"synced_folders": folderDetails,
	}

	topic := fmt.Sprintf("%s/summary/folders", s.config.MQTT.Topic)

	s.logger.Info(ComponentSynchronizer, "Sending sync folder summary to MQTT topic %s", topic)
	messageID, err := s.mqttSender.SendData(topic, summary)
	if err != nil {
		return fmt.Errorf("failed to send summary to MQTT: %w", err)
	}

	s.logger.Info(ComponentSynchronizer, "Successfully queued sync folder summary (Message ID: %d)", messageID)
	return nil
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// decryptAndReadBSON decrypts and reads a BSON file
func decryptAndReadBSON(filePath, fernetKey string) (map[string]interface{}, error) {
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	key, err := fernet.DecodeKey(fernetKey)
	if err != nil {
		return nil, fmt.Errorf("invalid Fernet key: %w", err)
	}

	msg := fernet.VerifyAndDecrypt(encryptedData, 0, []*fernet.Key{key})
	if msg == nil {
		return nil, fmt.Errorf("failed to decrypt: invalid token or key")
	}

	var result map[string]interface{}
	err = bson.Unmarshal(msg, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BSON: %w", err)
	}

	return result, nil
}
