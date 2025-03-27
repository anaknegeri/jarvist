package config

import (
	"encoding/json"
	"jarvist/pkg/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/tidwall/gjson"
)

type Config struct {
	AppName          string `json:"appName"`
	Version          string `json:"version"`
	DatabasePath     string `json:"databasePath"`
	ScreenshotDir    string `json:"screenshotDir"`
	FFmpegBinPath    string `json:"ffmpegBinPath"`
	SignToolBinPath  string `json:"signToolBinPath"`
	CameraConfigPath string `json:"cameraConfigPath"`
	CameraConfigName string `json:"cameraConfigName"`
	LogDir           string `json:"logDir"`
	ConfigDir        string `json:"configDir"`
	DevMode          bool   `json:"devMode"`
	MonitorEnabled   bool   `json:"monitorEnabled"`
	ApiKey           string `json:"apiKey"`
	TenantID         string `json:"tenantID"`
	Environment      string `json:"environment"`
}

// LoadEnv loads the environment variables from .env file
func LoadEnv() {
	// Default to development if not specified
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Try to load the specific environment file first
	envFile := ".env." + env
	if _, err := os.Stat(envFile); err == nil {
		godotenv.Load(envFile)
	}

	// Load the default .env file as fallback
	godotenv.Load()
}

func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}

	execDir := filepath.Dir(execPath)

	// Get environment or default to development
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	appDataDir := filepath.Join(homeDir, ".jarvist")
	if envDir := os.Getenv("APP_DATA_DIR"); envDir != "" {
		appDataDir = envDir
	}
	utils.EnsureDir(appDataDir)

	screenshotDir := filepath.Join(appDataDir, "temp", "screenshots")
	if envDir := os.Getenv("SCREENSHOT_DIR"); envDir != "" {
		screenshotDir = envDir
	}
	utils.EnsureDir(screenshotDir)

	logDir := filepath.Join(appDataDir, "logs")
	if envDir := os.Getenv("LOG_DIR"); envDir != "" {
		logDir = envDir
	}
	utils.EnsureDir(logDir)

	databaseDir := filepath.Join(appDataDir, "data")
	if envDir := os.Getenv("DATABASE_DIR"); envDir != "" {
		databaseDir = envDir
	}
	utils.EnsureDir(databaseDir)

	// Check if we're in dev mode
	devMode := env != "production"
	if envDevMode := os.Getenv("DEV_MODE"); envDevMode != "" {
		devMode = strings.ToLower(envDevMode) == "true"
	}

	// Check if monitoring is enabled
	monitorEnabled := false
	if envMonitor := os.Getenv("MONITOR_ENABLED"); envMonitor != "" {
		monitorEnabled = strings.ToLower(envMonitor) == "true"
	}

	return &Config{
		DatabasePath:     filepath.Join(databaseDir, "jarvist.db"),
		ScreenshotDir:    screenshotDir,
		FFmpegBinPath:    getEnvOrDefault("FFMPEG_BIN_PATH", filepath.Join(execDir, "bin", "ffmpeg")),
		SignToolBinPath:  getEnvOrDefault("SIGN_TOOL_BIN_PATH", filepath.Join(execDir, "bin", "signtool")),
		CameraConfigPath: getEnvOrDefault("CAMERA_CONFIG_PATH", filepath.Join(execDir, "bin", "services")),
		CameraConfigName: getEnvOrDefault("CAMERA_CONFIG_NAME", "config.camera.json"),
		LogDir:           logDir,
		ConfigDir:        appDataDir,
		DevMode:          devMode,
		MonitorEnabled:   monitorEnabled,
		ApiKey:           os.Getenv("API_KEY"),
		TenantID:         os.Getenv("TENANT_ID"),
		Environment:      env,
	}
}

// Helper function to get environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func LoadConfig(appInfo gjson.Result) (*Config, error) {
	// Load environment variables first
	LoadEnv()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".jarvist")
	if envDir := os.Getenv("CONFIG_DIR"); envDir != "" {
		configDir = envDir
	}
	utils.EnsureDir(configDir)

	configFile := filepath.Join(configDir, "config.json")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := config.Save(); err != nil {
			return nil, err
		}
		return config, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	productVersion := appInfo.Get("productVersion")
	productName := appInfo.Get("productName")

	config.Version = productVersion.String()
	config.AppName = productName.String()

	// Update environment-specific settings from env variables
	config.applyEnvironmentOverrides()

	if err := config.Save(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Apply any environment variable overrides to the existing config
func (c *Config) applyEnvironmentOverrides() {
	// Update environment field
	env := os.Getenv("APP_ENV")
	if env != "" {
		c.Environment = env
	}

	// Apply other environment overrides
	if v := os.Getenv("API_KEY"); v != "" {
		c.ApiKey = v
	}
	if v := os.Getenv("TENANT_ID"); v != "" {
		c.TenantID = v
	}

	// Parse boolean values
	if v := os.Getenv("DEV_MODE"); v != "" {
		c.DevMode = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("MONITOR_ENABLED"); v != "" {
		c.MonitorEnabled = strings.ToLower(v) == "true"
	}
}

func (c *Config) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".jarvist")
	utils.EnsureDir(configDir)

	configFile := filepath.Join(configDir, "config.json")

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func (c *Config) UpdateConfig(updates map[string]interface{}) error {
	currentData, err := json.Marshal(c)
	if err != nil {
		return err
	}

	var currentMap map[string]interface{}
	if err := json.Unmarshal(currentData, &currentMap); err != nil {
		return err
	}

	for key, value := range updates {
		currentMap[key] = value
	}

	updatedData, err := json.Marshal(currentMap)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(updatedData, c); err != nil {
		return err
	}

	return c.Save()
}
