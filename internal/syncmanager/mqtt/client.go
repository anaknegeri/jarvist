package mqtt

import (
	"encoding/json"
	"fmt"
	"jarvist/internal/syncmanager/config"
	"jarvist/pkg/logger"
	"jarvist/pkg/utils"
	"math"
	"sync"
	"time"

	"math/rand"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Connection retry settings
const (
	initialRetryDelay  = 1 * time.Second
	maxRetryDelay      = 60 * time.Second
	retryFactor        = 2.0
	stabilizationDelay = 3 * time.Second
	retryJitter        = 0.2 // 20% jitter
)

type Client struct {
	client          mqtt.Client
	cfg             *config.Config
	logger          *logger.Logger
	connected       bool
	lastActivity    time.Time
	mutex           sync.Mutex
	currentBackoff  time.Duration
	connectAttempt  int
	connectTimer    *time.Timer
	cleanDisconnect bool
	sentCache       map[string]bool
	sentCacheTimes  map[string]time.Time
	cacheMutex      sync.Mutex
	cacheTimeout    time.Duration
}

// NewClient creates a new MQTT client
func NewClient(cfg *config.Config, logger *logger.Logger) (*Client, error) {
	client := &Client{
		cfg:             cfg,
		logger:          logger,
		connected:       false,
		lastActivity:    time.Now(),
		mutex:           sync.Mutex{},
		currentBackoff:  initialRetryDelay,
		connectAttempt:  0,
		cleanDisconnect: false,
		sentCache:       make(map[string]bool),
		sentCacheTimes:  make(map[string]time.Time),
		cacheMutex:      sync.Mutex{},
		cacheTimeout:    30 * time.Second, // Pesan disimpan di cache selama 30 detik
	}

	// Mulai goroutine untuk membersihkan cache secara berkala
	go client.cleanupCache()

	return client, nil
}

// Fungsi untuk membersihkan cache secara berkala
func (c *Client) cleanupCache() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.cleanDisconnect {
				return
			}

			c.cacheMutex.Lock()

			if len(c.sentCache) > 1000 {
				c.sentCache = make(map[string]bool)
				c.sentCacheTimes = make(map[string]time.Time)
				c.cacheMutex.Unlock()
				continue
			}

			now := time.Now()
			for id, timestamp := range c.sentCacheTimes {
				if now.Sub(timestamp) > c.cacheTimeout {
					delete(c.sentCache, id)
					delete(c.sentCacheTimes, id)
				}
			}

			c.cacheMutex.Unlock()
		}
	}
}

// Connect connects to the MQTT broker with backoff retry logic
func (c *Client) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// If already trying to connect, don't try again
	if c.connectTimer != nil {
		return nil
	}

	c.logger.Info(ComponentMQTT, "Initiating connection to MQTT broker")

	// Reset the clean disconnect flag
	c.cleanDisconnect = false

	// Start the connection attempt immediately
	return c.connectWithBackoff()
}

// connectWithBackoff handles the actual connection with retry logic
func (c *Client) connectWithBackoff() error {
	// Create client options
	opts := mqtt.NewClientOptions()

	// Set broker address
	brokerURL := fmt.Sprintf("tcp://%s:%d", c.cfg.MQTT.Broker, c.cfg.MQTT.Port)
	if c.cfg.MQTT.EnableTLS {
		brokerURL = fmt.Sprintf("ssl://%s:%d", c.cfg.MQTT.Broker, c.cfg.MQTT.Port)
	}
	opts.AddBroker(brokerURL)

	// Generate a unique client ID if reusing the same one
	if c.connectAttempt > 0 {
		c.cfg.MQTT.ClientID = fmt.Sprintf("jarvist-%d", time.Now().Unix()%10000)
	}

	// Set client ID
	opts.SetClientID(c.cfg.MQTT.ClientID)

	// Set credentials if provided
	if c.cfg.MQTT.Username != "" {
		opts.SetUsername(c.cfg.MQTT.Username)
		opts.SetPassword(c.cfg.MQTT.Password)
	}

	// Set connection parameters
	opts.SetKeepAlive(time.Duration(c.cfg.MQTT.Keepalive) * time.Second)
	opts.SetCleanSession(true)   // Use clean session to avoid session conflicts
	opts.SetAutoReconnect(false) // We'll handle reconnections ourselves
	opts.SetConnectTimeout(30 * time.Second)
	opts.SetOrderMatters(false) // Don't block on message acks

	// Log lebih detail tentang konfigurasi MQTT
	c.logger.Info(ComponentMQTT, "MQTT Configuration - QoS: %d, ClientID: %s",
		c.cfg.MQTT.QoS, c.cfg.MQTT.ClientID)

	// Set TLS if enabled
	if c.cfg.MQTT.EnableTLS && c.cfg.MQTT.CACertPath != "" {
		tlsConfig, err := utils.NewTLSConfig(c.cfg.MQTT.CACertPath)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %v", err)
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// Set callbacks
	opts.SetOnConnectHandler(c.onConnect)
	opts.SetConnectionLostHandler(c.onDisconnect)

	// Create client
	c.client = mqtt.NewClient(opts)

	// Connect
	c.logger.Info(ComponentMQTT, "Connecting to MQTT broker at %s", brokerURL)
	token := c.client.Connect()

	// Wait for connection attempt to complete
	c.connectAttempt++
	if token.Wait() && token.Error() != nil {
		c.logger.Error(ComponentMQTT, "Failed to connect to MQTT broker (attempt %d): %v", c.connectAttempt, token.Error())

		// Schedule retry with backoff
		c.scheduleReconnect()
		return token.Error()
	}

	return nil
}

// scheduleReconnect schedules a reconnection attempt with exponential backoff
func (c *Client) scheduleReconnect() {
	if c.cleanDisconnect {
		c.logger.Info(ComponentMQTT, "Clean disconnect requested, not scheduling reconnection")
		return
	}

	// Calculate backoff with jitter
	jitter := 1.0 + (rand.Float64()*2-1)*retryJitter
	backoff := time.Duration(float64(c.currentBackoff) * jitter)

	// Log reconnection plan
	c.logger.Info(ComponentMQTT, "Scheduling reconnection attempt %d in %.2f seconds", c.connectAttempt+1, backoff.Seconds())

	// Schedule reconnection
	c.connectTimer = time.AfterFunc(backoff, func() {
		c.mutex.Lock()
		c.connectTimer = nil
		c.mutex.Unlock()

		if !c.cleanDisconnect {
			c.connectWithBackoff()
		}
	})

	// Increase backoff for next attempt (with max limit)
	c.currentBackoff = time.Duration(math.Min(
		float64(c.currentBackoff)*retryFactor,
		float64(maxRetryDelay),
	))
}

// Disconnect from the MQTT broker
func (c *Client) Disconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Mark this as a clean disconnect to avoid reconnection
	c.cleanDisconnect = true

	// Cancel any pending reconnect timers
	if c.connectTimer != nil {
		c.connectTimer.Stop()
		c.connectTimer = nil
	}

	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250) // Wait 250ms max for in-flight messages
		c.connected = false
		c.logger.Info(ComponentMQTT, "Disconnected from MQTT broker")
	}

	// Reset connection state for clean reconnect later
	c.currentBackoff = initialRetryDelay
	c.connectAttempt = 0
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.connected && c.client != nil && c.client.IsConnected()
}

// Fungsi untuk mengekstrak ID pesan dari payload
func extractMessageID(payload []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return ""
	}

	// Ekstrak ID dari struktur data
	if dataObj, ok := data["data"].(map[string]interface{}); ok {
		if id, ok := dataObj["id"].(string); ok {
			return id
		}
	}

	return ""
}

// Fungsi untuk memeriksa dan menambahkan ke cache
func (c *Client) checkAndCacheMessage(messageID string) bool {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	if _, exists := c.sentCache[messageID]; exists {
		return true
	}

	c.sentCache[messageID] = true
	c.sentCacheTimes[messageID] = time.Now()

	return false
}

// Fungsi utama untuk publish pesan MQTT
func (c *Client) Publish(topic string, payload []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected || c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("not connected to MQTT broker")
	}

	// Ekstrak ID pesan untuk pemeriksaan duplikat
	messageID := extractMessageID(payload)

	if messageID != "" {
		// Periksa apakah sudah pernah dikirim
		if c.checkAndCacheMessage(messageID) {
			c.logger.Info(ComponentMQTT, "Skipping duplicate message with ID: %s", messageID)
			return nil // Lewati jika ini adalah pesan duplikat
		}

		c.logger.Info(ComponentMQTT, "Publishing to topic %s with QoS %d, message ID: %s",
			topic, c.cfg.MQTT.QoS, messageID)
	} else {
		c.logger.Info(ComponentMQTT, "Publishing to topic %s with QoS %d (no message ID)",
			topic, c.cfg.MQTT.QoS)
	}

	// Publish pesan
	token := c.client.Publish(topic, c.cfg.MQTT.QoS, false, payload)
	token.Wait()

	if token.Error() != nil {
		return token.Error()
	}

	c.lastActivity = time.Now()
	return nil
}

// PublishHeartbeat publishes a heartbeat message
func (c *Client) PublishHeartbeat(heartbeatTopic string, payload []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected || c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("not connected to MQTT broker")
	}

	// Debug logging untuk heartbeat
	c.logger.Debug(ComponentMQTT, "Publishing heartbeat to topic %s with QoS 0", heartbeatTopic)

	// Publish with QoS 0 for heartbeat (no persistence needed)
	token := c.client.Publish(heartbeatTopic, 0, false, payload)
	token.Wait()

	if token.Error() != nil {
		return token.Error()
	}

	// Update last activity time
	c.lastActivity = time.Now()
	return nil
}

// GetLastActivity returns the time of last activity
func (c *Client) GetLastActivity() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.lastActivity
}

// SetLastActivity updates the last activity time
func (c *Client) SetLastActivity(t time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastActivity = t
}

// onConnect is called when connected to the broker
func (c *Client) onConnect(client mqtt.Client) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info(ComponentMQTT, "Connected to MQTT broker - stabilizing connection...")

	// Add a stabilization delay to ensure connection is stable
	time.AfterFunc(stabilizationDelay, func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		if client.IsConnected() {
			c.connected = true
			c.lastActivity = time.Now()
			c.currentBackoff = initialRetryDelay // Reset backoff on successful connection
			c.logger.Info(ComponentMQTT, "MQTT connection stabilized")
		}
	})
}

// onDisconnect is called when disconnected from the broker
func (c *Client) onDisconnect(client mqtt.Client, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.connected = false
	disconnectTime := time.Now().Format(time.RFC3339)

	if c.cleanDisconnect {
		c.logger.Info(ComponentMQTT, "Clean disconnect from MQTT broker")
		return
	}

	c.logger.Warning(ComponentMQTT, "Disconnected from MQTT broker at %s : %v", disconnectTime, err)
	c.logger.Warning(ComponentMQTT, "Data will be stored in local database")

	// Schedule reconnection with backoff
	c.scheduleReconnect()
}

func (c *Client) MeasurePing() time.Duration {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected || c.client == nil {
		return 0
	}

	start := time.Now()

	pingTopic := fmt.Sprintf("%s/ping", c.cfg.MQTT.Topic)
	payload := []byte(fmt.Sprintf(`{"timestamp":"%s"}`, time.Now().Format(time.RFC3339)))

	pingDone := make(chan bool, 1)
	var elapsed time.Duration

	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		elapsed = time.Since(start)
		pingDone <- true
	}

	responseTopic := fmt.Sprintf("%s/pong", c.cfg.MQTT.Topic)
	if token := c.client.Subscribe(responseTopic, 0, messageHandler); token.Wait() && token.Error() != nil {
		return 0
	}
	defer c.client.Unsubscribe(responseTopic)

	if token := c.client.Publish(pingTopic, 0, false, payload); token.Wait() && token.Error() != nil {
		return 0
	}

	select {
	case <-pingDone:
		return elapsed
	case <-time.After(500 * time.Millisecond):
		return 0
	}
}
