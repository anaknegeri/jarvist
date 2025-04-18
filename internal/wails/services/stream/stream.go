package stream

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type MJPEGStreamConfig struct {
	InputFile string `json:"inputFile"`
	FrameRate int    `json:"frameRate"`
	Port      int    `json:"port"`
	Quality   int    `json:"quality"`
	IsRunning bool   `json:"-"`
}

type StreamService struct {
	app          *application.App
	config       MJPEGStreamConfig
	server       *http.Server
	mu           sync.Mutex
	clients      map[chan []byte]bool
	clientsMu    sync.Mutex
	stopChan     chan struct{}
	frameCache   []byte
	ctx          context.Context
	cancelFunc   context.CancelFunc
	lastActivity map[chan []byte]time.Time
}

func New() *StreamService {
	ctx, cancel := context.WithCancel(context.Background())

	return &StreamService{
		config: MJPEGStreamConfig{
			InputFile: "bin/services/image/image.jpg",
			FrameRate: 10,
			Port:      8088,
			Quality:   100,
			IsRunning: false,
		},
		clients:      make(map[chan []byte]bool),
		stopChan:     make(chan struct{}),
		ctx:          ctx,
		cancelFunc:   cancel,
		lastActivity: make(map[chan []byte]time.Time),
	}
}

func (s *StreamService) OnStartup(ctx context.Context, options application.ServiceOptions) error {
	return nil
}

func (s *StreamService) InitService(app *application.App) {
	s.app = app
}

func (s *StreamService) UpdateConfig(config MJPEGStreamConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	isRunning := s.config.IsRunning
	s.config = config
	s.config.IsRunning = isRunning
}

func (s *StreamService) GetConfig() MJPEGStreamConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config
}

func (s *StreamService) GetStreamURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("http://localhost:%d/stream", s.config.Port)
}

func (s *StreamService) registerClient(client chan []byte) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.clients[client] = true
	s.lastActivity[client] = time.Now()

	if s.app != nil {
		s.app.Logger.Info("New client connected")
	}
}

func (s *StreamService) unregisterClient(client chan []byte) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if _, ok := s.clients[client]; ok {
		delete(s.clients, client)
		delete(s.lastActivity, client)
		close(client)

		if s.app != nil {
			s.app.Logger.Info("Client disconnected")
		}
	}
}

func (s *StreamService) cleanupInactiveClients() {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	now := time.Now()
	timeout := 2 * time.Minute

	for client, lastActive := range s.lastActivity {
		if now.Sub(lastActive) > timeout {
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				delete(s.lastActivity, client)
				close(client)

				if s.app != nil {
					s.app.Logger.Info("Removed inactive client due to timeout")
				}
			}
		}
	}
}

func (s *StreamService) broadcastFrame(frameData []byte) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	s.frameCache = frameData

	now := time.Now()
	for client := range s.clients {
		select {
		case client <- frameData:
			s.lastActivity[client] = now
		default:
		}
	}
}

func (s *StreamService) handleStream(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
	defer cancel()

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "close")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	frameChan := make(chan []byte, 10)

	s.registerClient(frameChan)
	defer s.unregisterClient(frameChan)

	if s.frameCache != nil {
		s.sendMJPEGFrame(w, s.frameCache)
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for {
		select {
		case frame := <-frameChan:
			err := s.sendMJPEGFrame(w, frame)
			if err != nil {
				if s.app != nil {
					s.app.Logger.Info(fmt.Sprintf("Error sending frame: %v", err))
				}
				return
			}

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case <-r.Context().Done():
			return

		case <-ctx.Done():
			return

		case <-s.stopChan:
			return

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *StreamService) sendMJPEGFrame(w io.Writer, frame []byte) error {
	_, err := fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame))
	if err != nil {
		return err
	}

	_, err = w.Write(frame)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "\r\n")
	return err
}

func (s *StreamService) handleOptions(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusOK)
}

func (s *StreamService) frameGenerator() {
	s.mu.Lock()
	inputFile := s.config.InputFile
	frameRate := s.config.FrameRate
	quality := s.config.Quality
	s.mu.Unlock()

	interval := time.Second / time.Duration(frameRate)

	buf := new(bytes.Buffer)
	buf.Grow(1024 * 1024)

	cleanupTicker := time.NewTicker(30 * time.Second)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-s.ctx.Done():
			return
		case <-cleanupTicker.C:
			s.cleanupInactiveClients()
		default:
			if _, err := os.Stat(inputFile); os.IsNotExist(err) {
				if s.app != nil {
					s.app.Logger.Error(fmt.Sprintf("Image file not found: %s", inputFile))
				}
				time.Sleep(interval)
				continue
			}

			file, err := os.Open(inputFile)
			if err != nil {
				if s.app != nil {
					s.app.Logger.Error(fmt.Sprintf("Error opening image: %v", err))
				}
				time.Sleep(interval)
				continue
			}

			img, _, err := image.Decode(file)
			file.Close()
			if err != nil {
				if s.app != nil {
					s.app.Logger.Error(fmt.Sprintf("Error decoding image: %v", err))
				}
				time.Sleep(interval)
				continue
			}

			buf.Reset()

			err = jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
			if err != nil {
				if s.app != nil {
					s.app.Logger.Error(fmt.Sprintf("Error encoding JPEG: %v", err))
				}
				time.Sleep(interval)
				continue
			}

			s.broadcastFrame(buf.Bytes())

			time.Sleep(interval)
		}
	}
}

func (s *StreamService) StartStream() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.IsRunning {
		return "Stream is already running", nil
	}

	if _, err := os.Stat(s.config.InputFile); os.IsNotExist(err) {
		execPath, _ := os.Executable()
		appDir := filepath.Dir(execPath)
		relativePath := filepath.Join(appDir, s.config.InputFile)

		if _, err := os.Stat(relativePath); os.IsNotExist(err) {
			return "", fmt.Errorf("input file not found: %s", s.config.InputFile)
		} else {
			s.config.InputFile = relativePath
		}
	}

	s.stopChan = make(chan struct{})
	s.clients = make(map[chan []byte]bool)
	s.lastActivity = make(map[chan []byte]time.Time)

	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/stream/", s.handleStream)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			s.handleOptions(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>MJPEG Stream</title>
				<style>
					body { margin: 0; background: #000; height: 100vh; display: flex; align-items: center; justify-content: center; }
					img { max-width: 100%%; max-height: 100%%; }
				</style>
			</head>
			<body>
				<img src="/stream" />
			</body>
			</html>
		`)))
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v\n", err)
			if s.app != nil {
				s.app.Logger.Error(fmt.Sprintf("HTTP server error: %v", err))
			}
		}
	}()

	go s.frameGenerator()

	s.config.IsRunning = true

	if s.app != nil {
		s.app.Logger.Info(fmt.Sprintf("MJPEG server started on port %d", s.config.Port))
	}

	return fmt.Sprintf("MJPEG stream started at http://localhost:%d/stream", s.config.Port), nil
}

func (s *StreamService) StopStream() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.config.IsRunning {
		return "No active stream to stop", nil
	}

	close(s.stopChan)

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			if s.app != nil {
				s.app.Logger.Error(fmt.Sprintf("HTTP server shutdown error: %v", err))
			}
		}

		s.server = nil
	}

	s.clientsMu.Lock()
	for client := range s.clients {
		close(client)
	}
	s.clients = make(map[chan []byte]bool)
	s.lastActivity = make(map[chan []byte]time.Time)
	s.clientsMu.Unlock()

	s.config.IsRunning = false

	if s.app != nil {
		s.app.Logger.Info("MJPEG server stopped")
	}

	return "MJPEG stream stopped successfully", nil
}

func (s *StreamService) IsStreamRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config.IsRunning
}

func (s *StreamService) OnShutdown() error {
	return s.Cleanup()
}

func (s *StreamService) Cleanup() error {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	_, err := s.StopStream()
	return err
}
