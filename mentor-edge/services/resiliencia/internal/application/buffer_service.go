package application

import (
	"context"
	"log"
	"time"

	"resiliencia/internal/domain"
	"resiliencia/internal/ports"
)

type MaintenanceConfig struct {
	RetentionDays int
	StaleHours    int
	MaxDiskBytes  int64
	EmergencyKeep int
}

func DefaultMaintenanceConfig() MaintenanceConfig {
	return MaintenanceConfig{
		RetentionDays: 180,
		StaleHours:    48,
		MaxDiskBytes:  5 * 1024 * 1024 * 1024,
		EmergencyKeep: 10000,
	}
}

type BufferService struct {
	storage     ports.EventStorage
	dedup       domain.DeduplicationPolicy
	queue       domain.QueuePolicy
	maintenance MaintenanceConfig
}

func NewBufferService(
	storage ports.EventStorage,
	dedup domain.DeduplicationPolicy,
	queue domain.QueuePolicy,
) *BufferService {
	return &BufferService{
		storage:     storage,
		dedup:       dedup,
		queue:       queue,
		maintenance: DefaultMaintenanceConfig(),
	}
}

func (s *BufferService) SetMaintenanceConfig(cfg MaintenanceConfig) {
	s.maintenance = cfg
}

func (s *BufferService) StoreEvent(ctx context.Context, event *domain.EventBuffer) error {
	if s.dedup.IsDuplicate(event.EventID) {
		return nil
	}

	if !s.queue.ShouldAccept(event) {
		return nil
	}

	if err := s.storage.Store(ctx, event); err != nil {
		return err
	}

	s.dedup.MarkProcessed(event.EventID)
	return nil
}

func (s *BufferService) GetPendingCount(ctx context.Context) (int, error) {
	return s.storage.GetPendingCount(ctx)
}

func (s *BufferService) GetRecentEvents(ctx context.Context, limit int, since *time.Time) ([]*domain.EventBuffer, error) {
	return s.storage.GetRecentEvents(ctx, limit, since)
}

func (s *BufferService) GetVisionCount(ctx context.Context, since, until time.Time) (*domain.VisionCount, error) {
	return s.storage.GetVisionCount(ctx, since, until)
}

func (s *BufferService) GetVisionCounter(ctx context.Context, until time.Time) (*domain.VisionCounter, error) {
	return s.storage.GetVisionCounter(ctx, until)
}

func (s *BufferService) GetPendingEvents(ctx context.Context, limit int) ([]*domain.EventBuffer, error) {
	return s.storage.GetPendingEvents(ctx, limit)
}

func (s *BufferService) GetBufferStats(ctx context.Context) (*domain.BufferStats, error) {
	return s.storage.GetBufferStats(ctx)
}

func (s *BufferService) RunMaintenance(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runMaintenanceCycle(ctx)
		}
	}
}

func (s *BufferService) runMaintenanceCycle(ctx context.Context) {
	purged, err := s.storage.PurgeExpired(ctx)
	if err != nil {
		log.Printf("Maintenance: purge expired failed: %v", err)
	} else if purged > 0 {
		log.Printf("Maintenance: purged %d expired events", purged)
	}

	stale, err := s.storage.MarkStaleEventsDead(ctx, s.maintenance.StaleHours)
	if err != nil {
		log.Printf("Maintenance: mark stale failed: %v", err)
	} else if stale > 0 {
		log.Printf("Maintenance: marked %d stale events as dead", stale)
	}

	stats, err := s.storage.GetBufferStats(ctx)
	if err != nil {
		log.Printf("Maintenance: get stats failed: %v", err)
		return
	}

	if stats.DiskBytes > s.maintenance.MaxDiskBytes {
		log.Printf("Maintenance: disk %d MB exceeds %d MB limit, emergency purge",
			stats.DiskBytes/(1024*1024), s.maintenance.MaxDiskBytes/(1024*1024))
		removed, err := s.storage.EmergencyPurge(ctx, s.maintenance.EmergencyKeep)
		if err != nil {
			log.Printf("Maintenance: emergency purge failed: %v", err)
		} else {
			log.Printf("Maintenance: emergency purged %d synced events", removed)
		}
	}

	log.Printf("Maintenance: total=%d pending=%d synced=%d dead=%d disk=%.1fMB",
		stats.TotalCount, stats.PendingCount, stats.SyncedCount, stats.DeadCount,
		float64(stats.DiskBytes)/(1024*1024))
}
