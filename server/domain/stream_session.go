package domain

import (
	"time"
)

type StreamSession struct {
	ID       string
	Name     string
	RoomPath string
	IsTail   bool
	JoinedAt time.Time
	Remote   string
}

func NewStreamSession(id, name, roomPath, remote string, isTail bool) StreamSession {
	return StreamSession{
		ID:       id,
		Name:     name,
		RoomPath: roomPath,
		IsTail:   isTail,
		JoinedAt: time.Now(),
		Remote:   remote,
	}
}

func (s StreamSession) IsValid() bool {
	return s.ID != "" && s.RoomPath != ""
}

func (s StreamSession) IsActive(timeout time.Duration) bool {
	return time.Since(s.JoinedAt) < timeout
}

func (s StreamSession) String() string {
	mode := "chat"
	if s.IsTail {
		mode = "tail"
	}
	return s.Name + "@" + s.RoomPath + "(" + mode + ")"
}
