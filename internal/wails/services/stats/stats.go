package stats

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jarvist/internal/common/config"
	"jarvist/pkg/hardware"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// StatsService untuk mengirim statistik
type StatsService struct {
	app          *application.App
	ticker       *time.Ticker
	stopChan     chan struct{}
	sendInterval time.Duration
	DeviceID     string
	startupTime  time.Time
	isRunning    bool
	cfg          *config.Config
	mu           sync.Mutex // Mutex untuk perlindungan isRunning
	client       *http.Client
	sessionID    uint
}

// NewStatsService membuat service statistik baru
func New(cfg *config.Config) *StatsService {
	currentHardwareID, _ := hardware.GetHardwareID()

	return &StatsService{
		sendInterval: 30 * time.Minute,
		DeviceID:     currentHardwareID,
		stopChan:     make(chan struct{}),
		isRunning:    false,
		startupTime:  time.Now(),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cfg: cfg,
	}
}

// OnStartup dijalankan saat aplikasi dimulai
func (s *StatsService) OnStartup(ctx context.Context, options application.ServiceOptions) error {
	s.start()

	sid, err := s.StartSession()
	if err != nil {
		//runtime.LogError(a.ctx, "Failed to start session: "+err.Error())
	} else {
		s.sessionID = sid
	}

	s.recordEvent("system", "app_start", map[string]interface{}{
		"startTime": time.Now().Format(time.RFC3339),
	})

	return nil
}

// ServiceShutdown dijalankan saat aplikasi ditutup
func (s *StatsService) OnShutdown() error {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		eventData := map[string]interface{}{
			"deviceId":  s.DeviceID,
			"eventType": "system",
			"eventName": "app_stop",
			"version":   s.cfg.AppVersion,
			"metadata": map[string]interface{}{
				"stopTime": time.Now().Format(time.RFC3339),
			},
		}

		// Buat timeout context untuk membatasi waktu menunggu
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Kirim dengan timeout
		done := make(chan error, 1)
		go func() {
			done <- s.sendRequest("stats/event", eventData)
		}()

		select {
		case err := <-done:
			if err != nil {
				s.app.Logger.Error("Failed to send app_stop event: " + err.Error())
			} else {
				s.app.Logger.Info("App_stop event sent successfully")
			}
		case <-timeoutCtx.Done():
			s.app.Logger.Warn("Timeout waiting for app_stop event to send")
		}
	}()

	if s.sessionID > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			sessionDuration := int(time.Since(s.startupTime).Seconds())

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- s.endSession(s.sessionID, sessionDuration)
			}()

			select {
			case err := <-done:
				if err != nil {
					s.app.Logger.Error("Failed to end session: " + err.Error())
				} else {
					s.app.Logger.Info("Session ended successfully")
				}
			case <-timeoutCtx.Done():
				s.app.Logger.Warn("Timeout waiting for session end")
			}
		}()
	}
	wg.Wait()

	s.stop()
	return nil
}

// Initialize menginisialisasi service dengan app instance dan konfigurasi
func (s *StatsService) InitService(app *application.App) {
	s.app = app
}

// Start memulai service statistik
func (s *StatsService) start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return
	}

	s.ticker = time.NewTicker(s.sendInterval)
	s.stopChan = make(chan struct{})
	s.isRunning = true

	// Kirim heartbeat pertama
	s.sendHeartbeat()

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.sendHeartbeat()
			case <-s.stopChan:
				return
			}
		}
	}()

	if s.app != nil {
		s.app.Logger.Info("Stats service started")
	}
}

// Stop menghentikan service statistik
func (s *StatsService) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	if s.ticker != nil {
		s.ticker.Stop()
	}

	close(s.stopChan) // Lebih baik menggunakan close daripada mengirim ke channel
	s.isRunning = false

	if s.app != nil {
		s.app.Logger.Info("Stats service stopped")
	}
}

// sendHeartbeat mengirim data heartbeat ke server
func (s *StatsService) sendHeartbeat() {
	heartbeat := map[string]interface{}{
		"version":  s.cfg.AppVersion,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"deviceId": s.DeviceID,
		"appName":  s.cfg.AppName,
		"metadata": map[string]interface{}{
			"memoryUsage": getMemoryUsage(),
			"uptime":      time.Since(s.startupTime).Seconds(),
		},
	}

	s.sendRequestAsync("stats/heartbeat", heartbeat)
}

// StartSession memulai sesi baru dan mengembalikan ID sesi
func (s *StatsService) StartSession() (uint, error) {
	isNewInstall := checkIfNewInstall()

	sessionData := map[string]interface{}{
		"deviceId":     s.DeviceID,
		"version":      s.cfg.AppVersion,
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"isNewInstall": isNewInstall,
		"metadata": map[string]interface{}{
			"startTime": time.Now().Format(time.RFC3339),
		},
	}

	var response struct {
		Message string `json:"message"`
		Data    struct {
			SessionID uint   `json:"sessionId"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
	}

	err := s.sendRequestWithResponse("stats/session/start", sessionData, &response)
	if err != nil {
		return 0, err
	}

	return response.Data.SessionID, nil
}

// endSession mengakhiri sesi dengan ID tertentu
func (s *StatsService) endSession(sessionID uint, durationSecs int) error {
	sessionData := map[string]interface{}{
		"deviceId":     s.DeviceID,
		"sessionId":    sessionID,
		"durationSecs": durationSecs,
		"metadata": map[string]interface{}{
			"endTime": time.Now().Format(time.RFC3339),
		},
	}

	return s.sendRequest("stats/session/end", sessionData)
}

// recordEvent merekam event ke server
func (s *StatsService) recordEvent(eventType, eventName string, metadata map[string]interface{}) {
	event := map[string]interface{}{
		"deviceId":  s.DeviceID,
		"eventType": eventType,
		"eventName": eventName,
		"version":   s.cfg.AppVersion,
		"metadata":  metadata,
	}

	s.sendRequestAsync("stats/event", event)
}

// SendRequest mengirim request ke endpoint tertentu
func (s *StatsService) sendRequest(endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		if s.app != nil {
			s.app.Logger.Error("Failed to marshal JSON: " + err.Error())
		}
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/app/%s", s.cfg.ApiUrl, endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		if s.app != nil {
			s.app.Logger.Error("Failed to create request: " + err.Error())
		}
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-API-Key", s.cfg.ApiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		if s.app != nil {
			s.app.Logger.Error("Failed to send request: " + err.Error())
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if s.app != nil {
			s.app.Logger.Debug(fmt.Sprintf("Request to %s successful", endpoint))
		}
		return nil
	} else {
		errMsg := fmt.Sprintf("Request to %s failed with status: %d", endpoint, resp.StatusCode)
		if s.app != nil {
			s.app.Logger.Error(errMsg)
		}
		return errors.New(errMsg)
	}
}

// sendRequestWithResponse mengirim request dan membaca responsenya
func (s *StatsService) sendRequestWithResponse(endpoint string, data interface{}, response interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		if s.app != nil {
			s.app.Logger.Error("Failed to marshal JSON: " + err.Error())
		}
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/app/%s", s.cfg.ApiUrl, endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		if s.app != nil {
			s.app.Logger.Error("Failed to create request: " + err.Error())
		}
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-API-Key", s.cfg.ApiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		if s.app != nil {
			s.app.Logger.Error("Failed to send request: " + err.Error())
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if s.app != nil {
			s.app.Logger.Debug(fmt.Sprintf("Request to %s successful", endpoint))
		}
		return json.NewDecoder(resp.Body).Decode(response)
	} else {
		errMsg := fmt.Sprintf("Request to %s failed with status: %d", endpoint, resp.StatusCode)
		if s.app != nil {
			s.app.Logger.Error(errMsg)
		}
		return errors.New(errMsg)
	}
}

// sendRequestAsync mengirim request secara asynchronous
func (s *StatsService) sendRequestAsync(endpoint string, data interface{}) {
	go func() {
		err := s.sendRequest(endpoint, data)
		if err != nil && s.app != nil {
			s.app.Logger.Error(fmt.Sprintf("Async request to %s failed: %s", endpoint, err.Error()))
		}
	}()
}

// checkIfNewInstall memeriksa apakah ini adalah instalasi baru
func checkIfNewInstall() bool {
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		return false
	}

	installFlagFile := filepath.Join(appDataDir, ".jarvist", "installed")
	if _, err := os.Stat(installFlagFile); os.IsNotExist(err) {
		// Pastikan direktori induk ada
		if err := os.MkdirAll(filepath.Dir(installFlagFile), 0755); err != nil {
			return false
		}
		// Tandai sebagai terinstal
		if err := os.WriteFile(installFlagFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
			return false
		}
		return true
	}

	return false
}

// getMemoryUsage mendapatkan penggunaan memori aplikasi
func getMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}
