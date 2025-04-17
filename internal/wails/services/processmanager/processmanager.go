package processmanager

import (
	"context"
	"fmt"
	"jarvist/internal/common/config"
	"jarvist/pkg/logger"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Config struct {
	ServicesDir string
	LogsDir     string
}

type EventData struct {
	ProcessId string    `json:"processId"`
	Message   string    `json:"message"`
	PID       int       `json:"pid,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Data      any       `json:"data,omitempty"`
}

type ProcessManagerService struct {
	app             *application.App
	processes       map[string]*exec.Cmd
	mu              sync.Mutex
	cfg             *config.Config
	config          Config
	logger          *logger.ContextLogger
	monitorStopChan chan struct{}
}

func New(cfg *config.Config, logger *logger.ContextLogger) *ProcessManagerService {
	return &ProcessManagerService{
		processes: make(map[string]*exec.Cmd),
		cfg:       cfg,
		config: Config{
			ServicesDir: filepath.Join(cfg.BinDir, "services"),
			LogsDir:     filepath.Join(cfg.BinDir, "services", "logs"),
		},
		logger: logger.WithComponent("processmanager"),
	}
}

func (s *ProcessManagerService) OnStartup(ctx context.Context, options application.ServiceOptions) error {
	return nil
}

func (s *ProcessManagerService) OnShutdown() error {
	s.StopStatusMonitor()
	return nil
}

func (s *ProcessManagerService) InitService(app *application.App) {
	s.app = app
	s.logger.Info("Initializing ProcessManagerService")
	s.logger.Info("Services directory: %s", s.config.ServicesDir)
	s.logger.Info("Logs directory: %s", s.config.LogsDir)

	if err := os.MkdirAll(s.config.LogsDir, 0755); err != nil {
		s.logger.Error("Failed to create logs directory: %v", err)
	} else {
		s.logger.Info("Created logs directory: %s", s.config.LogsDir)
	}

	s.StartStatusMonitor()
}

func (s *ProcessManagerService) StartStatusMonitor() {
	s.logger.Info("Starting process status monitor")

	if s.monitorStopChan != nil {
		close(s.monitorStopChan)
	}
	s.monitorStopChan = make(chan struct{})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.VerifyAllProcessStatusConsistency()
			case <-s.monitorStopChan:
				s.logger.Info("Process status monitor stopped")
				return
			}
		}
	}()
}

func (s *ProcessManagerService) StopStatusMonitor() {
	if s.monitorStopChan != nil {
		close(s.monitorStopChan)
		s.monitorStopChan = nil
	}
}

func (s *ProcessManagerService) CheckRunningProcesses() {
	s.logger.Debug("Checking running processes")
	s.UpdateProcessStatusOnMissing("people_counter.bat")
	s.UpdateProcessStatusOnMissing("sync_manager.bat")

	s.checkProcessWithStatus("people_counter_pid.txt", "people_counter_status.txt", "people_counter.bat")
	s.checkProcessWithStatus("sync_manager_pid.txt", "sync_manager_status.txt", "sync_manager.bat")
}

func (s *ProcessManagerService) checkProcessWithStatus(pidFileName, statusFileName, processId string) {
	pidPath := filepath.Join(s.config.LogsDir, pidFileName)
	statusPath := filepath.Join(s.config.LogsDir, statusFileName)

	if _, err := os.Stat(pidPath); err == nil {
		pidBytes, err := os.ReadFile(pidPath)
		if err == nil {
			pidStr := strings.TrimSpace(string(pidBytes))
			pid, err := strconv.Atoi(pidStr)

			if err == nil && s.isProcessRunningByPid(pid) {
				status := "Running"
				if _, err := os.Stat(statusPath); err == nil {
					statusBytes, err := os.ReadFile(statusPath)
					if err == nil {
						rawStatus := strings.TrimSpace(string(statusBytes))

						switch rawStatus {
						case "initializing":
							status = "Initializing"
						case "loading":
							status = "Loading"
						case "running":
							status = "Running"
						case "stopped":
							status = "Stopped"
						case "error":
							status = "Error"
						}
					}
				}

				s.logger.Info("Found running process %s with PID %d, status: %s", processId, pid, status)

				eventData := EventData{
					ProcessId: processId,
					Message:   "Process is already running",
					PID:       pid,
					Timestamp: time.Now(),
					Success:   true,
					Data:      map[string]interface{}{"status": status},
				}

				if s.app != nil {
					s.app.EmitEvent("process_running", eventData)
				}
			}
		}
	}
}

func (s *ProcessManagerService) IsProcessRunning(processId string) bool {
	s.mu.Lock()
	_, exists := s.processes[processId]
	s.mu.Unlock()

	if exists {
		s.logger.Debug("Process %s found in internal processes map", processId)
		return true
	}

	isRunning := s.checkProcessFromPidFile(processId)
	if !isRunning {
		s.logger.Debug("Process %s not found in PID file, updating status", processId)
		s.UpdateProcessStatusOnMissing(processId)
	} else {
		s.logger.Debug("Process %s found from PID file and is running", processId)
	}

	return isRunning
}

func (s *ProcessManagerService) UpdateProcessStatusOnMissing(processId string) bool {
	pidFilename := strings.Replace(processId, ".bat", "_pid.txt", 1)
	pidPath := filepath.Join(s.config.LogsDir, pidFilename)
	statusFilename := strings.Replace(processId, ".bat", "_status.txt", 1)
	statusPath := filepath.Join(s.config.LogsDir, statusFilename)

	s.logger.Debug("Checking status for %s (PID file: %s, Status file: %s)", processId, pidPath, statusPath)

	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		s.logger.Debug("Status file does not exist for %s", processId)
		return false
	}

	statusBytes, err := os.ReadFile(statusPath)
	if err != nil {
		s.logger.Error("Failed to read status file for %s: %v", processId, err)
		return false
	}
	currentStatus := strings.TrimSpace(string(statusBytes))
	s.logger.Debug("Current status for %s: %s", processId, currentStatus)

	if currentStatus == "stopped" || currentStatus == "error" {
		return false
	}

	pidExists := false
	var pid int
	if _, err := os.Stat(pidPath); err == nil {
		pidExists = true
		pidBytes, err := os.ReadFile(pidPath)
		if err == nil {
			pidStr := strings.TrimSpace(string(pidBytes))
			pid, _ = strconv.Atoi(pidStr)
			s.logger.Debug("Found PID %d for %s", pid, processId)
		} else {
			s.logger.Error("Failed to read PID file for %s: %v", processId, err)
		}
	} else {
		s.logger.Debug("PID file does not exist for %s", processId)
	}

	processRunning := false
	if pidExists && pid > 0 {
		processRunning = s.isProcessRunningByPid(pid)
		s.logger.Debug("Process %s with PID %d running status: %v", processId, pid, processRunning)
	}

	if !pidExists || !processRunning {
		s.logger.Info("Updating status to 'stopped' for %s (pid exists: %v, process running: %v)",
			processId, pidExists, processRunning)

		err := os.WriteFile(statusPath, []byte("stopped"), 0644)
		if err != nil {
			s.logger.Error("Failed to write status file for %s: %v", processId, err)
			return false
		}

		if pidExists && !processRunning {
			if err := os.Remove(pidPath); err != nil {
				s.logger.Error("Failed to remove PID file for %s: %v", processId, err)
			} else {
				s.logger.Debug("Removed stale PID file for %s", processId)
			}
		}

		eventData := EventData{
			ProcessId: processId,
			Message:   "Process status updated to stopped (process not found)",
			Timestamp: time.Now(),
			Success:   true,
		}

		if s.app != nil {
			s.app.EmitEvent("process_status_updated", eventData)
		}

		return true
	}

	return false
}

func (s *ProcessManagerService) checkProcessFromPidFile(processId string) bool {
	pidFilename := strings.Replace(processId, ".bat", "_pid.txt", 1)
	pidPath := filepath.Join(s.config.LogsDir, pidFilename)

	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		s.logger.Debug("PID file not found for %s", processId)
		return false
	}

	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		s.logger.Error("Failed to read PID file for %s: %v", processId, err)
		return false
	}

	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		s.logger.Error("Invalid PID format for %s: %s", processId, pidStr)
		return false
	}

	isRunning := s.isProcessRunningByPid(pid)
	s.logger.Debug("Process %s with PID %d is running: %v", processId, pid, isRunning)
	return isRunning
}

func (s *ProcessManagerService) isProcessRunningByPid(pid int) bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
		output, err := cmd.Output()
		if err != nil {
			s.logger.Error("Failed to check process status for PID %d: %v", pid, err)
			return false
		}
		return strings.Contains(string(output), fmt.Sprintf("%d", pid))
	} else {
		_, err := os.FindProcess(pid)
		return err == nil
	}
}

func (s *ProcessManagerService) StopProcess(processId string) bool {
	s.logger.Info("Attempting to stop process %s", processId)

	s.mu.Lock()
	cmd, exists := s.processes[processId]
	s.mu.Unlock()

	if exists {
		s.logger.Info("Found process %s in memory, stopping with PID %d", processId, cmd.Process.Pid)

		eventData := EventData{
			ProcessId: processId,
			Message:   "Stopping process",
			Timestamp: time.Now(),
			Success:   true,
		}

		if s.app != nil {
			s.app.EmitEvent("process_stopping", eventData)
		}

		if runtime.GOOS == "windows" {
			killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(cmd.Process.Pid))

			killCmd.SysProcAttr = &syscall.SysProcAttr{
				HideWindow:    true,
				CreationFlags: 0x08000000,
			}

			if output, err := killCmd.CombinedOutput(); err != nil {
				s.logger.Error("Failed to kill process %s (PID %d): %v, output: %s",
					processId, cmd.Process.Pid, err, string(output))
			} else {
				s.logger.Info("Successfully killed process %s (PID %d)", processId, cmd.Process.Pid)
			}
		} else {
			if err := cmd.Process.Kill(); err != nil {
				s.logger.Error("Failed to kill process %s (PID %d): %v", processId, cmd.Process.Pid, err)
			} else {
				s.logger.Info("Successfully killed process %s (PID %d)", processId, cmd.Process.Pid)
			}
		}

		s.mu.Lock()
		delete(s.processes, processId)
		s.mu.Unlock()

		statusFilename := strings.Replace(processId, ".bat", "_status.txt", 1)
		statusPath := filepath.Join(s.config.LogsDir, statusFilename)

		if _, err := os.Stat(statusPath); err == nil {
			if err := os.WriteFile(statusPath, []byte("stopped"), 0644); err != nil {
				s.logger.Error("Failed to update status file for %s: %v", processId, err)
			} else {
				s.logger.Debug("Updated status file for %s to 'stopped'", processId)
			}
		}

		eventData = EventData{
			ProcessId: processId,
			Message:   "Process stopped successfully",
			Timestamp: time.Now(),
			Success:   true,
		}

		if s.app != nil {
			s.app.EmitEvent("process_stopped", eventData)
		}
		return true
	}

	return s.stopProcessFromPidFile(processId)
}

func (s *ProcessManagerService) stopProcessFromPidFile(processId string) bool {
	pidFilename := strings.Replace(processId, ".bat", "_pid.txt", 1)
	pidPath := filepath.Join(s.config.LogsDir, pidFilename)
	statusFilename := strings.Replace(processId, ".bat", "_status.txt", 1)
	statusPath := filepath.Join(s.config.LogsDir, statusFilename)

	s.logger.Info("Attempting to stop process %s from PID file", processId)

	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		s.logger.Warn("PID file not found for %s", processId)
		return false
	}

	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		s.logger.Error("Failed to read PID file for %s: %v", processId, err)
		return false
	}

	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		s.logger.Error("Invalid PID format for %s: %s", processId, pidStr)
		return false
	}

	s.logger.Info("Stopping process %s with PID %d from PID file", processId, pid)

	eventData := EventData{
		ProcessId: processId,
		Message:   "Stopping process from PID file",
		PID:       pid,
		Timestamp: time.Now(),
		Success:   true,
	}

	if s.app != nil {
		s.app.EmitEvent("process_stopping", eventData)
	}

	if runtime.GOOS == "windows" {
		killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(pid))
		killCmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}

		if output, err := killCmd.CombinedOutput(); err != nil {
			s.logger.Error("Failed to kill process %s (PID %d): %v, output: %s",
				processId, pid, err, string(output))
		} else {
			s.logger.Info("Successfully killed process %s (PID %d)", processId, pid)
		}
	} else {
		if proc, err := os.FindProcess(pid); err == nil {
			if err := proc.Kill(); err != nil {
				s.logger.Error("Failed to kill process %s (PID %d): %v", processId, pid, err)
			} else {
				s.logger.Info("Successfully killed process %s (PID %d)", processId, pid)
			}
		}
	}

	if _, err := os.Stat(statusPath); err == nil {
		if err := os.WriteFile(statusPath, []byte("stopped"), 0644); err != nil {
			s.logger.Error("Failed to update status file for %s: %v", processId, err)
		} else {
			s.logger.Debug("Updated status file for %s to 'stopped'", processId)
		}
	}

	if err := os.Remove(pidPath); err != nil {
		s.logger.Error("Failed to remove PID file for %s: %v", processId, err)
	} else {
		s.logger.Debug("Removed PID file for %s", processId)
	}

	eventData = EventData{
		ProcessId: processId,
		Message:   "Process stopped successfully",
		Timestamp: time.Now(),
		Success:   true,
	}

	if s.app != nil {
		s.app.EmitEvent("process_stopped", eventData)
	}
	return true
}

func (s *ProcessManagerService) RunBatFile(batFilename string) error {
	processId := batFilename

	if s.IsProcessRunning(processId) {
		s.logger.Warn("Process %s is already running", processId)

		eventData := EventData{
			ProcessId: processId,
			Message:   "Process is already running",
			Timestamp: time.Now(),
			Success:   false,
		}

		if s.app != nil {
			s.app.EmitEvent("process_error", eventData)
		}
		return nil
	}

	// Pastikan direktori services dan logs ada
	if err := os.MkdirAll(s.config.LogsDir, 0755); err != nil {
		s.logger.Error("Failed to create logs directory: %v", err)
		return err
	}

	// Path lengkap ke file batch
	binPath := filepath.Join(s.config.ServicesDir, batFilename)
	s.logger.Info("Running batch file: %s", binPath)

	// Verify file exists
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		s.logger.Error("Batch file not found: %s", binPath)
		eventData := EventData{
			ProcessId: processId,
			Message:   "Batch file not found: " + binPath,
			Timestamp: time.Now(),
			Success:   false,
		}

		if s.app != nil {
			s.app.EmitEvent("process_error", eventData)
		}
		return fmt.Errorf("batch file not found: %s", binPath)
	}

	if runtime.GOOS != "windows" {
		s.logger.Error("Batch files can only be run on Windows")
		eventData := EventData{
			ProcessId: processId,
			Message:   "File .bat hanya dapat dijalankan di Windows",
			Timestamp: time.Now(),
			Success:   false,
		}

		if s.app != nil {
			s.app.EmitEvent("process_error", eventData)
		}
		return nil
	}

	// Log environment info for debugging
	s.logger.Debug("Environment info:")
	s.logger.Debug("  Working directory: %s", s.config.ServicesDir)
	s.logger.Debug("  Batch file full path: %s", binPath)
	s.logger.Debug("  Current directory: %s", getCurrentDir())
	s.logger.Debug("  BinDir from config: %s", s.cfg.BinDir)

	cmd := exec.Command("cmd", "/C", "call", binPath)
	cmd.Dir = filepath.Dir(binPath) // Set working directory ke lokasi file batch

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	// Log command yang akan dijalankan untuk debugging
	s.logger.Info("Running command: %s in directory: %s", cmd.String(), cmd.Dir)

	eventData := EventData{
		ProcessId: processId,
		Message:   "Process started",
		Timestamp: time.Now(),
		Success:   true,
		Data:      batFilename,
	}

	if s.app != nil {
		s.app.EmitEvent("process_started", eventData)
	}

	if err := cmd.Start(); err != nil {
		s.logger.Error("Failed to start process %s: %v", processId, err)

		eventData := EventData{
			ProcessId: processId,
			Message:   "Error: " + err.Error(),
			Timestamp: time.Now(),
			Success:   false,
		}

		if s.app != nil {
			s.app.EmitEvent("process_error", eventData)
		}
		return err
	}

	s.logger.Info("Process %s started with PID %d", processId, cmd.Process.Pid)

	s.mu.Lock()
	s.processes[processId] = cmd
	s.mu.Unlock()

	// Buat goroutine untuk menunggu proses selesai
	go func() {
		err := cmd.Wait()

		s.mu.Lock()
		delete(s.processes, processId)
		s.mu.Unlock()

		var eventData EventData
		if err != nil {
			s.logger.Error("Process %s failed: %v", processId, err)
			eventData = EventData{
				ProcessId: processId,
				Message:   "Error: " + err.Error(),
				Timestamp: time.Now(),
				Success:   false,
			}
		} else {
			s.logger.Info("Process %s completed successfully", processId)
			eventData = EventData{
				ProcessId: processId,
				Message:   "Proses " + batFilename + " selesai",
				Timestamp: time.Now(),
				Success:   true,
			}
		}

		if s.app != nil {
			s.app.EmitEvent("process_completed", eventData)
		}

		// Pastikan status file diperbarui saat proses selesai
		s.UpdateProcessStatusOnMissing(processId)
	}()

	return nil
}

// Helper function to get current directory
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "<error getting current dir>"
	}
	return dir
}

func (s *ProcessManagerService) GetDetailedProcessStatus(processId string) string {
	statusFileName := strings.Replace(processId, ".bat", "_status.txt", 1)
	statusPath := filepath.Join(s.config.LogsDir, statusFileName)

	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		if s.IsProcessRunning(processId) {
			return "Running"
		}
		return "Idle"
	}

	statusBytes, err := os.ReadFile(statusPath)
	if err != nil {
		s.logger.Error("Failed to read status file for %s: %v", processId, err)
		if s.IsProcessRunning(processId) {
			return "Running"
		}
		return "Unknown"
	}

	status := strings.TrimSpace(string(statusBytes))
	s.logger.Debug("Status for %s: %s", processId, status)

	switch status {
	case "initializing":
		return "Initializing"
	case "loading":
		return "Loading"
	case "running":
		return "Running"
	case "stopped":
		return "Stopped"
	case "error":
		return "Error"
	default:
		if s.IsProcessRunning(processId) {
			return "Running"
		}
		return status
	}
}

func (s *ProcessManagerService) GetStatusClass(processId string) string {
	status := s.GetDetailedProcessStatus(processId)

	switch status {
	case "Initializing":
		return "bg-blue-100 text-blue-800 hover:bg-blue-100"
	case "Loading":
		return "bg-yellow-100 text-yellow-800 hover:bg-yellow-100"
	case "Running":
		return "bg-green-100 text-green-800 hover:bg-green-100"
	case "Stopped":
		return "bg-gray-100 text-gray-800 hover:bg-gray-100"
	case "Error":
		return "bg-red-100 text-red-800 hover:bg-red-100"
	case "Idle":
		return "bg-gray-100 text-gray-800 hover:bg-gray-100"
	default:
		return "bg-gray-100 text-gray-800 hover:bg-gray-100"
	}
}

func (s *ProcessManagerService) VerifyProcessStatusConsistency(processId string) {
	pidFilename := strings.Replace(processId, ".bat", "_pid.txt", 1)
	pidPath := filepath.Join(s.config.LogsDir, pidFilename)
	statusFilename := strings.Replace(processId, ".bat", "_status.txt", 1)
	statusPath := filepath.Join(s.config.LogsDir, statusFilename)

	s.logger.Debug("Verifying status consistency for %s", processId)

	pidExists := false
	var pid int
	if _, err := os.Stat(pidPath); err == nil {
		pidExists = true
		pidBytes, err := os.ReadFile(pidPath)
		if err == nil {
			pidStr := strings.TrimSpace(string(pidBytes))
			pid, _ = strconv.Atoi(pidStr)
		}
	}

	statusExists := false
	var status string
	if _, err := os.Stat(statusPath); err == nil {
		statusExists = true
		statusBytes, err := os.ReadFile(statusPath)
		if err == nil {
			status = strings.TrimSpace(string(statusBytes))
		}
	}

	processRunning := false
	if pidExists && pid > 0 {
		processRunning = s.isProcessRunningByPid(pid)
	}

	s.logger.Debug("Status check for %s: pidExists=%v, pid=%d, statusExists=%v, status=%s, processRunning=%v",
		processId, pidExists, pid, statusExists, status, processRunning)

	if statusExists && !pidExists && status != "stopped" && status != "error" {
		s.logger.Info("Updating status to 'stopped' for %s (no PID file)", processId)
		os.WriteFile(statusPath, []byte("stopped"), 0644)
	} else if statusExists && pidExists && !processRunning && status != "stopped" && status != "error" {
		s.logger.Info("Updating status to 'stopped' for %s (PID %d not running)", processId, pid)
		os.WriteFile(statusPath, []byte("stopped"), 0644)
		os.Remove(pidPath)
	} else if statusExists && pidExists && processRunning && status == "stopped" {
		s.logger.Info("Updating status to 'running' for %s (PID %d is running)", processId, pid)
		os.WriteFile(statusPath, []byte("running"), 0644)
	}
}

func (s *ProcessManagerService) VerifyAllProcessStatusConsistency() {
	s.logger.Debug("Verifying all process status consistency")
	s.VerifyProcessStatusConsistency("people_counter.bat")
	s.VerifyProcessStatusConsistency("sync_manager.bat")
}

// ForceRemoveOrphanedStatusFiles menghapus file status yang tidak konsisten
func (s *ProcessManagerService) ForceRemoveOrphanedStatusFiles() {
	s.logger.Info("Forcing removal of orphaned status files")

	pidFiles, err := filepath.Glob(filepath.Join(s.config.LogsDir, "*_pid.txt"))
	if err != nil {
		s.logger.Error("Failed to glob PID files: %v", err)
		return
	}

	statusFiles, err := filepath.Glob(filepath.Join(s.config.LogsDir, "*_status.txt"))
	if err != nil {
		s.logger.Error("Failed to glob status files: %v", err)
		return
	}

	s.logger.Info("Found %d PID files and %d status files", len(pidFiles), len(statusFiles))

	// Map untuk melacak PID yang valid
	validPids := make(map[string]bool)

	// Periksa file PID dan simpan yang memiliki proses aktif
	for _, pidPath := range pidFiles {
		pidBytes, err := os.ReadFile(pidPath)
		if err != nil {
			s.logger.Error("Failed to read PID file %s: %v", pidPath, err)
			continue
		}

		pidStr := strings.TrimSpace(string(pidBytes))
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			// Hapus file PID yang tidak valid
			s.logger.Warn("Invalid PID in file %s: %s", pidPath, pidStr)
			os.Remove(pidPath)
			continue
		}

		// Periksa apakah proses masih berjalan
		if s.isProcessRunningByPid(pid) {
			// Simpan PID yang valid
			baseName := filepath.Base(pidPath)
			processId := strings.TrimSuffix(baseName, "_pid.txt")
			validPids[processId] = true
			s.logger.Debug("Found valid PID %d for process %s", pid, processId)
		} else {
			// Hapus file PID untuk proses yang tidak berjalan
			s.logger.Info("Removing stale PID file %s (PID %d not running)", pidPath, pid)
			os.Remove(pidPath)
		}
	}

	// Periksa file status dan perbarui yang tidak konsisten
	for _, statusPath := range statusFiles {
		baseName := filepath.Base(statusPath)
		processId := strings.TrimSuffix(baseName, "_status.txt")

		statusBytes, err := os.ReadFile(statusPath)
		if err != nil {
			s.logger.Error("Failed to read status file %s: %v", statusPath, err)
			continue
		}
		status := strings.TrimSpace(string(statusBytes))

		// Jika status bukan "stopped" atau "error" tapi tidak ada PID yang valid
		if status != "stopped" && status != "error" && !validPids[processId] {
			// Perbarui status menjadi "stopped"
			s.logger.Info("Updating status to 'stopped' for orphaned process %s", processId)
			os.WriteFile(statusPath, []byte("stopped"), 0644)
		}
	}
}

func (s *ProcessManagerService) RestartProcess(processId string) bool {
	s.logger.Info("Restarting process %s", processId)

	eventData := EventData{
		ProcessId: processId,
		Message:   "Restarting process",
		Timestamp: time.Now(),
		Success:   true,
	}

	if s.app != nil {
		s.app.EmitEvent("process_restarting", eventData)
	}

	// First stop the process
	stopped := s.StopProcess(processId)
	s.logger.Info("Process %s stop result: %v", processId, stopped)

	// Wait a bit to ensure the process is fully terminated
	time.Sleep(2 * time.Second)

	// Start the process again
	err := s.RunBatFile(processId)
	if err != nil {
		s.logger.Error("Failed to restart process %s: %v", processId, err)

		eventData := EventData{
			ProcessId: processId,
			Message:   "Failed to restart process: " + err.Error(),
			Timestamp: time.Now(),
			Success:   false,
		}

		if s.app != nil {
			s.app.EmitEvent("process_error", eventData)
		}
		return false
	}

	s.logger.Info("Process %s restarted successfully", processId)
	return stopped
}
