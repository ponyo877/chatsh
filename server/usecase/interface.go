package usecase

import (
	"github.com/ponyo877/chatsh/server/domain"
)

type Repository interface {
	// Node (Directory & Room)
	GetNodeByPath(path domain.Path) (domain.Node, error)
	ListNodes(parentDirID int) ([]domain.Node, error)

	// Directory
	CreateDirectory(parentDirID int, name string, ownerID int) error
	DeleteDirectory(dirID int) error
	UpdateDirectory(srcDirID, dstDirID int, name string) error

	// Room
	CreateRoom(parentDirID int, name string) error
	CreateExistRoom(roomID, dstDirID int, name string) error
	DeleteRoom(roomID int) error
	UpdateRoom(srcRoomID, dstDirID int, name string) error

	// Message
	CreateMessage(roomID, userID int, message string) error
	ListMessages(roomID, limit, offset int) ([]domain.Message, error)
	ListMessagesByQuery(roomID int, pattern string) ([]domain.Message, error)
}
