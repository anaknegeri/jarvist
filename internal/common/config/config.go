package config

import (
	"encoding/json"
	"fmt"
	"jarvist/internal/common/buildinfo"
	"jarvist/pkg/utils"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
)

// Config berisi konfigurasi aplikasi
type Config struct {
	// App info
	AppName     string `json:"appName"`
	AppVersion  string `json:"appVersion"`
	Environment string `json:"environment"`

	// Database
	DatabasePath string `json:"databasePath"`

	// Paths
	BinDir    string `json:"binDir"`
	LogDir    string `json:"logDir"`
	DataDir   string `json:"dataDir"`
	TempDir   string `json:"tempDir"`
	AssetsDir string `json:"assetsDir"`

	// Features
	DebugMode       bool `json:"debugMode"`
	EnableAnalytics bool `json:"enableAnalytics"`

	// API Settings
	ApiUrl   string `json:"apiUrl"`
	ApiKey   string `json:"apiKey"`
	TenantId string `json:"tenantId"`
	ClientId string `json:"clientId"`

	// Camera setting
	CameraConfigPath string `json:"cameraConfigPath"`
	CameraConfigName string `json:"cameraConfigName"`
	ScreenshotDir    string `json:"screenshotDir"`
	ServicesDir      string `json:"servicesDir"`
	ServicesDataDir  string `json:"servicesDarDir"`

	SyncApi string `json:"sync_api"`

	BuildInfo buildinfo.BuildInfo `json:"buildInfo"`
}

// IsDev mengembalikan true jika aplikasi berjalan dalam mode development
func (c *Config) IsDev() bool {
	return c.Environment == "development"
}

// GetDBConnectionString mengembalikan string koneksi database
func (c *Config) GetDBConnectionString() string {
	return fmt.Sprintf("file:%s?mode=rwc&cache=shared", c.DatabasePath)
}

// LoadConfig memuat konfigurasi dari file dan environment variables
func LoadConfig(buildMode string, buildInfo *buildinfo.BuildInfoService) (*Config, error) {
	info := buildInfo.LoadBuildInfo()

	// Default is development unless built with production flag
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		if buildMode == "production" {
			environment = "production"
		} else {
			environment = "development"
		}
	}

	// Load appropriate .env file
	LoadEnv(environment)

	execPath, _ := os.Executable()
	currentDir := filepath.Dir(execPath)

	// Initialize config with defaults
	config := &Config{
		AppName:          info.ProductName,
		AppVersion:       strings.TrimPrefix(info.ProductVersion, "v"),
		Environment:      environment,
		DebugMode:        environment == "development",
		EnableAnalytics:  environment == "production",
		BuildInfo:        info,
		BinDir:           filepath.Join(currentDir, "bin"),
		CameraConfigName: "config.camera.json",
		CameraConfigPath: filepath.Join(currentDir, "bin", "services"),
		ServicesDir:      filepath.Join(currentDir, "bin", "services"),
		ServicesDataDir:  filepath.Join(currentDir, "bin", "services", "data"),
	}

	// Setup paths based on environment
	if environment == "development" {
		setupDevPaths(config)
	} else {
		setupProdPaths(config)
	}

	// Override with environment variables
	applyEnvOverrides(config)

	// Create necessary directories
	ensureDirectories(config)

	return config, nil
}

// LoadEnv loads environment variables from .env files
func LoadEnv(environment string) {
	// Try environment-specific .env file first
	envFile := ".env." + environment
	if _, err := os.Stat(envFile); err == nil {
		godotenv.Load(envFile)
	}

	// Then load default .env
	godotenv.Load()
}

// setupDevPaths configures paths for development environment
func setupDevPaths(config *Config) {
	execPath, _ := os.Executable()

	currentDir := filepath.Dir(execPath)
	config.DataDir = filepath.Join(currentDir, "dev-data")
	config.LogDir = filepath.Join(currentDir, "dev-logs")
	config.TempDir = filepath.Join(currentDir, "dev-temp")
	config.DatabasePath = filepath.Join(config.DataDir, "dev.db")
	config.AssetsDir = filepath.Join(currentDir, "assets")

	config.ScreenshotDir = filepath.Join(config.TempDir, "screenshots")

	// Default API settings for development
	config.ApiUrl = "http://localhost:3001/api"
	config.SyncApi = "http://localhost:8081/api"
}

// setupProdPaths configures paths for production environment
func setupProdPaths(config *Config) {
	var appDataDir string
	// Get appropriate app data directory by platform
	switch runtime.GOOS {
	case "windows":
		appDataDir = filepath.Join(os.Getenv("APPDATA"), config.AppName)
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		appDataDir = filepath.Join(homeDir, "Library", "Application Support", config.AppName)
	default: // linux and others
		homeDir, _ := os.UserHomeDir()
		appDataDir = filepath.Join(homeDir, "."+strings.ToLower(config.AppName))
	}

	config.DataDir = filepath.Join(appDataDir, "data")
	config.LogDir = filepath.Join(appDataDir, "logs")
	config.TempDir = filepath.Join(appDataDir, "temp")
	config.DatabasePath = filepath.Join(config.DataDir, "app.db")

	config.ScreenshotDir = filepath.Join(config.TempDir, "screenshots")
	// For production, assets are embedded but we still define a path for any dynamic assets
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	config.AssetsDir = filepath.Join(execDir, "assets")

	// Production API settings
	config.ApiUrl = "https://api-mayene.indward.com/api"
	config.SyncApi = "http://localhost:8722/api"
}

// applyEnvOverrides applies environment variable overrides to config
func applyEnvOverrides(config *Config) {
	if val := os.Getenv("DATABASE_PATH"); val != "" {
		config.DatabasePath = val
	}

	if val := os.Getenv("LOG_DIR"); val != "" {
		config.LogDir = val
	}

	if val := os.Getenv("DATA_DIR"); val != "" {
		config.DataDir = val
	}

	if val := os.Getenv("TEMP_DIR"); val != "" {
		config.TempDir = val
	}

	if val := os.Getenv("API_URL"); val != "" {
		config.ApiUrl = val
	}

	if val := os.Getenv("API_KEY"); val != "" {
		config.ApiKey = val
	}

	if val := os.Getenv("DEBUG_MODE"); val != "" {
		config.DebugMode = val == "true"
	}

	if val := os.Getenv("ENABLE_ANALYTICS"); val != "" {
		config.EnableAnalytics = val == "true"
	}
}

// ensureDirectories membuat direktori yang diperlukan jika belum ada
func ensureDirectories(config *Config) {
	dirs := []string{
		config.DataDir,
		config.LogDir,
		config.TempDir,
		config.BinDir,
		config.CameraConfigPath,
		config.ScreenshotDir,
	}

	for _, dir := range dirs {
		utils.EnsureDir(dir)
	}
}

// SaveConfig menyimpan konfigurasi saat ini ke file
func (c *Config) SaveConfig() error {
	configFilePath := filepath.Join(c.DataDir, "config.json")

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFilePath, data, 0644)
}
