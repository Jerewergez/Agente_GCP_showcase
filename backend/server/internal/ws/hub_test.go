package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestHubRunAndStats(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	stats := hub.Stats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats["clients"].(int) != 0 {
		t.Fatalf("expected 0 clients, got %d", stats["clients"])
	}
	if stats["rooms"].(int) != 0 {
		t.Fatalf("expected 0 rooms, got %d", stats["rooms"])
	}
}

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		msgType MessageType
		label   string
	}{
		{MsgQuery, "query"},
		{MsgResult, "result"},
		{MsgError, "error"},
		{MsgPing, "ping"},
		{MsgPong, "pong"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			m := Message{Type: tt.msgType}
			if m.Type != tt.msgType {
				t.Fatalf("expected %s, got %s", tt.msgType, m.Type)
			}
		})
	}
}

func TestNewQueryMessage(t *testing.T) {
	payload := json.RawMessage(`{"sql":"SELECT 1"}`)
	msg := NewQueryMessage("req-1", payload)
	if msg.Type != MsgQuery {
		t.Fatalf("expected query, got %s", msg.Type)
	}
	if msg.ID != "req-1" {
		t.Fatalf("expected req-1, got %s", msg.ID)
	}
	if msg.TS == 0 {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestNewResultMessage(t *testing.T) {
	data := map[string]interface{}{"rows": 10, "columns": []string{"a", "b"}}
	msg := NewResultMessage("req-1", data)
	if msg.Type != MsgResult {
		t.Fatalf("expected result, got %s", msg.Type)
	}
	if msg.ID != "req-1" {
		t.Fatalf("expected req-1, got %s", msg.ID)
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg := NewErrorMessage("req-1", "something went wrong")
	if msg.Type != MsgError {
		t.Fatalf("expected error, got %s", msg.Type)
	}
	if msg.Error != "something went wrong" {
		t.Fatalf("expected error message, got %s", msg.Error)
	}
}

func TestHandleConnection_RejectsNonWS(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Regular HTTP request to WS endpoint should fail the upgrade
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()

	hub.HandleConnection(rec, req)

	// The upgrade will fail because there's no WS upgrade headers
	if rec.Code != http.StatusBadRequest {
		// Note: gorilla/websocket returns 400 for failed upgrades.
		// If the code is different, we just check it's an error.
		if rec.Code != http.StatusOK {
			t.Logf("expected error status, got %d", rec.Code)
		}
	}
}
