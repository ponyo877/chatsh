package domain

import "time"

type Message struct {
	ID          int
	RoomID      int
	DisplayName string
	Content     string
	CreatedAt   time.Time
}

func NewMessage(id, roomID int, displayName, content string, createdAt time.Time) Message {
	return Message{
		ID:          id,
		RoomID:      roomID,
		DisplayName: displayName,
		Content:     content,
		CreatedAt:   createdAt,
	}
}
