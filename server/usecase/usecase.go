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
	return &Usecase{
		repo:  repo,
		rooms: sync.Map{},
	}
}

func (u *Usecase) CheckDirectoryExists(path domain.Path) (bool, error) {
	return u.repo.CheckDirectoryExists(path)
}

func (u *Usecase) GetConfig(ownerToken string) (domain.Config, error) {
	config, err := u.repo.GetConfig(ownerToken)
	if err != nil {
		return domain.Config{}, fmt.Errorf("error getting config: %w", err)
	}
	return config, nil
}

func (u *Usecase) SetConfig(config domain.Config) error {
	if err := u.repo.CreateConfig(config); err != nil {
		return fmt.Errorf("error setting config: %w", err)
	}
	return nil
}

func (u *Usecase) ListMessages(path domain.Path, limit int32) ([]domain.Message, error) {
	node, err := u.repo.GetNodeByPath(path)
	if err != nil {
		return nil, fmt.Errorf("error getting room: %w", err)
	}
	if node.Type != domain.NodeTypeRoom {
		return nil, fmt.Errorf("path '%s' is not a room", path)
	}

	messages, err := u.repo.ListMessages(node.ID, int(limit), 0)
	if err != nil {
		return nil, fmt.Errorf("error listing messages for roomID %d: %w", node.ID, err)
	}
	return messages, nil
}

func (u *Usecase) CreateRoom(path domain.Path, ownerToken string) error {
	parentNode, err := u.repo.GetNodeByPath(path.Parent())
	if err != nil {
		return fmt.Errorf("error getting parent room: %w", err)
	}
	return u.repo.CreateRoom(parentNode.ID, path.Parent().String(), path.NodeName(), ownerToken)
}

func (u *Usecase) CreateDirectory(path domain.Path, ownerToken string) error {
	parentNode, err := u.repo.GetNodeByPath(path.Parent())
	if err != nil {
		return fmt.Errorf("error getting parent directory: %w", err)
	}
	return u.repo.CreateDirectory(parentNode.ID, path.Parent().String(), path.NodeName(), ownerToken)
}

func (u *Usecase) DeletePath(path domain.Path, ownerToken string) error {
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

func (u *Usecase) CopyPath(srcPath, dstPath domain.Path, ownerToken string) error {
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

func (u *Usecase) MovePath(srcPath, dstPath domain.Path, ownerToken string) error {
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

func (u *Usecase) ListNodes(path domain.Path) ([]domain.Node, error) {
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

const (
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

func (u *Usecase) getRoom(name string) *Room {
	if v, ok := u.rooms.Load(name); ok {
		return v.(*Room)
	}
	r := newRoom()
	actual, _ := u.rooms.LoadOrStore(name, r)
	return actual.(*Room)
}

func (u *Usecase) StreamMessage(stream pb.ChatshService_StreamMessageServer) error {
	p, _ := peer.FromContext(stream.Context())
	remote := p.Addr.String()

	first, err := stream.Recv()
	if err != nil {
		return err
	}

	var clientName string
	var targetRoomPath string
	var roomID int
	var isTailMode bool = false

	if join := first.GetJoin(); join != nil {
		clientName = join.GetName()
		if clientName == "" {
			clientName = remote
		}
		targetRoomPath = join.GetRoom()
		if targetRoomPath == "" {
			targetRoomPath = defaultRoom
		}
		isTailMode = false
	} else if tailReq := first.GetTail(); tailReq != nil {
		targetRoomPath = tailReq.GetRoomPath()
		if targetRoomPath == "" {
			return fmt.Errorf("TailRequest must specify a room_path")
		}
		clientName = "tail_client_" + remote
		isTailMode = true
	} else {
		return fmt.Errorf("first message must be Join or TailRequest")
	}

	roomNode, err := u.repo.GetNodeByPath(domain.NewPath(targetRoomPath))
	if err != nil {
		return fmt.Errorf("failed to get room details from DB for '%s': %w", targetRoomPath, err)
	}
	if roomNode.Type != domain.NodeTypeRoom {
		return fmt.Errorf("path '%s' is not a room", targetRoomPath)
	}
	roomID = roomNode.ID

	room := u.getRoom(targetRoomPath)
	cli := &clientStream{name: clientName, room: room, out: make(chan *pb.ServerMessage, 32)}
	room.add(cli)
	defer room.remove(cli)

	go func() {
		for msg := range cli.out {
			if errSend := stream.Send(msg); errSend != nil {
				fmt.Printf("error sending message to %s: %v\n", cli.name, errSend)
				return
			}
		}
	}()

	if !isTailMode {
		joinText := fmt.Sprintf("joined #%s as %s", targetRoomPath, cli.name)
		room.publish(cli.name, joinText)
		if errDb := u.repo.CreateMessage(roomID, cli.name, joinText); errDb != nil {
			fmt.Printf("Error saving join message to DB for roomID %d: %v\n", roomID, errDb)
		}

		defer func() {
			leftText := fmt.Sprintf("left #%s", targetRoomPath)
			room.publish(cli.name, leftText)
			if errDb := u.repo.CreateMessage(roomID, cli.name, leftText); errDb != nil {
				fmt.Printf("Error saving left message to DB for roomID %d: %v\n", roomID, errDb)
			}
		}()
	}

	for {
		if isTailMode {
			fmt.Printf("Received unexpected message from tail client %s\n", cli.name)
			continue
		}
		in, err := stream.Recv()
		if err != nil {
			fmt.Printf("Client %s disconnected: %v\n", cli.name, err)
			return nil
		}

		if chat := in.GetChat(); chat != nil {
			txt := strings.TrimSpace(chat.GetText())
			if txt == "" {
				continue
			}
			room.publish(cli.name, txt)
			if errDb := u.repo.CreateMessage(roomID, cli.name, txt); errDb != nil {
				fmt.Printf("Error saving chat message to DB for roomID %d: %v\n", roomID, errDb)
			}
		}
	}
}

func (u *Usecase) SearchMessage(path domain.Path, pattern string) ([]domain.Message, error) {
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

func (u *Usecase) WriteMessage(path domain.Path, message, ownerToken string) error {
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
