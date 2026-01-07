package service

import (
	"context"
	"sync"
	"time"

	"github.com/bravo68web/stasis/pkg/logger"
)

// MirrorCronService handles scheduled syncing of mirror repositories
type MirrorCronService struct {
	syncService *MirrorSyncService
	interval    time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
	log         *logger.Logger
}

// NewMirrorCronService creates a new mirror cron service
func NewMirrorCronService(
	syncService *MirrorSyncService,
	interval time.Duration,
) *MirrorCronService {
	if interval == 0 {
		interval = 1 * time.Hour // Default to 1 hour
	}

	return &MirrorCronService{
		syncService: syncService,
		interval:    interval,
		stopChan:    make(chan struct{}),
		log:         logger.Get(),
	}
}

// Start starts the cron scheduler
func (s *MirrorCronService) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.log.Warn("Mirror cron service already running")
		return
	}

	s.running = true
	s.wg.Add(1)

	go s.run()

	s.log.Info("Mirror cron service started",
		logger.String("interval", s.interval.String()),
	)
}

// Stop stops the cron scheduler
func (s *MirrorCronService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		s.log.Warn("Mirror cron service not running")
		return
	}

	s.log.Info("Stopping mirror cron service")
	close(s.stopChan)
	s.running = false

	// Wait for goroutine to finish
	s.wg.Wait()

	s.log.Info("Mirror cron service stopped")
}

// run is the main loop for the cron scheduler
func (s *MirrorCronService) run() {
	defer s.wg.Done()

	// Create a ticker for the sync interval
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.log.Info("Mirror cron scheduler running",
		logger.String("interval", s.interval.String()),
	)

	// Run initial sync immediately
	s.syncAllMirrors()

	for {
		select {
		case <-ticker.C:
			s.syncAllMirrors()
		case <-s.stopChan:
			s.log.Info("Mirror cron scheduler received stop signal")
			return
		}
	}
}

// syncAllMirrors performs the sync operation
func (s *MirrorCronService) syncAllMirrors() {
	s.log.Info("Starting scheduled mirror sync")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := s.syncService.SyncAllMirrors(ctx); err != nil {
		s.log.Error("Failed to sync mirrors",
			logger.Error(err),
		)
	}
}

// IsRunning returns whether the cron service is running
func (s *MirrorCronService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// GetInterval returns the current sync interval
func (s *MirrorCronService) GetInterval() time.Duration {
	return s.interval
}

// SetInterval updates the sync interval
func (s *MirrorCronService) SetInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if interval == 0 {
		s.log.Warn("Invalid interval provided, ignoring")
		return
	}

	wasRunning := s.running

	// Restart if running
	if wasRunning {
		s.mu.Unlock()
		s.Stop()
		s.mu.Lock()
	}

	s.interval = interval

	if wasRunning {
		s.mu.Unlock()
		s.Start()
		s.mu.Lock()
	}

	s.log.Info("Mirror sync interval updated",
		logger.String("new_interval", interval.String()),
	)
}
