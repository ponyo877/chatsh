package domain

import "time"

type StreamEventType int

const (
	EventJoin StreamEventType = iota
	EventLeave
	EventMessage
	EventError
)

func (t StreamEventType) String() string {
	switch t {
	case EventJoin:
		return "join"
	case EventLeave:
		return "leave"
	case EventMessage:
		return "message"
	case EventError:
		return "error"
	default:
		return "unknown"
	}
}

type StreamEvent struct {
	Type      StreamEventType
	SessionID string
	RoomPath  string
	Sender    string
	Message   string
	Timestamp time.Time
	Error     error
}

func NewJoinEvent(sessionID, roomPath, sender string) StreamEvent {
	return StreamEvent{
		Type:      EventJoin,
		SessionID: sessionID,
		RoomPath:  roomPath,
		Sender:    sender,
		Message:   "joined #" + roomPath + " as " + sender,
		Timestamp: time.Now(),
	}
}

func NewLeaveEvent(sessionID, roomPath, sender string) StreamEvent {
	return StreamEvent{
		Type:      EventLeave,
		SessionID: sessionID,
		RoomPath:  roomPath,
		Sender:    sender,
		Message:   "left #" + roomPath,
		Timestamp: time.Now(),
	}
}

func NewMessageEvent(sessionID, roomPath, sender, message string) StreamEvent {
	return StreamEvent{
		Type:      EventMessage,
		SessionID: sessionID,
		RoomPath:  roomPath,
		Sender:    sender,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func NewErrorEvent(sessionID, roomPath string, err error) StreamEvent {
	return StreamEvent{
		Type:      EventError,
		SessionID: sessionID,
		RoomPath:  roomPath,
		Error:     err,
		Timestamp: time.Now(),
	}
}

func (e StreamEvent) IsValid() bool {
	switch e.Type {
	case EventJoin, EventLeave, EventMessage:
		return e.SessionID != "" && e.RoomPath != "" && e.Sender != ""
	case EventError:
		return e.Error != nil
	default:
		return false
	}
}

func (e StreamEvent) String() string {
	if e.Type == EventError {
		return e.Type.String() + ": " + e.Error.Error()
	}
	return e.Type.String() + ": " + e.Sender + " - " + e.Message
}
