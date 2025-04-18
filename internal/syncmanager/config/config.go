package config

import (
	"fmt"
	baseConfig "jarvist/internal/common/config"
	"time"
)

const (
	// Default configuration values
	DefaultMQTTBroker    = "localhost"
	DefaultMQTTPort      = 1883
	DefaultMQTTUsername  = "user"
	DefaultMQTTPassword  = "password"
	DefaultMQTTTopic     = "jarvist"
	DefaultMQTTQoS       = 2
	DefaultMQTTKeepalive = 5

	// Service information
	ServiceName        = "jarvist-sync"
	ServiceDisplayName = "JARVIST Sync Manager"
	ServiceDescription = "Service to sync data to server"

	// Encryption
	DefaultEncryptionKey = "default-encryption-key-please-change-me-now!"
)

type Config struct {
	BaseConfig *baseConfig.Config `json:"baseConfig"`
	MQTT       struct {
		Broker      string `json:"broker"`
		Port        int    `json:"port"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		ClientID    string `json:"client_id"`
		Topic       string `json:"topic"`
		QoS         byte   `json:"qos"`
		Keepalive   int    `json:"keepalive"`
		EnableTLS   bool   `json:"enable_tls"`
		CACertPath  string `json:"ca_cert_path"`
		EncryptData bool   `json:"encrypt_data"`
	} `json:"mqtt"`

	// API settings
	API struct {
		Enabled   bool   `json:"enabled"`
		Port      int    `json:"port"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		EnableTLS bool   `json:"enable_tls"`
		CertFile  string `json:"cert_file"`
		KeyFile   string `json:"key_file"`
	} `json:"api"`

	// Service settings
	Service struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	} `json:"service"`

	// Logging settings
	Advanced struct {
		MaxQueueWorkers int    `json:"max_queue_workers"`
		FernetKey       string `json:"fernetKey"`
	}

	//Sync setting
	Sync struct {
		Interval int `json:"sync_interval"`
	}

	Logger struct {
		EnableMQTTLogs bool `json:"mqtt_Logs"`
		EnableDBLogs   bool `json:"db_Logs"`
	}
}

func LoadConfig(buildMode string, baseConfig *baseConfig.Config) *Config {
	cfg := &Config{}

	cfg.BaseConfig = baseConfig

	cfg.API.Enabled = true
	cfg.API.Port = 8081
	cfg.API.Username = "admin"
	cfg.API.Password = "admin"
	cfg.API.EnableTLS = false
	cfg.API.CertFile = ""
	cfg.API.KeyFile = ""

	cfg.MQTT.ClientID = fmt.Sprintf("jarvist-%d", time.Now().Unix()%10000)
	cfg.MQTT.Topic = DefaultMQTTTopic
	cfg.MQTT.QoS = DefaultMQTTQoS
	cfg.MQTT.Keepalive = DefaultMQTTKeepalive
	cfg.MQTT.EnableTLS = false
	cfg.MQTT.CACertPath = ""
	cfg.MQTT.EncryptData = true

	cfg.Service.Name = ServiceName
	cfg.Service.DisplayName = ServiceDisplayName
	cfg.Service.Description = ServiceDescription

	cfg.Advanced.MaxQueueWorkers = 5
	cfg.Advanced.FernetKey = "0yhvieBf7ZfOWRAQdeKOtzTAvGD5OCFSIivbfOjn3Ug="

	cfg.Sync.Interval = 60
	cfg.Logger.EnableMQTTLogs = true
	cfg.Logger.EnableDBLogs = true

	if buildMode == "production" {
		setupProdConfigs(cfg)
	} else {
		setupDevConfig(cfg)
	}

	return cfg
}

func setupProdConfigs(cfg *Config) {
	cfg.MQTT.Broker = "tqty.indward.com"
	cfg.MQTT.Port = 1883
	cfg.MQTT.Username = "admin"
	cfg.MQTT.Password = "B_I-Kk4XMSF!93R1@i1ko!SF"

	cfg.API.Port = 8722
}

func setupDevConfig(cfg *Config) {
	cfg.MQTT.Broker = DefaultMQTTBroker
	cfg.MQTT.Port = DefaultMQTTPort
	cfg.MQTT.Username = DefaultMQTTUsername
	cfg.MQTT.Password = DefaultMQTTPassword
}
