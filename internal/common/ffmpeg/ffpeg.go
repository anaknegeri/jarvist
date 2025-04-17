package ffmpeg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jarvist/internal/common/config" // Update this import to match your project structure
	"jarvist/pkg/logger"             // Update this import to match your project structure
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

type RTSPConfig struct {
	Schema   string `json:"schema"`
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	Path     string `json:"path,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type RTSPOptions struct {
	TakeScreenshot bool `json:"takeScreenshot"`
}

type ResponseJSON struct {
	Success        bool      `json:"success"`
	Message        string    `json:"message"`
	URL            string    `json:"url,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	Error          string    `json:"error,omitempty"`
	Data           any       `json:"data,omitempty"`
	ScreenshotPath string    `json:"screenshotPath,omitempty"`
}

var (
	FFmpegPath       string
	runningProcesses []*exec.Cmd
	processMutex     sync.Mutex
	cfg              *config.Config        // Global configuration
	logr             *logger.ContextLogger // Global logger
)

// SetupFFmpeg initializes FFmpeg with the provided configuration
func SetupFFmpeg(config *config.Config, logger *logger.Logger) error {
	// Store references to config and logger
	cfg = config
	logr = logger.WithComponent("ffpeg")

	// Check if config is nil
	if cfg == nil {
		return errors.New("nil config provided to SetupFFmpeg")
	}

	// Get FFmpeg path based on config
	ffmpegDir := filepath.Join(cfg.BinDir, "ffmpeg")
	FFmpegPath = filepath.Join(ffmpegDir, "ffmpeg.exe")

	// Check if FFmpeg exists
	if _, err := os.Stat(FFmpegPath); err != nil {
		if logger != nil {
			logger.Error("FFmpeg not found at: %s", FFmpegPath)
		}
		return err
	}

	// Log success
	if logger != nil {
		logger.Info("FFmpeg found at: %s", FFmpegPath)
	}

	return nil
}

// GetFFmpegPath returns the current FFmpeg path
func GetFFmpegPath() string {
	return FFmpegPath
}

// GenerateRTSPURL constructs an RTSP URL from the provided configuration
func GenerateRTSPURL(rtspConfig RTSPConfig) (string, error) {
	if rtspConfig.Host == "" {
		return "", errors.New("host is required")
	}

	if rtspConfig.Schema == "" {
		rtspConfig.Schema = "rtsp"
	}

	var credentials string
	if rtspConfig.Username != "" {
		if rtspConfig.Password != "" {
			credentials = fmt.Sprintf("%s:%s@", rtspConfig.Username, url.QueryEscape(rtspConfig.Password))
		} else {
			credentials = fmt.Sprintf("%s@", rtspConfig.Username)
		}
	}

	var portPart string
	if rtspConfig.Port != 0 {
		portPart = fmt.Sprintf(":%d", rtspConfig.Port)
	}

	var pathPart string
	if rtspConfig.Path != "" {
		if strings.HasPrefix(rtspConfig.Path, "/") {
			pathPart = rtspConfig.Path
		} else {
			pathPart = "/" + rtspConfig.Path
		}
	}

	rtspURL := fmt.Sprintf("%s://%s%s%s%s", rtspConfig.Schema, credentials, rtspConfig.Host, portPart, pathPart)
	return rtspURL, nil
}

// CheckRTSPConnectionWithConfig checks an RTSP connection using the provided configuration
func CheckRTSPConnectionWithConfig(config RTSPConfig, options RTSPOptions) string {
	// Initialize response with timestamp and config data
	response := ResponseJSON{
		Timestamp: time.Now(),
		Data:      config,
	}

	// Generate RTSP URL from config
	rtspURL, err := GenerateRTSPURL(config)
	if err != nil {
		response.Success = false
		response.Message = "Failed to create RTSP URL"
		response.Error = err.Error()
		jsonResponse, _ := json.Marshal(response)
		return string(jsonResponse)
	}

	response.URL = rtspURL

	// Check the connection
	return checkRTSPConnectionInternal(rtspURL, options, response)
}

// CheckRTSPConnection checks an RTSP connection with the provided URL
func CheckRTSPConnection(rtspURL string, options RTSPOptions) string {
	response := ResponseJSON{
		URL:       rtspURL,
		Timestamp: time.Now(),
	}

	return checkRTSPConnectionInternal(rtspURL, options, response)
}

// checkRTSPConnectionInternal handles the actual connection check
func checkRTSPConnectionInternal(rtspURL string, options RTSPOptions, response ResponseJSON) string {
	// Make sure we have FFmpeg
	ffmpegPath := GetFFmpegPath()
	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		response.Success = false
		response.Message = "FFmpeg not found"
		response.Error = "ffmpeg not found at " + ffmpegPath
		jsonResponse, _ := json.Marshal(response)
		return string(jsonResponse)
	}

	// Set up a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run FFmpeg command to check connection
	cmd := exec.CommandContext(ctx, ffmpegPath, "-rtsp_transport", "tcp", "-i", rtspURL, "-t", "2", "-f", "null", "-")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	// Track process for cleanup
	TrackProcess(cmd)
	defer UntrackProcess(cmd)

	// Run the command
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		response.Success = false
		response.Message = "RTSP connection timed out after 30 seconds"
		response.Error = "operation timed out"
		jsonResponse, _ := json.Marshal(response)
		return string(jsonResponse)
	}

	// Handle connection errors
	if err != nil {
		response.Success = false

		if strings.Contains(outputStr, "Connection refused") {
			response.Message = "RTSP connection refused"
		} else if strings.Contains(outputStr, "Connection timed out") {
			response.Message = "RTSP connection timed out"
		} else if strings.Contains(outputStr, "401 Unauthorized") {
			response.Message = "Invalid RTSP credentials"
		} else if strings.Contains(outputStr, "404 Not Found") {
			response.Message = "RTSP URL not found"
		} else {
			response.Message = "Failed to connect to RTSP stream"
		}

		response.Error = err.Error()
		response.Data = map[string]string{"output": limitOutputSize(outputStr, 500)}
		jsonResponse, _ := json.Marshal(response)
		return string(jsonResponse)
	}

	// Connection successful
	response.Success = true
	response.Message = "RTSP connection successful"

	// Take screenshot if requested
	if options.TakeScreenshot {
		// Add panic recovery for screenshot function
		func() {
			defer func() {
				if r := recover(); r != nil {
					if logr != nil {
						logr.Error(fmt.Sprintf("Panic in screenshot capture: %v", r))
					}
					response.Data = map[string]string{
						"screenshotError": fmt.Sprintf("Panic in screenshot capture: %v", r),
					}
				}
			}()

			screenshotPath, screenshotErr := captureScreenshot(rtspURL)
			if screenshotErr != nil {
				response.Data = map[string]string{
					"screenshotError": screenshotErr.Error(),
				}
			} else {
				response.ScreenshotPath = screenshotPath
			}
		}()
	}

	// Return JSON response
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse)
}

// captureScreenshot captures a frame from the RTSP stream
func captureScreenshot(rtspURL string) (string, error) {
	// Check for valid configuration
	if cfg == nil {
		return "", fmt.Errorf("configuration not initialized, cannot capture screenshot")
	}

	// Get screenshot directory with fallback
	screenshotDir := cfg.ScreenshotDir
	if screenshotDir == "" {
		screenshotDir = os.TempDir()
	}

	// Ensure directory exists
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	// Create filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("rtsp_screenshot_%s.jpg", timestamp)
	outputPath := filepath.Join(screenshotDir, filename)

	// Make sure FFmpeg exists
	ffmpegPath := GetFFmpegPath()
	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		return "", fmt.Errorf("ffmpeg not found at %s", ffmpegPath)
	}

	// Create FFmpeg command
	cmd := exec.Command(
		ffmpegPath,
		"-y",
		"-rtsp_transport", "tcp",
		"-i", rtspURL,
		"-frames:v", "1",
		"-q:v", "2",
		"-s", "480x360",
		outputPath,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	// Track and run the process
	TrackProcess(cmd)
	defer UntrackProcess(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w: %s", err, string(output))
	}

	// Verify screenshot was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("screenshot file not created")
	}

	return outputPath, nil
}

// TrackProcess adds a process to the tracking list
func TrackProcess(cmd *exec.Cmd) {
	processMutex.Lock()
	defer processMutex.Unlock()
	runningProcesses = append(runningProcesses, cmd)
}

// UntrackProcess removes a process from the tracking list
func UntrackProcess(cmd *exec.Cmd) {
	processMutex.Lock()
	defer processMutex.Unlock()

	for i, p := range runningProcesses {
		if p == cmd {
			runningProcesses = append(runningProcesses[:i], runningProcesses[i+1:]...)
			break
		}
	}
}

// KillAllProcesses terminates all tracked processes
func KillAllProcesses() {
	processMutex.Lock()
	defer processMutex.Unlock()

	for _, cmd := range runningProcesses {
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
		}
	}

	runningProcesses = []*exec.Cmd{}

	// Kill any stray FFmpeg processes
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/F", "/IM", "ffmpeg.exe")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
		cmd.Run()
	} else {
		exec.Command("pkill", "-9", "ffmpeg").Run()
	}
}

// limitOutputSize restricts the output string length
func limitOutputSize(output string, maxSize int) string {
	if len(output) <= maxSize {
		return output
	}
	return output[:maxSize] + "..."
}
