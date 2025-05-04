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
	DeleteRoom(roomID int) error
	UpdateRoom(srcRoomID, dstDirID int, name string) error

	// Message
	ListMessages(roomID, limit, offset int) ([]domain.Message, error)
	ListMessagesByQuery(roomID int, pattern string) ([]domain.Message, error)
}

// UseCase defines the interface for the use case layer.
// Adaptor layer depends on this interface.
type UseCase interface {
	// Example method based on adaptor's needs
	ChangeDirectory(path string) (string, error)
	// Add other methods corresponding to gRPC service methods here
	// e.g., CreateDirectory(path string, parents bool) error
	//       DeletePath(path string, recursive, force bool) error
	//       CopyPath(src, dst string, recursive, overwrite bool) error
	//       MovePath(src, dst string, overwrite bool) error
	//       ListNodes(path string, showAll, longFormat bool) ([]domain.FileInfo, error) // Assuming FileInfo is defined in domain
	//       GetMessage(path string) ([]byte, error)
	//       StreamMessage(ctx context.Context, path string, initialLines uint32, follow bool) (<-chan domain.MessageChunk, error) // Example stream
	//       SearchMessage(ctx context.Context, pattern string, paths []string, recursive, ignoreCase bool) (<-chan domain.MessageMatch, error) // Example stream
	//       WriteMessage(path, content string, appendMode bool) error
	//       GetCurrentDirectory() (string, error) // Added for Pwd/GetCurrentDirectory
	//       CreateFile(path string) error // Added for Touch/CreateFile
}
