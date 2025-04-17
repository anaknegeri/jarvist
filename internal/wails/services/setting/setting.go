package setting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"jarvist/internal/common/config"
	"jarvist/internal/common/models"
	licenseservice "jarvist/internal/wails/services/license"
	"jarvist/pkg/logger"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gorm.io/gorm"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type SettingsService struct {
	db             *gorm.DB
	config         *config.Config
	logger         *logger.ContextLogger
	requiredKeys   []string
	licenseService *licenseservice.LicenseService
}

type EnvConfigItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// EnvConfig represents all environment configuration items
type EnvConfig struct {
	Items []EnvConfigItem `json:"items"`
}

func New(db *gorm.DB, cfg *config.Config, logger *logger.ContextLogger, licenseService *licenseservice.LicenseService) *SettingsService {
	return &SettingsService{
		db:     db,
		config: cfg,
		logger: logger.WithComponent("setting"),
		requiredKeys: []string{
			"default_timezone",
			"camera_sync_interval",
			"log_level",
			"site_code",
			"site_name",
		},
		licenseService: licenseService,
	}
}

// ServiceStartup initializes the license service
func (s *SettingsService) OnStartup(ctx context.Context, options application.ServiceOptions) error {
	s.logger.Info("Settings service starting")
	s.CreateDefaultSettings()
	s.CheckAndCreateEnvFile()
	return nil
}
func (s *SettingsService) IsConfigured() bool {
	var count int64
	s.db.Model(&models.Setting{}).Count(&count)

	if count == 0 {
		return false
	}

	for _, key := range s.requiredKeys {
		var setting models.Setting
		result := s.db.Where("key = ?", key).First(&setting)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			s.logger.Error("Required setting '%s' not found", key)
			return false
		} else if result.Error != nil {
			s.logger.Error("Error checking setting '%s': %v", key, result.Error)
			return false
		}
	}

	return true
}

func (s *SettingsService) GetSetting(key string) (string, error) {
	var setting models.Setting
	result := s.db.Where("key = ?", key).First(&setting)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "", errors.New("setting not found")
	} else if result.Error != nil {
		return "", result.Error
	}

	return setting.Value, nil
}

func (s *SettingsService) GetAllSettings() (map[string]string, error) {
	var settings []models.Setting
	if err := s.db.Find(&settings).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}

	return result, nil
}

func (s *SettingsService) GetSettingWithDefault(key, defaultValue string) string {
	value, err := s.GetSetting(key)
	if err != nil {
		return defaultValue
	}
	return value
}

func (s *SettingsService) GetTimeZones() ([]models.TimeZone, error) {
	var timezones []models.TimeZone
	result := s.db.Find(&timezones)
	if result.Error != nil {
		return nil, result.Error
	}
	return timezones, nil
}

func (s *SettingsService) SavePlaceConfig(input models.SettingInput) (map[string]interface{}, error) {
	// Prepare the settings map
	setting := make(map[string]string)
	setting["site_code"] = input.SiteCode
	setting["site_name"] = input.SiteName
	setting["site_category"] = strconv.FormatInt(int64(input.SiteCategory), 10)
	setting["default_timezone"] = input.DefaultTimezone
	setting["tenant_id"] = s.config.TenantId
	setting["client_id"] = s.config.ClientId

	// Prepare API request data
	requestData := map[string]interface{}{
		"place_code":  input.SiteCode,
		"place_name":  input.SiteName,
		"category_id": input.SiteCategory,
		"timezone":    input.DefaultTimezone,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request data: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/app/site-register", s.config.ApiUrl), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-API-Key", s.config.ApiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var apiResp struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			PlaceID float64 `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned error: %s (code: %d)", apiResp.Message, apiResp.Code)
	}

	setting["site_id"] = strconv.FormatInt(int64(apiResp.Data.PlaceID), 10)

	// Only save settings locally AFTER successful API call
	if err := s.SaveSettings(setting); err != nil {
		return nil, fmt.Errorf("API call was successful but failed to save settings to local database: %w", err)
	}

	if err := s.GenerateEnvFile(setting); err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		s.logger.Error("Failed to parse license server response: %v", err)
		return nil, fmt.Errorf("invalid response from license server: %v", err)
	}

	log.Printf("Successfully save place with code: %s and saved to local database", input.SiteCode)
	return response, nil
}

func (s *SettingsService) UpdatePlaceConfig(placeId uint, input models.SettingInput) (map[string]interface{}, error) {
	// Prepare the settings map
	setting := make(map[string]string)
	setting["site_code"] = input.SiteCode
	setting["site_name"] = input.SiteName
	setting["site_category"] = strconv.FormatInt(int64(input.SiteCategory), 10)
	setting["default_timezone"] = input.DefaultTimezone

	// Prepare API request data
	requestData := map[string]interface{}{
		"place_code":  input.SiteCode,
		"place_name":  input.SiteName,
		"category_id": input.SiteCategory,
		"timezone":    input.DefaultTimezone,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request data: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/app/site-update/%d", s.config.ApiUrl, placeId), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-API-Key", s.config.ApiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var apiResp struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			PlaceID float64 `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned error: %s (code: %d)", apiResp.Message, apiResp.Code)
	}

	setting["site_id"] = strconv.FormatInt(int64(apiResp.Data.PlaceID), 10)

	if err := s.SaveSettings(setting); err != nil {
		return nil, fmt.Errorf("API call was successful but failed to save settings to local database: %w", err)
	}

	if err := s.GenerateEnvFile(setting); err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		s.logger.Error("Failed to parse license server response: %v", err)
		return nil, fmt.Errorf("invalid response from license server: %v", err)
	}

	log.Printf("Successfully updated place with code: %s and saved to local database", input.SiteCode)
	return response, nil
}

func (s *SettingsService) SaveSetting(key, value string) error {
	var setting models.Setting
	result := s.db.Where("key = ?", key).First(&setting)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		setting = models.Setting{
			Key:   key,
			Value: value,
		}
		return s.db.Create(&setting).Error
	} else if result.Error != nil {
		return result.Error
	}

	setting.Value = value
	return s.db.Save(&setting).Error
}

func (s *SettingsService) SaveSettings(settings map[string]string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range settings {
			var setting models.Setting
			result := tx.Where("key = ?", key).First(&setting)

			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				setting = models.Setting{
					Key:   key,
					Value: value,
				}
				if err := tx.Create(&setting).Error; err != nil {
					return err
				}
			} else if result.Error != nil {
				return result.Error
			} else {
				setting.Value = value
				if err := tx.Save(&setting).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *SettingsService) CreateDefaultSettings() error {
	defaultSettings := map[string]string{
		"default_timezone":     "Asia/Jakarta",
		"camera_sync_interval": "60",
		"log_level":            "info",
	}

	return s.SaveSettings(defaultSettings)
}

func (s *SettingsService) DeleteSetting(key string) error {
	return s.db.Where("key = ?", key).Delete(&models.Setting{}).Error
}

func (s *SettingsService) CheckAndCreateEnvFile() error {
	savePath := filepath.Join(filepath.Dir(s.config.BinDir), "bin", "services", ".env")

	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		if !s.IsConfigured() {
			s.logger.Info(".env file not created - application is not fully configured yet")
			return nil
		}

		settings, err := s.GetAllSettings()
		if err != nil {
			return fmt.Errorf("failed to get settings: %w", err)
		}

		if err := s.GenerateEnvFile(settings); err != nil {
			return fmt.Errorf("failed to create .env file: %w", err)
		}

		s.logger.Info("Created new .env file at %s", savePath)
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking for .env file: %w", err)
	}

	s.logger.Info(".env file already exists at %s", savePath)
	return nil
}

func (s *SettingsService) GenerateEnvFile(settings map[string]string) error {
	cfg := EnvConfig{
		Items: []EnvConfigItem{
			{Key: "CLIENT_ID", Value: "1937", Description: "Client identifier"},
			{Key: "PLACE_ID", Value: "0", Description: "Place identifier"},
			{Key: "LOCAL_SAVE_DATA_PATH", Value: "data", Description: "Path for storing local data"},
			{Key: "IMAGE_LOG_PATH", Value: "image", Description: "Path for storing image logs"},
			{Key: "MODEL_PATH", Value: "models/yolo11n.pt", Description: "Path to the model file"},
			{Key: "CONFIG_PATH", Value: "config.camera.json", Description: "Path to camera configuration file"},
			{Key: "SAVE_TO_LOCAL_INTERVAL", Value: "60", Description: "Interval for saving data locally (in seconds)"},
			{Key: "RESET_TIME", Value: "00:01", Description: "Time to reset the application daily"},
			{Key: "API_ENDPOINT", Value: "https://vision-map.pitds.my.id/v1/people-counting", Description: "API endpoint for data uploads"},
			{Key: "API_KEY", Value: "4pPk3y1", Description: "API key for authentication"},
		},
	}

	if s.licenseService != nil {
		licenseDetails := s.licenseService.GetLicenseDetails()

		if clientID, ok := licenseDetails["clientId"]; ok && clientID != nil {
			clientIDStr := fmt.Sprintf("%.0f", clientID.(float64))
			for i, item := range cfg.Items {
				if item.Key == "CLIENT_ID" {
					cfg.Items[i].Value = clientIDStr
					break
				}
			}
			s.logger.Info("Updated CLIENT_ID in env file to: %s", clientIDStr)
		}
	}

	if siteID, exists := settings["site_id"]; exists && siteID != "" {
		for i, item := range cfg.Items {
			if item.Key == "PLACE_ID" {
				cfg.Items[i].Value = siteID
				break
			}
		}
	}

	savePath := filepath.Join(filepath.Dir(s.config.BinDir), "bin", "services", ".env")

	err := os.MkdirAll(filepath.Dir(savePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	var content strings.Builder

	for _, item := range cfg.Items {
		// if item.Description != "" {
		// 	content.WriteString(fmt.Sprintf("# %s\n", item.Description))
		// }
		content.WriteString(fmt.Sprintf("%s=%s\n", item.Key, item.Value))

		// Add an empty line after each group for readability
		if item != cfg.Items[len(cfg.Items)-1] {
			content.WriteString("\n")
		}
	}

	err = os.WriteFile(savePath, []byte(content.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}
