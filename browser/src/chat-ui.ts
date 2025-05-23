import { Terminal } from 'xterm';
import { grpcClient } from './grpc-client';

export interface ChatMessage {
    name: string;
    text: string;
    timestamp: Date;
}

export class ChatUI {
    private term: Terminal;
    private roomPath: string;
    private userName: string;
    private ownerToken: string;
    private messages: ChatMessage[] = [];
    private isActive: boolean = false;
    private currentInput: string = '';
    private messageDisplayHeight: number = 0;
    private inputLine: number = 0;

    constructor(term: Terminal, roomPath: string, userName: string, ownerToken: string) {
        this.term = term;
        this.roomPath = roomPath;
        this.userName = userName;
        this.ownerToken = ownerToken;
    }

    async start(): Promise<void> {
        this.isActive = true;

        // Clear terminal and setup chat UI
        this.term.clear();
        this.setupChatInterface();

        // Load past messages
        await this.loadPastMessages();

        // Start message polling (mock for now)
        this.startMessagePolling();

        // Setup input handling
        this.setupInputHandling();

        // Show initial interface
        this.renderMessages();
        this.renderInputLine();
    }

    private setupChatInterface(): void {
        const rows = this.term.rows;
        this.messageDisplayHeight = rows - 3; // Reserve 3 lines for input area
        this.inputLine = rows - 2;

        // Draw header
        this.term.writeln(`\x1b[36m=== Chat Room: ${this.roomPath} ===\x1b[0m`);
        this.term.writeln(`\x1b[90mPress Ctrl+C to exit chat mode\x1b[0m`);
        this.term.writeln(''); // Empty line separator
    }

    private async loadPastMessages(): Promise<void> {
        try {
            // Mock past messages for now
            const mockMessages: ChatMessage[] = [
                {
                    name: 'alice',
                    text: 'Hello everyone!',
                    timestamp: new Date(Date.now() - 300000) // 5 minutes ago
                },
                {
                    name: 'bob',
                    text: 'Hey Alice! How are you?',
                    timestamp: new Date(Date.now() - 240000) // 4 minutes ago
                },
                {
                    name: 'alice',
                    text: 'I\'m doing great, thanks for asking!',
                    timestamp: new Date(Date.now() - 180000) // 3 minutes ago
                }
            ];

            this.messages = mockMessages;
        } catch (error) {
            this.addSystemMessage(`Error loading past messages: ${error}`);
        }
    }

    private startMessagePolling(): void {
        // Mock message polling - in real implementation, this would use gRPC streaming
        const pollInterval = setInterval(() => {
            if (!this.isActive) {
                clearInterval(pollInterval);
                return;
            }

            // Simulate receiving a message occasionally
            if (Math.random() < 0.1) { // 10% chance every poll
                const mockUsers = ['charlie', 'diana', 'eve'];
                const mockTexts = [
                    'Anyone working on the new feature?',
                    'Just pushed some updates',
                    'Great work everyone!',
                    'Let\'s discuss this in the meeting',
                    'I found a bug in the latest build'
                ];

                const randomUser = mockUsers[Math.floor(Math.random() * mockUsers.length)];
                const randomText = mockTexts[Math.floor(Math.random() * mockTexts.length)];

                if (randomUser !== this.userName) { // Don't simulate messages from current user
                    this.addMessage({
                        name: randomUser,
                        text: randomText,
                        timestamp: new Date()
                    });
                }
            }
        }, 2000); // Poll every 2 seconds
    }

    private setupInputHandling(): void {
        this.term.onKey(({ key, domEvent }) => {
            if (!this.isActive) return;

            if (domEvent.ctrlKey && domEvent.key === 'c') {
                this.exit();
                return;
            }

            const printable = !domEvent.altKey && !domEvent.ctrlKey && !domEvent.metaKey;

            if (domEvent.key === 'Enter') {
                this.sendMessage();
            } else if (domEvent.key === 'Backspace') {
                if (this.currentInput.length > 0) {
                    this.currentInput = this.currentInput.slice(0, -1);
                    this.renderInputLine();
                }
            } else if (printable && key.length === 1) {
                this.currentInput += key;
                this.renderInputLine();
            }
        });
    }

    private addMessage(message: ChatMessage): void {
        this.messages.push(message);

        // Keep only recent messages to prevent memory issues
        if (this.messages.length > 100) {
            this.messages = this.messages.slice(-100);
        }

        this.renderMessages();
        this.renderInputLine();
    }

    private addSystemMessage(text: string): void {
        this.addMessage({
            name: 'system',
            text: text,
            timestamp: new Date()
        });
    }

    private async sendMessage(): Promise<void> {
        if (this.currentInput.trim() === '') return;

        const messageText = this.currentInput.trim();
        this.currentInput = '';

        // Add message locally first
        const newMessage: ChatMessage = {
            name: this.userName,
            text: messageText,
            timestamp: new Date()
        };

        this.addMessage(newMessage);

        // Send to server (mock for now)
        try {
            // In real implementation, this would send via gRPC
            // await grpcClient.sendMessage({
            //     room_path: this.roomPath,
            //     text: messageText,
            //     owner_token: this.ownerToken
            // });
        } catch (error) {
            this.addSystemMessage(`Failed to send message: ${error}`);
        }
    }

    private renderMessages(): void {
        // Calculate how many messages we can display
        const availableLines = this.messageDisplayHeight - 3; // Account for header
        const messagesToShow = this.messages.slice(-availableLines);

        // Clear message area (preserve header)
        for (let i = 3; i < this.messageDisplayHeight + 3; i++) {
            this.term.write(`\x1b[${i + 1};1H\x1b[K`); // Move to line and clear
        }

        // Render messages
        let lineOffset = 3; // Start after header
        messagesToShow.forEach((message, index) => {
            const timestamp = message.timestamp.toLocaleTimeString('en-US', {
                hour12: false,
                hour: '2-digit',
                minute: '2-digit'
            });

            let nameColor = '\x1b[32m'; // Default green
            if (message.name === this.userName) {
                nameColor = '\x1b[33m'; // Yellow for own messages
            } else if (message.name === 'system') {
                nameColor = '\x1b[31m'; // Red for system messages
            }

            const line = `\x1b[90m[${timestamp}]\x1b[0m ${nameColor}${message.name}\x1b[0m: ${message.text}`;

            this.term.write(`\x1b[${lineOffset + index + 1};1H${line}`);
        });
    }

    private renderInputLine(): void {
        const inputLineNum = this.inputLine + 1;

        // Clear input area
        this.term.write(`\x1b[${inputLineNum};1H\x1b[K`);
        this.term.write(`\x1b[${inputLineNum + 1};1H\x1b[K`);

        // Draw input prompt and current input
        const prompt = `\x1b[32m${this.userName}\x1b[0m> `;
        this.term.write(`\x1b[${inputLineNum};1H${prompt}${this.currentInput}`);

        // Position cursor at end of input
        const cursorCol = prompt.length - 8 + this.currentInput.length + 1; // -8 for ANSI codes
        this.term.write(`\x1b[${inputLineNum};${cursorCol}H`);
    }

    private exit(): void {
        this.isActive = false;
        this.term.clear();

        // Show exit message
        this.term.writeln('\x1b[36mExited chat mode\x1b[0m');

        // Return control to main application
        // This will be handled by the main application
        if (this.onExit) {
            this.onExit();
        }
    }

    // Callback for when chat UI exits
    public onExit?: () => void;

    public stop(): void {
        this.isActive = false;
    }
}
