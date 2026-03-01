package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/enunezf/sentinel/internal/domain"
	"github.com/enunezf/sentinel/internal/repository/postgres"
)

// AuditService handles asynchronous audit log persistence.
type AuditService struct {
	repo   *postgres.AuditRepository
	ch     chan *domain.AuditLog
	logger *slog.Logger
}

// NewAuditService creates an AuditService with a buffered channel and starts the worker.
func NewAuditService(repo *postgres.AuditRepository, log *slog.Logger) *AuditService {
	svc := &AuditService{
		repo:   repo,
		ch:     make(chan *domain.AuditLog, 1000),
		logger: log.With("component", "audit"),
	}
	go svc.worker()
	return svc
}

// worker reads from the channel and persists audit logs.
func (s *AuditService) worker() {
	for entry := range s.ch {
		ctx := context.Background()
		if err := s.repo.Insert(ctx, entry); err != nil {
			s.logger.Error("failed to insert audit log", "event_type", entry.EventType, "error", err)
		}
	}
}

// LogEvent submits an audit log entry asynchronously.
// If the channel is full, the event is dropped and an error is logged.
func (s *AuditService) LogEvent(entry *domain.AuditLog) {
	if entry.ID == (uuid.UUID{}) {
		entry.ID = uuid.New()
	}
	select {
	case s.ch <- entry:
	default:
		s.logger.Warn("audit channel full, dropping event", "event_type", entry.EventType)
	}
}

// Close drains and closes the audit channel. Call on graceful shutdown.
func (s *AuditService) Close() {
	close(s.ch)
}
