package adaptor

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/ponyo877/chatsh/server/domain"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Adaptor struct {
	uc Usecase
	pb.UnimplementedChatshServiceServer
}

func NewAdaptor(uc Usecase) *Adaptor {
	return &Adaptor{uc: uc}
}

func toPbNodeInfo(node domain.Node) *pb.NodeInfo {
	var nodeType pb.NodeType
	switch node.Type {
	case domain.NodeTypeDirectory:
		nodeType = pb.NodeType_DIRECTORY
	case domain.NodeTypeRoom:
		nodeType = pb.NodeType_ROOM
	default:
		nodeType = pb.NodeType_UNKNOWN
	}
	return &pb.NodeInfo{
		Name:      node.Name,
		OwnerName: node.OwnerName,
		Type:      nodeType,
		Modified:  timestamppb.New(node.CreatedAt),
	}
}

func (a *Adaptor) GetConfig(ctx context.Context, in *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	config, err := a.uc.GetConfig(in.GetOwnerToken())
	if err != nil {
		log.Printf("Error getting config: %v", err)
		return nil, err
	}
	return &pb.GetConfigResponse{
		DisplayName: config.DisplayName,
	}, nil
}

func (a *Adaptor) CheckDirectoryExists(ctx context.Context, in *pb.CheckDirectoryExistsRequest) (*pb.CheckDirectoryExistsResponse, error) {
	exists, err := a.uc.CheckDirectoryExists(domain.NewPath(in.GetPath()))
	if err != nil {
		log.Printf("Error checking directory existence: %v", err)
		return nil, err
	}
	return &pb.CheckDirectoryExistsResponse{Exists: exists}, nil
}

func (a *Adaptor) SetConfig(ctx context.Context, in *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
	config := domain.NewConfig(in.GetDisplayName(), in.GetOwnerToken())
	if err := a.uc.SetConfig(config); err != nil {
		log.Printf("Error setting config: %v", err)
		return &pb.SetConfigResponse{Status: &pb.Status{Ok: false, Message: err.Error()}}, nil
	}
	return &pb.SetConfigResponse{Status: &pb.Status{Ok: true}}, nil
}

func (a *Adaptor) CreateRoom(ctx context.Context, in *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	err := a.uc.CreateRoom(domain.NewPath(in.GetPath()), in.GetOwnerToken())
	if err != nil {
		log.Printf("Error creating room: %v", err)
		return &pb.CreateRoomResponse{Status: &pb.Status{Ok: false, Message: err.Error()}}, nil
	}
	return &pb.CreateRoomResponse{Status: &pb.Status{Ok: true}}, nil
}

func (a *Adaptor) CreateDirectory(ctx context.Context, in *pb.CreateDirectoryRequest) (*pb.CreateDirectoryResponse, error) {
	err := a.uc.CreateDirectory(domain.NewPath(in.GetPath()), in.GetOwnerToken())
	if err != nil {
		log.Printf("Error creating directory: %v", err)
		return &pb.CreateDirectoryResponse{Status: &pb.Status{Ok: false, Message: err.Error()}}, nil
	}
	return &pb.CreateDirectoryResponse{Status: &pb.Status{Ok: true}}, nil
}

func (a *Adaptor) DeletePath(ctx context.Context, in *pb.DeletePathRequest) (*pb.DeletePathResponse, error) {
	err := a.uc.DeletePath(domain.NewPath(in.GetPath()), in.GetOwnerToken())
	if err != nil {
		log.Printf("Error deleting path: %v", err)
		return &pb.DeletePathResponse{Status: &pb.Status{Ok: false, Message: err.Error()}}, nil
	}
	return &pb.DeletePathResponse{Status: &pb.Status{Ok: true}}, nil
}

func (a *Adaptor) CopyPath(ctx context.Context, in *pb.CopyPathRequest) (*pb.CopyPathResponse, error) {
	err := a.uc.CopyPath(domain.NewPath(in.GetSourcePath()), domain.NewPath(in.GetDestinationPath()), in.GetOwnerToken())
	if err != nil {
		log.Printf("Error copying path: %v", err)
		return &pb.CopyPathResponse{Status: &pb.Status{Ok: false, Message: err.Error()}}, nil
	}
	return &pb.CopyPathResponse{Status: &pb.Status{Ok: true}}, nil
}

func (a *Adaptor) MovePath(ctx context.Context, in *pb.MovePathRequest) (*pb.MovePathResponse, error) {
	err := a.uc.MovePath(domain.NewPath(in.GetSourcePath()), domain.NewPath(in.GetDestinationPath()), in.GetOwnerToken())
	if err != nil {
		log.Printf("Error moving path: %v", err)
		return &pb.MovePathResponse{Status: &pb.Status{Ok: false, Message: err.Error()}}, nil
	}
	return &pb.MovePathResponse{Status: &pb.Status{Ok: true}}, nil
}

func (a *Adaptor) ListNodes(ctx context.Context, in *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	nodes, err := a.uc.ListNodes(domain.NewPath(in.GetPath()))
	if err != nil {
		log.Printf("Error listing nodes: %v", err)
		return nil, err
	}

	pbNodeInfos := make([]*pb.NodeInfo, len(nodes))
	for i, node := range nodes {
		pbNodeInfos[i] = toPbNodeInfo(node)
	}
	return &pb.ListNodesResponse{Entries: pbNodeInfos}, nil
}

func (a *Adaptor) SearchMessage(ctx context.Context, in *pb.SearchMessageRequest) (*pb.SearchMessageResponse, error) {
	messages, err := a.uc.SearchMessage(domain.NewPath(in.GetPath()), in.GetPattern())
	if err != nil {
		log.Printf("Error searching messages: %v", err)
		return nil, err
	}

	pbMessages := make([]*pb.Message, len(messages))
	for i, message := range messages {
		pbMessages[i] = &pb.Message{
			TextContent: message.Content,
			OwnerName:   message.DisplayName,
			Created:     timestamppb.New(message.CreatedAt),
		}
	}
	return &pb.SearchMessageResponse{Messages: pbMessages}, nil
}

func (a *Adaptor) WriteMessage(ctx context.Context, in *pb.WriteMessageRequest) (*pb.WriteMessageResponse, error) {
	err := a.uc.WriteMessage(domain.NewPath(in.GetDestinationPath()), in.GetTextContent(), in.GetOwnerToken())
	if err != nil {
		log.Printf("Error writing message: %v", err)
		return &pb.WriteMessageResponse{Status: &pb.Status{Ok: false, Message: err.Error()}}, nil
	}
	return &pb.WriteMessageResponse{Status: &pb.Status{Ok: true}}, nil
}

func (a *Adaptor) StreamMessage(stream pb.ChatshService_StreamMessageServer) error {

	p, _ := peer.FromContext(stream.Context())
	remote := p.Addr.String()

	sessionID := fmt.Sprintf("%s-%d", remote, time.Now().UnixNano())

	requestChan := make(chan domain.StreamRequest, 32)
	responseChan := make(chan domain.StreamResponse, 32)

	usecaseErr := make(chan error, 1)
	go func() {
		defer close(responseChan)
		if err := a.uc.HandleStreamSession(requestChan, responseChan, sessionID, remote); err != nil {
			usecaseErr <- err
		}
	}()

	responseErr := make(chan error, 1)
	go func() {
		for response := range responseChan {
			if response.IsError() {
				log.Printf("StreamMessage: error response: %v", response.Error)
				continue
			}

			pbMessage := &pb.ServerMessage{
				Name: response.Name,
				Text: response.Message,
			}

			if err := stream.Send(pbMessage); err != nil {
				responseErr <- fmt.Errorf("failed to send response: %w", err)
				return
			}
		}
	}()

	defer close(requestChan)

	for {
		in, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("Client %s disconnected normally", sessionID)
			} else {
				log.Printf("Client %s disconnected with error: %v", sessionID, err)
			}
			break
		}

		domainRequest, err := a.convertPbToDomainRequest(in)
		if err != nil {
			log.Printf("StreamMessage: failed to convert request: %v", err)
			continue
		}

		select {
		case requestChan <- domainRequest:
		case <-stream.Context().Done():
			return stream.Context().Err()
		case err := <-usecaseErr:
			return err
		case err := <-responseErr:
			return err
		}
	}

	select {
	case err := <-usecaseErr:
		if err != nil {
			log.Printf("StreamMessage: usecase error: %v", err)
			return err
		}
	case err := <-responseErr:
		if err != nil {
			log.Printf("StreamMessage: response error: %v", err)
			return err
		}
	case <-stream.Context().Done():
		return stream.Context().Err()
	}

	return nil
}

func (a *Adaptor) convertPbToDomainRequest(in *pb.ClientMessage) (domain.StreamRequest, error) {
	if join := in.GetJoin(); join != nil {
		return domain.NewJoinRequest(join.GetName(), join.GetRoom()), nil
	}

	if tail := in.GetTail(); tail != nil {
		return domain.NewTailRequest(tail.GetRoomPath()), nil
	}

	if chat := in.GetChat(); chat != nil {
		return domain.NewChatRequest(chat.GetText()), nil
	}

	return domain.StreamRequest{}, fmt.Errorf("unknown request type")
}

func (a *Adaptor) ListMessages(ctx context.Context, in *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	messages, err := a.uc.ListMessages(domain.NewPath(in.GetRoomPath()), in.GetLimit())
	if err != nil {
		log.Printf("Error getting past messages for room %s: %v", in.GetRoomPath(), err)
		return nil, err
	}

	pbMessages := make([]*pb.Message, len(messages))
	for i, message := range messages {
		pbMessages[i] = &pb.Message{
			TextContent: message.Content,
			OwnerName:   message.DisplayName,
			Created:     timestamppb.New(message.CreatedAt),
		}
	}
	return &pb.ListMessagesResponse{Messages: pbMessages}, nil
}
