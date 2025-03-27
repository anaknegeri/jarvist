package mjpeg

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

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type MJPEGStreamConfig struct {
	InputFile string `json:"inputFile"`
	FrameRate int    `json:"frameRate"`
	Port      int    `json:"port"`
	Quality   int    `json:"quality"` // 1-100, semakin tinggi semakin baik
	IsRunning bool   `json:"-"`
}

// MJPEGServer menangani streaming MJPEG langsung dari Go
type MJPEGServer struct {
	ctx        context.Context
	config     MJPEGStreamConfig
	server     *http.Server
	mu         sync.Mutex
	clients    map[chan []byte]bool
	clientsMu  sync.Mutex
	stopChan   chan struct{}
	frameCache []byte // Cache frame terbaru
}

// NewMJPEGServer membuat instance baru server MJPEG
func NewMJPEGServer() *MJPEGServer {
	return &MJPEGServer{
		config: MJPEGStreamConfig{
			InputFile: "bin/services/image/image.jpg",
			FrameRate: 10,
			Port:      8088,
			Quality:   75,
			IsRunning: false,
		},
		clients: make(map[chan []byte]bool),
	}
}

// SetContext menyimpan context Wails
func (s *MJPEGServer) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// UpdateConfig memperbarui konfigurasi server
func (s *MJPEGServer) UpdateConfig(config MJPEGStreamConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	isRunning := s.config.IsRunning
	s.config = config
	s.config.IsRunning = isRunning
}

// GetConfig mengembalikan konfigurasi server saat ini
func (s *MJPEGServer) GetConfig() MJPEGStreamConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config
}

// GetStreamURL mengembalikan URL untuk mengakses stream
func (s *MJPEGServer) GetStreamURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("http://localhost:%d/stream", s.config.Port)
}

// registerClient menambahkan klien baru yang akan menerima frame
func (s *MJPEGServer) registerClient(client chan []byte) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.clients[client] = true

	if s.ctx != nil {
		runtime.LogInfo(s.ctx, "New client connected")
	}
}

// unregisterClient menghapus klien ketika koneksi terputus
func (s *MJPEGServer) unregisterClient(client chan []byte) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if _, ok := s.clients[client]; ok {
		delete(s.clients, client)
		close(client)

		if s.ctx != nil {
			runtime.LogInfo(s.ctx, "Client disconnected")
		}
	}
}

// broadcastFrame mengirim frame ke semua klien yang terhubung
func (s *MJPEGServer) broadcastFrame(frameData []byte) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	// Update cache frame
	s.frameCache = frameData

	// Kirim ke semua klien
	for client := range s.clients {
		// Non-blocking send untuk mencegah klien lambat menghambat server
		select {
		case client <- frameData:
			// Frame berhasil dikirim
		default:
			// Skip klien yang lambat (buffer channel penuh)
		}
	}
}

// handleStream adalah handler untuk request MJPEG stream
func (s *MJPEGServer) handleStream(w http.ResponseWriter, r *http.Request) {
	// MJPEG stream memerlukan header khusus
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "close")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	// Channel untuk menerima frame
	frameChan := make(chan []byte, 10) // Buffer beberapa frame

	// Register klien baru
	s.registerClient(frameChan)
	defer s.unregisterClient(frameChan)

	// Jika ada frame di cache, kirim segera
	if s.frameCache != nil {
		s.sendMJPEGFrame(w, s.frameCache)
	}

	// Flush header ke client
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Loop untuk mengirim frame ke browser
	for {
		select {
		case frame := <-frameChan:
			err := s.sendMJPEGFrame(w, frame)
			if err != nil {
				if s.ctx != nil {
					runtime.LogInfo(s.ctx, fmt.Sprintf("Error sending frame: %v", err))
				}
				return
			}

			// Flush frame ke client
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case <-r.Context().Done():
			// Client terputus
			return

		case <-s.stopChan:
			// Server dihentikan
			return
		}
	}
}

// sendMJPEGFrame mengirim sebuah frame sebagai MJPEG ke client
func (s *MJPEGServer) sendMJPEGFrame(w io.Writer, frame []byte) error {
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

// handleOptions merespon pre-flight CORS requests
func (s *MJPEGServer) handleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusOK)
}

// frameGenerator membaca image input, convert ke JPEG dan broadcastnya
func (s *MJPEGServer) frameGenerator() {
	s.mu.Lock()
	inputFile := s.config.InputFile
	frameRate := s.config.FrameRate
	quality := s.config.Quality
	s.mu.Unlock()

	interval := time.Second / time.Duration(frameRate)

	for {
		select {
		case <-s.stopChan:
			return
		default:
			// Cek apakah file ada
			if _, err := os.Stat(inputFile); os.IsNotExist(err) {
				if s.ctx != nil {
					runtime.LogError(s.ctx, fmt.Sprintf("Image file not found: %s", inputFile))
				}
				time.Sleep(interval)
				continue
			}

			// Baca file
			file, err := os.Open(inputFile)
			if err != nil {
				if s.ctx != nil {
					runtime.LogError(s.ctx, fmt.Sprintf("Error opening image: %v", err))
				}
				time.Sleep(interval)
				continue
			}

			// Decode image
			img, _, err := image.Decode(file)
			file.Close()
			if err != nil {
				if s.ctx != nil {
					runtime.LogError(s.ctx, fmt.Sprintf("Error decoding image: %v", err))
				}
				time.Sleep(interval)
				continue
			}

			// Encode ke JPEG dengan kualitas yang dikonfigurasi
			buf := new(bytes.Buffer)
			err = jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
			if err != nil {
				if s.ctx != nil {
					runtime.LogError(s.ctx, fmt.Sprintf("Error encoding JPEG: %v", err))
				}
				time.Sleep(interval)
				continue
			}

			// Broadcast frame
			s.broadcastFrame(buf.Bytes())

			// Tunggu untuk frame berikutnya berdasarkan framerate
			time.Sleep(interval)
		}
	}
}

// StartStream memulai server MJPEG
func (s *MJPEGServer) StartStream() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.IsRunning {
		return "Stream is already running", nil
	}

	// Validate input file exists
	if _, err := os.Stat(s.config.InputFile); os.IsNotExist(err) {
		// If the input file doesn't exist at the absolute path, check relative to executable
		execPath, _ := os.Executable()
		appDir := filepath.Dir(execPath)
		relativePath := filepath.Join(appDir, s.config.InputFile)

		if _, err := os.Stat(relativePath); os.IsNotExist(err) {
			return "", fmt.Errorf("input file not found: %s", s.config.InputFile)
		} else {
			s.config.InputFile = relativePath
		}
	}

	// Init channels
	s.stopChan = make(chan struct{})
	s.clients = make(map[chan []byte]bool)

	// Setup server mux
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/stream/", s.handleStream) // Handle /stream/ juga
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Root handler untuk menampilkan HTML sederhana dengan embed video stream
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

	// Buat HTTP server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: mux,
	}

	// Start server di goroutine terpisah
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v\n", err)
			if s.ctx != nil {
				runtime.LogError(s.ctx, fmt.Sprintf("HTTP server error: %v", err))
			}
		}
	}()

	// Start frame generator di goroutine terpisah
	go s.frameGenerator()

	s.config.IsRunning = true

	if s.ctx != nil {
		runtime.LogInfo(s.ctx, fmt.Sprintf("MJPEG server started on port %d", s.config.Port))
	}

	return fmt.Sprintf("MJPEG stream started at http://localhost:%d/stream", s.config.Port), nil
}

// StopStream menghentikan server MJPEG
func (s *MJPEGServer) StopStream() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.config.IsRunning {
		return "No active stream to stop", nil
	}

	// Signal all goroutines to stop
	close(s.stopChan)

	// Shutdown HTTP server
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			if s.ctx != nil {
				runtime.LogError(s.ctx, fmt.Sprintf("HTTP server shutdown error: %v", err))
			}
		}

		s.server = nil
	}

	// Clear all clients
	s.clientsMu.Lock()
	for client := range s.clients {
		close(client)
	}
	s.clients = make(map[chan []byte]bool)
	s.clientsMu.Unlock()

	s.config.IsRunning = false

	if s.ctx != nil {
		runtime.LogInfo(s.ctx, "MJPEG server stopped")
	}

	return "MJPEG stream stopped successfully", nil
}

func (s *MJPEGServer) IsStreamRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config.IsRunning
}

func (s *MJPEGServer) Cleanup() {
	s.StopStream()
}
