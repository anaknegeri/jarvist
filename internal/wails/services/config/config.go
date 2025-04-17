package configservice

import (
	"jarvist/internal/common/buildinfo"
	"jarvist/internal/common/config"
	"jarvist/pkg/logger"
)

type Config struct {
	// App info
	AppName     string `json:"appName"`
	AppVersion  string `json:"appVersion"`
	Environment string `json:"environment"`

	// Features
	IsDev           bool `json:"isDev"`
	DebugMode       bool `json:"debugMode"`
	EnableAnalytics bool `json:"enableAnalytics"`

	BuildInfo buildinfo.BuildInfo `json:"buildInfo"`
}

type ConfigService struct {
	config *config.Config
	logger *logger.ContextLogger
}

func New(cfg *config.Config, logger *logger.ContextLogger) *ConfigService {
	return &ConfigService{
		config: cfg,
		logger: logger,
	}
}

func (s *ConfigService) GetConfig() Config {
	return Config{
		AppName:         s.config.AppName,
		AppVersion:      s.config.AppVersion,
		Environment:     s.config.Environment,
		IsDev:           s.config.IsDev(),
		DebugMode:       s.config.DebugMode,
		EnableAnalytics: s.config.EnableAnalytics,
		BuildInfo:       s.config.BuildInfo,
	}
}
