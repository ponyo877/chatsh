package usecase

import (
	"errors"

	"github.com/ponyo877/chatsh/server/domain"
)

type Repository interface {
	// Config
	GetConfig(ownerToken string) (domain.Config, error)
	CreateConfig(config domain.Config) error

	// Node (Directory & Room)
	GetNodeByPath(path domain.Path) (domain.Node, error)
	ListNodes(parentDirID int) ([]domain.Node, error)
	CheckDirectoryExists(path domain.Path) (bool, error)

	// Directory
	CreateDirectory(parentDirID int, parentDirPath, name, ownerToken string) error
	DeleteDirectory(dirID int) error
	UpdateDirectory(srcDirID, dstDirID int, dstDirPath, name string) error

	// Room
	CreateRoom(parentDirID int, parentDirPath, name, ownerToken string) error
	CreateExistRoom(roomID, dstDirID int, dstDirPath, name, ownerToken string) error
	DeleteRoom(roomID int) error
	UpdateRoom(srcRoomID, dstDirID int, dstDirPath, name string) error

	// Message
	CreateMessage(roomID int, displayName, message string) error
	ListMessages(roomID, limit, offset int) ([]domain.Message, error)
	ListMessagesByQuery(roomID int, pattern string) ([]domain.Message, error)
}

var ErrNotFound = errors.New("not found")

var ErrAlreadyExists = errors.New("already exists")
