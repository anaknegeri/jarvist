package sync

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"jarvist/internal/common/models"
	"jarvist/internal/syncmanager/config"
	"jarvist/internal/syncmanager/mqtt"
	"jarvist/pkg/logger"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/fernet/fernet-go"
	"gopkg.in/mgo.v2/bson"
	"gorm.io/gorm"
)

const (
	ComponentSynchronizer = "synchronizer"
	DateFolderPattern     = "20060102"
)

type Synchronizer struct {
	config        *config.Config
	logger        *logger.Logger
	stopCh        chan struct{}
	inSyncProcess bool
	mu            sync.Mutex
	db            *gorm.DB
	mqttSender    *mqtt.Sender // Added MQTT sender
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

type PayloadData struct {
	Data []DataEntry `json:"data"`
}

type CSVRecord struct {
	Filename   string
	SyncStatus bool
}

// NewSynchronizer creates a new synchronizer with MQTT sender
func NewSynchronizer(config *config.Config, logger *logger.Logger, db *gorm.DB, mqttSender *mqtt.Sender) *Synchronizer {
	return &Synchronizer{
		config:     config,
		logger:     logger,
		stopCh:     make(chan struct{}),
		db:         db,
		mqttSender: mqttSender,
	}
}

func (s *Synchronizer) Start() error {
	s.logger.Info(ComponentSynchronizer, "Starting synchronizer")

	interval := time.Duration(s.config.Sync.Interval) * time.Second

	ticker := time.NewTicker(interval)

	// Run initial sync
	go func() {
		s.syncData()
		if err := s.UpdateFolderSyncStatus(); err != nil {
			s.logger.Warn(ComponentSynchronizer, "Failed to update folder sync status: %v", err)
		}
	}()

	// Start background goroutine for periodic syncing
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				if !s.inSyncProcess {
					s.inSyncProcess = true
					s.mu.Unlock()

					func() {
						defer func() {
							s.mu.Lock()
							s.inSyncProcess = false
							s.mu.Unlock()
						}()
						s.syncData()
					}()
				} else {
					s.mu.Unlock()
					s.logger.Info(ComponentSynchronizer, "Skipping sync cycle as previous sync is still in progress")
				}

			case <-s.stopCh:
				s.logger.Info(ComponentSynchronizer, "Synchronizer stopped")
				return
			}
		}
	}()

	return nil
}

func (s *Synchronizer) Stop() error {
	s.logger.Info(ComponentSynchronizer, "Stopping synchronizer...")
	close(s.stopCh)

	time.Sleep(1 * time.Second)

	s.logger.Info(ComponentSynchronizer, "Synchronizer stopped")
	return nil
}

func (s *Synchronizer) syncData() {
	s.logger.Info(ComponentSynchronizer, "Starting data synchronization")

	syncedFolders, err := s.getFullySyncedFolders()
	if err != nil {
		s.logger.Warn(ComponentSynchronizer, "Could not retrieve synced folders: %v", err)
	}

	dateFolders, err := s.findDateFolders()
	if err != nil {
		s.logger.Warn(ComponentSynchronizer, "Failed to find date folders: %v", err)
		return
	}

	if len(dateFolders) == 0 {
		s.logger.Info(ComponentSynchronizer, "No date folders found matching the YYYYMMDD pattern")
		return
	}

	for _, dateFolder := range dateFolders {
		folderName := filepath.Base(dateFolder)

		// Check if this folder is fully synced
		isSynced := false
		var syncedFileCount int = 0
		if syncedFolders != nil {
			if _, ok := syncedFolders[folderName]; ok {
				isSynced = true
				// Get the synced file count from database
				var folder models.SyncedFolder
				if err := s.db.Where("folder_name = ?", folderName).First(&folder).Error; err == nil {
					syncedFileCount = folder.TotalFiles
				}
			}
		}

		if err := s.processDateFolder(dateFolder, folderName, isSynced, syncedFileCount); err != nil {
			s.logger.Error(ComponentSynchronizer, "Error processing folder %s: %v", folderName, err)
		}
	}

	if err := s.UpdateFolderSyncStatus(); err != nil {
		s.logger.Warn(ComponentSynchronizer, "Failed to update folder sync status: %v", err)
	}
}

func (s *Synchronizer) processDateFolder(dateFolder, folderName string, previouslySynced bool, syncedFileCount int) error {
	csvPath := filepath.Join(dateFolder, "data.csv")

	csvEntries, err := readCSVFile(csvPath)
	if err != nil {
		return fmt.Errorf("could not read CSV file for %s: %w", folderName, err)
	}

	if previouslySynced && len(csvEntries) == syncedFileCount && isAllSynced(csvEntries) {
		s.logger.Info(ComponentSynchronizer, "Folder %s is already synced with %d files and no new files detected",
			folderName, syncedFileCount)
		return nil
	}

	if previouslySynced {
		s.logger.Info(ComponentSynchronizer, "Folder %s was previously synced with %d files, but now has %d files. Processing new files.",
			folderName, syncedFileCount, len(csvEntries))
	}

	for i := range csvEntries {
		if csvEntries[i].SyncStatus {
			s.logger.Info(ComponentSynchronizer, "Skipping already processed file: %s", csvEntries[i].Filename)
			continue
		}

		var count int64
		if err := s.db.Model(&models.ProcessedFile{}).
			Where("filename = ? AND date_folder = ?", csvEntries[i].Filename, folderName).
			Count(&count).Error; err != nil {
			s.logger.Error(ComponentSynchronizer, "Database error checking file %s: %v",
				csvEntries[i].Filename, err)
			continue
		}

		if count > 0 {
			s.logger.Info(ComponentSynchronizer, "File %s is already processed in database", csvEntries[i].Filename)
			csvEntries[i].SyncStatus = true
			continue
		}

		filePath := filepath.Join(s.config.BaseConfig.ServicesDataDir, csvEntries[i].Filename)
		if err := s.processFile(filePath, csvEntries[i].Filename, folderName); err != nil {
			s.logger.Error(ComponentSynchronizer, "Error processing file %s: %v", filePath, err)
			continue
		}

		csvEntries[i].SyncStatus = true
	}

	if err := writeCSVFile(csvPath, csvEntries); err != nil {
		return fmt.Errorf("error updating CSV file %s: %w", csvPath, err)
	}

	if isAllSynced(csvEntries) {
		return s.markFolderAsSynced(folderName, len(csvEntries))
	}

	return nil
}

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

func (s *Synchronizer) processFile(filePath, filename, folderName string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", filePath)
	}

	s.logger.Info(ComponentSynchronizer, "Processing file: %s", filePath)

	data, err := decryptAndReadBSON(filePath, s.config.Advanced.FernetKey)
	if err != nil {
		return fmt.Errorf("error decrypting and reading file: %w", err)
	}

	// Mark file as processed in database
	if err := s.markFileAsProcessed(filename, folderName, data); err != nil {
		return fmt.Errorf("error marking file as processed: %w", err)
	}

	if err := s.sendDecryptedData(filename, folderName, data); err != nil {
		s.logger.Error(ComponentSynchronizer, "Error sending decrypted data for file %s: %v", filename, err)
	}

	return nil
}

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

// sendDecryptedData sends the decrypted data to MQTT
func (s *Synchronizer) sendDecryptedData(filename, folderName string, data map[string]interface{}) error {

	if s.mqttSender == nil {
		return fmt.Errorf("MQTT sender not initialized")
	}

	// Convert the data to DataEntry for more control
	dataEntry, err := s.mapToDataEntry(data)
	if err != nil {
		return fmt.Errorf("error converting data to DataEntry: %w", err)
	}

	siteId, _ := s.GetSetting("site_id")
	tenantId, _ := s.GetSetting("tenant_id")
	clientId, _ := s.GetSetting("client_id")

	// Create payload with metadata
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

func isAllSynced(entries []CSVRecord) bool {
	for _, entry := range entries {
		if !entry.SyncStatus {
			return false
		}
	}
	return len(entries) > 0
}

func readCSVFile(filePath string) ([]CSVRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data: %w", err)
	}

	var entries []CSVRecord

	if len(records) == 0 {
		return entries, nil
	}

	header := records[0]
	if len(header) != 2 || header[0] != "filename" || header[1] != "sync_status" {
		return nil, fmt.Errorf("invalid CSV header: expected [filename, sync_status], got %v", header)
	}

	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 2 {
			continue
		}

		syncStatus, err := strconv.ParseBool(record[1])
		if err != nil {
			syncStatus = false
		}

		entries = append(entries, CSVRecord{
			Filename:   record[0],
			SyncStatus: syncStatus,
		})
	}

	return entries, nil
}

func writeCSVFile(filePath string, entries []CSVRecord) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"filename", "sync_status"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, entry := range entries {
		if err := writer.Write([]string{
			entry.Filename,
			strconv.FormatBool(entry.SyncStatus),
		}); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

func (s *Synchronizer) getFullySyncedFolders() (map[string]bool, error) {
	result := make(map[string]bool)

	var syncedFolders []models.SyncedFolder
	if err := s.db.Where("fully_synced = ?", true).Find(&syncedFolders).Error; err != nil {
		return nil, fmt.Errorf("database error getting synced folders: %w", err)
	}

	for _, folder := range syncedFolders {
		result[folder.FolderName] = true
	}

	return result, nil
}

func (s *Synchronizer) markFolderAsSynced(folderName string, totalFiles int) error {
	var folder models.SyncedFolder

	result := s.db.Where("folder_name = ?", folderName).First(&folder)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error checking folder: %w", result.Error)
	}

	now := time.Now()

	if result.Error == gorm.ErrRecordNotFound {
		folder = models.SyncedFolder{
			FolderName:  folderName,
			LastChecked: now,
			FullySynced: true,
			TotalFiles:  totalFiles,
		}
		return s.db.Create(&folder).Error
	}

	folder.LastChecked = now
	folder.FullySynced = true
	folder.TotalFiles = totalFiles

	return s.db.Save(&folder).Error
}

func (s *Synchronizer) markFileAsProcessed(filename, dateFolder string, data map[string]interface{}) error {
	dataEntry, err := s.mapToDataEntry(data)
	if err != nil {
		return fmt.Errorf("error converting data to DataEntry: %w", err)
	}

	// Create JSON representation of essential data
	dataJSON := fmt.Sprintf(`{"id":"%s","cctv_id":%d,"device_id":"%s","in_count":%d,"out_count":%d}`,
		dataEntry.ID, dataEntry.CCTVID, dataEntry.DeviceID, dataEntry.InCount, dataEntry.OutCount)

	// Create processed file record
	processedFile := models.ProcessedFile{
		Filename:    filename,
		DateFolder:  dateFolder,
		DataJSON:    dataJSON,
		ProcessedAt: time.Now(),
	}

	return s.db.Create(&processedFile).Error
}

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

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (s *Synchronizer) UpdateFolderSyncStatus() error {
	var dateFolders []string

	err := filepath.WalkDir(s.config.BaseConfig.ServicesDataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || path == s.config.BaseConfig.ServicesDataDir {
			return nil
		}

		dirName := filepath.Base(path)
		if len(dirName) == 8 && isNumeric(dirName) {
			_, err := time.Parse(DateFolderPattern, dirName)
			if err == nil {
				dateFolders = append(dateFolders, path)
			}
		}

		return filepath.SkipDir
	})

	if err != nil {
		return fmt.Errorf("error walking data directory: %w", err)
	}

	for _, folder := range dateFolders {
		folderName := filepath.Base(folder)
		csvPath := filepath.Join(folder, "data.csv")

		entries, err := readCSVFile(csvPath)
		if err != nil {
			s.logger.Warn(ComponentSynchronizer, "Could not read CSV for folder %s: %v", folderName, err)
			continue
		}

		// Check if the folder exists in the database
		var existingFolder models.SyncedFolder
		result := s.db.Where("folder_name = ?", folderName).First(&existingFolder)

		if result.Error == nil {
			// Folder exists in database
			if len(entries) != existingFolder.TotalFiles || !isAllSynced(entries) {
				// Either the number of files changed or not all files are synced
				existingFolder.FullySynced = isAllSynced(entries)
				existingFolder.TotalFiles = len(entries)
				existingFolder.LastChecked = time.Now()

				if err := s.db.Save(&existingFolder).Error; err != nil {
					s.logger.Warn(ComponentSynchronizer, "Failed to update sync status for folder %s: %v", folderName, err)
				}
			}
		} else if result.Error == gorm.ErrRecordNotFound {
			// Folder doesn't exist yet
			if isAllSynced(entries) {
				err = s.markFolderAsSynced(folderName, len(entries))
				if err != nil {
					s.logger.Warn(ComponentSynchronizer, "Failed to update sync status for folder %s: %v", folderName, err)
				}
			}
		} else {
			// Other database error
			s.logger.Warn(ComponentSynchronizer, "Database error checking folder %s: %v", folderName, result.Error)
		}
	}

	return nil
}

// SendSyncFolderSummary sends a summary of synced folders via MQTT
func (s *Synchronizer) SendSyncFolderSummary() error {
	if s.mqttSender == nil {
		return fmt.Errorf("MQTT sender not initialized")
	}

	var syncedFolders []models.SyncedFolder
	if err := s.db.Find(&syncedFolders).Error; err != nil {
		return fmt.Errorf("database error getting synced folders: %w", err)
	}

	// Create summary
	summary := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"total_folders":  len(syncedFolders),
		"synced_folders": syncedFolders,
	}

	// Create topic for the summary
	topic := fmt.Sprintf("%s/summary/folders", s.config.MQTT.Topic)

	// Send the summary
	s.logger.Info(ComponentSynchronizer, "Sending sync folder summary to MQTT topic %s", topic)
	messageID, err := s.mqttSender.SendData(topic, summary)
	if err != nil {
		return fmt.Errorf("failed to send summary to MQTT: %w", err)
	}

	s.logger.Info(ComponentSynchronizer, "Successfully queued sync folder summary (Message ID: %d)", messageID)
	return nil
}

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

func (s *Synchronizer) SyncData() {
	s.syncData()
}

// GetStatus returns the current synchronizer status
func (s *Synchronizer) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"running":         true,
		"in_sync_process": s.inSyncProcess,
		"last_sync_time":  time.Now().Format(time.RFC3339), // Replace with actual last sync time
	}
}

// GetSyncedFoldersList returns a list of fully synced folders
func (s *Synchronizer) GetSyncedFoldersList() ([]string, error) {
	syncedFolders, err := s.getFullySyncedFolders()
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(syncedFolders))
	for folder := range syncedFolders {
		result = append(result, folder)
	}

	return result, nil
}

// GetSyncedFoldersDetails returns detailed information about synced folders
func (s *Synchronizer) GetSyncedFoldersDetails() ([]map[string]interface{}, error) {
	var syncedFolders []models.SyncedFolder
	if err := s.db.Find(&syncedFolders).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(syncedFolders))
	for _, folder := range syncedFolders {
		result = append(result, map[string]interface{}{
			"folder_name":  folder.FolderName,
			"last_checked": folder.LastChecked.Format(time.RFC3339),
			"fully_synced": folder.FullySynced,
			"total_files":  folder.TotalFiles,
		})
	}

	return result, nil
}

// ResyncFolder forces a resync of a specific folder
func (s *Synchronizer) ResyncFolder(folderName string) error {
	// Mark folder as not fully synced in database
	var folder models.SyncedFolder
	result := s.db.Where("folder_name = ?", folderName).First(&folder)

	if result.Error != nil {
		return result.Error
	}

	folder.FullySynced = false
	folder.LastChecked = time.Now()

	if err := s.db.Save(&folder).Error; err != nil {
		return err
	}

	// Trigger sync process
	go s.syncData()

	return nil
}

// GetFileProcessingStatus gets processing status for a file
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
