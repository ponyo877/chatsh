package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"regexp"

	"github.com/mattn/go-sqlite3"
	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/ponyo877/chatsh/server/adaptor"
	"github.com/ponyo877/chatsh/server/repository"
	"github.com/ponyo877/chatsh/server/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port int = 50051
)

func regex(re, s string) (bool, error) {
	return regexp.MatchString(re, s)
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	sql.Register("sqlite3_with_go_func",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("regexp", regex, true)
			},
		})
	conn, err := sql.Open("sqlite3_with_go_func", "./chatsh.db")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer conn.Close()
	rp := repository.NewRepository(conn)
	uc := usecase.NewUsecase(rp)
	ad := adaptor.NewAdaptor(uc)
	s := grpc.NewServer()
	pb.RegisterChatshServiceServer(s, ad)
	reflection.Register(s)

	log.Printf("Server is running on port %d", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
