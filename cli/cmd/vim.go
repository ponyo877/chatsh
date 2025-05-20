package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	pb "github.com/ponyo877/chatsh/grpc"

	"github.com/gdamore/tcell/v2" // tviewが内部で使用するが、直接は使わないので削除しても良い場合がある
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var vimCmd = &cobra.Command{
	Use:   "vim [room_path]",
	Short: "Starts a chat session in a tview-based interface",
	Long: `Starts a chat session using StreamMessage RPC with a tview-based interface.
You can type messages at the bottom and see the chat history above.`,
	Args: cobra.ExactArgs(1),
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
		userName, _ := cmd.Flags().GetString("name")

		if userName == "" {
			ownerToken := viper.GetString("owner_token")
			if ownerToken == "" {
				fmt.Println("Error: owner_token is not set. Cannot get display name.")
				fmt.Println("Please set owner_token via config file or environment variable, or use the -n flag.")
				os.Exit(1)
			}
			configResp, err_cfg := chatshClient.GetConfig(context.Background(), &pb.GetConfigRequest{OwnerToken: ownerToken})
			if err_cfg != nil {
				fmt.Printf("Error getting config to retrieve display name: %v\n", err_cfg)
				fmt.Println("Please ensure the server is running and owner_token is correct, or use the -n flag.")
				os.Exit(1)
			}
			userName = configResp.GetDisplayName()
			if userName == "" {
				fmt.Println("Error: DisplayName is empty in config. Please set it or use the -n flag.")
				os.Exit(1)
			}
		}

		if err := runChatUITview(chatshClient, userName, targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Chat UI error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(vimCmd)
	vimCmd.Flags().StringP("name", "n", "", "Your name for the chat session (optional, defaults to DisplayName in config)")
}

func runChatUITview(client pb.ChatshServiceClient, userName string, roomPath string) error {
	app := tview.NewApplication()

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true).
		ScrollToEnd()

	inputField := tview.NewInputField().
		SetLabel(userName + " ❯❯ ").
		SetFieldWidth(0).
		SetAcceptanceFunc(tview.InputFieldMaxLength(256))

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, false).
		AddItem(inputField, 1, 0, true)

	app.SetRoot(flex, true).SetFocus(inputField)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load past messages first
	pastMessagesLimit := int32(50)
	// Use ListMessages as per user feedback and current proto definition
	pastMsgsResp, err := client.ListMessages(ctx, &pb.ListMessagesRequest{RoomPath: roomPath, Limit: pastMessagesLimit})
	if err != nil {
		fmt.Fprintf(textView, "[red]Error loading past messages: %v\n", err)
	} else {
		for _, msg := range slices.Backward(pastMsgsResp.Messages) {
			fmt.Fprintf(textView, "[white][%s] [blue]%s[white]: %s\n",
				msg.GetCreated().AsTime().Format("15:04:05"),
				msg.GetOwnerName(),
				msg.GetTextContent())
		}
	}
	textView.ScrollToEnd()

	stream, err := client.StreamMessage(ctx)
	if err != nil {
		return fmt.Errorf("StreamMessage failed: %w", err)
	}

	// Join the room
	joinMsg := &pb.ClientMessage{
		Payload: &pb.ClientMessage_Join{
			Join: &pb.Join{Name: userName, Room: roomPath},
		},
	}
	if err := stream.Send(joinMsg); err != nil {
		return fmt.Errorf("failed to send join message: %w", err)
	}
	fmt.Fprintf(textView, "[green]Welcome to %s! You are %s. (Ctrl+C to exit)\n", roomPath, userName)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				serverMsg, err := stream.Recv()
				if err == io.EOF {
					app.QueueUpdateDraw(func() {
						fmt.Fprintln(textView, "[red]Stream closed by server.")
					})
					cancel()
					return
				}
				if err != nil {
					app.QueueUpdateDraw(func() {
						fmt.Fprintf(textView, "[red]Error receiving message: %v\n", err)
					})
					cancel()
					return
				}
				app.QueueUpdateDraw(func() {
					fmt.Fprintf(textView, "[white][%s] [blue]%s[white]: %s\n",
						time.Now().Format("15:04:05"),
						serverMsg.GetName(),
						serverMsg.GetText())
					textView.ScrollToEnd()
				})
			}
		}
	}()

	// Send messages when Enter is pressed
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := strings.TrimSpace(inputField.GetText())
			if text == "" {
				return
			}

			chatMsg := &pb.ClientMessage{
				Payload: &pb.ClientMessage_Chat{
					Chat: &pb.Chat{Name: userName, Text: text},
				},
			}
			if err := stream.Send(chatMsg); err != nil {
				app.QueueUpdateDraw(func() {
					fmt.Fprintf(textView, "[red]Failed to send message: %v\n", err)
				})
			}
			inputField.SetText("")
		}
	})

	// Logout and exit on Ctrl+C
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			cancel()
			app.Stop()
			return nil
		}
		return event
	})

	if err := app.Run(); err != nil {
		cancel()
		return err
	}

	cancel()
	return nil
}
