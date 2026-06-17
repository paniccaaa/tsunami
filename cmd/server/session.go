package server

import (
	"sync"
	"time"

	"github.com/paniccaaa/tsunami/internal/attack"
	"github.com/paniccaaa/tsunami/internal/grpcattack"
)

type SessionStatus string

const (
	StatusIdle      SessionStatus = "idle"
	StatusRunning   SessionStatus = "running"
	StatusCompleted SessionStatus = "completed"
	StatusStopped   SessionStatus = "stopped"
	StatusError     SessionStatus = "error"
)

type TestSession struct {
	ID          string
	Status      SessionStatus
	Protocol    string
	Config      *attack.AttackConfig
	GRPCConfig  *grpcattack.Config
	Metrics     *attack.GlobalMetrics
	StartTime   time.Time
	EndTime     time.Time
	ElapsedTime time.Duration
	StopCh      chan struct{}
	Error       error
	mu          sync.RWMutex
}

type SessionManager struct {
	current *TestSession
	mu      sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

func (sm *SessionManager) GetCurrent() *TestSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

func (sm *SessionManager) CreateSession(id string, cfg *attack.AttackConfig) *TestSession {
	return sm.createSession(id, "http", cfg, nil)
}

func (sm *SessionManager) CreateGRPCSession(id string, cfg *grpcattack.Config) *TestSession {
	return sm.createSession(id, "grpc", nil, cfg)
}

func (sm *SessionManager) createSession(id, protocol string, httpCfg *attack.AttackConfig, grpcCfg *grpcattack.Config) *TestSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.current != nil && sm.current.Status == StatusRunning {
		sm.current.Stop()
	}

	session := &TestSession{
		ID:         id,
		Status:     StatusIdle,
		Protocol:   protocol,
		Config:     httpCfg,
		GRPCConfig: grpcCfg,
		StartTime:  time.Now(),
		StopCh:     make(chan struct{}),
	}

	sm.current = session
	return session
}

func (s *TestSession) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusRunning
	s.StartTime = time.Now()
}

func (s *TestSession) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusRunning {
		select {
		case <-s.StopCh:
		default:
			close(s.StopCh)
		}
		s.Status = StatusStopped
		s.EndTime = time.Now()
	}
}

func (s *TestSession) Complete(metrics *attack.GlobalMetrics, elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusCompleted
	s.Metrics = metrics
	s.ElapsedTime = elapsed
	s.EndTime = time.Now()
}

func (s *TestSession) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusError
	s.Error = err
	s.EndTime = time.Now()
}

func (s *TestSession) SetMetrics(metrics *attack.GlobalMetrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Metrics = metrics
}

func (s *TestSession) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

func (s *TestSession) GetMetrics() *attack.GlobalMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Metrics
}

func (s *TestSession) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status == StatusRunning
}
