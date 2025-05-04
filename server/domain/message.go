package domain

import "time"

type Message struct {
	ID        int
	RoomID    int
	UserID    int
	Content   string
	CreatedAt time.Time
}

func NewMessage(id, roomID, userID int, content string, createdAt time.Time) Message {
	return Message{
		ID:        id,
		RoomID:    roomID,
		UserID:    userID,
		Content:   content,
		CreatedAt: createdAt,
	}
}
