package main

import (
	"fmt"
	"jarvist/internal/common/buildinfo"
	baseConfig "jarvist/internal/common/config"
	"jarvist/internal/common/database"
	"jarvist/internal/syncmanager/config"
	"jarvist/internal/syncmanager/interfaces"
	"jarvist/pkg/logger"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kardianos/service"
)

const (
	defaultServiceName  = "jarvist-sync"
	defaultDisplayName  = "Jarvist Transmitter Service"
	defaultDescription  = "Service to transmit data to MQTT broker with persistence"
	serviceStartTimeout = 30 * time.Second
	serviceStopTimeout  = 30 * time.Second
	defaultAPIPort      = 8080
)

var (
	serviceConfig *config.Config
	configMutex   sync.RWMutex
)

// SetConfig sets the service configuration
func SetConfig(cfg *config.Config) {
	configMutex.Lock()
	defer configMutex.Unlock()
	serviceConfig = cfg
}

// GetServiceName returns the service name from config
func GetServiceName() string {
	if serviceConfig == nil {
		return defaultServiceName
	}
	return serviceConfig.Service.Name
}

// GetServiceDisplayName returns the service display name from config
func GetServiceDisplayName() string {
	if serviceConfig == nil {
		return defaultDisplayName
	}
	return serviceConfig.Service.DisplayName
}

// GetServiceDescription returns the service description from config
func GetServiceDescription() string {
	if serviceConfig == nil {
		return defaultDescription
	}
	return serviceConfig.Service.Description
}

// Program implements service.Program interface for kardianos/service
type Program struct {
	components []interfaces.ServiceComponent
	apiServer  interfaces.ServerController
	logger     *logger.Logger
	stopChan   chan struct{}
	stopWg     sync.WaitGroup
	exit       chan struct{}
	apiRunning bool
}

// Start starts the service
func (p *Program) Start(s service.Service) error {
	p.logger.Info("service", "Starting %s", GetServiceDisplayName())
	p.exit = make(chan struct{})
	p.stopChan = make(chan struct{})
	p.apiRunning = false

	// Load configuration
	buildInfoService := buildinfo.NewBuildInfoService()
	baseConfig, err := baseConfig.LoadConfig(buildMode, buildInfoService)
	if err != nil {
		p.logger.Error("service", "Failed to load base config: %v", err)
		return err
	}

	// Initialize database
	dbLogger := p.logger.WithComponent("database")
	err = database.SetupDatabase(baseConfig, dbLogger)
	if err != nil {
		p.logger.Error("service", "Failed to setup database: %v", err)
		return err
	}

	// Run database migrations
	err = database.RunMigrations(dbLogger)
	if err != nil {
		p.logger.Error("service", "Failed to run database migrations: %v", err)
		return err
	}

	// Start all service components with enhanced logging
	for i, component := range p.components {
		componentName := fmt.Sprintf("Component %d", i+1)
		if named, ok := component.(interface{ Name() string }); ok {
			componentName = named.Name()
		}

		p.logger.Info("service", "Starting component: %s", componentName)
		if err := component.Start(); err != nil {
			p.logger.Error("service", "Failed to start component %s: %v", componentName, err)
			return err
		}
		p.logger.Info("service", "Component started successfully: %s", componentName)
	}

	// Start API server if available
	if p.apiServer != nil {
		apiStartedChan := make(chan bool, 1)
		p.stopWg.Add(1)

		go func() {
			defer p.stopWg.Done()

			configMutex.RLock()
			port := defaultAPIPort
			if serviceConfig != nil && serviceConfig.API.Port > 0 {
				port = serviceConfig.API.Port
			}
			configMutex.RUnlock()

			p.logger.Info("service", "Starting API server on port %d", port)

			// Signal that we're attempting to start the API
			p.apiRunning = true
			apiStartedChan <- true

			if err := p.apiServer.Start(port); err != nil {
				if err.Error() != "http: Server closed" {
					p.logger.Error("service", "API server error: %v", err)
				} else {
					p.logger.Info("service", "API server shutdown gracefully")
				}
				p.apiRunning = false
			}
		}()

		// Wait for API server to start or timeout
		select {
		case <-apiStartedChan:
			p.logger.Info("service", "API server initialization complete")
		case <-time.After(5 * time.Second):
			p.logger.Warning("service", "Timeout waiting for API server to initialize")
		}
	}

	p.logger.Info("service", "Service started successfully")

	// Run forever
	go func() {
		<-p.exit
	}()

	return nil
}

// Stop stops the service
func (p *Program) Stop(s service.Service) error {
	p.logger.Info("service", "Stopping %s", GetServiceDisplayName())

	// Signal all goroutines to terminate
	close(p.stopChan)

	// First step: Stop API server if running
	if p.apiServer != nil && p.apiRunning {
		p.logger.Info("service", "Stopping API server")

		// Stop the API server with timeout handling
		apiStopDone := make(chan error, 1)

		go func() {
			err := p.apiServer.Stop()
			apiStopDone <- err
		}()

		// Wait for API server to stop with timeout
		select {
		case err := <-apiStopDone:
			if err != nil {
				p.logger.Error("service", "Error stopping API server: %v", err)
			} else {
				p.logger.Info("service", "API server stopped successfully")
			}
		case <-time.After(20 * time.Second):
			p.logger.Warning("service", "Timeout waiting for API server to stop")
		}

		p.apiRunning = false
	}

	// Second step: Stop components in reverse order
	for i := len(p.components) - 1; i >= 0; i-- {
		componentName := fmt.Sprintf("Component %d", i+1)
		if named, ok := p.components[i].(interface{ Name() string }); ok {
			componentName = named.Name()
		}

		p.logger.Info("service", "Stopping component: %s", componentName)
		if err := p.components[i].Stop(); err != nil {
			p.logger.Error("service", "Error stopping component %s: %v", componentName, err)
		} else {
			p.logger.Info("service", "Component %s stopped successfully", componentName)
		}
	}

	// Third step: Close database connection
	p.logger.Info("service", "Closing database connection")
	database.CloseDatabase()
	p.logger.Info("service", "Database connection closed")

	// Wait for any remaining goroutines to finish with timeout
	waitChan := make(chan struct{})
	go func() {
		p.stopWg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		p.logger.Info("service", "All components stopped successfully")
	case <-time.After(serviceStopTimeout):
		p.logger.Warning("service", "Timeout waiting for components to stop after %v", serviceStopTimeout)
	}

	if p.exit != nil {
		close(p.exit)
	}

	p.logger.Info("service", "Service stopped")
	return nil
}

// NewProgram creates a new service program
func NewProgram(components []interfaces.ServiceComponent, apiServer interfaces.ServerController, logger *logger.Logger) *Program {
	return &Program{
		components: components,
		apiServer:  apiServer,
		logger:     logger,
		stopChan:   make(chan struct{}),
		apiRunning: false,
	}
}

// Run runs the Windows service
func Run(components []interfaces.ServiceComponent, apiServer interfaces.ServerController, logger *logger.Logger) error {
	configMutex.RLock()
	if serviceConfig == nil {
		configMutex.RUnlock()
		return fmt.Errorf("service configuration not set")
	}
	configMutex.RUnlock()

	svcConfig := &service.Config{
		Name:        GetServiceName(),
		DisplayName: GetServiceDisplayName(),
		Description: GetServiceDescription(),
	}

	prg := NewProgram(components, apiServer, logger)
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	return svc.Run()
}

// Install installs the Windows service
func Install() error {
	svcConfig := &service.Config{
		Name:        GetServiceName(),
		DisplayName: GetServiceDisplayName(),
		Description: GetServiceDescription(),
		Executable:  getExecutablePath(),
		Arguments:   []string{"-service"},
	}

	s, err := service.New(nil, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	err = s.Install()
	if err != nil {
		return fmt.Errorf("failed to install service: %v", err)
	}

	return nil
}

// Uninstall removes the Windows service
func Uninstall() error {
	svcConfig := &service.Config{
		Name: GetServiceName(),
	}

	s, err := service.New(nil, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	// Cek status service terlebih dahulu
	status, err := s.Status()
	if err == nil && status == service.StatusRunning {
		// Hentikan service jika sedang berjalan
		fmt.Println("Stopping service before uninstall...")
		if err := s.Stop(); err != nil {
			fmt.Printf("Warning: Failed to stop service: %v\n", err)
		}

		// Beri waktu service untuk benar-benar berhenti
		fmt.Println("Waiting for service to stop...")
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			status, _ := s.Status()
			if status != service.StatusRunning {
				break
			}
		}
	}

	// Coba uninstall
	err = s.Uninstall()
	if err != nil {
		return fmt.Errorf("failed to uninstall service: %v", err)
	}

	// Pastikan tidak ada proses yang tersisa
	if runtime.GOOS == "windows" {
		// Coba temukan dan kill proses dengan nama executable yang sama
		exePath, _ := os.Executable()
		execName := filepath.Base(exePath)
		fmt.Printf("Checking for remaining %s processes...\n", execName)

		// Cari proses yang namanya sama dengan executable
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", execName), "/FO", "CSV")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}

		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), execName) {
			fmt.Println("Found remaining processes, attempting to terminate...")

			// Kill proses yang tersisa
			killCmd := exec.Command("taskkill", "/F", "/IM", execName)
			killCmd.SysProcAttr = &syscall.SysProcAttr{
				HideWindow:    true,
				CreationFlags: 0x08000000,
			}

			killCmd.Run() // Ignore errors
		}
	}

	return nil
}

// StartService starts an installed service
func StartService() error {
	// Make sure database is properly initialized before starting the service
	buildInfoService := buildinfo.NewBuildInfoService()
	baseConfig, err := baseConfig.LoadConfig(buildMode, buildInfoService)
	if err != nil {
		return fmt.Errorf("failed to load base config: %v", err)
	}

	// Create a simple logger for database setup
	logOptions := logger.DefaultOptions()
	logOptions.LogDir = baseConfig.LogDir
	appLogger := logger.New(logOptions)
	dbLogger := appLogger.WithComponent("database")

	// Setup database
	err = database.SetupDatabase(baseConfig, dbLogger)
	if err != nil {
		return fmt.Errorf("failed to setup database: %v", err)
	}

	// Run migrations
	err = database.RunMigrations(dbLogger)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	// Close database connection after setup
	database.CloseDatabase()

	// Start the service
	svcConfig := &service.Config{
		Name: GetServiceName(),
	}

	s, err := service.New(nil, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	// Check if already running
	status, _ := s.Status()
	if status == service.StatusRunning {
		return fmt.Errorf("service is already running")
	}

	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	// Wait for service to start
	fmt.Println("Waiting for service to start...")
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		status, _ := s.Status()
		if status == service.StatusRunning {
			fmt.Println("Service started successfully")
			break
		}
	}

	return nil
}

// StopService stops a running service
func StopService() error {
	svcConfig := &service.Config{
		Name: GetServiceName(),
	}

	s, err := service.New(nil, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	// Check if already stopped
	status, _ := s.Status()
	if status == service.StatusStopped {
		return fmt.Errorf("service is already stopped")
	}

	err = s.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	// Wait for service to stop
	fmt.Println("Waiting for service to stop...")
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		status, _ := s.Status()
		if status != service.StatusRunning {
			fmt.Println("Service stopped successfully")
			break
		}
	}

	return nil
}

// RestartService restarts the service
func RestartService() error {
	svcConfig := &service.Config{
		Name: GetServiceName(),
	}

	s, err := service.New(nil, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	err = s.Restart()
	if err != nil {
		return fmt.Errorf("failed to restart service: %v", err)
	}

	// Wait for service to restart
	fmt.Println("Waiting for service to restart...")
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		status, _ := s.Status()
		if status == service.StatusRunning {
			fmt.Println("Service restarted successfully")
			break
		}
	}

	return nil
}

// GetServiceStatus returns the current status of the service
func GetServiceStatus() (string, error) {
	svcConfig := &service.Config{
		Name: GetServiceName(),
	}

	s, err := service.New(nil, svcConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create service: %v", err)
	}

	status, err := s.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get service status: %v", err)
	}

	switch status {
	case service.StatusRunning:
		return "Running", nil
	case service.StatusStopped:
		return "Stopped", nil
	case service.StatusUnknown:
		return "Unknown", nil
	default:
		return fmt.Sprintf("Status: %d", int(status)), nil
	}
}

// Helper functions
func getExecutablePath() string {
	ex, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	return filepath.Clean(ex)
}
