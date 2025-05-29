package domain

type StreamRequestType int

const (
	RequestJoin StreamRequestType = iota
	RequestTail
	RequestChat
)

func (t StreamRequestType) String() string {
	switch t {
	case RequestJoin:
		return "join"
	case RequestTail:
		return "tail"
	case RequestChat:
		return "chat"
	default:
		return "unknown"
	}
}

type StreamRequest struct {
	Type     StreamRequestType
	Name     string
	RoomPath string
	Message  string
}

func NewJoinRequest(name, roomPath string) StreamRequest {
	return StreamRequest{
		Type:     RequestJoin,
		Name:     name,
		RoomPath: roomPath,
	}
}

func NewTailRequest(roomPath string) StreamRequest {
	return StreamRequest{
		Type:     RequestTail,
		RoomPath: roomPath,
	}
}

func NewChatRequest(message string) StreamRequest {
	return StreamRequest{
		Type:    RequestChat,
		Message: message,
	}
}

func (r StreamRequest) IsValid() bool {
	switch r.Type {
	case RequestJoin:
		return r.Name != "" && r.RoomPath != ""
	case RequestTail:
		return r.RoomPath != ""
	case RequestChat:
		return r.Message != ""
	default:
		return false
	}
}

func (r StreamRequest) String() string {
	switch r.Type {
	case RequestJoin:
		return r.Type.String() + ": " + r.Name + " -> " + r.RoomPath
	case RequestTail:
		return r.Type.String() + ": " + r.RoomPath
	case RequestChat:
		return r.Type.String() + ": " + r.Message
	default:
		return r.Type.String()
	}
}
