package licenseservice

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"jarvist/internal/common/config"
	"jarvist/internal/wails/services/device"
	"jarvist/pkg/hardware"
	"jarvist/pkg/logger"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type LicenseStatus int

const (
	StatusInvalid LicenseStatus = iota
	StatusValid
	StatusExpired
	StatusWrongMachine
)

type LicenseType int

const (
	TypeTrial LicenseType = iota
	TypeStandard
	TypeProfessional
	TypeEnterprise
)

type DeviceInfo struct {
	ID           int       `json:"id"`
	DeviceID     string    `json:"device_id"`
	DeviceName   *string   `json:"device_name"`
	DeviceOs     *string   `json:"device_os"`
	RegisteredAt time.Time `json:"registered_at"`
}

type ClientInfo struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type LicenseInfo struct {
	LicenseKey  string    `json:"licenseKey"`
	HardwareID  string    `json:"hardwareID"`
	Company     string    `json:"company"`
	ContactName string    `json:"contactName"`
	Email       string    `json:"email"`
	IssuedDate  time.Time `json:"issuedDate"`
	ExpiryDate  time.Time `json:"expiryDate"`
	Activated   bool      `json:"activated"`
	ApiKey      string    `json:"apiKey"`
	TenantId    string    `json:"tenantId"`
	ClientID    float64   `json:"clientID"`
}

type LicenseValidation struct {
	Valid       bool          `json:"valid"`
	Status      LicenseStatus `json:"status"`
	Message     string        `json:"message"`
	License     *LicenseInfo  `json:"license,omitempty"`
	DaysLeft    int           `json:"daysLeft"`
	GracePeriod bool          `json:"gracePeriod"`
}

type LicenseService struct {
	config      *config.Config
	logger      *logger.ContextLogger
	licenseInfo *LicenseInfo
	encryption  *EncryptionConfig
	device      *device.DeviceInfo
}

type EncryptionConfig struct {
	Key         []byte
	Salt        string
	LicensePath string
}

func New(cfg *config.Config, logger *logger.ContextLogger, secretKey, salt string) *LicenseService {
	if cfg.IsDev() && secretKey == "dev_test_license_key_not_for_production" {
		logger.Warning("Using INSECURE default license key for development!")
	}

	hash := sha256.Sum256([]byte(secretKey))
	device := device.New()

	return &LicenseService{
		config: cfg,
		logger: logger.WithComponent("license"),
		encryption: &EncryptionConfig{
			Key:         hash[:],
			Salt:        salt,
			LicensePath: filepath.Join(cfg.DataDir, "license.dat"),
		},
		device: device.GetDeviceInfo(),
	}
}

// ServiceStartup initializes the license service
func (s *LicenseService) OnStartup(ctx context.Context, options application.ServiceOptions) error {
	s.LoadLicense()
	return nil
}

// GetHardwareFingerprint returns a unique identifier for the current machine
func (s *LicenseService) GetHardwareFingerprint() (string, error) {
	hardwareID, err := hardware.GetHardwareID()
	if err != nil {
		s.logger.Error("Failed to get hardware ID: %v", err)
		return "", err
	}
	return hardwareID, nil
}

// GetLicenseStatus returns current license status
func (s *LicenseService) GetLicenseStatus() map[string]interface{} {
	validation := s.validateLicense()

	// Don't send sensitive data to frontend
	result := map[string]interface{}{
		"valid":       validation.Valid,
		"status":      int(validation.Status),
		"message":     validation.Message,
		"daysLeft":    validation.DaysLeft,
		"gracePeriod": validation.GracePeriod,
	}

	if validation.License != nil {
		result["company"] = validation.License.Company
		result["contactName"] = validation.License.ContactName
		result["email"] = validation.License.Email
		result["expiryDate"] = validation.License.ExpiryDate.Format("2006-01-02")
	}

	return result
}

// RegisterLicense activates a license with the given key for this machine
func (s *LicenseService) RegisterLicense(licenseKey string) map[string]interface{} {
	s.logger.Info("Attempting to register license: %s", licenseKey)

	result := map[string]interface{}{
		"success": false,
		"message": "",
	}

	// Get hardware fingerprint
	hardwareID, err := s.GetHardwareFingerprint()
	if err != nil {
		s.logger.Error("Failed to get hardware ID: %v", err)
		result["message"] = "Failed to identify this machine"
		return result
	}

	// Contact license server to activate
	activationResult, err := s.activateLicenseWithServer(licenseKey, hardwareID)
	if err != nil {
		s.logger.Error("License activation failed: %v", err)
		result["message"] = "Failed to contact activation server: " + err.Error()
		return result
	}

	// Check if success field exists and is true
	success, ok := activationResult["success"].(bool)
	if !ok || !success {
		message := "Unknown error occurred during activation"
		if msg, ok := activationResult["message"].(string); ok {
			message = msg
		}
		result["message"] = message
		return result
	}

	// Create license info from activation result
	licenseData := activationResult["data"].(map[string]interface{})
	clientInfo := licenseData["client_info"].(map[string]interface{})

	// Extract name and email safely
	var companyName, contactName, email string
	if name, ok := clientInfo["name"]; ok && name != nil {
		companyName = name.(string)
		contactName = name.(string)
	}
	if emailVal, ok := clientInfo["email"]; ok && emailVal != nil {
		email = emailVal.(string)
	}

	license := &LicenseInfo{
		LicenseKey:  licenseKey,
		HardwareID:  hardwareID,
		ApiKey:      licenseData["api_key"].(string),
		TenantId:    licenseData["tenant_id"].(string),
		ClientID:    licenseData["client_id"].(float64),
		Company:     companyName,
		ContactName: contactName,
		Email:       email,
		Activated:   true,
	}

	// Parse dates - they include time information in the JSON
	issuedDate, err := time.Parse(time.RFC3339, licenseData["valid_from"].(string))
	if err != nil {
		s.logger.Warning("Failed to parse valid_from date: %v", err)
		// Fallback to date-only parsing
		issuedDate, _ = time.Parse("2006-01-02T15:04:05Z", licenseData["valid_from"].(string))
	}

	expiryDate, err := time.Parse(time.RFC3339, licenseData["valid_until"].(string))
	if err != nil {
		s.logger.Warning("Failed to parse valid_until date: %v", err)
		// Fallback to date-only parsing
		expiryDate, _ = time.Parse("2006-01-02T15:04:05Z", licenseData["valid_until"].(string))
	}

	license.IssuedDate = issuedDate
	license.ExpiryDate = expiryDate

	// Save license to disk
	s.licenseInfo = license
	if err := s.saveLicense(); err != nil {
		s.logger.Error("Failed to save license: %v", err)
		result["message"] = "License validated but failed to save locally: " + err.Error()
		return result
	}

	result["success"] = true
	result["message"] = "License successfully activated"
	return result
}

// activateLicenseWithServer contacts license server for activation
func (s *LicenseService) activateLicenseWithServer(licenseKey, hardwareID string) (map[string]interface{}, error) {
	// Create activation request
	requestData := map[string]interface{}{
		"license_key": licenseKey,
		"device_id":   hardwareID,
		"device_name": s.device.Name,
		"device_os":   s.device.OS,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", s.config.ApiUrl+"/v1/license/activate", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.ApiKey)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		s.logger.Error("Failed to parse license server response: %v", err)
		return nil, fmt.Errorf("invalid response from license server: %v", err)
	}

	return response, nil
}

// DeactivateLicense deactivates the current license
func (s *LicenseService) DeactivateLicense() map[string]interface{} {
	result := map[string]interface{}{
		"success": false,
		"message": "",
	}

	if s.licenseInfo == nil {
		result["message"] = "No license is currently active"
		return result
	}

	// Contact license server to deactivate
	deactivationResult, err := s.deactivateLicenseWithServer(s.licenseInfo.LicenseKey, s.licenseInfo.HardwareID)
	if err != nil {
		s.logger.Error("License deactivation failed: %v", err)
		result["message"] = "Failed to contact activation server: " + err.Error()
		return result
	}

	// Remove local license file even if server deactivation fails
	if err := os.Remove(s.encryption.LicensePath); err != nil && !os.IsNotExist(err) {
		s.logger.Warning("Failed to remove license file: %v", err)
	}

	s.licenseInfo = nil
	result["success"] = true
	result["message"] = "License successfully deactivated"

	// Log the server response as well
	if message, ok := deactivationResult["message"].(string); ok {
		s.logger.Info("Server deactivation response: %s", message)
	}

	return result
}

// deactivateLicenseWithServer contacts license server for deactivation
func (s *LicenseService) deactivateLicenseWithServer(licenseKey, hardwareID string) (map[string]interface{}, error) {
	// Create deactivation request
	requestData := map[string]interface{}{
		"licenseKey": licenseKey,
		"hardwareID": hardwareID,
		"appVersion": s.config.AppVersion,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", s.config.ApiUrl+"/license/deactivate", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.ApiKey)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response, nil
}

// loadLicense loads and validates license from disk
func (s *LicenseService) LoadLicense() {
	if _, err := os.Stat(s.encryption.LicensePath); os.IsNotExist(err) {
		// No license file exists
		return
	}

	// Read encrypted license file
	encryptedData, err := os.ReadFile(s.encryption.LicensePath)
	if err != nil {
		s.logger.Error("Failed to read license file: %v", err)
		return
	}

	// Decrypt license data
	data, err := s.decrypt(encryptedData)
	if err != nil {
		s.logger.Error("Failed to decrypt license: %v", err)
		return
	}

	// Parse license info
	var license LicenseInfo
	if err := json.Unmarshal(data, &license); err != nil {
		s.logger.Error("Failed to parse license data: %v", err)
		return
	}

	// Verify hardware ID
	currentHardwareID, err := hardware.GetHardwareID()
	if err != nil {
		s.logger.Error("Failed to get current hardware ID: %v", err)
		return
	}

	if license.HardwareID != currentHardwareID {
		s.logger.Warning("License hardware ID mismatch: stored=%s, current=%s",
			license.HardwareID, currentHardwareID)
		// still load the license but it will be marked as invalid
	}

	s.config.ApiKey = license.ApiKey
	s.config.TenantId = license.TenantId
	s.config.ClientId = fmt.Sprintf("%f", license.ClientID)

	s.licenseInfo = &license
}

// saveLicense encrypts and saves license to disk
func (s *LicenseService) saveLicense() error {
	if s.licenseInfo == nil {
		return errors.New("no license information to save")
	}

	// Convert license to JSON
	data, err := json.Marshal(s.licenseInfo)
	if err != nil {
		return err
	}

	// Encrypt license data
	encryptedData, err := s.encrypt(data)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.encryption.LicensePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write encrypted license to file
	return os.WriteFile(s.encryption.LicensePath, encryptedData, 0644)
}

// validateLicense checks if the current license is valid
func (s *LicenseService) validateLicense() LicenseValidation {
	s.LoadLicense()

	validation := LicenseValidation{
		Valid:  false,
		Status: StatusInvalid,
	}

	// No license loaded
	if s.licenseInfo == nil {
		validation.Message = "No license installed"
		return validation
	}

	// Verify hardware ID
	currentHardwareID, err := hardware.GetHardwareID()
	if err != nil {
		validation.Message = "Failed to get hardware ID"
		return validation
	}

	if s.licenseInfo.HardwareID != currentHardwareID {
		validation.Status = StatusWrongMachine
		validation.Message = "License is not valid for this machine"
		validation.License = s.licenseInfo
		return validation
	}

	// Check expiration
	now := time.Now()
	if now.After(s.licenseInfo.ExpiryDate) {
		validation.Status = StatusExpired
		validation.Message = "License has expired"
		validation.License = s.licenseInfo

		// Calculate days since expiration
		daysSinceExpiry := int(now.Sub(s.licenseInfo.ExpiryDate).Hours() / 24)
		validation.DaysLeft = -daysSinceExpiry

		// Check for grace period (7 days)
		if daysSinceExpiry <= 7 {
			validation.GracePeriod = true
			validation.Valid = true // Still valid during grace period
		}

		return validation
	}

	// License is valid
	validation.Valid = true
	validation.Status = StatusValid
	validation.Message = "License is valid"
	validation.License = s.licenseInfo

	// Calculate days left
	validation.DaysLeft = int(s.licenseInfo.ExpiryDate.Sub(now).Hours() / 24)

	return validation
}

// encrypt encrypts data using AES-256-GCM
func (s *LicenseService) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryption.Key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Prefix salt + nonce to ciphertext
	saltBytes := []byte(s.encryption.Salt)
	ciphertext := gcm.Seal(nil, nonce, data, saltBytes)

	// Format: nonce + ciphertext
	result := make([]byte, 0, len(nonce)+len(ciphertext))
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// decrypt decrypts data using AES-256-GCM
func (s *LicenseService) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryption.Key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(data) < gcm.NonceSize() {
		return nil, errors.New("encrypted data too short")
	}

	// Extract nonce and ciphertext
	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	// Additional data for authentication
	saltBytes := []byte(s.encryption.Salt)

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, saltBytes)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// HasFeature checks if current license includes a specific feature
func (s *LicenseService) HasFeature(featureName string) bool {
	if s.licenseInfo == nil {
		return false
	}

	validation := s.validateLicense()
	if !validation.Valid {
		return false
	}

	return false
}

// IsLicensed returns true if the application has a valid license
func (s *LicenseService) IsLicensed() bool {
	validation := s.validateLicense()
	return validation.Valid
}

// GetLicenseDetails returns full license details (safe for settings UI)
func (s *LicenseService) GetLicenseDetails() map[string]interface{} {
	if s.licenseInfo == nil {
		return map[string]interface{}{
			"licensed": false,
		}
	}

	validation := s.validateLicense()

	result := map[string]interface{}{
		"licensed":    validation.Valid,
		"apiKey":      s.licenseInfo.ApiKey,
		"licenseKey":  s.licenseInfo.LicenseKey,
		"clientId":    s.licenseInfo.ClientID,
		"company":     s.licenseInfo.Company,
		"contactName": s.licenseInfo.ContactName,
		"email":       s.licenseInfo.Email,
		"issueDate":   s.licenseInfo.IssuedDate.Format("2006-01-02"),
		"expiryDate":  s.licenseInfo.ExpiryDate.Format("2006-01-02"),
		"daysLeft":    validation.DaysLeft,
		"status":      int(validation.Status),
		"message":     validation.Message,
		"deviceInfo":  s.device,
	}

	return result
}

// maskLicenseKey returns a masked version of the license key (e.g., "XXXX-XXXX-XXXX-1234")
func (s *LicenseService) maskLicenseKey(key string) string {
	if len(key) <= 4 {
		return key
	}

	// Only show last 4 characters
	maskedPart := strings.Repeat("X", len(key)-4)
	return maskedPart + key[len(key)-4:]
}
