package adaptor

import "github.com/ponyo877/chatsh/server/domain"

type Usecase interface {
	CopyPath(srcPath domain.Path, dstPath domain.Path, ownerTone string) error
	CreateDirectory(path domain.Path, ownerToken string) error
	DeletePath(path domain.Path, ownerTone string) error
	ListMessage(path domain.Path) ([]domain.Message, error)
	ListNodes(path domain.Path) ([]domain.Node, error)
	MovePath(srcPath domain.Path, dstPath domain.Path, ownerTone string) error
	SearchMessage(path domain.Path, pattern string) ([]domain.Message, error)
	StreamMessage() error
	WriteMessage(path domain.Path, message string, userID int) error
}
