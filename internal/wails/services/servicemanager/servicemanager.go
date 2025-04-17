package servicemanager

import (
	"fmt"
	"jarvist/internal/common/config"
	"jarvist/pkg/logger"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

// ServiceManager handles interactions with the Windows system service
type ServiceManager struct {
	config        *config.Config
	logger        *logger.ContextLogger
	serviceBinary string
	serviceName   string
}

func New(config *config.Config, logger *logger.Logger) *ServiceManager {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)
	serviceBinary := filepath.Join(execDir, "sync-manager.exe")

	return &ServiceManager{
		config:        config,
		logger:        logger.WithComponent("service-manager"),
		serviceBinary: serviceBinary,
		serviceName:   "jarvist-sync",
	}
}

func (s *ServiceManager) runCommand(args ...string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("service management is only supported on Windows")
	}

	cmd := exec.Command(s.serviceBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (s *ServiceManager) InstallService() (string, error) {
	s.logger.Info("Installing service...")
	output, err := s.runCommand("--install")
	if err != nil {
		return output, fmt.Errorf("failed to install service: %w", err)
	}

	time.Sleep(1 * time.Second)

	s.logger.Info("Running service in service mode...")
	serviceOutput, serviceErr := s.runCommand("--service")
	if serviceErr != nil {
		return output + "\n" + serviceOutput, fmt.Errorf("service installed but failed to run in service mode: %w", serviceErr)
	}

	return output + "\n" + serviceOutput, nil
}

func (s *ServiceManager) UninstallService() (string, error) {
	s.logger.Info("Uninstalling service...")
	return s.runCommand("--uninstall")
}

func (s *ServiceManager) StartService() (string, error) {
	s.logger.Info("Starting service...")
	return s.runCommand("--start")
}

func (s *ServiceManager) StopService() (string, error) {
	s.logger.Info("Stopping service...")
	return s.runCommand("--stop")
}

func (s *ServiceManager) GetServiceStatus() (string, error) {
	s.logger.Info("Getting service status...")

	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("service management is only supported on Windows")
	}

	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	service, err := m.OpenService(s.serviceName)
	if err != nil {
		if err == windows.ERROR_SERVICE_DOES_NOT_EXIST {
			return "Not installed", nil
		}
		return "", fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return "", fmt.Errorf("failed to query service: %w", err)
	}

	switch status.State {
	case windows.SERVICE_STOPPED:
		return "Stopped", nil
	case windows.SERVICE_START_PENDING:
		return "Starting", nil
	case windows.SERVICE_STOP_PENDING:
		return "Stopping", nil
	case windows.SERVICE_RUNNING:
		return "Running", nil
	case windows.SERVICE_CONTINUE_PENDING:
		return "Continue Pending", nil
	case windows.SERVICE_PAUSE_PENDING:
		return "Pause Pending", nil
	case windows.SERVICE_PAUSED:
		return "Paused", nil
	default:
		return "Unknown", nil
	}
}

func (s *ServiceManager) IsServiceInstalled() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, fmt.Errorf("service management is only supported on Windows")
	}

	m, err := mgr.Connect()
	if err != nil {
		return false, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	service, err := m.OpenService(s.serviceName)
	if err != nil {
		if err == windows.ERROR_SERVICE_DOES_NOT_EXIST {
			return false, nil
		}
		return false, fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()

	return true, nil
}

func (s *ServiceManager) IsServiceRunning() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, fmt.Errorf("service management is only supported on Windows")
	}

	m, err := mgr.Connect()
	if err != nil {
		return false, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	service, err := m.OpenService(s.serviceName)
	if err != nil {
		if err == windows.ERROR_SERVICE_DOES_NOT_EXIST {
			return false, nil
		}
		return false, fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return false, fmt.Errorf("failed to query service: %w", err)
	}

	return status.State == windows.SERVICE_RUNNING, nil
}

func (s *ServiceManager) GetServiceDetails() (map[string]interface{}, error) {
	s.logger.Info("Getting service details...")

	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("service management is only supported on Windows")
	}

	details := map[string]interface{}{
		"installed": false,
		"name":      s.serviceName,
	}

	m, err := mgr.Connect()
	if err != nil {
		return details, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	service, err := m.OpenService(s.serviceName)
	if err != nil {
		if err == windows.ERROR_SERVICE_DOES_NOT_EXIST {
			return details, nil // Return basic details without error
		}
		return details, fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()

	// Service is installed
	details["installed"] = true

	// Get service configuration
	config, err := service.Config()
	if err == nil {
		details["displayName"] = config.DisplayName
		details["description"] = config.Description
		details["binaryPath"] = config.BinaryPathName

		switch config.StartType {
		case mgr.StartAutomatic:
			details["startType"] = "Automatic"
		case mgr.StartManual:
			details["startType"] = "Manual"
		case mgr.StartDisabled:
			details["startType"] = "Disabled"
		default:
			details["startType"] = "Unknown"
		}
	}

	// Get service status
	status, err := service.Query()
	if err == nil {
		switch status.State {
		case windows.SERVICE_STOPPED:
			details["status"] = "Stopped"
		case windows.SERVICE_START_PENDING:
			details["status"] = "Starting"
		case windows.SERVICE_STOP_PENDING:
			details["status"] = "Stopping"
		case windows.SERVICE_RUNNING:
			details["status"] = "Running"
		case windows.SERVICE_CONTINUE_PENDING:
			details["status"] = "Continue Pending"
		case windows.SERVICE_PAUSE_PENDING:
			details["status"] = "Pause Pending"
		case windows.SERVICE_PAUSED:
			details["status"] = "Paused"
		default:
			details["status"] = "Unknown"
		}

		details["pid"] = status.ProcessId
		details["exitCode"] = status.Win32ExitCode
	} else {
		details["status"] = "Unknown"
	}

	return details, nil
}

// RestartService restarts the Windows service
func (s *ServiceManager) RestartService() (string, error) {
	s.logger.Info("Restarting service...")
	_, err := s.StopService()
	if err != nil {
		return "", fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait a moment for the service to fully stop
	time.Sleep(2 * time.Second)

	return s.StartService()
}

func (s *ServiceManager) CheckAndInstallService() (string, error) {
	s.logger.Info("Checking and installing service if needed...")

	installed, err := s.IsServiceInstalled()
	if err != nil {
		return "", fmt.Errorf("failed to check installation status: %w", err)
	}

	if installed {
		return "Service is already installed", nil
	}

	s.logger.Info("Service not installed, installing...")
	return s.InstallService()
}

func (s *ServiceManager) EnsureServiceRunning() (string, error) {
	s.logger.Info("Ensuring service is installed and running...")

	installed, err := s.IsServiceInstalled()
	if err != nil {
		return "", fmt.Errorf("failed to check installation status: %w", err)
	}

	if !installed {
		s.logger.Info("Service not installed, installing...")
		return s.InstallService()
	}

	running, err := s.IsServiceRunning()
	if err != nil {
		return "", fmt.Errorf("failed to check service status: %w", err)
	}

	if !running {
		s.logger.Info("Service not running, starting...")
		return s.StartService()
	}

	return "Service is already installed and running", nil
}
