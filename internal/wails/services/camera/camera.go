package camera

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"jarvist/internal/common/config"
	"jarvist/internal/common/ffmpeg"
	"jarvist/internal/common/models"
	"jarvist/internal/wails/services/processmanager"
	"jarvist/internal/wails/services/setting"
	"jarvist/pkg/logger"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gorm.io/gorm"
)

type CameraService struct {
	DB                 *gorm.DB
	app                *application.App // Ubah dari ctx ke app
	connectionStatuses map[string]CameraConnectionStatus
	statusMutex        sync.RWMutex
	checkInterval      time.Duration
	backgroundRunning  bool
	backgroundCtx      context.Context
	backgroundCancelFn context.CancelFunc
	concurrencyLimit   int
	settingService     *setting.SettingsService
	config             *config.Config
	process            *processmanager.ProcessManagerService
	logger             *logger.ContextLogger
}

type SyncResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type CameraConnectionStatus struct {
	CameraID      uint      `json:"camera_id"`
	CameraUUID    string    `json:"camera_uuid"`
	IsConnected   bool      `json:"is_connected"`
	LastChecked   time.Time `json:"last_checked"`
	StatusMessage string    `json:"status_message,omitempty"`
	Error         string    `json:"error,omitempty"`
}

type CameraSync struct {
	ID          uint   `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	LocationID  string `json:"location_id"`
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags,omitempty"`
	Schema      string `json:"schema"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Path        string `json:"path"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Direction   string `json:"direction,omitempty"`
	Status      string `json:"status,omitempty"`
	Payload     string `json:"payload,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func New(db *gorm.DB, settingService *setting.SettingsService, cfg *config.Config, logger *logger.ContextLogger, process *processmanager.ProcessManagerService) *CameraService {
	return &CameraService{
		DB:                 db,
		connectionStatuses: make(map[string]CameraConnectionStatus),
		checkInterval:      5 * time.Minute,
		concurrencyLimit:   5,
		backgroundRunning:  false,
		settingService:     settingService,
		config:             cfg,
		process:            process,
		logger:             logger,
	}
}

func (s *CameraService) InitService(app *application.App) {
	s.app = app
	s.backgroundCtx, s.backgroundCancelFn = context.WithCancel(context.Background())
}

func (s *CameraService) StartBackgroundChecking() {
	if s.backgroundRunning {
		return
	}

	s.backgroundRunning = true
	go s.runBackgroundChecker()
}

func (s *CameraService) StopBackgroundChecking() {
	if !s.backgroundRunning {
		return // Not running
	}

	if s.backgroundCancelFn != nil {
		s.backgroundCancelFn()
	}
	s.backgroundRunning = false
}

func (s *CameraService) SetCheckInterval(interval time.Duration) {
	s.checkInterval = interval
}

func (s *CameraService) runBackgroundChecker() {
	s.logger.Info("Starting background camera connection checker")

	s.checkAllCameraConnections()

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAllCameraConnections()
		case <-s.backgroundCtx.Done():
			s.logger.Info("Background camera connection checker stopped")
			return
		}
	}
}

func (s *CameraService) checkAllCameraConnections() {
	s.logger.Info("Running camera connection check for all cameras")

	cameras, err := s.ListCamera()
	if err != nil {
		s.logger.Error("Error getting cameras for connection check: %v", err)
		return
	}

	var wg sync.WaitGroup

	semaphore := make(chan struct{}, s.concurrencyLimit)

	for _, camera := range cameras {
		wg.Add(1)

		semaphore <- struct{}{}

		go func(cam models.Camera) {
			defer wg.Done()
			defer func() { <-semaphore }()

			s.checkCameraConnection(&cam)
		}(camera)
	}

	wg.Wait()
	s.broadcastStatusUpdate()
}

func (s *CameraService) checkCameraConnection(camera *models.Camera) {
	s.logger.Info("Checking connection for camera %s (ID: %d)", camera.Name, camera.ID)

	rtspConfig := ffmpeg.RTSPConfig{
		Schema:   camera.Schema,
		Host:     camera.Host,
		Port:     camera.Port,
		Path:     camera.Path,
		Username: camera.Username,
		Password: camera.Password,
	}

	options := ffmpeg.RTSPOptions{
		TakeScreenshot: false,
	}

	responseStr := ffmpeg.CheckRTSPConnectionWithConfig(rtspConfig, options)

	var response ffmpeg.ResponseJSON
	if err := json.Unmarshal([]byte(responseStr), &response); err != nil {
		s.logger.Error("Error parsing RTSP check response: %v", err)
		return
	}

	status := CameraConnectionStatus{
		CameraID:      camera.ID,
		CameraUUID:    camera.UUID,
		IsConnected:   response.Success,
		LastChecked:   time.Now(),
		StatusMessage: response.Message,
	}

	if !response.Success && response.Error != "" {
		status.Error = response.Error
	}

	s.statusMutex.Lock()
	s.connectionStatuses[camera.UUID] = status
	s.statusMutex.Unlock()

	newStatus := "online"
	if !response.Success {
		newStatus = "offline"
	}

	if camera.Status != newStatus {
		camera.Status = newStatus
		if err := s.DB.Save(camera).Error; err != nil {
			s.logger.Error("Error updating camera status: %v", err)
		}
	}
}

func (s *CameraService) broadcastStatusUpdate() {
	if s.app == nil {
		return
	}

	s.statusMutex.RLock()
	defer s.statusMutex.RUnlock()

	statusesCopy := make(map[string]CameraConnectionStatus)
	for k, v := range s.connectionStatuses {
		statusesCopy[k] = v
	}

	// Menggunakan EmitEvent dari app instance untuk Wails v3
	s.app.EmitEvent("camera:status-update", statusesCopy)
}

func (s *CameraService) GetConnectionStatus(cameraUUID string) (CameraConnectionStatus, bool) {
	s.statusMutex.RLock()
	defer s.statusMutex.RUnlock()

	status, exists := s.connectionStatuses[cameraUUID]
	return status, exists
}

func (s *CameraService) GetAllConnectionStatuses() map[string]CameraConnectionStatus {
	s.statusMutex.RLock()
	defer s.statusMutex.RUnlock()

	statusesCopy := make(map[string]CameraConnectionStatus)
	for k, v := range s.connectionStatuses {
		statusesCopy[k] = v
	}

	return statusesCopy
}

func (s *CameraService) CheckCameraConnectionNow(id uint) (CameraConnectionStatus, error) {
	camera, err := s.GetCameraByID(id)
	if err != nil {
		return CameraConnectionStatus{}, err
	}

	s.checkCameraConnection(camera)

	status, exists := s.GetConnectionStatus(camera.UUID)
	if !exists {
		return CameraConnectionStatus{}, errors.New("status not found after check")
	}

	return status, nil
}

func (s *CameraService) GetCamerasWithStatus() ([]map[string]interface{}, error) {
	cameras, err := s.ListCamera()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(cameras))

	for _, camera := range cameras {
		cameraMap := map[string]interface{}{
			"ID":        camera.ID,
			"UUID":      camera.UUID,
			"Name":      camera.Name,
			"Location":  camera.Location,
			"Schema":    camera.Schema,
			"Host":      camera.Host,
			"Port":      camera.Port,
			"Path":      camera.Path,
			"Username":  camera.Username,
			"Direction": camera.Direction,
			"Status":    camera.Status,
			"CreatedAt": camera.CreatedAt,
		}

		if status, exists := s.GetConnectionStatus(camera.UUID); exists {
			cameraMap["is_connected"] = status.IsConnected
			cameraMap["last_checked"] = status.LastChecked.Format(time.RFC3339)
			if status.StatusMessage != "" {
				cameraMap["status_message"] = status.StatusMessage
			}
		}

		result = append(result, cameraMap)
	}

	return result, nil
}

func (s *CameraService) CreateCamera(input models.CameraInput) (*models.Camera, error) {
	var location models.Location
	if err := s.DB.Where("id = ?", input.Location).First(&location).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("location not found")
		}
		return nil, err
	}

	camera := &models.Camera{
		Name:        input.Name,
		LocationID:  location.ID,
		Schema:      input.Schema,
		Host:        input.Host,
		Port:        input.Port,
		Path:        input.Path,
		Username:    input.Username,
		Password:    input.Password,
		ImageData:   input.ImageData,
		Direction:   input.Direction,
		Description: input.Description,
		Tags:        input.Tags,
		Status:      "offline",
	}

	payload := map[string]interface{}{
		"lines": input.Lines,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	camera.Payload = string(payloadJSON)

	if err := s.DB.Create(camera).Error; err != nil {
		return nil, err
	}

	if s.backgroundRunning {
		go s.checkCameraConnection(camera)
	}

	s.autoExportConfig()

	s.syncCamerasAsync()

	if s.process.RestartProcess("people_counter.bat") {
		s.logger.Info("Restarting people_counter.bat")
	}

	return camera, nil
}

func (s *CameraService) ListCamera() ([]models.Camera, error) {
	var cameras []models.Camera

	if err := s.DB.Where("deleted_at IS NULL").Preload("Location").Order("created_at").Find(&cameras).Error; err != nil {
		return nil, err
	}

	return cameras, nil
}

func (s *CameraService) GetCameraByID(id uint) (*models.Camera, error) {
	var camera models.Camera
	if err := s.DB.Preload("Location").First(&camera, id).Error; err != nil {
		return nil, err
	}
	return &camera, nil
}

func (s *CameraService) UpdateCamera(id uint, input models.CameraInput) (*models.Camera, error) {

	var camera models.Camera
	if err := s.DB.First(&camera, id).Error; err != nil {
		return nil, err
	}

	if input.Location != "" {
		var location models.Location
		if err := s.DB.Where("id = ?", input.Location).First(&location).Error; err != nil {
			return nil, errors.New("location not found")
		}
		camera.LocationID = location.ID
	}

	camera.Name = input.Name
	camera.Schema = input.Schema
	camera.Host = input.Host
	camera.Port = input.Port
	camera.Path = input.Path
	camera.Username = input.Username
	camera.Password = input.Password
	camera.ImageData = input.ImageData
	camera.Direction = input.Direction
	camera.Description = input.Description
	camera.Tags = input.Tags

	var existingPayload map[string]interface{}
	if camera.Payload != "" {
		if err := json.Unmarshal([]byte(camera.Payload), &existingPayload); err != nil {
			existingPayload = make(map[string]interface{})
		}
	} else {
		existingPayload = make(map[string]interface{})
	}

	existingPayload["lines"] = input.Lines

	payloadJSON, err := json.Marshal(existingPayload)
	if err != nil {
		return nil, err
	}
	camera.Payload = string(payloadJSON)

	if err := s.DB.Save(&camera).Error; err != nil {
		return nil, err
	}

	if s.backgroundRunning {
		go s.checkCameraConnection(&camera)
	}

	s.autoExportConfig()

	s.syncCamerasAsync()

	// if s.process.RestartProcess("people_counter.bat") {
	// 	s.logger.Info("Restarting people_counter.bat")
	// }

	return &camera, nil
}

func (s *CameraService) DeleteCamera(id uint) error {
	var camera models.Camera
	if err := s.DB.First(&camera, id).Error; err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	camera.DeletedAt = &now

	if err := s.DB.Delete(&camera).Error; err != nil {
		return err
	}

	s.autoExportConfig()
	s.syncCamerasAsync()

	if s.process.RestartProcess("people_counter.bat") {
		s.logger.Info("Restarting people_counter.bat")
	}

	return nil
}

func (s *CameraService) GetPayloadData(camera *models.Camera) (map[string]any, error) {
	if camera.Payload == "" {
		return make(map[string]any), nil
	}

	var payloadData map[string]any
	if err := json.Unmarshal([]byte(camera.Payload), &payloadData); err != nil {
		return nil, err
	}

	return payloadData, nil
}

func (s *CameraService) GetCameraWithLines(id uint) (*models.Camera, []models.LineData, error) {
	camera, err := s.GetCameraByID(id)
	if err != nil {
		return nil, nil, err
	}

	payloadData, err := s.GetPayloadData(camera)
	if err != nil {
		return camera, nil, err
	}

	var lines []models.LineData
	if linesData, ok := payloadData["lines"]; ok {
		linesJSON, err := json.Marshal(linesData)
		if err != nil {
			return camera, nil, err
		}

		if err := json.Unmarshal(linesJSON, &lines); err != nil {
			return camera, nil, err
		}
	}

	return camera, lines, nil
}

func (s *CameraService) autoExportConfig() {
	go func() {
		if err := s.ExportCameraConfig(); err != nil {
			s.logger.Error("Error auto-exporting camera config: %v", err)
		}
	}()
}

func (s *CameraService) CheckConnection(rtspURL string, takeScreenshot bool) string {
	options := ffmpeg.RTSPOptions{
		TakeScreenshot: takeScreenshot,
	}
	return ffmpeg.CheckRTSPConnection(rtspURL, options)
}

func (s *CameraService) CheckConnectionWithConfig(config ffmpeg.RTSPConfig, takeScreenshot bool) string {
	var result string

	// Add recovery for panics
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("PANIC in CheckConnectionWithConfig: %v", r)
			response := map[string]interface{}{
				"success":   false,
				"message":   "Internal server error",
				"error":     fmt.Sprintf("Panic occurred: %v", r),
				"timestamp": time.Now(),
			}
			jsonResponse, _ := json.Marshal(response)
			result = string(jsonResponse)
		}
	}()

	// Verify options initialization
	options := ffmpeg.RTSPOptions{
		TakeScreenshot: takeScreenshot,
	}

	// Log before calling method
	s.logger.Info("Calling CheckRTSPConnectionWithConfig with config: %+v", config)

	// Call the RTSP connection check
	result = ffmpeg.CheckRTSPConnectionWithConfig(config, options)

	// Log the result type for debugging
	s.logger.Debug("Result type: %T", result)

	return result
}

func (s *CameraService) GenerateRTSPURL(config ffmpeg.RTSPConfig) (string, error) {
	return ffmpeg.GenerateRTSPURL(config)
}

func (s *CameraService) GetImageAsBase64(imagePath string) string {
	fileBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return ""
	}

	var mimeType string
	ext := strings.ToLower(filepath.Ext(imagePath))
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	default:
		mimeType = "image/jpeg" // Default to JPEG
	}

	base64Str := base64.StdEncoding.EncodeToString(fileBytes)
	return "data:" + mimeType + ";base64," + base64Str
}

func (s *CameraService) ExportCameraConfig() error {
	cameras, err := s.ListCamera()
	if err != nil {
		return err
	}

	configs := make([]models.Config, 0, len(cameras))

	for _, camera := range cameras {
		if camera.DeletedAt != nil {
			continue
		}

		_, lines, err := s.GetCameraWithLines(camera.ID)
		if err != nil {
			s.logger.Warn("Warning: Error getting lines for camera %d: %v", camera.ID, err)
			continue
		}

		var mainLine models.LineJson
		var allLines []models.LineJson

		if len(lines) > 0 {
			for _, line := range lines {
				startX := int(line.Start.X)
				startY := int(line.Start.Y)
				endX := int(line.End.X)
				endY := int(line.End.Y)

				lineJson := models.LineJson{
					START: [2]float64{float64(startX), float64(startY)},
					END:   [2]float64{float64(endX), float64(endY)},
				}
				allLines = append(allLines, lineJson)

				if len(allLines) == 1 {
					mainLine = lineJson
				}
			}
		} else {
			mainLine = models.LineJson{
				START: [2]float64{0, 0},
				END:   [2]float64{0, 0},
			}
		}

		rtspConfig := ffmpeg.RTSPConfig{
			Schema:   camera.Schema,
			Host:     camera.Host,
			Port:     camera.Port,
			Path:     camera.Path,
			Username: camera.Username,
			Password: camera.Password,
		}

		rtspURL, err := ffmpeg.GenerateRTSPURL(rtspConfig)
		if err != nil {
			return fmt.Errorf("failed to create RTSP URL: %w", err)
		}

		config := models.Config{
			IP:              rtspURL,
			ID:              camera.ID,
			UUID:            camera.UUID,
			PORT:            camera.Port,
			IS_INSIDE_Y:     checkDirection(camera.Direction, []string{"btt", "ttb"}),
			IS_INSIDE_UNDER: checkDirection(camera.Direction, []string{"btt"}),
			IS_INSIDE_LEFT:  checkDirection(camera.Direction, []string{"ltr"}),
			LINE:            mainLine,
			LINES:           allLines,
		}

		configs = append(configs, config)
	}

	siteId, err := s.settingService.GetSetting("site_id")
	if err != nil {
		return fmt.Errorf("failed to get location id: %w", err)
	}

	siteIdInt, err := strconv.Atoi(siteId)
	if err != nil {
		return fmt.Errorf("failed to convert site id to integer: %w", err)
	}

	cameraConfig := models.CameraConfig{
		TENANT_ID:    s.config.TenantId,
		SITE_ID:      siteIdInt,
		CCTV_NUMBER:  len(configs),
		FRAME_HEIGHT: 360,
		FRAME_WIDTH:  480,
		CONFIG:       configs,
	}

	jsonData, err := json.MarshalIndent(cameraConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal camera config to JSON: %w", err)
	}

	filePath := filepath.Join(s.config.CameraConfigPath, s.config.CameraConfigName)
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	s.logger.Info("Successfully exported camera config to %s", filePath)
	return nil
}

func checkDirection(direction string, validValues []string) bool {
	return slices.Contains(validValues, direction)
}

func (s *CameraService) syncCameras() error {
	s.logger.Info("Starting camera synchronization to server")

	cameras, err := s.ListCamera()
	if err != nil {
		return fmt.Errorf("failed to list cameras: %w", err)
	}

	siteId, err := s.settingService.GetSetting("site_id")
	if err != nil {
		return fmt.Errorf("failed to get location id: %w", err)
	}

	siteIdInt, err := strconv.Atoi(siteId)
	if err != nil {
		return fmt.Errorf("failed to convert site id to integer: %w", err)
	}

	var camerasSync []CameraSync
	for _, camera := range cameras {
		camerasSync = append(camerasSync, CameraSync{
			ID:          camera.ID,
			UUID:        camera.UUID,
			Name:        camera.Name,
			LocationID:  camera.LocationID,
			Description: camera.Description,
			Tags:        camera.Tags,
			Schema:      camera.Schema,
			Host:        camera.Host,
			Port:        camera.Port,
			Username:    camera.Username,
			Password:    camera.Password,
			Path:        camera.Path,
			Direction:   camera.Direction,
			Status:      camera.Status,
			Payload:     camera.Payload,
			CreatedAt:   camera.CreatedAt,
		})
	}

	// Buat request body
	requestBody := map[string]interface{}{
		"site_id": siteIdInt,
		"cameras": camerasSync,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal camera data: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/app/cameras/sync", s.config.ApiUrl), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-API-Key", s.config.ApiKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send sync request: %w", err)
	}
	defer resp.Body.Close()

	var syncResponse SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResponse); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !syncResponse.Success {
		return fmt.Errorf("sync failed: %s", syncResponse.Error)
	}

	s.logger.Info("Camera synchronization completed successfully: %s", syncResponse.Message)
	return nil
}

func (s *CameraService) syncCamerasAsync() {
	go func() {
		if err := s.syncCameras(); err != nil {
			s.logger.Error("Background camera sync error: %v", err)
		}
	}()
}
