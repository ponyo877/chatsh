package usecase

import (
	"github.com/ponyo877/chatsh/server/domain"
)

type Repository interface {
	// Node (Directory & Room)
	GetNodeByPath(path domain.Path) (domain.Node, error)
	ListNodes(parentDirID int) ([]domain.Node, error)

	// Directory
	CreateDirectory(parentDirID int, parentDirPath, name, ownerToken string) error
	DeleteDirectory(dirID int) error
	UpdateDirectory(srcDirID, dstDirID int, dstDirPath, name string) error

	// Room
	CreateRoom(parentDirID int, parentDirPath, name, ownerToken string) error
	CreateExistRoom(roomID, dstDirID int, dstDirPath, name string) error
	DeleteRoom(roomID int) error
	UpdateRoom(srcRoomID, dstDirID int, dstDirPath, name string) error

	// Message
	CreateMessage(roomID int, displayName, message string) error
	ListMessages(roomID, limit, offset int) ([]domain.Message, error)
	ListMessagesByQuery(roomID int, pattern string) ([]domain.Message, error)
}
