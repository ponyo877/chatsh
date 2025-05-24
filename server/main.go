package main

import (
	"database/sql"
	"fmt" // ioutilを追加
	"log"
	"net"
	"os"
	"regexp"
	"strings" // stringsを追加

	"github.com/mattn/go-sqlite3"
	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/ponyo877/chatsh/server/adaptor"
	"github.com/ponyo877/chatsh/server/repository"
	"github.com/ponyo877/chatsh/server/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func regex(re, s string) (bool, error) {
	return regexp.MatchString(re, s)
}

func main() {
	portEnv := os.Getenv("PORT")
	if portEnv == "" {
		portEnv = "50051"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", portEnv))
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

	// Enable WAL mode for better concurrency
	// if _, err := conn.Exec("PRAGMA journal_mode=WAL;"); err != nil {
	// 	log.Fatalf("failed to set WAL mode: %v", err)
	// }

	rows, err := conn.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='users';")
	if err != nil {
		log.Fatalf("failed to query sqlite_master: %v", err)
	}
	defer rows.Close()

	tableExists := false
	if rows.Next() {
		tableExists = true
	}

	if !tableExists {
		log.Println("Users table not found, initializing database from schema/chatsh.sql...")
		schemaContent, err := os.ReadFile("./schema/chatsh.sql")
		if err != nil {
			log.Fatalf("failed to read schema/chatsh.sql: %v", err)
		}

		statements := strings.Split(string(schemaContent), ";")
		for _, stmt := range statements {
			trimmedStmt := strings.TrimSpace(stmt)
			if trimmedStmt == "" {
				continue
			}
			_, err := conn.Exec(trimmedStmt)
			if err != nil {
				log.Fatalf("failed to execute schema statement: %s, error: %v", trimmedStmt, err)
			}
		}
		log.Println("Database initialized successfully.")
	}

	rp := repository.NewRepository(conn)
	uc := usecase.NewUsecase(rp)
	ad := adaptor.NewAdaptor(uc)
	s := grpc.NewServer(
		grpc.MaxConcurrentStreams(1000),
		grpc.NumStreamWorkers(10),
	)
	pb.RegisterChatshServiceServer(s, ad)
	reflection.Register(s)

	log.Printf("Server is running on port %s", portEnv)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
