package adaptor

import (
	"context" // Add errors package
	"log"     // Add log package

	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/ponyo877/chatsh/server/domain"           // Add domain package
	"google.golang.org/protobuf/types/known/timestamppb" // Add timestamppb package
)

type Adaptor struct {
	pb.UnimplementedFileSystemServiceServer // Embed UnimplementedFileSystemServiceServer
	uc                                      Usecase
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
		pbNodeInfos[i] = toPbNodeInfo(node) // Use the corrected helper
	}
	return &pb.ListNodesResponse{Entries: pbNodeInfos}, nil
}

func (a *Adaptor) ListMessage(ctx context.Context, in *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	messages, err := a.uc.ListMessage(domain.NewPath(in.GetPath()))
	if err != nil {
		log.Printf("Error listing messages: %v", err)
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
