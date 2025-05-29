package domain

import (
	"fmt"
	"sync"
	"time"
)

const (
	ringSize       = 256
	sessionTimeout = 30 * time.Minute
)

type streamManagerImpl struct {
	mu            sync.RWMutex
	rooms         map[string]*roomImpl
	sessions      map[string]StreamSession
	responseChans map[string]chan<- StreamResponse
	stats         StreamStats
	startTime     time.Time
}

type roomImpl struct {
	mu        sync.RWMutex
	path      string
	clients   map[string]StreamSession
	broadcast chan StreamEvent
	manager   *streamManagerImpl
}

func NewStreamManager() StreamManager {
	sm := &streamManagerImpl{
		rooms:         make(map[string]*roomImpl),
		sessions:      make(map[string]StreamSession),
		responseChans: make(map[string]chan<- StreamResponse),
		startTime:     time.Now(),
	}
	return sm
}

func newRoom(path string, manager *streamManagerImpl) *roomImpl {
	r := &roomImpl{
		path:      path,
		clients:   make(map[string]StreamSession),
		broadcast: make(chan StreamEvent, ringSize),
		manager:   manager,
	}
	go r.fanout()
	return r
}

func (r *roomImpl) fanout() {
	for event := range r.broadcast {
		r.mu.RLock()
		for sessionID := range r.clients {
			if responseChan, exists := r.manager.responseChans[sessionID]; exists {
				response := StreamResponse{
					Name:    event.Sender,
					Message: event.Message,
				}
				select {
				case responseChan <- response:
				default:

				}
			}
		}
		r.mu.RUnlock()
	}
}

func (sm *streamManagerImpl) JoinRoom(session StreamSession) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions[session.ID] = session

	room, exists := sm.rooms[session.RoomPath]
	if !exists {
		room = newRoom(session.RoomPath, sm)
		sm.rooms[session.RoomPath] = room
	}

	room.mu.Lock()
	room.clients[session.ID] = session
	room.mu.Unlock()

	sm.stats.ActiveSessions++
	sm.stats.ActiveRooms = len(sm.rooms)

	return nil
}

func (sm *streamManagerImpl) LeaveRoom(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if room, roomExists := sm.rooms[session.RoomPath]; roomExists {
		room.mu.Lock()
		delete(room.clients, sessionID)
		clientCount := len(room.clients)
		room.mu.Unlock()

		if clientCount == 0 {
			close(room.broadcast)
			delete(sm.rooms, session.RoomPath)
		}
	}

	delete(sm.sessions, sessionID)

	sm.stats.ActiveSessions--
	sm.stats.ActiveRooms = len(sm.rooms)

	return nil
}

func (sm *streamManagerImpl) BroadcastMessage(roomPath, sender, message string) error {
	sm.mu.RLock()
	room, exists := sm.rooms[roomPath]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("room not found: %s", roomPath)
	}

	event := NewMessageEvent("", roomPath, sender, message)
	select {
	case room.broadcast <- event:
		sm.stats.TotalMessages++
		return nil
	default:

		return fmt.Errorf("room broadcast channel is full")
	}
}

func (sm *streamManagerImpl) GetActiveClients(roomPath string) []StreamSession {
	sm.mu.RLock()
	room, exists := sm.rooms[roomPath]
	sm.mu.RUnlock()

	if !exists {
		return []StreamSession{}
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	clients := make([]StreamSession, 0, len(room.clients))
	for _, session := range room.clients {
		clients = append(clients, session)
	}
	return clients
}

func (sm *streamManagerImpl) GetSession(sessionID string) (StreamSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

func (sm *streamManagerImpl) IsRoomActive(roomPath string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	_, exists := sm.rooms[roomPath]
	return exists
}

func (sm *streamManagerImpl) GetActiveRooms() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	rooms := make([]string, 0, len(sm.rooms))
	for roomPath := range sm.rooms {
		rooms = append(rooms, roomPath)
	}
	return rooms
}

func (sm *streamManagerImpl) GetRoomClientCount(roomPath string) int {
	sm.mu.RLock()
	room, exists := sm.rooms[roomPath]
	sm.mu.RUnlock()

	if !exists {
		return 0
	}

	room.mu.RLock()
	defer room.mu.RUnlock()
	return len(room.clients)
}

func (sm *streamManagerImpl) ValidateSession(sessionID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	return session.IsActive(sessionTimeout)
}

func (sm *streamManagerImpl) Broadcast(event StreamEvent) error {
	return sm.BroadcastToRoom(event.RoomPath, event)
}

func (sm *streamManagerImpl) SendToSession(sessionID string, event StreamEvent) error {
	sm.mu.RLock()
	responseChan, exists := sm.responseChans[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not registered: %s", sessionID)
	}

	response := StreamResponse{
		Name:    event.Sender,
		Message: event.Message,
	}

	select {
	case responseChan <- response:
		return nil
	default:
		return fmt.Errorf("session response channel is full")
	}
}

func (sm *streamManagerImpl) BroadcastToRoom(roomPath string, event StreamEvent) error {
	sm.mu.RLock()
	room, exists := sm.rooms[roomPath]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("room not found: %s", roomPath)
	}

	select {
	case room.broadcast <- event:
		return nil
	default:
		return fmt.Errorf("room broadcast channel is full")
	}
}

func (sm *streamManagerImpl) RegisterSession(sessionID string, responseChan chan<- StreamResponse) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.responseChans[sessionID] = responseChan
	return nil
}

func (sm *streamManagerImpl) UnregisterSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.responseChans, sessionID)
	return nil
}

func (sm *streamManagerImpl) IsSessionRegistered(sessionID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	_, exists := sm.responseChans[sessionID]
	return exists
}

func (sm *streamManagerImpl) GetRegisteredSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.responseChans)
}

func (sm *streamManagerImpl) HandleJoinRequest(request StreamRequest, sessionID, remote string) (StreamSession, error) {
	if !request.IsValid() || request.Type != RequestJoin {
		return StreamSession{}, fmt.Errorf("invalid join request")
	}

	session := NewStreamSession(sessionID, request.Name, request.RoomPath, remote, false)
	if err := sm.JoinRoom(session); err != nil {
		return StreamSession{}, fmt.Errorf("failed to join room: %w", err)
	}

	return session, nil
}

func (sm *streamManagerImpl) HandleLeaveRequest(sessionID string) error {
	return sm.LeaveRoom(sessionID)
}

func (sm *streamManagerImpl) HandleChatRequest(sessionID string, message string) error {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return sm.BroadcastMessage(session.RoomPath, session.Name, message)
}

func (sm *streamManagerImpl) Cleanup() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, room := range sm.rooms {
		close(room.broadcast)
	}

	sm.rooms = make(map[string]*roomImpl)
	sm.sessions = make(map[string]StreamSession)
	sm.responseChans = make(map[string]chan<- StreamResponse)
	sm.stats = StreamStats{}

	return nil
}

func (sm *streamManagerImpl) GetStats() StreamStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := sm.stats
	stats.Uptime = time.Since(sm.startTime).String()
	return stats
}
