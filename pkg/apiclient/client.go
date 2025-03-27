// Package apiclient menyediakan client untuk berkomunikasi dengan API dengan dukungan retry otomatis
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// Client mengelola komunikasi dengan API dan retry otomatis
type Client struct {
	baseURL    string
	retryMutex sync.Mutex
	retryQueue map[string]RetryItem
	tenantIDFn func() string
	stopped    bool
	stopCh     chan struct{}
}

// RequestData mengemas data untuk request API
type RequestData interface{}

// RetryItem merepresentasikan item yang ada dalam antrian retry
type RetryItem struct {
	Endpoint string      // Endpoint API yang dituju
	Method   string      // HTTP method (POST, GET, etc)
	Data     RequestData // Data yang akan dikirim
	ID       string      // ID unik untuk item ini
}

// Response adalah struktur untuk menampung respons dari API
type Response struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// New membuat instance baru dari Client
func New(baseURL string, tenantIDFn func() string) *Client {
	client := &Client{
		baseURL:    baseURL,
		retryQueue: make(map[string]RetryItem),
		tenantIDFn: tenantIDFn,
		stopCh:     make(chan struct{}),
	}

	// Start background retry worker
	go client.startRetryWorker()

	return client
}

// SendRequest mengirim request ke API dengan endpoint tertentu
func (c *Client) SendRequest(method, endpoint string, data RequestData) (*Response, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, endpoint)

	var req *http.Request
	var err error

	if data != nil && method != "GET" {
		// Marshal data menjadi JSON jika ada dan bukan GET request
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("error marshalling request data: %w", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
	} else {
		// Tanpa body untuk GET request
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set header
	req.Header.Set("Content-Type", "application/json")

	// Get tenant ID if function is provided
	if c.tenantIDFn != nil {
		tenantID := c.tenantIDFn()
		if tenantID != "" {
			req.Header.Set("X-Tenant-ID", tenantID)
		}
	}

	// Kirim request dengan timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Baca respons
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Parse respons JSON
	var apiResp Response
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Periksa status respons
	if !apiResp.Success {
		return &apiResp, fmt.Errorf("API returned error: %s (code: %d)", apiResp.Message, apiResp.Code)
	}

	return &apiResp, nil
}

// AddToRetryQueue menambahkan item ke antrian retry dengan informasi lengkap
func (c *Client) AddToRetryQueue(id, method, endpoint string, data RequestData) {
	c.retryMutex.Lock()
	defer c.retryMutex.Unlock()

	c.retryQueue[id] = RetryItem{
		Endpoint: endpoint,
		Method:   method,
		Data:     data,
		ID:       id,
	}

	log.Printf("Added request to %s with ID %s to retry queue", endpoint, id)
}

// startRetryWorker memulai worker yang akan mencoba kembali mengirim data yang gagal
func (c *Client) startRetryWorker() {
	ticker := time.NewTicker(1 * time.Minute) // Coba ulang setiap 1 menit
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.processRetryQueue()
		case <-c.stopCh:
			log.Println("Retry worker stopped")
			return
		}
	}
}

// processRetryQueue memproses antrian retry
func (c *Client) processRetryQueue() {
	c.retryMutex.Lock()
	// Buat salinan queue untuk diproses
	queue := make(map[string]RetryItem)
	for k, v := range c.retryQueue {
		queue[k] = v
	}
	c.retryMutex.Unlock()

	if len(queue) == 0 {
		return
	}

	log.Printf("Processing retry queue with %d items", len(queue))

	// Proses setiap item dalam antrian
	for key, item := range queue {
		log.Printf("Retrying request to %s with ID %s", item.Endpoint, item.ID)

		_, err := c.SendRequest(item.Method, item.Endpoint, item.Data)
		if err == nil {
			// Jika berhasil, hapus dari antrian
			c.retryMutex.Lock()
			delete(c.retryQueue, key)
			c.retryMutex.Unlock()
			log.Printf("Successfully processed request to %s with ID %s from retry queue", item.Endpoint, item.ID)
		} else {
			log.Printf("Retry failed for request to %s with ID %s: %v. Will try again later.", item.Endpoint, item.ID, err)
		}

		// Tunggu sedikit antara request untuk menghindari flood
		time.Sleep(2 * time.Second)
	}
}

// Stop menghentikan worker retry dan membersihkan resource
func (c *Client) Stop() {
	if !c.stopped {
		close(c.stopCh)
		c.stopped = true
	}
}

// QueueSize mengembalikan jumlah item dalam antrian retry
func (c *Client) QueueSize() int {
	c.retryMutex.Lock()
	defer c.retryMutex.Unlock()
	return len(c.retryQueue)
}

// ClearQueue membersihkan antrian retry
func (c *Client) ClearQueue() {
	c.retryMutex.Lock()
	defer c.retryMutex.Unlock()
	c.retryQueue = make(map[string]RetryItem)
	log.Println("Retry queue cleared")
}
