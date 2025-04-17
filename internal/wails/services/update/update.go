package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"jarvist/internal/common/config"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type UpdateInfo struct {
	Version     string `json:"version"`
	ReleaseDate string `json:"releaseDate"`
	DownloadURL string `json:"downloadUrl"`
	Notes       string `json:"notes"`
	IsForced    bool   `json:"isForced"`
	Checksum    string `json:"checksum"`
}

type UpdateResponse struct {
	Success bool       `json:"success"`
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    UpdateInfo `json:"data"`
}

type UpdateEvent struct {
	Event     string    `json:"event"`
	Message   string    `json:"message"`
	Progress  int       `json:"progress,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Data      any       `json:"data,omitempty"`
}

type UpdateService struct {
	currentVersion  string
	updateServerURL string
	updateProcess   *exec.Cmd
	mu              sync.Mutex
	isChecking      bool
	isDownloading   bool
	isInstalling    bool
	cfg             *config.Config
	app             *application.App
}

func New(cfg *config.Config) *UpdateService {
	updateServerURL := fmt.Sprintf("%s/v1/app/updates", cfg.ApiUrl)

	return &UpdateService{
		currentVersion:  cfg.AppVersion,
		updateServerURL: updateServerURL,
		cfg:             cfg,
	}
}

func (s *UpdateService) InitService(app *application.App) {
	s.app = app
}

func (s *UpdateService) GetCurrentVersion() string {
	return s.currentVersion
}

func (s *UpdateService) SetCurrentVersion(version string) {
	s.currentVersion = version
}

func (s *UpdateService) CheckForUpdates() (*UpdateInfo, error) {
	s.mu.Lock()
	if s.isChecking {
		s.mu.Unlock()
		return nil, fmt.Errorf("already checking for updates")
	}
	s.isChecking = true
	s.mu.Unlock()

	s.emitEvent("update_checking", "Checking for updates...", true, nil)

	defer func() {
		s.mu.Lock()
		s.isChecking = false
		s.mu.Unlock()
	}()

	checkURL := fmt.Sprintf("%s/check?version=%s&os=%s&arch=%s",
		s.updateServerURL,
		s.currentVersion,
		runtime.GOOS,
		runtime.GOARCH)

	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		s.emitEvent("update_check_error", "Error creating request: "+err.Error(), false, nil)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-API-Key", s.cfg.ApiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		s.emitEvent("update_check_error", "Error: "+err.Error(), false, nil)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("HTTP error: %d", resp.StatusCode)
		s.emitEvent("update_check_error", errorMsg, false, nil)
		return nil, fmt.Errorf("%s", errorMsg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.emitEvent("update_check_error", "Error: "+err.Error(), false, nil)
		return nil, err
	}

	var response UpdateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		s.emitEvent("update_check_error", "Error: "+err.Error(), false, nil)
		return nil, err
	}

	updateInfo := response.Data

	if updateInfo.Version == s.currentVersion {
		s.emitEvent("update_check_complete", "You are using the latest version", true, nil)
		return nil, nil
	}

	s.emitEvent("update_available", fmt.Sprintf("Version %s is available", updateInfo.Version), true, updateInfo)
	return &updateInfo, nil
}

func (s *UpdateService) DownloadUpdate(updateInfo *UpdateInfo) error {
	s.mu.Lock()
	if s.isDownloading {
		s.mu.Unlock()
		return fmt.Errorf("already downloading update")
	}
	s.isDownloading = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isDownloading = false
		s.mu.Unlock()
	}()

	s.emitEvent("update_download_start", "Downloading update...", true, nil)

	downloadDir := filepath.Join(s.cfg.TempDir, "updates")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		s.emitEvent("update_download_error", "Error: "+err.Error(), false, nil)
		return err
	}

	var fileName string
	switch runtime.GOOS {
	case "windows":
		fileName = fmt.Sprintf("update-%s.exe", updateInfo.Version)
	case "darwin":
		fileName = fmt.Sprintf("update-%s.dmg", updateInfo.Version)
	default:
		fileName = fmt.Sprintf("update-%s.tar.gz", updateInfo.Version)
	}

	downloadPath := filepath.Join(downloadDir, fileName)

	file, err := os.Create(downloadPath)
	if err != nil {
		s.emitEvent("update_download_error", "Error: "+err.Error(), false, nil)
		return err
	}
	defer file.Close()

	resp, err := http.Get(updateInfo.DownloadURL)
	if err != nil {
		s.emitEvent("update_download_error", "Error: "+err.Error(), false, nil)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("HTTP error: %d", resp.StatusCode)
		s.emitEvent("update_download_error", errorMsg, false, nil)
		return fmt.Errorf("%s", errorMsg)
	}

	total := resp.ContentLength
	buf := make([]byte, 1024*32) // 32KB chunks
	var downloaded int64
	hash := sha256.New()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				s.emitEvent("update_download_error", "Error: "+writeErr.Error(), false, nil)
				return writeErr
			}

			hash.Write(buf[:n])
			downloaded += int64(n)

			if total > 0 {
				progress := int((float64(downloaded) / float64(total)) * 100)
				s.emitEvent("update_download_progress", fmt.Sprintf("Downloading: %d%%", progress), true, progress)
			}
		}

		if err != nil {
			if err != io.EOF {
				s.emitEvent("update_download_error", "Error: "+err.Error(), false, nil)
				return err
			}
			break
		}
	}

	if updateInfo.Checksum != "" {
		s.emitEvent("update_download_progress", "Verifying checksum...", true, 100)
		actualChecksum := hex.EncodeToString(hash.Sum(nil))
		if actualChecksum != updateInfo.Checksum {
			s.emitEvent("update_download_error", "Invalid checksum! File may be corrupted.", false, nil)
			return fmt.Errorf("invalid checksum: expected %s, got %s", updateInfo.Checksum, actualChecksum)
		}
		s.emitEvent("update_download_progress", "Checksum verified", true, 100)
	}

	pendingUpdateInfo := map[string]string{
		"path":      downloadPath,
		"version":   updateInfo.Version,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	pendingUpdateJson, _ := json.Marshal(pendingUpdateInfo)
	pendingUpdatePath := filepath.Join(filepath.Dir(downloadPath), "pending_update.json")

	if err := os.WriteFile(pendingUpdatePath, pendingUpdateJson, 0644); err != nil {
		s.emitEvent("update_flag_error", "Error creating update flag: "+err.Error(), false, nil)
	}

	s.emitEvent("update_download_complete", "Download completed. Update will be installed on restart.", true, downloadPath)
	return nil
}

func (s *UpdateService) InstallUpdate(downloadPath string) error {
	s.mu.Lock()
	if s.isInstalling {
		s.mu.Unlock()
		return fmt.Errorf("already installing update")
	}
	s.isInstalling = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isInstalling = false
		s.mu.Unlock()
	}()

	s.emitEvent("update_install_start", "Installing update...", true, nil)

	if !fileExists(downloadPath) {
		s.emitEvent("update_install_error", "Installer file not found", false, nil)
		return fmt.Errorf("file not found: %s", downloadPath)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command(downloadPath, "/SILENT", "/NORESTART")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
	case "darwin":
		cmd = exec.Command("open", downloadPath)
	default:
		cmd = exec.Command("tar", "-xzf", downloadPath, "-C", filepath.Dir(downloadPath))
	}

	s.updateProcess = cmd

	if err := cmd.Start(); err != nil {
		s.emitEvent("update_install_error", "Error: "+err.Error(), false, nil)
		return err
	}

	go func() {
		err := cmd.Wait()

		if err != nil {
			s.emitEvent("update_install_error", "Error: "+err.Error(), false, nil)
		} else {
			s.emitEvent("update_install_complete", "Installation completed. Please restart the application.", true, nil)
		}

		s.mu.Lock()
		s.updateProcess = nil
		s.isInstalling = false
		s.mu.Unlock()
	}()

	return nil
}

func (s *UpdateService) InstallUpdateOnRestart(downloadPath string) error {
	pendingUpdateInfo := map[string]string{
		"path":      downloadPath,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	pendingUpdateJson, _ := json.Marshal(pendingUpdateInfo)
	pendingUpdatePath := filepath.Join(filepath.Dir(downloadPath), "pending_update.json")

	if err := os.WriteFile(pendingUpdatePath, pendingUpdateJson, 0644); err != nil {
		return err
	}

	s.emitEvent("update_restart_ready", "Update ready to install on restart", true, nil)
	return nil
}

func (s *UpdateService) CancelUpdate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.updateProcess != nil && s.updateProcess.Process != nil {
		if err := s.updateProcess.Process.Kill(); err != nil {
			s.emitEvent("update_cancel_error", "Error: "+err.Error(), false, nil)
			return err
		}

		s.updateProcess = nil
		s.emitEvent("update_cancelled", "Update cancelled", true, nil)
	}

	return nil
}

func (s *UpdateService) CleanupDownloads() error {
	downloadDir := filepath.Join(s.cfg.TempDir, "updates")

	if !fileExists(downloadDir) {
		return nil
	}

	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(downloadDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				s.emitEvent("update_cleanup_error", fmt.Sprintf("Error removing %s: %v", filePath, err), false, nil)
			}
		}
	}

	s.emitEvent("update_cleanup_complete", "Download files cleanup completed", true, nil)
	return nil
}

func (s *UpdateService) CheckPendingUpdates() bool {
	pendingUpdatePath, _ := s.getPendingUpdatePath()

	updatePath, err := s.readPendingUpdateFile(pendingUpdatePath)
	if err != nil || !fileExists(updatePath) {
		if err == nil && !fileExists(updatePath) {
			os.Remove(pendingUpdatePath)
		}
		return false
	}

	return true
}

func (s *UpdateService) InstallPendingUpdates() error {
	pendingUpdatePath, err := s.getPendingUpdatePath()
	if err != nil {
		return err
	}

	updatePath, err := s.readPendingUpdateFile(pendingUpdatePath)
	if err != nil {
		return err
	}

	if !fileExists(updatePath) {
		os.Remove(pendingUpdatePath)
		return fmt.Errorf("update installer not found: %s", updatePath)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command(updatePath, "/SILENT", "/NORESTART")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}

	case "darwin":
		cmd = exec.Command("open", updatePath)
	default:
		cmd = exec.Command("bash", "-c", fmt.Sprintf("nohup %s &", updatePath))
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	os.Remove(pendingUpdatePath)
	if runtime.GOOS != "darwin" {
		os.Exit(0)
	}

	return nil
}

func (s *UpdateService) SetUpdateServerURL(url string) {
	s.updateServerURL = url
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func (s *UpdateService) emitEvent(event, message string, success bool, data any) {
	updateEvent := UpdateEvent{
		Event:     event,
		Message:   message,
		Timestamp: time.Now(),
		Success:   success,
		Data:      data,
	}

	jsonData, _ := json.Marshal(updateEvent)
	s.app.EmitEvent("update_event", string(jsonData))
}

func (s *UpdateService) getPendingUpdatePath() (string, error) {
	updateDir := filepath.Join(s.cfg.TempDir, "updates")
	return filepath.Join(updateDir, "pending_update.json"), nil
}

func (s *UpdateService) readPendingUpdateFile(pendingUpdatePath string) (string, error) {
	if !fileExists(pendingUpdatePath) {
		return "", fmt.Errorf("no pending update")
	}

	pendingUpdateData, err := os.ReadFile(pendingUpdatePath)
	if err != nil {
		return "", err
	}

	var pendingUpdate map[string]string
	if err := json.Unmarshal(pendingUpdateData, &pendingUpdate); err != nil {
		return "", err
	}

	return pendingUpdate["path"], nil
}
