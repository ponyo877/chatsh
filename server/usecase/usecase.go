package usecase

import (
	"fmt"
	"strings"
	"sync"

	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/ponyo877/chatsh/server/adaptor"
	"github.com/ponyo877/chatsh/server/domain"
	"google.golang.org/grpc/peer"
)

var (
	messageLimit int = 1000
)

type Usecase struct {
	repo  Repository
	rooms sync.Map
}

func NewUsecase(repo Repository) adaptor.Usecase {
	return Usecase{
		repo:  repo,
		rooms: sync.Map{},
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

const (
	addr        = ":9000"
	ringSize    = 1024
	defaultRoom = "lobby"
)

type Room struct {
	mu        sync.RWMutex
	clients   map[*clientStream]struct{}
	broadcast chan *pb.ServerMessage
}

func newRoom() *Room {
	r := &Room{
		clients:   make(map[*clientStream]struct{}),
		broadcast: make(chan *pb.ServerMessage, ringSize),
	}
	go r.fanout()
	return r
}

func (r *Room) fanout() {
	for msg := range r.broadcast {
		r.mu.RLock()
		for c := range r.clients {
			select {
			case c.out <- msg:
			default:
			}
		}
		r.mu.RUnlock()
	}
}

func (r *Room) add(c *clientStream) {
	r.mu.Lock()
	r.clients[c] = struct{}{}
	r.mu.Unlock()
}

func (r *Room) remove(c *clientStream) {
	r.mu.Lock()
	delete(r.clients, c)
	r.mu.Unlock()
}

func (r *Room) publish(name, text string) {
	msg := &pb.ServerMessage{Name: name, Text: text}
	select {
	case r.broadcast <- msg:
	default:
		<-r.broadcast
		r.broadcast <- msg
	}
}

type clientStream struct {
	name string
	room *Room
	out  chan *pb.ServerMessage
}

type chatService struct {
	pb.UnimplementedChatshServiceServer

	rooms sync.Map
}

func (u Usecase) getRoom(name string) *Room {
	if v, ok := u.rooms.Load(name); ok {
		return v.(*Room)
	}
	r := newRoom()
	actual, _ := u.rooms.LoadOrStore(name, r)
	return actual.(*Room)
}

func (u Usecase) StreamMessage(stream pb.ChatshService_StreamMessageServer) error {
	p, _ := peer.FromContext(stream.Context())
	remote := p.Addr.String()

	first, err := stream.Recv()
	if err != nil {
		return err
	}
	join := first.GetJoin()
	if join == nil {
		return fmt.Errorf("first message must be Join")
	}
	if join.Name == "" {
		join.Name = remote
	}
	if join.Room == "" {
		join.Room = defaultRoom
	}

	room := u.getRoom(join.Room)
	cli := &clientStream{name: join.Name, room: room, out: make(chan *pb.ServerMessage, 32)}
	room.add(cli)
	defer room.remove(cli)

	room.publish(cli.name, fmt.Sprintf("🟢 joined #%s", join.Room))
	defer room.publish(cli.name, fmt.Sprintf("🔴 left #%s", join.Room))

	go func() {
		for msg := range cli.out {
			_ = stream.Send(msg)
		}
	}()

	for {
		in, err := stream.Recv()
		if err != nil {
			return nil
		}
		if chat := in.GetChat(); chat != nil {
			txt := strings.TrimSpace(chat.Text)
			if txt == "" {
				continue
			}
			room.publish(cli.name, txt)
		}
	}
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
