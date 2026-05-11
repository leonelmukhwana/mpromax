package notifications

import (
	"encoding/json"
	"fmt"
)

type WebProvider struct {
	Hub interface{}
}

// NewWebProvider was missing in your previous version, causing the undefined error.
func NewWebProvider() *WebProvider {
	return &WebProvider{
		Hub: nil,
	}
}

func (p *WebProvider) Broadcast(userID string, payload interface{}) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal web payload: %v", err)
	}

	// This currently prints to console until you plug in your WebSocket Hub
	fmt.Printf("📢 [WebPush] User %s: %s\n", userID, string(message))

	return nil
}
