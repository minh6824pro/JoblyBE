package server

import (
	"JobblyBE/internal/biz"
)

// HubAdapter adapts Hub to implement biz.WebSocketHub interface
type HubAdapter struct {
	hub *Hub
}

// NewHubAdapter wraps Hub to implement WebSocketHub interface
func NewHubAdapter(hub *Hub) biz.WebSocketHub {
	return &HubAdapter{hub: hub}
}

// SendNotification implements biz.WebSocketHub interface
// It converts biz.NotificationMessage to NotificationPayload and sends via WebSocket
func (a *HubAdapter) SendNotification(userID string, payload interface{}) {
	// Handle biz.NotificationMessage
	if notif, ok := payload.(*biz.NotificationMessage); ok {
		wsPayload := &NotificationPayload{
			ID:        notif.ID,
			Title:     notif.Title,
			Content:   notif.Content,
			Type:      notif.Type,
			ObjectID:  notif.ObjectID,
			CreatedAt: notif.CreatedAt,
		}
		a.hub.SendNotification(userID, wsPayload)
		return
	}

	// Fallback: try to use payload directly if it's already NotificationPayload
	if wsPayload, ok := payload.(*NotificationPayload); ok {
		a.hub.SendNotification(userID, wsPayload)
	}
}
