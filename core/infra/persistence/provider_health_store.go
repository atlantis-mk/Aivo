package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) SaveProviderHealth(ctx context.Context, health domain.ProviderHealth) error {
	if health.UpdatedAt == "" {
		health.UpdatedAt = domain.NowString(time.Now())
	}
	if health.Status == "" {
		health.Status = "unknown"
	}
	row := providerHealthRow{
		ProviderID:       health.ProviderID,
		Status:           health.Status,
		LastSuccessAt:    health.LastSuccessAt,
		LastFailureAt:    health.LastFailureAt,
		LastLatencyMs:    health.LastLatencyMs,
		LastErrorClass:   health.LastErrorClass,
		LastErrorMessage: health.LastErrorMessage,
		LastHTTPStatus:   health.LastHTTPStatus,
		FailureCount:     health.FailureCount,
		UpdatedAt:        health.UpdatedAt,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (s *Store) LoadProviderHealth(ctx context.Context, providerID string) (*domain.ProviderHealth, error) {
	var row providerHealthRow
	err := s.db.WithContext(ctx).Where("provider_id = ?", providerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	health := providerHealthFromRow(row)
	return &health, nil
}

func (s *Store) ListProviderHealth(ctx context.Context) ([]domain.ProviderHealth, error) {
	var rows []providerHealthRow
	if err := s.db.WithContext(ctx).Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	health := make([]domain.ProviderHealth, 0, len(rows))
	for _, row := range rows {
		health = append(health, providerHealthFromRow(row))
	}
	return health, nil
}

func providerHealthFromRow(row providerHealthRow) domain.ProviderHealth {
	return domain.ProviderHealth{
		ProviderID:       row.ProviderID,
		Status:           row.Status,
		LastSuccessAt:    row.LastSuccessAt,
		LastFailureAt:    row.LastFailureAt,
		LastLatencyMs:    row.LastLatencyMs,
		LastErrorClass:   row.LastErrorClass,
		LastErrorMessage: row.LastErrorMessage,
		LastHTTPStatus:   row.LastHTTPStatus,
		FailureCount:     row.FailureCount,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (s *Store) SaveProviderCallEvent(ctx context.Context, event domain.ProviderCallEvent) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt == "" {
		event.CreatedAt = domain.NowString(time.Now())
	}
	streaming := 0
	if event.Streaming {
		streaming = 1
	}
	estimated := 0
	if event.Estimated {
		estimated = 1
	}
	row := providerCallEventRow{
		ID:            event.ID,
		ProviderID:    event.ProviderID,
		ModelID:       event.ModelID,
		Transport:     event.Transport,
		Status:        event.Status,
		ErrorClass:    event.ErrorClass,
		ErrorMessage:  event.ErrorMessage,
		HTTPStatus:    event.HTTPStatus,
		LatencyMs:     event.LatencyMs,
		InputTokens:   event.InputTokens,
		OutputTokens:  event.OutputTokens,
		TotalTokens:   event.TotalTokens,
		CostMicros:    event.CostMicros,
		Estimated:     estimated,
		Attempt:       event.Attempt,
		FallbackIndex: event.FallbackIndex,
		Streaming:     streaming,
		ToolCallCount: event.ToolCallCount,
		CreatedAt:     event.CreatedAt,
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *Store) ListProviderCallEvents(ctx context.Context, providerID string, limit int) ([]domain.ProviderCallEvent, error) {
	if limit <= 0 || limit > 5000 {
		limit = 50
	}
	query := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if strings.TrimSpace(providerID) != "" {
		query = query.Where("provider_id = ?", providerID)
	}
	var rows []providerCallEventRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	events := make([]domain.ProviderCallEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, providerCallEventFromRow(row))
	}
	return events, nil
}

func providerCallEventFromRow(row providerCallEventRow) domain.ProviderCallEvent {
	return domain.ProviderCallEvent{
		ID:            row.ID,
		ProviderID:    row.ProviderID,
		ModelID:       row.ModelID,
		Transport:     row.Transport,
		Status:        row.Status,
		ErrorClass:    row.ErrorClass,
		ErrorMessage:  row.ErrorMessage,
		HTTPStatus:    row.HTTPStatus,
		LatencyMs:     row.LatencyMs,
		InputTokens:   row.InputTokens,
		OutputTokens:  row.OutputTokens,
		TotalTokens:   row.TotalTokens,
		CostMicros:    row.CostMicros,
		Estimated:     row.Estimated == 1,
		Attempt:       row.Attempt,
		FallbackIndex: row.FallbackIndex,
		Streaming:     row.Streaming == 1,
		ToolCallCount: row.ToolCallCount,
		CreatedAt:     row.CreatedAt,
	}
}
