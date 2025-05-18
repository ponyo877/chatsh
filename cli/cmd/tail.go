package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"slices"
	"time"

	pb "github.com/ponyo877/chatsh/grpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tailCmd = &cobra.Command{
	Use:   "tail [room_path]",
	Short: "Continuously stream messages from a room",
	Long:  `Tails a chat room, displaying new messages as they arrive. Does not send messages.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pathArg := args[0]
		currentBaseDir := viper.GetString(currentDirectoryKey)
		if currentBaseDir == "" {
			currentBaseDir = viper.GetString(homeDirectoryKey)
		}

		var targetPath string
		if filepath.IsAbs(pathArg) {
			targetPath = pathArg
		} else {
			targetPath = filepath.Join(currentBaseDir, pathArg)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		pastMessagesLimit := int32(10)
		listReq := &pb.ListMessagesRequest{RoomPath: targetPath, Limit: pastMessagesLimit}
		pastMsgsResp, err := chatshClient.ListMessages(ctx, listReq)
		if err != nil {
			log.Printf("Warning: Failed to load past messages for %s: %v", targetPath, err)
		}
		if len(pastMsgsResp.Messages) > 0 {
			for _, msg := range slices.Backward(pastMsgsResp.Messages) {
				fmt.Printf("[%s] %s: %s\n",
					msg.GetCreated().AsTime().Format("15:04:05"),
					msg.GetOwnerName(),
					msg.GetTextContent())
			}
		}

		// Start streaming new messages
		stream, err := chatshClient.StreamMessage(ctx)
		if err != nil {
			log.Fatalf("StreamMessage failed: %v", err)
		}

		// Send TailRequest
		tailReq := &pb.ClientMessage{
			Payload: &pb.ClientMessage_Tail{
				Tail: &pb.Tail{RoomPath: targetPath},
			},
		}
		if err := stream.Send(tailReq); err != nil {
			log.Fatalf("Failed to send tail request: %v", err)
		}
		if err := stream.CloseSend(); err != nil {
			log.Printf("Failed to close send stream: %v", err)
		}

		// Receive and display new messages
		for {
			select {
			case <-ctx.Done():
				return // Exit if context is cancelled (e.g., by Ctrl+C)
			default:
				serverMsg, err := stream.Recv()
				if err == io.EOF {
					fmt.Println("Stream closed by server.")
					return
				}
				if err != nil {
					if ctx.Err() == context.Canceled {
						return
					}
					log.Printf("Error receiving message: %v", err)
					return
				}
				fmt.Printf("[%s] %s: %s\n",
					time.Now().Format("15:04:05"),
					serverMsg.GetName(),
					serverMsg.GetText())
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(tailCmd)
	// No flags needed for tail for now, room_path is an argument.
}
