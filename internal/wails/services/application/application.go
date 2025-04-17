package application

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type ApplicationService struct {
	app        *application.App // Store the application instance
	stopSignal chan struct{}
}

// Constructor that takes the application instance
func New(app *application.App) *ApplicationService {
	return &ApplicationService{
		app:        app,
		stopSignal: make(chan struct{}),
	}
}

func (s *ApplicationService) OnStartup(ctx context.Context, options application.ServiceOptions) error {
	return nil
}

func (s *ApplicationService) OnShutdown(ctx context.Context, options application.ServiceOptions) error {
	close(s.stopSignal)
	time.Sleep(100 * time.Millisecond)
	os.Exit(0)
	return nil
}

func (s *ApplicationService) InitService(app *application.App) {
	s.app = app
}

func (s *ApplicationService) Restart() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Windows-specific approach
	cmd := exec.Command("cmd", "/C", "start", "", execPath)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start new instance: %w", err)
	}

	// Give the new process time to start
	time.Sleep(500 * time.Millisecond)

	// Use the stored app instance to quit
	s.app.Quit()

	return nil
}

func (s *ApplicationService) CloseApp() {
	s.app.Quit()
}
