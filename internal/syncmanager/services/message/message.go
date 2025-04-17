package message

import (
	"encoding/json"
	"fmt"
	"jarvist/internal/common/database"
	"jarvist/internal/common/models"
	"jarvist/pkg/logger"
	"time"

	"gorm.io/gorm"
)

type MessageService struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewMessageService(db *gorm.DB, logger *logger.Logger) *MessageService {
	return &MessageService{
		db:     db,
		logger: logger,
	}
}

func (s *MessageService) StoreMessage(topic string, payload interface{}, connected bool) (uint, error) {
	extraInfo, err := json.Marshal(map[string]interface{}{
		"stored_at":         time.Now().Format(time.RFC3339),
		"connection_status": connected,
		"processing":        false, // Add processing flag to track message state
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create extra info: %w", err)
	}

	var encodedPayload string
	switch v := payload.(type) {
	case string:
		encodedPayload = v
	case []byte:
		encodedPayload = string(v)
	default:
		bytes, err := json.Marshal(payload)
		if err != nil {
			return 0, fmt.Errorf("error serializing data: %v", err)
		}
		encodedPayload = string(bytes)
	}

	message := models.PendingMessage{
		Topic:           topic,
		Payload:         encodedPayload,
		Timestamp:       time.Now(),
		Sent:            false,
		ConnectionState: connected,
		ExtraInfo:       string(extraInfo),
	}

	result := s.db.Create(&message)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to insert message: %w", result.Error)
	}

	s.logger.Info(database.ComponentMessages, "Stored message ID %d for topic %s", message.ID, topic)
	return message.ID, nil
}

// New method to get and mark a specific message as processing
func (s *MessageService) GetAndMarkProcessing(messageID uint) (*models.PendingMessage, error) {
	var message models.PendingMessage

	// First get the message
	err := s.db.Where("id = ? AND sent = ?", messageID, false).First(&message).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find message: %w", err)
	}

	// Parse and update extra info to mark as processing
	var extraInfo map[string]interface{}
	if err := json.Unmarshal([]byte(message.ExtraInfo), &extraInfo); err != nil {
		return nil, fmt.Errorf("failed to parse extra info: %w", err)
	}

	extraInfo["processing"] = true
	extraInfo["processing_started"] = time.Now().Format(time.RFC3339)

	updatedExtraInfo, err := json.Marshal(extraInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to update extra info: %w", err)
	}

	// Update the message
	if err := s.db.Model(&models.PendingMessage{}).
		Where("id = ?", messageID).
		Update("extra_info", string(updatedExtraInfo)).Error; err != nil {
		return nil, fmt.Errorf("failed to mark message as processing: %w", err)
	}

	// Get the updated message
	if err := s.db.Where("id = ?", messageID).First(&message).Error; err != nil {
		return nil, fmt.Errorf("failed to get updated message: %w", err)
	}

	s.logger.Info(database.ComponentMessages, "Marked message ID %d as processing", messageID)
	return &message, nil
}

func (s *MessageService) GetPendingMessages(limit int) ([]models.PendingMessage, error) {
	var messages []models.PendingMessage

	result := s.db.Where("sent = ? AND (JSON_EXTRACT(extra_info, '$.processing') IS NULL OR JSON_EXTRACT(extra_info, '$.processing') = false)", false).
		Order("id").
		Limit(limit).
		Find(&messages)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query pending messages: %w", result.Error)
	}

	for i := range messages {
		var extraInfo map[string]interface{}
		if err := json.Unmarshal([]byte(messages[i].ExtraInfo), &extraInfo); err != nil {
			s.logger.Warning(database.ComponentMessages, "Failed to parse extra info for message ID %d: %v",
				messages[i].ID, err)
			continue
		}

		extraInfo["processing"] = true
		extraInfo["processing_started"] = time.Now().Format(time.RFC3339)

		updatedExtraInfo, err := json.Marshal(extraInfo)
		if err != nil {
			s.logger.Warning(database.ComponentMessages, "Failed to update extra info for message ID %d: %v",
				messages[i].ID, err)
			continue
		}

		if err := s.db.Model(&models.PendingMessage{}).
			Where("id = ?", messages[i].ID).
			Update("extra_info", string(updatedExtraInfo)).Error; err != nil {
			s.logger.Warning(database.ComponentMessages, "Failed to mark message ID %d as processing: %v",
				messages[i].ID, err)
		}
	}

	s.logger.Info(database.ComponentMessages, "Retrieved and marked %d messages as processing", len(messages))
	return messages, nil
}

// Reset processing status for any stuck messages
func (s *MessageService) ResetProcessingStatus() error {
	var messages []models.PendingMessage

	result := s.db.Raw(`
		SELECT * FROM pending_messages
		WHERE sent = false
		AND JSON_EXTRACT(extra_info, '$.processing') = true
	`).Scan(&messages)

	if result.Error != nil {
		return fmt.Errorf("failed to query messages to reset: %w", result.Error)
	}

	s.logger.Info(database.ComponentMessages, "Found %d stuck processing messages to reset", len(messages))

	// Reset processing status for each message
	for _, msg := range messages {
		var extraInfo map[string]interface{}
		if err := json.Unmarshal([]byte(msg.ExtraInfo), &extraInfo); err != nil {
			s.logger.Warning(database.ComponentMessages, "Failed to parse extra info for message ID %d: %v",
				msg.ID, err)
			continue
		}

		extraInfo["processing"] = false
		if extraInfo["processing_started"] != nil {
			extraInfo["processing_reset"] = time.Now().Format(time.RFC3339)
			extraInfo["processing_started"] = nil
		}

		updatedExtraInfo, err := json.Marshal(extraInfo)
		if err != nil {
			s.logger.Warning(database.ComponentMessages, "Failed to update extra info for message ID %d: %v",
				msg.ID, err)
			continue
		}

		if err := s.db.Model(&models.PendingMessage{}).
			Where("id = ?", msg.ID).
			Update("extra_info", string(updatedExtraInfo)).Error; err != nil {
			s.logger.Warning(database.ComponentMessages, "Failed to reset processing for message ID %d: %v",
				msg.ID, err)
		} else {
			s.logger.Info(database.ComponentMessages, "Reset processing for message ID %d", msg.ID)
		}
	}

	return nil
}

func (s *MessageService) MarkMessageSent(id uint) error {
	updateTime := time.Now().Format(time.RFC3339)

	var message models.PendingMessage
	if err := s.db.First(&message, id).Error; err != nil {
		return fmt.Errorf("failed to find message: %w", err)
	}

	var extraInfo map[string]interface{}
	if err := json.Unmarshal([]byte(message.ExtraInfo), &extraInfo); err != nil {
		return fmt.Errorf("failed to parse extra info: %w", err)
	}

	extraInfo["sent_at"] = updateTime
	extraInfo["processing"] = false // Clear processing flag when sent
	updatedExtraInfo, err := json.Marshal(extraInfo)
	if err != nil {
		return fmt.Errorf("failed to update extra info: %w", err)
	}

	result := s.db.Model(&models.PendingMessage{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"sent":        true,
			"retry_count": gorm.Expr("retry_count + 1"),
			"extra_info":  string(updatedExtraInfo),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark message as sent: %w", result.Error)
	}

	s.logger.Info(database.ComponentMessages, "Marked message ID %d as sent", id)
	return nil
}

func (s *MessageService) CountPendingMessages() (int64, error) {
	var count int64

	result := s.db.Model(&models.PendingMessage{}).
		Where("sent = ?", false).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to count pending messages: %w", result.Error)
	}

	return count, nil
}

func (s *MessageService) HasOldPendingMessages(age time.Duration) (bool, error) {
	var count int64
	cutoffTime := time.Now().Add(-age)

	result := s.db.Model(&models.PendingMessage{}).
		Where("sent = ? AND timestamp < ?", false, cutoffTime).
		Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("failed to check for old pending messages: %w", result.Error)
	}

	return count > 0, nil
}
