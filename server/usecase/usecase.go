package usecase

import (
	"fmt"

	"github.com/ponyo877/chatsh/server/adaptor"
	"github.com/ponyo877/chatsh/server/domain"
)

var (
	messageLimit int = 1000
)

type Usecase struct {
	repo Repository
}

func NewUsecase(repo Repository) adaptor.Usecase {
	return &Usecase{
		repo: repo,
	}
}

func (u Usecase) CheckDirectoryExists(path domain.Path) (bool, error) {
	return u.repo.CheckDirectoryExists(path)
}

func (u Usecase) GetConfig(ownerToken string) (domain.Config, error) {
	config, err := u.repo.GetConfig(ownerToken)
	if err != nil {
		return domain.Config{}, fmt.Errorf("error getting config: %w", err)
	}
	return config, nil
}

func (u Usecase) SetConfig(config domain.Config) error {
	if err := u.repo.CreateConfig(config); err != nil {
		return fmt.Errorf("error setting config: %w", err)
	}
	return nil
}

func (u Usecase) CreateRoom(path domain.Path, ownerToken string) error {
	parentNode, err := u.repo.GetNodeByPath(path.Parent())
	if err != nil {
		return fmt.Errorf("error getting parent room: %w", err)
	}
	return u.repo.CreateRoom(parentNode.ID, path.Parent().String(), path.NodeName(), ownerToken)
}

func (u Usecase) CreateDirectory(path domain.Path, ownerToken string) error {
	parentNode, err := u.repo.GetNodeByPath(path.Parent())
	if err != nil {
		return fmt.Errorf("error getting parent directory: %w", err)
	}
	return u.repo.CreateDirectory(parentNode.ID, path.Parent().String(), path.NodeName(), ownerToken)
}

func (u Usecase) DeletePath(path domain.Path, ownerToken string) error {
	node, err := u.repo.GetNodeByPath(path)
	if err != nil {
		return fmt.Errorf("error getting path: %w", err)
	}
	if node.OwnerToken != ownerToken {
		return fmt.Errorf("permission denied")
	}

	switch node.Type {
	case domain.NodeTypeRoom:
		return u.repo.DeleteRoom(node.ID)
	case domain.NodeTypeDirectory:
		return u.repo.DeleteDirectory(node.ID)
	default:
		return fmt.Errorf("broken node")
	}
}

func (u Usecase) CopyPath(srcPath, dstPath domain.Path, ownerToken string) error {
	srcNode, err := u.repo.GetNodeByPath(srcPath)
	if err != nil {
		return fmt.Errorf("error getting source path: %w", err)
	}
	if srcNode.Type != domain.NodeTypeRoom {
		return fmt.Errorf("source path is not a file")
	}
	if srcNode.OwnerToken != ownerToken {
		return fmt.Errorf("permission denied")
	}
	dstParentNode, err := u.repo.GetNodeByPath(dstPath.Parent())
	if err != nil {
		return fmt.Errorf("error getting destination path: %w", err)
	}
	if dstParentNode.Type != domain.NodeTypeDirectory {
		return fmt.Errorf("source path is not a file")
	}
	dstNode, err := u.repo.GetNodeByPath(dstPath)
	if err != nil && err != ErrNotFound {
		return fmt.Errorf("error getting destination path: %w", err)
	}
	newName := srcNode.Name
	newDstPath := dstPath.String()
	newDstDirID := dstNode.ID

	// dst is room type (not exists)
	if err == ErrNotFound {
		newName = dstPath.NodeName()
		newDstPath = dstPath.Parent().String()
		newDstDirID = dstParentNode.ID
	} else if dstNode.Type != domain.NodeTypeDirectory {
		return fmt.Errorf("destination room is already exists")
	}
	if err := u.repo.CreateExistRoom(srcNode.ID, newDstDirID, newDstPath, newName, ownerToken); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}
	return nil
}

func (u Usecase) MovePath(srcPath, dstPath domain.Path, ownerToken string) error {
	srcNode, err := u.repo.GetNodeByPath(srcPath)
	if err != nil {
		return fmt.Errorf("error getting source path: %w", err)
	}
	if srcNode.Type != domain.NodeTypeRoom {
		return fmt.Errorf("source path is not a file")
	}
	if srcNode.OwnerToken != ownerToken {
		return fmt.Errorf("permission denied")
	}
	dstParentNode, err := u.repo.GetNodeByPath(dstPath.Parent())
	if err != nil {
		return fmt.Errorf("error getting destination path: %w", err)
	}
	if dstParentNode.Type != domain.NodeTypeDirectory {
		return fmt.Errorf("source path is not a file")
	}
	dstNode, err := u.repo.GetNodeByPath(dstPath)
	if err != nil && err != ErrNotFound {
		return fmt.Errorf("error getting destination path: %w", err)
	}
	newName := srcNode.Name
	newDstPath := dstPath.String()
	newDstDirID := dstNode.ID

	// dst is room type (not exists)
	if err == ErrNotFound {
		newName = dstPath.NodeName()
		newDstPath = dstPath.Parent().String()
		newDstDirID = dstParentNode.ID
	} else if dstNode.Type != domain.NodeTypeDirectory {
		return fmt.Errorf("destination room is already exists")
	}
	if err := u.repo.UpdateRoom(srcNode.ID, newDstDirID, newDstPath, newName); err != nil {
		return fmt.Errorf("error moving file: %w", err)
	}
	return nil

}

func (u Usecase) ListNodes(path domain.Path) ([]domain.Node, error) {
	node, err := u.repo.GetNodeByPath(path)
	if err != nil {
		return nil, fmt.Errorf("error getting path: %w", err)
	}
	if node.Type == domain.NodeTypeRoom {
		return []domain.Node{node}, nil
	}
	nodes, err := u.repo.ListNodes(node.ID)
	if err != nil {
		return nil, fmt.Errorf("error listing nodes: %w", err)
	}
	return nodes, nil
}

func (u Usecase) ListMessage(path domain.Path) ([]domain.Message, error) {
	node, err := u.repo.GetNodeByPath(path)
	if err != nil {
		return nil, fmt.Errorf("error getting room: %w", err)
	}
	if node.Type != domain.NodeTypeRoom {
		return nil, fmt.Errorf("path is not a room")
	}
	messages, err := u.repo.ListMessages(node.ID, messageLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("error listing messages: %w", err)
	}
	return messages, nil
}

func (u Usecase) StreamMessage() error {
	// TODO: not implemented
	return nil
}

func (u Usecase) SearchMessage(path domain.Path, pattern string) ([]domain.Message, error) {
	node, err := u.repo.GetNodeByPath(path)
	if err != nil {
		return nil, fmt.Errorf("error getting room: %w", err)
	}
	if node.Type != domain.NodeTypeRoom {
		return nil, fmt.Errorf("path is not a room")
	}
	messages, err := u.repo.ListMessagesByQuery(node.ID, pattern)
	if err != nil {
		return nil, fmt.Errorf("error searching messages: %w", err)
	}
	return messages, nil
}

func (u Usecase) WriteMessage(path domain.Path, message, ownerToken string) error {
	node, err := u.repo.GetNodeByPath(path)
	if err != nil {
		return fmt.Errorf("error getting room: %w", err)
	}
	if node.Type != domain.NodeTypeRoom {
		return fmt.Errorf("path is not a room")
	}
	if err := u.repo.CreateMessage(node.ID, ownerToken, message); err != nil {
		return fmt.Errorf("error writing message: %w", err)
	}
	return nil
}
