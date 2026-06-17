//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paniccaaa/tsunami/cmd/server"
)

func newHandlers() (*server.Handlers, *server.SessionManager) {
	sm := server.NewSessionManager()
	hub := server.NewHub()
	go hub.Run()
	return server.NewHandlers(sm, hub), sm
}

func TestHandleStartAttack(t *testing.T) {
	h, sm := newHandlers()

	body := map[string]any{
		"url":      testBaseURL + "/get",
		"method":   "GET",
		"rate":     "5/1s",
		"duration": "3s",
		"workers":  uint(5),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/attack/start", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.HandleStartAttack(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty session ID in response")
	}
	if resp["status"] != "running" {
		t.Errorf("expected status 'running', got %v", resp["status"])
	}
	if resp["started_at"] == nil {
		t.Error("expected started_at in response")
	}

	t.Cleanup(func() {
		if session := sm.GetCurrent(); session != nil {
			session.Stop()
		}
	})
}

func TestHandleStartAttack_ValidationError(t *testing.T) {
	h, _ := newHandlers()

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing url",
			body: map[string]any{"rate": "10/1s"},
		},
		{
			name: "invalid url scheme",
			body: map[string]any{"url": "ftp://example.com", "rate": "10/1s"},
		},
		{
			name: "invalid rate format",
			body: map[string]any{"url": testBaseURL, "rate": "invalid"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/attack/start", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.HandleStartAttack(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleStopAttack(t *testing.T) {
	h, _ := newHandlers()

	startBody := map[string]any{
		"url":      testBaseURL + "/get",
		"method":   "GET",
		"rate":     "5/1s",
		"duration": "60s",
		"workers":  uint(5),
	}
	startBytes, _ := json.Marshal(startBody)

	startReq := httptest.NewRequest(http.MethodPost, "/api/attack/start", bytes.NewReader(startBytes))
	startReq.Header.Set("Content-Type", "application/json")
	startRR := httptest.NewRecorder()
	h.HandleStartAttack(startRR, startReq)

	if startRR.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRR.Code, startRR.Body.String())
	}

	time.Sleep(50 * time.Millisecond)

	stopReq := httptest.NewRequest(http.MethodPost, "/api/attack/stop", nil)
	stopRR := httptest.NewRecorder()
	h.HandleStopAttack(stopRR, stopReq)

	if stopRR.Code != http.StatusOK {
		t.Fatalf("expected 200 on stop, got %d: %s", stopRR.Code, stopRR.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(stopRR.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode stop response: %v", err)
	}

	if resp["status"] != "stopped" {
		t.Errorf("expected status 'stopped', got %v", resp["status"])
	}
	if resp["id"] == nil {
		t.Error("expected session ID in stop response")
	}
	if resp["stopped_at"] == nil {
		t.Error("expected stopped_at in stop response")
	}
}

func TestHandleStopAttack_NoSession(t *testing.T) {
	h, _ := newHandlers()

	req := httptest.NewRequest(http.MethodPost, "/api/attack/stop", nil)
	rr := httptest.NewRecorder()
	h.HandleStopAttack(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when no session, got %d", rr.Code)
	}
}

func TestHandleStartAttack_WrongMethod(t *testing.T) {
	h, _ := newHandlers()

	req := httptest.NewRequest(http.MethodGet, "/api/attack/start", nil)
	rr := httptest.NewRecorder()
	h.HandleStartAttack(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}
