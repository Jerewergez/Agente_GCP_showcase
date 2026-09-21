package ws

// Message types are defined in hub.go. This file provides helper
// constructors for common message types.

import (
	"encoding/json"
	"time"
)

// NewQueryMessage creates a new query message.
func NewQueryMessage(id string, payload json.RawMessage) Message {
	return Message{
		Type:    MsgQuery,
		ID:      id,
		Payload: payload,
		TS:      time.Now().UnixMilli(),
	}
}

// NewResultMessage creates a new result message.
func NewResultMessage(id string, payload interface{}) Message {
	raw, _ := json.Marshal(payload)
	return Message{
		Type:    MsgResult,
		ID:      id,
		Payload: raw,
		TS:      time.Now().UnixMilli(),
	}
}

// NewErrorMessage creates a new error message.
func NewErrorMessage(id string, err string) Message {
	return Message{
		Type:  MsgError,
		ID:    id,
		Error: err,
		TS:    time.Now().UnixMilli(),
	}
}
