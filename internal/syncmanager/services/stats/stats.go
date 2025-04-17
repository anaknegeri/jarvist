package stats

import (
	"fmt"
	"jarvist/internal/common/models"
	"jarvist/internal/syncmanager/services/log"

	"gorm.io/gorm"
)

type StatsService struct {
	db         *gorm.DB
	logService *log.LogService
}

func NewStatsService(db *gorm.DB, logService *log.LogService) *StatsService {
	return &StatsService{
		db:         db,
		logService: logService,
	}
}

func (s *StatsService) GetDatabaseStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	var totalCount int64
	if err := s.db.Model(&models.PendingMessage{}).Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count total messages: %w", err)
	}
	stats["total_messages"] = totalCount

	var pendingCount int64
	if err := s.db.Model(&models.PendingMessage{}).Where("sent = ?", false).Count(&pendingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count pending messages: %w", err)
	}
	stats["pending_messages"] = pendingCount

	var sentCount int64
	if err := s.db.Model(&models.PendingMessage{}).Where("sent = ?", true).Count(&sentCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count sent messages: %w", err)
	}
	stats["sent_messages"] = sentCount

	logStats, err := s.logService.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get log stats: %w", err)
	}

	for k, v := range logStats {
		stats[k] = v
	}

	return stats, nil
}

func (s *StatsService) GetTopicStats() (map[string]interface{}, error) {
	type TopicStat struct {
		Topic string
		Count int64
	}

	var topicStats []TopicStat

	if err := s.db.Model(&models.PendingMessage{}).
		Select("topic, count(*) as count").
		Where("sent = ?", true).
		Group("topic").
		Scan(&topicStats).Error; err != nil {
		return nil, fmt.Errorf("failed to get topic stats: %w", err)
	}

	result := map[string]interface{}{}
	for _, stat := range topicStats {
		result[stat.Topic] = stat.Count
	}

	return result, nil
}
