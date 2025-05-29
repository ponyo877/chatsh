package usecase

import (
	"fmt"
	"strings"

	"github.com/ponyo877/chatsh/server/domain"
)

// StreamUsecase handles streaming-related business logic
type StreamUsecase struct {
	repo          Repository
	streamManager domain.StreamManager
}

// NewStreamUsecase creates a new stream usecase
func NewStreamUsecase(repo Repository, streamManager domain.StreamManager) *StreamUsecase {
	return &StreamUsecase{
		repo:          repo,
		streamManager: streamManager,
	}
}

// HandleStreamSession processes streaming requests and responses using domain types only
func (u *StreamUsecase) HandleStreamSession(
	requestChan <-chan domain.StreamRequest,
	responseChan chan<- domain.StreamResponse,
	sessionID, remote string,
) error {
	// Register the response channel with the stream manager
	if err := u.streamManager.RegisterSession(sessionID, responseChan); err != nil {
		return fmt.Errorf("failed to register session: %w", err)
	}
	defer u.streamManager.UnregisterSession(sessionID)

	var sessionInitialized bool

	// Process incoming requests
	for request := range requestChan {
		if !sessionInitialized {
			// Handle initial request (join or tail)
			_, err := u.HandleInitialRequest(request, sessionID, remote)
			if err != nil {
				responseChan <- domain.NewStreamError(err)
				return err
			}
			sessionInitialized = true
			continue
		}

		// Handle subsequent requests (chat messages)
		if request.Type == domain.RequestChat {
			if err := u.HandleChatMessage(sessionID, request.Message); err != nil {
				responseChan <- domain.NewStreamError(err)
				// Don't return error for chat message failures, continue processing
			}
		}
	}

	// Clean up session when request channel is closed
	if sessionInitialized {
		if err := u.HandleSessionEnd(sessionID); err != nil {
			fmt.Printf("Error ending session %s: %v\n", sessionID, err)
		}
	}

	return nil
}

// HandleInitialRequest processes the first request from a streaming client
func (u *StreamUsecase) HandleInitialRequest(
	request domain.StreamRequest,
	sessionID, remote string,
) (domain.StreamSession, error) {
	// Validate request
	if !request.IsValid() {
		return domain.StreamSession{}, fmt.Errorf("invalid initial request")
	}

	// Set default room if not specified
	roomPath := request.RoomPath
	if roomPath == "" {
		roomPath = defaultRoom
	}

	// Validate room exists and is actually a room
	if err := u.validateRoom(roomPath); err != nil {
		return domain.StreamSession{}, fmt.Errorf("room validation failed: %w", err)
	}

	// Handle different request types
	switch request.Type {
	case domain.RequestJoin:
		return u.handleJoinRequest(request, sessionID, remote, roomPath)
	case domain.RequestTail:
		return u.handleTailRequest(request, sessionID, remote, roomPath)
	default:
		return domain.StreamSession{}, fmt.Errorf("invalid initial request type: %s", request.Type)
	}
}

// HandleChatMessage processes a chat message from an active session
func (u *StreamUsecase) HandleChatMessage(sessionID, message string) error {
	// Get session
	session, exists := u.streamManager.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Validate message
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return nil // Ignore empty messages
	}

	// Get room details for database storage
	roomNode, err := u.repo.GetNodeByPath(domain.NewPath(session.RoomPath))
	if err != nil {
		return fmt.Errorf("failed to get room details: %w", err)
	}

	// Broadcast message to room
	if err := u.streamManager.BroadcastMessage(session.RoomPath, session.Name, trimmedMessage); err != nil {
		return fmt.Errorf("failed to broadcast message: %w", err)
	}

	// Save message to database
	if err := u.repo.CreateMessage(roomNode.ID, session.Name, trimmedMessage); err != nil {
		// Log error but don't fail the broadcast
		fmt.Printf("Error saving chat message to DB for roomID %d: %v\n", roomNode.ID, err)
	}

	return nil
}

// HandleSessionEnd processes the end of a streaming session
func (u *StreamUsecase) HandleSessionEnd(sessionID string) error {
	// Get session before removing it
	session, exists := u.streamManager.GetSession(sessionID)
	if !exists {
		return nil // Session already ended
	}

	// Don't send leave message for tail mode
	if !session.IsTail {
		// Get room details for database storage
		roomNode, err := u.repo.GetNodeByPath(domain.NewPath(session.RoomPath))
		if err != nil {
			fmt.Printf("Error getting room details for leave message: %v\n", err)
		} else {
			// Broadcast leave message
			leaveMessage := fmt.Sprintf("left #%s", session.RoomPath)
			if err := u.streamManager.BroadcastMessage(session.RoomPath, session.Name, leaveMessage); err != nil {
				fmt.Printf("Error broadcasting leave message: %v\n", err)
			}

			// Save leave message to database
			if err := u.repo.CreateMessage(roomNode.ID, session.Name, leaveMessage); err != nil {
				fmt.Printf("Error saving leave message to DB for roomID %d: %v\n", roomNode.ID, err)
			}
		}
	}

	// Remove session from stream manager
	return u.streamManager.LeaveRoom(sessionID)
}

// GetStreamStats returns streaming statistics
func (u *StreamUsecase) GetStreamStats() domain.StreamStats {
	return u.streamManager.GetStats()
}

// validateRoom checks if the given path is a valid room
func (u *StreamUsecase) validateRoom(roomPath string) error {
	roomNode, err := u.repo.GetNodeByPath(domain.NewPath(roomPath))
	if err != nil {
		return fmt.Errorf("failed to get room details from DB for '%s': %w", roomPath, err)
	}
	if roomNode.Type != domain.NodeTypeRoom {
		return fmt.Errorf("path '%s' is not a room", roomPath)
	}
	return nil
}

// handleJoinRequest processes a join request
func (u *StreamUsecase) handleJoinRequest(
	request domain.StreamRequest,
	sessionID, remote, roomPath string,
) (domain.StreamSession, error) {
	// Set default client name if not provided
	clientName := request.Name
	if clientName == "" {
		clientName = remote
	}

	// Create session
	session := domain.NewStreamSession(sessionID, clientName, roomPath, remote, false)

	// Join room
	if err := u.streamManager.JoinRoom(session); err != nil {
		return domain.StreamSession{}, fmt.Errorf("failed to join room: %w", err)
	}

	// Get room details for database storage
	roomNode, err := u.repo.GetNodeByPath(domain.NewPath(roomPath))
	if err != nil {
		// Clean up session on error
		u.streamManager.LeaveRoom(sessionID)
		return domain.StreamSession{}, fmt.Errorf("failed to get room details: %w", err)
	}

	// Broadcast join message
	joinMessage := fmt.Sprintf("joined #%s as %s", roomPath, clientName)
	if err := u.streamManager.BroadcastMessage(roomPath, clientName, joinMessage); err != nil {
		fmt.Printf("Error broadcasting join message: %v\n", err)
	}

	// Save join message to database
	if err := u.repo.CreateMessage(roomNode.ID, clientName, joinMessage); err != nil {
		fmt.Printf("Error saving join message to DB for roomID %d: %v\n", roomNode.ID, err)
	}

	return session, nil
}

// handleTailRequest processes a tail request
func (u *StreamUsecase) handleTailRequest(
	request domain.StreamRequest,
	sessionID, remote, roomPath string,
) (domain.StreamSession, error) {
	// Create tail session (no client name needed for tail mode)
	session := domain.NewStreamSession(sessionID, remote, roomPath, remote, true)

	// Join room (but don't broadcast join message for tail mode)
	if err := u.streamManager.JoinRoom(session); err != nil {
		return domain.StreamSession{}, fmt.Errorf("failed to join room for tail: %w", err)
	}

	return session, nil
}
