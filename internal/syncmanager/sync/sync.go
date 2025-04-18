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
	"strings"
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
	mqttSender    *mqtt.Sender
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

	go func() {
		s.SyncData()
		if err := s.UpdateFolderSyncStatus(); err != nil {
			s.logger.Warn(ComponentSynchronizer, "Failed to update folder sync status: %v", err)
		}
	}()

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
						s.SyncData()
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

func (s *Synchronizer) SyncData() {
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

		isSynced := false
		var syncedFileCount int = 0
		if syncedFolders != nil {
			if _, ok := syncedFolders[folderName]; ok {
				isSynced = true
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

func (s *Synchronizer) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"running":         true,
		"in_sync_process": s.inSyncProcess,
		"last_sync_time":  time.Now().Format(time.RFC3339),
	}
}

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

func (s *Synchronizer) ResyncFolder(folderName string) error {
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

	go s.SyncData()

	return nil
}

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

func (s *Synchronizer) processDateFolder(dateFolder, folderName string, previouslySynced bool, syncedFileCount int) error {
	if previouslySynced {
		actualFileCount := 0
		entries, err := os.ReadDir(dateFolder)
		if err != nil {
			return fmt.Errorf("could not read directory %s: %w", folderName, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && entry.Name() != "data.csv" && strings.HasSuffix(entry.Name(), ".json.bson") {
				actualFileCount++
			}
		}

		if actualFileCount == syncedFileCount {
			s.logger.Info(ComponentSynchronizer, "Folder %s is already synced with %d files and file count matches, skipping",
				folderName, syncedFileCount)
			return nil
		}

		s.logger.Info(ComponentSynchronizer, "Folder %s has %d files but database shows %d synced files, checking for changes",
			folderName, actualFileCount, syncedFileCount)
	}

	csvPath := filepath.Join(dateFolder, "data.csv")

	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		s.logger.Info(ComponentSynchronizer, "Creating new CSV file for folder %s", folderName)

		if err := writeCSVFile(csvPath, []CSVRecord{}); err != nil {
			return fmt.Errorf("failed to create CSV file for %s: %w", folderName, err)
		}
	}

	csvEntries, err := readCSVFile(csvPath)
	if err != nil {
		return fmt.Errorf("could not read CSV file for %s: %w", folderName, err)
	}

	allDataFiles, err := s.getDataFilesInDirectory(dateFolder)
	if err != nil {
		return fmt.Errorf("could not read directory %s: %w", folderName, err)
	}

	existingEntries := make(map[string]bool)
	for _, entry := range csvEntries {
		existingEntries[entry.Filename] = true
	}

	var newFiles []string
	for _, file := range allDataFiles {
		fileName := fmt.Sprintf("%s/%s", folderName, file)
		if !existingEntries[fileName] {
			newFiles = append(newFiles, file)
			csvEntries = append(csvEntries, CSVRecord{
				Filename:   fileName,
				SyncStatus: false,
			})
			s.logger.Info(ComponentSynchronizer, "Found new file: %s in folder %s", file, folderName)
		}
	}

	if len(newFiles) > 0 {
		s.logger.Info(ComponentSynchronizer, "Adding %d new files to CSV for folder %s", len(newFiles), folderName)

		if err := writeCSVFile(csvPath, csvEntries); err != nil {
			return fmt.Errorf("error updating CSV file with new files %s: %w", csvPath, err)
		}

		previouslySynced = false
	}

	if previouslySynced && len(csvEntries) == syncedFileCount && isAllSynced(csvEntries) {
		s.logger.Info(ComponentSynchronizer, "Folder %s is already synced with %d files and no new files detected",
			folderName, syncedFileCount)
		return nil
	}

	if previouslySynced && len(csvEntries) != syncedFileCount {
		s.logger.Warning(ComponentSynchronizer,
			"Folder %s was previously synced with %d files, but now has %d files. Processing all unsynced files.",
			folderName, syncedFileCount, len(csvEntries))
	}

	for i := range csvEntries {
		if csvEntries[i].SyncStatus {
			s.logger.Debug(ComponentSynchronizer, "Skipping already processed file: %s", csvEntries[i].Filename)
			continue
		}

		filePath := filepath.Join(s.config.BaseConfig.ServicesDataDir, csvEntries[i].Filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			s.logger.Warning(ComponentSynchronizer, "File %s is in CSV but does not exist in filesystem", csvEntries[i].Filename)
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

		if err := s.processFile(filePath, csvEntries[i].Filename, folderName); err != nil {
			s.logger.Error(ComponentSynchronizer, "Error processing file %s: %v", filePath, err)
			continue
		}

		csvEntries[i].SyncStatus = true
	}

	if err := writeCSVFile(csvPath, csvEntries); err != nil {
		return fmt.Errorf("error updating CSV file %s: %w", csvPath, err)
	}

	finalFiles, err := s.getDataFilesInDirectory(dateFolder)
	if err == nil {
		existingEntries = make(map[string]bool)
		for _, entry := range csvEntries {
			existingEntries[entry.Filename] = true
		}

		var missedFiles []string
		for _, file := range finalFiles {
			if !existingEntries[file] {
				missedFiles = append(missedFiles, file)
			}
		}

		if len(missedFiles) > 0 {
			s.logger.Warning(ComponentSynchronizer, "Found %d files after processing that weren't in CSV. Adding them now.", len(missedFiles))

			for _, file := range missedFiles {
				fileName := fmt.Sprintf("%s/%s", folderName, file)
				csvEntries = append(csvEntries, CSVRecord{
					Filename:   fileName,
					SyncStatus: false,
				})
			}

			if err := writeCSVFile(csvPath, csvEntries); err != nil {
				s.logger.Error(ComponentSynchronizer, "Error updating CSV file with missed files: %v", err)
			}

			return nil
		}
	}

	if isAllSynced(csvEntries) {
		return s.markFolderAsSynced(folderName, len(csvEntries))
	}

	return nil
}

func (s *Synchronizer) getDataFilesInDirectory(dirPath string) ([]string, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dataFiles []string
	for _, file := range files {
		if file.IsDir() || file.Name() == "data.csv" {
			continue
		}

		if strings.HasSuffix(file.Name(), ".json.bson") {
			dataFiles = append(dataFiles, file.Name())
		}
	}

	return dataFiles, nil
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

		var existingFolder models.SyncedFolder
		result := s.db.Where("folder_name = ?", folderName).First(&existingFolder)

		if result.Error == nil && existingFolder.FullySynced {
			actualFileCount := 0
			entries, err := os.ReadDir(folder)
			if err != nil {
				s.logger.Warn(ComponentSynchronizer, "Could not read directory %s: %v", folderName, err)
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() && entry.Name() != "data.csv" && strings.HasSuffix(entry.Name(), ".json.bson") {
					actualFileCount++
				}
			}

			if actualFileCount == existingFolder.TotalFiles {
				s.logger.Debug(ComponentSynchronizer, "Folder %s file count unchanged (%d files), skipping detailed check",
					folderName, actualFileCount)

				existingFolder.LastChecked = time.Now()
				if err := s.db.Save(&existingFolder).Error; err != nil {
					s.logger.Warn(ComponentSynchronizer, "Failed to update last checked timestamp for folder %s: %v",
						folderName, err)
				}
				continue
			}

			s.logger.Info(ComponentSynchronizer, "Folder %s file count changed (DB: %d, Actual: %d), performing detailed check",
				folderName, existingFolder.TotalFiles, actualFileCount)
		}

		csvPath := filepath.Join(folder, "data.csv")

		if _, err := os.Stat(csvPath); os.IsNotExist(err) {
			s.logger.Info(ComponentSynchronizer, "Creating new CSV file for folder %s during status update", folderName)
			if err := writeCSVFile(csvPath, []CSVRecord{}); err != nil {
				s.logger.Warn(ComponentSynchronizer, "Could not create CSV for folder %s: %v", folderName, err)
				continue
			}
		}

		actualFiles, err := s.getDataFilesInDirectory(folder)
		if err != nil {
			s.logger.Warn(ComponentSynchronizer, "Could not read directory %s: %v", folderName, err)
			continue
		}

		entries, err := readCSVFile(csvPath)
		if err != nil {
			s.logger.Warn(ComponentSynchronizer, "Could not read CSV for folder %s: %v", folderName, err)
			continue
		}

		csvFiles := make(map[string]bool)
		for _, entry := range entries {
			csvFiles[entry.Filename] = true
		}

		var newFiles []string
		for _, file := range actualFiles {
			fileName := fmt.Sprintf("%s/%s", folderName, file)
			if !csvFiles[fileName] {
				newFiles = append(newFiles, file)
				entries = append(entries, CSVRecord{
					Filename:   fileName,
					SyncStatus: false,
				})
			}
		}

		if len(newFiles) > 0 {
			s.logger.Info(ComponentSynchronizer, "Found %d new files for folder %s during status update", len(newFiles), folderName)
			if err := writeCSVFile(csvPath, entries); err != nil {
				s.logger.Warn(ComponentSynchronizer, "Failed to update CSV for folder %s: %v", folderName, err)
				continue
			}
		}

		if result.Error == nil {
			if len(entries) != existingFolder.TotalFiles || !isAllSynced(entries) {
				existingFolder.FullySynced = isAllSynced(entries)
				existingFolder.TotalFiles = len(entries)
				existingFolder.LastChecked = time.Now()

				if err := s.db.Save(&existingFolder).Error; err != nil {
					s.logger.Warn(ComponentSynchronizer, "Failed to update sync status for folder %s: %v", folderName, err)
				}
			} else {
				existingFolder.LastChecked = time.Now()
				if err := s.db.Save(&existingFolder).Error; err != nil {
					s.logger.Warn(ComponentSynchronizer, "Failed to update last checked timestamp for folder %s: %v",
						folderName, err)
				}
			}
		} else if result.Error == gorm.ErrRecordNotFound {
			if isAllSynced(entries) {
				err = s.markFolderAsSynced(folderName, len(entries))
				if err != nil {
					s.logger.Warn(ComponentSynchronizer, "Failed to update sync status for folder %s: %v", folderName, err)
				}
			} else if len(entries) > 0 {
				folder := models.SyncedFolder{
					FolderName:  folderName,
					LastChecked: time.Now(),
					FullySynced: false,
					TotalFiles:  len(entries),
				}
				if err := s.db.Create(&folder).Error; err != nil {
					s.logger.Warn(ComponentSynchronizer, "Failed to create sync status for folder %s: %v", folderName, err)
				}
			}
		} else {
			s.logger.Warn(ComponentSynchronizer, "Database error checking folder %s: %v", folderName, result.Error)
		}
	}

	return nil
}

func (s *Synchronizer) SendSyncFolderSummary() error {
	if s.mqttSender == nil {
		return fmt.Errorf("MQTT sender not initialized")
	}

	var syncedFolders []models.SyncedFolder
	if err := s.db.Find(&syncedFolders).Error; err != nil {
		return fmt.Errorf("database error getting synced folders: %w", err)
	}

	summary := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"total_folders":  len(syncedFolders),
		"synced_folders": syncedFolders,
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
