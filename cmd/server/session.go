/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package server

import (
	"sync"
	"time"

	"github.com/paniccaaa/tsunami/internal/attack"
)

// SessionStatus represents the current state of a test session
type SessionStatus string

const (
	StatusIdle      SessionStatus = "idle"
	StatusRunning   SessionStatus = "running"
	StatusCompleted SessionStatus = "completed"
	StatusStopped   SessionStatus = "stopped"
	StatusError     SessionStatus = "error"
)

// TestSession represents a single load test session
type TestSession struct {
	ID          string
	Status      SessionStatus
	Config      *attack.AttackConfig
	Metrics     *attack.GlobalMetrics
	StartTime   time.Time
	EndTime     time.Time
	ElapsedTime time.Duration
	StopCh      chan struct{}
	Error       error
	mu          sync.RWMutex
}

// SessionManager manages the current test session (single session at a time)
type SessionManager struct {
	current *TestSession
	mu      sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

// GetCurrent returns the current session (may be nil)
func (sm *SessionManager) GetCurrent() *TestSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// CreateSession creates a new test session, stopping any existing one
func (sm *SessionManager) CreateSession(id string, cfg *attack.AttackConfig) *TestSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Stop existing session if running
	if sm.current != nil && sm.current.Status == StatusRunning {
		sm.current.Stop()
	}

	session := &TestSession{
		ID:        id,
		Status:    StatusIdle,
		Config:    cfg,
		StartTime: time.Now(),
		StopCh:    make(chan struct{}),
	}

	sm.current = session
	return session
}

// Start marks the session as running
func (s *TestSession) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusRunning
	s.StartTime = time.Now()
}

// Stop signals the session to stop
func (s *TestSession) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusRunning {
		select {
		case <-s.StopCh:
			// Already closed
		default:
			close(s.StopCh)
		}
		s.Status = StatusStopped
		s.EndTime = time.Now()
	}
}

// Complete marks the session as completed with results
func (s *TestSession) Complete(metrics *attack.GlobalMetrics, elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusCompleted
	s.Metrics = metrics
	s.ElapsedTime = elapsed
	s.EndTime = time.Now()
}

// SetError marks the session as failed with an error
func (s *TestSession) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusError
	s.Error = err
	s.EndTime = time.Now()
}

// SetMetrics updates the session metrics (used during streaming)
func (s *TestSession) SetMetrics(metrics *attack.GlobalMetrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Metrics = metrics
}

// GetStatus returns the current session status
func (s *TestSession) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// GetMetrics returns the current metrics
func (s *TestSession) GetMetrics() *attack.GlobalMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Metrics
}

// IsRunning returns true if the session is currently running
func (s *TestSession) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status == StatusRunning
}
