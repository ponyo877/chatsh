package domain

type RoomService interface {
	JoinRoom(session StreamSession) error
	LeaveRoom(sessionID string) error

	BroadcastMessage(roomPath, sender, message string) error

	GetActiveClients(roomPath string) []StreamSession
	GetSession(sessionID string) (StreamSession, bool)

	IsRoomActive(roomPath string) bool
	GetActiveRooms() []string

	GetRoomClientCount(roomPath string) int

	ValidateSession(sessionID string) bool
}

type MessageBroadcaster interface {
	Broadcast(event StreamEvent) error

	SendToSession(sessionID string, event StreamEvent) error

	BroadcastToRoom(roomPath string, event StreamEvent) error

	RegisterSession(sessionID string, responseChan chan<- StreamResponse) error
	UnregisterSession(sessionID string) error

	IsSessionRegistered(sessionID string) bool
	GetRegisteredSessionCount() int
}

type StreamManager interface {
	RoomService
	MessageBroadcaster

	HandleJoinRequest(request StreamRequest, sessionID, remote string) (StreamSession, error)
	HandleLeaveRequest(sessionID string) error
	HandleChatRequest(sessionID string, message string) error

	Cleanup() error
	GetStats() StreamStats
}

type StreamStats struct {
	ActiveRooms    int
	ActiveSessions int
	TotalMessages  int64
	Uptime         string
}
