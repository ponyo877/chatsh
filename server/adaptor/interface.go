package adaptor

import (
	"github.com/ponyo877/chatsh/server/domain"
)

type Usecase interface {
	CheckDirectoryExists(path domain.Path) (bool, error)
	GetConfig(ownerToken string) (domain.Config, error)
	SetConfig(config domain.Config) error
	CopyPath(srcPath domain.Path, dstPath domain.Path, ownerToken string) error
	CreateRoom(path domain.Path, ownerToken string) error
	CreateDirectory(path domain.Path, ownerToken string) error
	DeletePath(path domain.Path, ownerToken string) error
	ListNodes(path domain.Path) ([]domain.Node, error)
	MovePath(srcPath domain.Path, dstPath domain.Path, ownerToken string) error
	SearchMessage(path domain.Path, pattern string) ([]domain.Message, error)
	HandleStreamSession(
		requestChan <-chan domain.StreamRequest,
		responseChan chan<- domain.StreamResponse,
		sessionID, remote string,
	) error
	WriteMessage(path domain.Path, message, ownerToken string) error
	ListMessages(path domain.Path, limit int32) ([]domain.Message, error)
}
