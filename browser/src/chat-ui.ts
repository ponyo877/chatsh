import { Terminal } from 'xterm';
// Updated import: ChatMessage is now the primary type for UI, Message from grpc-client is not directly used here.
// We use the ChatMessage interface defined in grpc-client.ts which is already mapped from gRPC types.
import { grpcClient, ChatMessage as AppChatMessage, ListMessagesParams, WriteMessageParams } from './grpc-client';


// Local ChatMessage type for this UI module, can be same as AppChatMessage or adapted
export interface UIMessage {
    name: string;
    text: string;
    timestamp: Date;
}

export class ChatUI {
    private term: Terminal;
    private roomPath: string;
    private userName: string;
    private ownerToken: string;
    private messages: UIMessage[] = []; // Use UIMessage
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
        this.term.clear();
        this.setupChatInterface();
        await this.loadPastMessages();
        this.startMessagePolling();
        this.setupInputHandling();
        this.renderMessages();
        this.renderInputLine();
    }

    private setupChatInterface(): void {
        const rows = this.term.rows;
        this.messageDisplayHeight = rows - 3;
        this.inputLine = rows - 2;
        this.term.writeln(`\x1b[36m=== Chat Room: ${this.roomPath} ===\x1b[0m`);
        this.term.writeln(`\x1b[90mPress Ctrl+C to exit chat mode\x1b[0m`);
        this.term.writeln('');
    }

    private async loadPastMessages(): Promise<void> {
        try {
            const params: ListMessagesParams = {
                roomPath: this.roomPath, // Corrected: roomPath
                limit: 50
            };
            const response = await grpcClient.listMessages(params);

            this.messages = response.messages.map(msg => ({
                name: msg.ownerName,     // Corrected: ownerName
                text: msg.textContent,   // Corrected: textContent
                timestamp: msg.created
            }));

            if (this.messages.length === 0) {
                this.addSystemMessage('No previous messages in this room.');
            }
        } catch (error) {
            this.addSystemMessage(`Error loading past messages: ${error}`);
            this.loadMockMessages(); // Fallback
        }
    }

    private loadMockMessages(): void { // Fallback if gRPC fails
        const mockMessages: UIMessage[] = [
            { name: 'alice', text: 'Hello everyone! (mock)', timestamp: new Date(Date.now() - 300000) },
            { name: 'bob', text: 'Hey Alice! How are you? (mock)', timestamp: new Date(Date.now() - 240000) },
        ];
        this.messages = mockMessages;
    }

    private startMessagePolling(): void {
        const pollInterval = setInterval(async () => {
            if (!this.isActive) {
                clearInterval(pollInterval);
                return;
            }
            try {
                const params: ListMessagesParams = {
                    roomPath: this.roomPath, // Corrected
                    limit: 10
                };
                const response = await grpcClient.listMessages(params);
                const latestAppMessages = response.messages;

                const lastMessageTime = this.messages.length > 0
                    ? this.messages[this.messages.length - 1].timestamp
                    : new Date(0);

                const newUiMessages: UIMessage[] = latestAppMessages
                    .filter(msg => msg.created > lastMessageTime && msg.ownerName !== this.userName)
                    .map(msg => ({
                        name: msg.ownerName,   // Corrected
                        text: msg.textContent, // Corrected
                        timestamp: msg.created
                    }));

                newUiMessages.forEach(msg => this.addMessage(msg));
            } catch (error) {
                // console.warn('Message polling error:', error); // Optional: log non-critical polling errors
                // this.simulateMockMessage(); // Fallback to mock if polling fails
            }
        }, 3000);
    }

    // simulateMockMessage removed as polling fallback is now part of startMessagePolling's catch

    private setupInputHandling(): void {
        this.term.onKey(({ key, domEvent }) => {
            if (!this.isActive) return;
            if (domEvent.ctrlKey && domEvent.key === 'c') {
                this.exit();
                return;
            }
            const printable = !domEvent.altKey && !domEvent.ctrlKey && !domEvent.metaKey;
            if (domEvent.key === 'Enter') this.sendMessage();
            else if (domEvent.key === 'Backspace') {
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

    private addMessage(message: UIMessage): void {
        this.messages.push(message);
        if (this.messages.length > 100) this.messages = this.messages.slice(-100);
        this.renderMessages();
        this.renderInputLine();
    }

    private addSystemMessage(text: string): void {
        this.addMessage({ name: 'system', text: text, timestamp: new Date() });
    }

    private async sendMessage(): Promise<void> {
        if (this.currentInput.trim() === '') return;
        const messageText = this.currentInput.trim();
        this.currentInput = '';

        const newMessage: UIMessage = {
            name: this.userName,
            text: messageText,
            timestamp: new Date()
        };
        this.addMessage(newMessage); // Add locally for immediate feedback

        try {
            const params: WriteMessageParams = {
                textContent: messageText, // Corrected
                destinationPath: this.roomPath,
                ownerToken: this.ownerToken
            };
            const response = await grpcClient.writeMessage(params);
            if (!response.status.ok) {
                this.addSystemMessage(`Failed to send message: ${response.status.message}`);
            }
        } catch (error) {
            this.addSystemMessage(`Failed to send message: ${error}`);
        }
    }

    private renderMessages(): void {
        const availableLines = this.messageDisplayHeight - 3;
        const messagesToShow = this.messages.slice(-availableLines);
        for (let i = 3; i < this.messageDisplayHeight + 3; i++) {
            this.term.write(`\x1b[${i + 1};1H\x1b[K`);
        }
        let lineOffset = 3;
        messagesToShow.forEach((message, index) => {
            const timestamp = message.timestamp.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit' });
            let nameColor = '\x1b[32m'; // Green
            if (message.name === this.userName) nameColor = '\x1b[33m'; // Yellow
            else if (message.name === 'system') nameColor = '\x1b[31m'; // Red
            const line = `\x1b[90m[${timestamp}]\x1b[0m ${nameColor}${message.name}\x1b[0m: ${message.text}`;
            this.term.write(`\x1b[${lineOffset + index + 1};1H${line}`);
        });
    }

    private renderInputLine(): void {
        const inputLineNum = this.inputLine + 1;
        this.term.write(`\x1b[${inputLineNum};1H\x1b[K`);
        this.term.write(`\x1b[${inputLineNum + 1};1H\x1b[K`);
        const prompt = `\x1b[32m${this.userName}\x1b[0m> `;
        this.term.write(`\x1b[${inputLineNum};1H${prompt}${this.currentInput}`);
        const cursorCol = prompt.length - 8 + this.currentInput.length + 1;
        this.term.write(`\x1b[${inputLineNum};${cursorCol}H`);
    }

    private exit(): void {
        this.isActive = false;
        this.term.clear();
        this.term.writeln('\x1b[36mExited chat mode\x1b[0m');
        if (this.onExit) this.onExit();
    }

    public onExit?: () => void;

    public stop(): void {
        this.isActive = false;
        // Consider clearing any intervals here if not handled by isActive flag
    }
}
