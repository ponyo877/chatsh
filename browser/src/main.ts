import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

const terminalContainer = document.getElementById('terminal-container');

function writePrompt(term: Terminal) {
    term.write('\r\n$ ');
}

if (terminalContainer) {
    var baseTheme = {
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

    var term = new Terminal({
        fontFamily: '"Cascadia Code", Menlo, monospace',
        theme: baseTheme,
        cursorBlink: true,
        allowProposedApi: true
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    term.open(terminalContainer);
    fitAddon.fit();


    window.addEventListener('resize', () => {
        fitAddon.fit();
    });

    term.writeln('Welcome to chatsh Web Client!');
    term.writeln('This is a basic xterm.js instance.');
    writePrompt(term);

    let currentLine = '';
    term.onKey(({ key, domEvent }) => {
        const printable = !domEvent.altKey && !domEvent.ctrlKey && !domEvent.metaKey;

        if (domEvent.key === 'Enter') {
            term.writeln('');
            if (currentLine.trim() !== '') {
                term.writeln(`You typed: ${currentLine}`);
            }
            currentLine = '';
            writePrompt(term);
        } else if (domEvent.key === 'Backspace') {
            if (currentLine.length > 0) {
                term.write('\b \b');
                currentLine = currentLine.slice(0, -1);
            }
        } else if (printable && key.length === 1) {
            currentLine += key;
            term.write(key);
        }
    }
    );

} else {
    console.error("Could not find terminal container element with ID 'terminal-container'");
}
