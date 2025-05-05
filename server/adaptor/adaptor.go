package adaptor

import (
	"context"

	pb "github.com/ponyo877/chatsh/grpc"
	"google.golang.org/grpc"
)

type Adaptor struct {
	// pb.UnimplementedFileSystemServiceServer
	uc Usecase
}

func NewAdaptor(uc Usecase) *Adaptor {
	return &Adaptor{uc: uc}
}

func (a *Adaptor) CreateDirectory(ctx context.Context, in *pb.CreateDirectoryRequest) (*pb.CreateDirectoryResponse, error) {
	return nil, nil
}

func (a *Adaptor) DeletePath(ctx context.Context, in *pb.DeletePathRequest) (*pb.DeletePathResponse, error) {
	return nil, nil
}

func (a *Adaptor) CopyPath(ctx context.Context, in *pb.CopyPathRequest) (*pb.CopyPathResponse, error) {
	return nil, nil
}

func (a *Adaptor) MovePath(ctx context.Context, in *pb.MovePathRequest) (*pb.MovePathResponse, error) {
	return nil, nil
}

func (a *Adaptor) ListNodes(ctx context.Context, in *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	return nil, nil
}

func (a *Adaptor) GetMessage(ctx context.Context, in *pb.GetMessageRequest) (*pb.GetMessageResponse, error) {
	return nil, nil
}

func (a *Adaptor) StreamMessage(ctx context.Context, in *pb.StreamMessageRequest) (grpc.ServerStreamingClient[pb.MessageChunk], error) {
	return nil, nil
}

func (a *Adaptor) SearchMessage(ctx context.Context, in *pb.SearchMessageRequest) (grpc.ServerStreamingClient[pb.MessageMatch], error) {
	return nil, nil
}

func (a *Adaptor) WriteMessage(ctx context.Context, in *pb.WriteMessageRequest) (*pb.WriteMessageResponse, error) {
	return nil, nil
}
