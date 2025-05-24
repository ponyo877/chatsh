import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { grpcClient, NodeType } from './grpc-client';
import { ChatUI } from './chat-ui';

// Types and interfaces
interface CommandResult {
    output: string;
    error?: string;
}

interface AppState {
    currentPath: string;
    userName: string;
    ownerToken: string;
    isInChatMode: boolean;
}

// Application state
const appState: AppState = {
    currentPath: '/home',
    userName: 'user',
    ownerToken: 'demo-token',
    isInChatMode: false
};

// Global variables for terminal and chat UI
let globalTerm: Terminal;
let currentChatUI: ChatUI | null = null;

// Command parser
function parseCommand(input: string): { command: string; args: string[] } {
    const trimmed = input.trim();
    if (!trimmed) {
        return { command: '', args: [] };
    }

    const parts = trimmed.split(/\s+/);
    const command = parts[0];
    const args = parts.slice(1);

    return { command, args };
}

// Async command handlers
const commands: Record<string, (args: string[], term: Terminal) => Promise<void>> = {
    pwd: async (args: string[], term: Terminal) => {
        term.writeln(appState.currentPath);
    },

    ls: async (args: string[], term: Terminal) => {
        try {
            term.write('Loading...');
            const response = await grpcClient.listNodes({ path: appState.currentPath });

            // Clear the "Loading..." text
            term.write('\r\x1b[K');

            if (response.entries.length === 0) {
                term.writeln('(empty directory)');
                return;
            }

            // Format output similar to Unix ls
            const entries = response.entries.map(entry => {
                const typeIndicator = entry.type === NodeType.DIRECTORY ? '/' :
                    entry.type === NodeType.ROOM ? '@' : '';
                const colorCode = entry.type === NodeType.DIRECTORY ? '\x1b[34m' : // Blue for directories
                    entry.type === NodeType.ROOM ? '\x1b[32m' : // Green for rooms
                        '\x1b[0m'; // Default for unknown
                return `${colorCode}${entry.name}${typeIndicator}\x1b[0m`;
            });

            term.writeln(entries.join('  '));
        } catch (error) {
            term.writeln(`\x1b[31mError listing directory: ${error}\x1b[0m`);
        }
    },

    cd: async (args: string[], term: Terminal) => {
        if (args.length === 0) {
            term.writeln('\x1b[31mcd: missing argument\x1b[0m');
            return;
        }

        const targetPath = args[0];
        let newPath: string;

        if (targetPath.startsWith('/')) {
            newPath = targetPath;
        } else {
            // Relative path
            if (targetPath === '..') {
                const parts = appState.currentPath.split('/').filter(p => p);
                if (parts.length > 1) {
                    parts.pop();
                    newPath = '/' + parts.join('/');
                } else {
                    newPath = '/';
                }
            } else {
                newPath = appState.currentPath === '/'
                    ? `/${targetPath}`
                    : `${appState.currentPath}/${targetPath}`;
            }
        }

        try {
            // Check if directory exists
            const exists = await grpcClient.checkDirectoryExists(newPath);
            if (exists) {
                appState.currentPath = newPath;
            } else {
                term.writeln(`\x1b[31mcd: ${targetPath}: No such directory\x1b[0m`);
            }
        } catch (error) {
            term.writeln(`\x1b[31mError checking directory: ${error}\x1b[0m`);
        }
    },

    touch: async (args: string[], term: Terminal) => {
        if (args.length === 0) {
            term.writeln('\x1b[31mtouch: missing argument\x1b[0m');
            return;
        }

        const roomName = args[0];
        const roomPath = appState.currentPath === '/'
            ? `/${roomName}`
            : `${appState.currentPath}/${roomName}`;

        try {
            term.write('Creating room...');
            const response = await grpcClient.createRoom({
                path: roomPath,
                ownerToken: appState.ownerToken // Corrected
            });

            // Clear the "Creating room..." text
            term.write('\r\x1b[K');

            if (response.status.ok) {
                term.writeln(`\x1b[32mCreated room: ${roomName}\x1b[0m`);
            } else {
                term.writeln(`\x1b[31mError: ${response.status.message}\x1b[0m`);
            }
        } catch (error) {
            term.write('\r\x1b[K');
            term.writeln(`\x1b[31mError creating room: ${error}\x1b[0m`);
        }
    },

    mkdir: async (args: string[], term: Terminal) => {
        if (args.length === 0) {
            term.writeln('\x1b[31mmkdir: missing argument\x1b[0m');
            return;
        }

        const dirName = args[0];
        const dirPath = appState.currentPath === '/'
            ? `/${dirName}`
            : `${appState.currentPath}/${dirName}`;

        try {
            term.write('Creating directory...');
            const response = await grpcClient.createDirectory({
                path: dirPath,
                ownerToken: appState.ownerToken // Corrected
            });

            // Clear the "Creating directory..." text
            term.write('\r\x1b[K');

            if (response.status.ok) {
                term.writeln(`\x1b[32mCreated directory: ${dirName}\x1b[0m`);
            } else {
                term.writeln(`\x1b[31mError: ${response.status.message}\x1b[0m`);
            }
        } catch (error) {
            term.write('\r\x1b[K');
            term.writeln(`\x1b[31mError creating directory: ${error}\x1b[0m`);
        }
    },

    vim: async (args: string[], term: Terminal) => {
        if (args.length === 0) {
            term.writeln('\x1b[31mvim: missing argument (room name)\x1b[0m');
            return;
        }

        const roomName = args[0];
        const roomPath = appState.currentPath === '/'
            ? `/${roomName}`
            : `${appState.currentPath}/${roomName}`;

        try {
            // Enter chat mode
            appState.isInChatMode = true;

            // Create and start chat UI
            currentChatUI = new ChatUI(term, roomPath, appState.userName, appState.ownerToken);

            // Set up exit callback
            currentChatUI.onExit = () => {
                appState.isInChatMode = false;
                currentChatUI = null;

                // Re-enable normal command handling
                setupNormalInputHandling();

                // Show prompt
                writePrompt(term);
            };

            // Start chat UI
            await currentChatUI.start();

        } catch (error) {
            term.writeln(`\x1b[31mError entering chat mode: ${error}\x1b[0m`);
            appState.isInChatMode = false;
            currentChatUI = null;
        }
    },

    help: async (args: string[], term: Terminal) => {
        const helpText = [
            'Available commands:',
            '  \x1b[33mpwd\x1b[0m     - show current path',
            '  \x1b[33mls\x1b[0m      - list rooms and directories',
            '  \x1b[33mcd\x1b[0m      - change directory/room',
            '  \x1b[33mtouch\x1b[0m   - create a new chat room',
            '  \x1b[33mmkdir\x1b[0m   - create a new directory',
            '  \x1b[33mvim\x1b[0m     - enter chat mode for a room',
            '  \x1b[33mclear\x1b[0m   - clear the screen',
            '  \x1b[33mhelp\x1b[0m    - show this help',
            '',
            'Chat mode usage:',
            '  \x1b[33mvim <room_name>\x1b[0m - enter chat room',
            '  \x1b[90mIn chat mode: type messages and press Enter to send\x1b[0m',
            '  \x1b[90mPress Ctrl+C to exit chat mode\x1b[0m'
        ];
        term.writeln(helpText.join('\n'));
    },

    clear: async (args: string[], term: Terminal) => {
        term.clear();
    }
};

// Execute command (now async)
async function executeCommand(term: Terminal, input: string): Promise<void> {
    const { command, args } = parseCommand(input);

    if (!command) {
        return;
    }

    const handler = commands[command];
    if (!handler) {
        term.writeln(`chatsh: command not found: ${command}`);
        return;
    }

    try {
        await handler(args, term);
    } catch (error) {
        term.writeln(`\x1b[31mError executing command: ${error}\x1b[0m`);
    }
}

// Generate prompt
function getPrompt(): string {
    return `\x1b[32m${appState.userName}\x1b[0m:\x1b[34m${appState.currentPath}\x1b[0m$ `;
}

function writePrompt(term: Terminal): void {
    term.write('\r\n' + getPrompt());
}

// Setup normal input handling (for command mode)
function setupNormalInputHandling(): void {
    let currentLine = '';
    let isExecutingCommand = false;

    globalTerm.onKey(({ key, domEvent }) => {
        // Skip if in chat mode
        if (appState.isInChatMode) {
            return;
        }

        // Prevent input while executing commands
        if (isExecutingCommand) {
            return;
        }

        const printable = !domEvent.altKey && !domEvent.ctrlKey && !domEvent.metaKey;

        if (domEvent.key === 'Enter') {
            globalTerm.writeln(''); // Move to next line

            if (currentLine.trim() !== '') {
                isExecutingCommand = true;
                executeCommand(globalTerm, currentLine).finally(() => {
                    isExecutingCommand = false;
                    currentLine = '';
                    writePrompt(globalTerm);
                });
            } else {
                writePrompt(globalTerm);
            }
            currentLine = '';
        } else if (domEvent.key === 'Backspace') {
            if (currentLine.length > 0) {
                globalTerm.write('\b \b');
                currentLine = currentLine.slice(0, -1);
            }
        } else if (domEvent.key === 'Tab') {
            // TODO: Implement tab completion
            domEvent.preventDefault();
        } else if (printable && key.length === 1) {
            currentLine += key;
            globalTerm.write(key);
        }
    });
}

// Initialize terminal
const terminalContainer = document.getElementById('terminal-container');

if (terminalContainer) {
    const baseTheme = {
        foreground: '#F8F8F8',
        background: '#2D2E2C',
        selection: '#5DA5D533',
        black: '#1E1E1D',
        brightBlack: '#262625',
        red: '#CE5C5C',
        brightRed: '#FF7272',
        green: '#5BCC5B',
        brightGreen: '#72FF72',
        yellow: '#CCCC5B',
        brightYellow: '#FFFF72',
        blue: '#5D5DD3',
        brightBlue: '#7279FF',
        magenta: '#BC5ED1',
        brightMagenta: '#E572FF',
        cyan: '#5DA5D5',
        brightCyan: '#72F0FF',
        white: '#F8F8F8',
        brightWhite: '#FFFFFF'
    };

    globalTerm = new Terminal({
        fontFamily: '"Cascadia Code", Menlo, monospace',
        theme: baseTheme,
        cursorBlink: true,
        allowProposedApi: true
    });

    const fitAddon = new FitAddon();
    globalTerm.loadAddon(fitAddon);

    globalTerm.open(terminalContainer);
    fitAddon.fit();

    window.addEventListener('resize', () => {
        fitAddon.fit();
    });

    // Welcome message
    globalTerm.writeln('\x1b[36mWelcome to chatsh Web Client!\x1b[0m');
    globalTerm.writeln('Type \x1b[33mhelp\x1b[0m to see available commands.');
    globalTerm.writeln('\x1b[90m(Currently using mock data - gRPC integration in progress)\x1b[0m');
    writePrompt(globalTerm);

    // Setup input handling
    setupNormalInputHandling();

} else {
    console.error("Could not find terminal container element with ID 'terminal-container'");
}
