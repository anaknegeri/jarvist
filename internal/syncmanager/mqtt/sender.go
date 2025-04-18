package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jarvist/internal/common/models"
	"jarvist/internal/syncmanager/config"
	"jarvist/internal/syncmanager/services/message"
	"jarvist/internal/syncmanager/services/stats"
	"jarvist/pkg/logger"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// Constants for component names
const (
	ComponentSender    = "mqtt-sender"
	ComponentMQTT      = "mqtt-client"
	ComponentWorker    = "msg-worker"
	ComponentMonitor   = "conn-monitor"
	ComponentHeartbeat = "heartbeat"
)

// Constants for connection management
const (
	ConnectionTimeout   = 10 // seconds
	ConnectionCheckFreq = 2  // seconds
	HeartbeatInterval   = 3  // seconds
)

type Sender struct {
	client            *Client
	db                *gorm.DB
	cfg               *config.Config
	logger            *logger.Logger
	running           bool
	shutdown          bool
	startTime         time.Time
	quitChan          chan struct{}
	heartbeatTime     *time.Timer
	wg                sync.WaitGroup
	mutex             sync.Mutex
	queueMutex        sync.Mutex
	messageQueue      chan models.PendingMessage
	pendingQueue      []models.PendingMessage
	messagesProcessed uint64
	processingTime    time.Duration
	processingMutex   sync.Mutex
	checkingPending   bool
	pendingMutex      sync.Mutex
	ctx               context.Context
	cancel            context.CancelFunc
	messageService    *message.MessageService
	statsService      *stats.StatsService
	workerSemaphore   chan struct{}
}

// NewSender creates a new MQTT sender
func NewSender(ctx context.Context, cfg *config.Config, db *gorm.DB, messageService *message.MessageService, statsService *stats.StatsService, logger *logger.Logger) (*Sender, error) {
	// Create MQTT client
	client, err := NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %v", err)
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	// Create sender
	t := &Sender{
		client:          client,
		db:              db,
		cfg:             cfg,
		logger:          logger,
		running:         false,
		shutdown:        false,
		messageQueue:    make(chan models.PendingMessage, 1000), // Large buffer for better performance
		pendingQueue:    make([]models.PendingMessage, 0),       // Initially empty backing queue
		quitChan:        make(chan struct{}),
		mutex:           sync.Mutex{},
		queueMutex:      sync.Mutex{},
		startTime:       time.Now(),
		checkingPending: false,
		pendingMutex:    sync.Mutex{},
		ctx:             ctxWithCancel,
		cancel:          cancel,
		messageService:  messageService,
		statsService:    statsService,
		workerSemaphore: make(chan struct{}, 5),
	}

	return t, nil
}

// Start starts the sender service
func (t *Sender) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.running {
		t.logger.Warning(ComponentSender, "Sender already running")
		return nil
	}

	t.logger.Info(ComponentSender, "Starting MQTT sender service")
	t.running = true
	t.shutdown = false

	// Reset processing status for any messages that were being processed when the system stopped
	if err := t.messageService.ResetProcessingStatus(); err != nil {
		t.logger.Warning(ComponentSender, "Failed to reset processing status: %v", err)
	}

	// Connect to MQTT broker
	if err := t.client.Connect(); err != nil {
		t.logger.Warning(ComponentMQTT, "Failed to connect to MQTT broker: %v", err)
	}

	// Start worker goroutines
	t.wg.Add(4)
	go t.messageWorker()
	go t.connectionMonitor()
	go t.heartbeatWorker()
	go t.pendingQueueWorker()

	// Check for pending messages after startup
	go t.checkPendingMessages()

	t.logger.Info(ComponentSender, "MQTT sender service started")
	return nil
}

// Stop stops the sender service
func (t *Sender) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.running {
		t.logger.Warning(ComponentSender, "Sender not running")
		return nil
	}

	t.logger.Info(ComponentSender, "Stopping MQTT sender service")
	t.running = false
	t.shutdown = true

	// Cancel the context to signal all workers
	if t.cancel != nil {
		t.cancel()
	}

	// Signal all workers to stop
	close(t.quitChan)

	// Stop heartbeat timer
	if t.heartbeatTime != nil {
		t.heartbeatTime.Stop()
	}

	// Final attempt to send pending messages
	pendingCount, err := t.messageService.CountPendingMessages()
	if err != nil {
		t.logger.Warning(ComponentSender, "Failed to count pending messages during shutdown: %v", err)
	}

	t.queueMutex.Lock()
	pendingQueueLen := len(t.pendingQueue)
	t.queueMutex.Unlock()

	totalPending := int(pendingCount) + pendingQueueLen + len(t.messageQueue)

	if totalPending > 0 && t.client.IsConnected() {
		t.logger.Info(ComponentSender, "Attempting to send %d pending messages before shutdown", totalPending)

		// Make sure we process our in-memory queue
		t.drainQueues()

		// And check database
		t.checkPendingMessages()

		// Wait for some processing to complete
		time.Sleep(5 * time.Second)
	}

	// Disconnect MQTT client
	t.client.Disconnect()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers completed normally
		t.logger.Info(ComponentSender, "All workers completed successfully")
	case <-time.After(10 * time.Second):
		t.logger.Warning(ComponentSender, "Timed out waiting for workers to finish")
	}

	t.logger.Info(ComponentSender, "MQTT sender service stopped")
	return nil
}

// drainQueues attempts to drain in-memory queues during shutdown
func (t *Sender) drainQueues() {
	t.logger.Info(ComponentSender, "Draining in-memory queues")

	// First drain the channel-based queue
	timeout := time.After(10 * time.Second)
	drained := 0

	// Loop until queue is empty or timeout
drainLoop:
	for {
		select {
		case msg, ok := <-t.messageQueue:
			if !ok {
				break drainLoop
			}

			if t.client.IsConnected() {
				if err := t.client.Publish(msg.Topic, []byte(msg.Payload)); err == nil {
					t.messageService.MarkMessageSent(msg.ID)
					drained++
				}
			}
		case <-timeout:
			t.logger.Warning(ComponentSender, "Queue drain timed out")
			break drainLoop
		default:
			// Channel is empty
			break drainLoop
		}
	}

	// Then drain the slice-based pendingQueue
	t.queueMutex.Lock()
	pendingQueueCopy := t.pendingQueue
	t.pendingQueue = nil // Clear the queue
	t.queueMutex.Unlock()

	for _, msg := range pendingQueueCopy {
		if t.client.IsConnected() {
			if err := t.client.Publish(msg.Topic, []byte(msg.Payload)); err == nil {
				t.messageService.MarkMessageSent(msg.ID)
				drained++
			}
		}
	}

	t.logger.Info(ComponentSender, "Drained %d messages from in-memory queues", drained)
}

// SendData sends data to the MQTT broker
func (t *Sender) SendData(topic string, data interface{}) (uint, error) {
	if t.shutdown {
		return 0, errors.New("sender is shutting down")
	}

	startTime := time.Now()

	// Use provided topic or default if empty
	if topic == "" {
		topic = t.cfg.MQTT.Topic
	}

	// Store message in database
	messageID, err := t.messageService.StoreMessage(topic, data, t.client.IsConnected())
	if err != nil {
		return 0, fmt.Errorf("failed to store message: %v", err)
	}

	// Immediately mark it as processing and get it for sending
	message, err := t.messageService.GetAndMarkProcessing(messageID)
	if err != nil {
		t.logger.Warning(ComponentSender, "Failed to mark message as processing: %v", err)
	} else if message != nil {
		// Queue the message for sending
		t.enqueueMessage(*message)
	}

	// Track processing time and count
	elapsed := time.Since(startTime)
	t.processingMutex.Lock()
	t.processingTime += elapsed
	atomic.AddUint64(&t.messagesProcessed, 1)
	t.processingMutex.Unlock()

	return messageID, nil
}

// enqueueMessage adds a message to the send queue with unlimited capacity
func (t *Sender) enqueueMessage(msg models.PendingMessage) {
	// Try to send to channel queue first (non-blocking)
	select {
	case t.messageQueue <- msg:
		t.logger.Debug(ComponentWorker, "Queued message ID %d for sending", msg.ID)
		return
	default:
		// Channel is full, add to backing queue
		t.queueMutex.Lock()
		t.pendingQueue = append(t.pendingQueue, msg)
		qLen := len(t.pendingQueue)
		t.queueMutex.Unlock()

		if qLen%100 == 1 { // Log at 1, 101, 201, etc to avoid excessive logging
			t.logger.Warning(ComponentWorker, "Channel queue full, added message ID %d to pendingQueue (size: %d)",
				msg.ID, qLen)
		} else {
			t.logger.Debug(ComponentWorker, "Added message ID %d to pendingQueue (size: %d)", msg.ID, qLen)
		}
	}
}

// pendingQueueWorker processes the pendingQueue when the main queue has capacity
func (t *Sender) pendingQueueWorker() {
	defer t.wg.Done()
	t.logger.Info(ComponentWorker, "Pending queue worker started")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if t.shutdown {
				return
			}

			// Check if we have pending messages to process
			t.queueMutex.Lock()
			qLen := len(t.pendingQueue)
			if qLen == 0 {
				t.queueMutex.Unlock()
				continue
			}

			// Take one message from pendingQueue
			msg := t.pendingQueue[0]
			t.pendingQueue = t.pendingQueue[1:]
			t.queueMutex.Unlock()

			// Try to put it into the main queue (with timeout)
			select {
			case t.messageQueue <- msg:
				t.logger.Debug(ComponentWorker, "Moved message ID %d from pendingQueue to main queue", msg.ID)

				// If we successfully moved one, try moving more immediately
				if qLen > 1 {
					// No need for select/case here since we just log this event
					ticker.Reset(50 * time.Millisecond)
				}
			case <-time.After(1 * time.Second):
				// If we can't move to main queue, put back at front of pendingQueue
				t.queueMutex.Lock()
				t.pendingQueue = append([]models.PendingMessage{msg}, t.pendingQueue...)
				t.queueMutex.Unlock()
				t.logger.Debug(ComponentWorker, "Failed to move message ID %d to main queue, requeued", msg.ID)
			}

		case <-t.quitChan:
			t.logger.Info(ComponentWorker, "Pending queue worker stopping")
			return
		}
	}
}

// messageWorker processes the message queue
func (t *Sender) messageWorker() {
	defer t.wg.Done()
	t.logger.Info(ComponentWorker, "Message worker started")

	for {
		select {
		case <-t.ctx.Done():
			t.logger.Info(ComponentWorker, "Message worker stopping due to context cancellation")
			return
		case msg := <-t.messageQueue:
			if t.shutdown {
				return
			}

			// Add semaphore here
			t.workerSemaphore <- struct{}{} // Acquire semaphore

			var existingMsg models.PendingMessage
			if err := t.db.Where("id = ?", msg.ID).First(&existingMsg).Error; err != nil {
				t.logger.Warning(ComponentWorker, "Failed to check message %d status: %v", msg.ID, err)
			} else if existingMsg.Sent {
				t.logger.Info(ComponentWorker, "Skipping already sent message ID %d", msg.ID)
				<-t.workerSemaphore // Release semaphore
				continue
			}

			if t.client.IsConnected() {
				if err := t.client.Publish(msg.Topic, []byte(msg.Payload)); err != nil {
					t.logger.Error(ComponentWorker, "Failed to publish message ID %d: %v", msg.ID, err)
					if !t.shutdown {
						t.enqueueMessage(msg)
					}
				} else {
					if err := t.messageService.MarkMessageSent(msg.ID); err != nil {
						t.logger.Error(ComponentWorker, "Failed to mark message as sent: %v", err)
					} else {
						t.logger.Info(ComponentWorker, "Message ID %d sent successfully", msg.ID)
					}
				}
			} else {
				if !t.shutdown {
					t.enqueueMessage(msg)
				}
			}

			<-t.workerSemaphore
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// connectionMonitor monitors the connection status
func (t *Sender) connectionMonitor() {
	defer t.wg.Done()
	t.logger.Info(ComponentMonitor, "Connection monitor started")

	healthCheckTicker := time.NewTicker(ConnectionCheckFreq * time.Second)
	statusLogTicker := time.NewTicker(60 * time.Second)  // Log status every minute
	queueCheckTicker := time.NewTicker(10 * time.Second) // Check queue status regularly
	defer healthCheckTicker.Stop()
	defer statusLogTicker.Stop()
	defer queueCheckTicker.Stop()

	consecutiveFails := 0
	wasConnected := false // Track connection transitions

	for {
		select {
		case <-healthCheckTicker.C:
			// Check if connected
			isConnected := t.client.IsConnected()

			// Detect connection established
			if isConnected && !wasConnected {
				t.logger.Info(ComponentMonitor, "Connection established - checking pending messages")
				// Trigger check for pending messages on connection restore
				go t.checkPendingMessages()
			}

			if isConnected {
				// Reset failure counter on successful connection
				if consecutiveFails > 0 {
					t.logger.Info(ComponentMonitor, "Connection restored after %d failures", consecutiveFails)
					consecutiveFails = 0
				}

				// Only check for activity timeout after 3 successful health checks
				// This prevents false positives during initial connection stabilization
				if consecutiveFails == 0 {
					elapsed := time.Since(t.client.GetLastActivity()).Seconds()
					if elapsed > ConnectionTimeout {
						t.logger.Warning(ComponentMonitor, "No activity for %f seconds (timeout: %d)", elapsed, ConnectionTimeout)
						t.logger.Warning(ComponentMonitor, "Manual detection: broker may be down without triggering disconnect callback")

						// Force manual disconnect
						t.client.Disconnect()
					}
				}
			} else if t.running && !t.shutdown {
				// Increment failure counter
				consecutiveFails++

				// Only log reconnection attempts for notable failures to avoid spamming logs
				if consecutiveFails == 1 || consecutiveFails == 3 ||
					consecutiveFails == 5 || consecutiveFails%10 == 0 {
					t.logger.Info(ComponentMonitor, "Connection check failed (attempt %d) - reconnection being handled by client", consecutiveFails)
				}

				// After 10 consecutive failures, attempt to "reset" the connection
				if consecutiveFails == 10 {
					t.logger.Warning(ComponentMonitor, "10 consecutive connection failures - forcing client reconnect")
					t.client.Disconnect() // Clean disconnect first
					time.Sleep(1 * time.Second)
					t.client.Connect() // Attempt fresh reconnect
				}
			}

			// Update connection state
			wasConnected = isConnected

		case <-statusLogTicker.C:
			if t.running && !t.shutdown {
				pendingCount, err := t.messageService.CountPendingMessages()
				if err != nil {
					t.logger.Warning(ComponentMonitor, "Failed to count pending messages: %v", err)
					pendingCount = 0
				}

				t.queueMutex.Lock()
				pendingQueueLen := len(t.pendingQueue)
				t.queueMutex.Unlock()

				t.logger.Info(ComponentMonitor, "Status update: Connected=%v, Running=%v, DB pending=%d, Queue size=%d/%d, Backing queue=%d",
					t.client.IsConnected(),
					t.running,
					pendingCount,
					len(t.messageQueue), cap(t.messageQueue),
					pendingQueueLen)

				if pendingCount > 0 && t.client.IsConnected() {
					needsCheck, err := t.messageService.HasOldPendingMessages(5 * time.Minute)
					if err != nil {
						t.logger.Warning(ComponentMonitor, "Failed to check for old pending messages: %v", err)
					} else if needsCheck {
						t.logger.Info(ComponentMonitor, "Found old pending messages, triggering check")
						go t.checkPendingMessages()
					}
				}
			}

		case <-queueCheckTicker.C:
			// Check queue sizes periodically
			if t.running && !t.shutdown {
				t.queueMutex.Lock()
				pendingQueueLen := len(t.pendingQueue)
				t.queueMutex.Unlock()

				// Log only if we have items in the backing queue
				if pendingQueueLen > 0 {
					t.logger.Debug(ComponentMonitor, "Queue stats: Main=%d/%d, Backing=%d",
						len(t.messageQueue), cap(t.messageQueue),
						pendingQueueLen)
				}
			}

		case <-t.quitChan:
			t.logger.Info(ComponentMonitor, "Connection monitor stopping")
			return
		}
	}
}

// heartbeatWorker sends periodic heartbeats
func (t *Sender) heartbeatWorker() {
	defer t.wg.Done()
	t.logger.Info(ComponentHeartbeat, "Heartbeat worker started")

	ticker := time.NewTicker(HeartbeatInterval * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if t.client.IsConnected() {
				t.sendHeartbeat()
			}

		case <-t.quitChan:
			t.logger.Info(ComponentHeartbeat, "Heartbeat worker stopping")
			return
		}
	}
}

// sendHeartbeat sends a heartbeat message
func (t *Sender) sendHeartbeat() {
	// Generate heartbeat data
	heartbeatData := map[string]interface{}{
		"type":      "heartbeat",
		"client_id": t.cfg.MQTT.ClientID,
		"timestamp": time.Now().Format(time.RFC3339),
		"random_id": fmt.Sprintf("hb_%d", time.Now().UnixNano()%10000),
	}

	payload, err := json.Marshal(heartbeatData)
	if err != nil {
		t.logger.Error(ComponentHeartbeat, "Failed to serialize heartbeat: %v", err)
		return
	}

	// Send heartbeat
	heartbeatTopic := t.cfg.MQTT.Topic + "/heartbeat"
	if err := t.client.PublishHeartbeat(heartbeatTopic, payload); err != nil {
		t.logger.Warning(ComponentHeartbeat, "Heartbeat failed: %v", err)
	} else {
		t.logger.Debug(ComponentHeartbeat, "Heartbeat sent: %s", heartbeatData["random_id"])
	}
}

// checkPendingMessages checks for and sends pending messages
func (t *Sender) checkPendingMessages() {
	// Use a mutex to ensure only one instance runs at a time
	t.pendingMutex.Lock()
	if t.checkingPending {
		t.pendingMutex.Unlock()
		return
	}
	t.checkingPending = true
	t.pendingMutex.Unlock()

	// Always reset the flag when done
	defer func() {
		t.pendingMutex.Lock()
		t.checkingPending = false
		t.pendingMutex.Unlock()
	}()

	if t.shutdown {
		return
	}

	// Try to connect if not connected
	if !t.client.IsConnected() {
		t.logger.Debug(ComponentWorker, "Not connected when checking pending, trying to connect")
		t.client.Connect()
		time.Sleep(1 * time.Second)
	}

	// Get total pending count
	pendingTotal, err := t.messageService.CountPendingMessages()
	if err != nil {
		t.logger.Error(ComponentWorker, "Failed to count pending messages: %v", err)
		return
	}

	if pendingTotal == 0 {
		t.logger.Info(ComponentWorker, "No pending messages to process")
		return
	}

	t.logger.Info(ComponentWorker, "Total pending messages: %d", pendingTotal)

	// Process in batches
	processed := 0
	batchSize := 20

	for processed < int(pendingTotal) && !t.shutdown {
		if !t.client.IsConnected() {
			t.logger.Warning(ComponentWorker, "Lost connection while processing pending messages")
			break
		}

		// Get pending messages from database
		msgs, err := t.messageService.GetPendingMessages(batchSize)
		if err != nil {
			t.logger.Error(ComponentWorker, "Failed to get pending messages: %v", err)
			break
		}

		if len(msgs) == 0 {
			t.logger.Warning(ComponentWorker, "No more pending messages found despite count being %d", pendingTotal)
			break
		}

		t.logger.Info(ComponentWorker, "Processing batch of %d pending messages", len(msgs))

		// Queue messages for sending with a small delay to avoid flooding
		for i, msg := range msgs {
			// Acquire semaphore to limit concurrent database operations
			t.workerSemaphore <- struct{}{} // Acquire semaphore

			// Process the message
			func(message models.PendingMessage) {
				defer func() {
					<-t.workerSemaphore // Release semaphore when done
				}()

				t.enqueueMessage(message)
			}(msg)

			processed++

			// Add a small delay every few messages
			if i > 0 && i%5 == 0 {
				time.Sleep(200 * time.Millisecond)
			}
		}

		// Brief pause between batches
		time.Sleep(500 * time.Millisecond)
	}

	t.logger.Info(ComponentWorker, "Processed %d of %d pending messages", processed, pendingTotal)

	// Schedule another check if there are still pending messages
	if processed < int(pendingTotal) && !t.shutdown {
		t.logger.Info(ComponentWorker, "Scheduling check for remaining %d pending messages", pendingTotal-int64(processed))
		time.AfterFunc(5*time.Second, t.checkPendingMessages)
	}
}

func formatDuration(d time.Duration) string {
	hours := d / time.Hour
	minutes := (d % time.Hour) / time.Minute
	seconds := (d % time.Minute) / time.Second
	milliseconds := (d % time.Second) / time.Millisecond

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	if milliseconds > 0 {
		parts = append(parts, fmt.Sprintf("%dms", milliseconds))
	}

	return strings.Join(parts, "")
}

// GetStatus returns the current status of the sender
func (t *Sender) GetStatus() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.processingMutex.Lock()
	processed := atomic.LoadUint64(&t.messagesProcessed)
	avgTime := time.Duration(0)
	if processed > 0 {
		avgTime = t.processingTime / time.Duration(processed)
	}
	rate := float64(processed) / time.Since(t.startTime).Seconds()
	t.processingMutex.Unlock()

	t.queueMutex.Lock()
	pendingQueueLen := len(t.pendingQueue)
	t.queueMutex.Unlock()

	ping := t.client.MeasurePing()

	status := map[string]interface{}{
		"running":             t.running,
		"connected":           t.client.IsConnected(),
		"broker":              t.cfg.MQTT.Broker,
		"port":                t.cfg.MQTT.Port,
		"client_id":           t.cfg.MQTT.ClientID,
		"last_active":         t.client.GetLastActivity().Format(time.RFC3339),
		"uptime":              time.Since(t.startTime).Truncate(time.Second).String(),
		"uptime_seconds":      int(time.Since(t.startTime).Seconds()),
		"messages_processed":  processed,
		"processing_rate":     fmt.Sprintf("%.2f msg/s", rate),
		"avg_processing_time": formatDuration(avgTime),
		"avg_processing_ms":   avgTime.Milliseconds(),
		"ping_ms":             ping.Milliseconds(),
		"channel_queue_len":   len(t.messageQueue),
		"channel_capacity":    cap(t.messageQueue),
		"backing_queue_len":   pendingQueueLen,
		"total_queued":        len(t.messageQueue) + pendingQueueLen,
	}

	dbStats, err := t.statsService.GetDatabaseStats()
	if err == nil {
		for k, v := range dbStats {
			status[k] = v
		}
	}

	return status
}

// Refresh forces a reconnect and pending message check
func (t *Sender) Refresh() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.logger.Info(ComponentSender, "Refreshing MQTT sender service")

	if !t.client.IsConnected() && t.running && !t.shutdown {
		t.logger.Info(ComponentSender, "Reconnecting to MQTT broker during refresh")
		t.client.Connect()
	}

	go t.checkPendingMessages()

	t.processingMutex.Lock()
	processingRate := float64(t.messagesProcessed) / time.Since(t.startTime).Seconds()
	t.processingMutex.Unlock()

	t.logger.Info(ComponentSender, "MQTT sender service refreshed (rate: %.2f msg/s)", processingRate)

	return nil
}
