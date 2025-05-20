# chatsh: Your Terminal, Supercharged with Conversation! 🚀

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Tired of context-switching between your terminal and chat apps? Wish your command line was a bit more... conversational?**

**chatsh is an innovative interactive shell that seamlessly blends powerful command-line operations with real-time chat, all within your familiar terminal environment!**

Built with Go and powered by gRPC, chatsh aims to be your go-to interface for both productive work and engaging discussions.

## 🎬 See it in Action!

https://github.com/user-attachments/assets/79b90e61-2d2c-4421-a0d0-1eba00193e73

---

## 💾 Installation

The easiest way to install chatsh is using Homebrew (macOS or Linux):

```bash
brew install ponyo877/tap/chatsh
```

---

## ✨ Why chatsh?

*   **Conversational CLI:** Imagine an `ls` that understands context, or a `vim`-like interface not just for files, but for dedicated chat rooms!
*   **Real-time Chat Rooms:** Jump into persistent chat rooms directly from your terminal. Discuss projects, share snippets, or just hang out.
*   **Familiar Shell Experience:** Use common commands in a new, interactive way.
*   **gRPC Powered:** Robust and efficient client-server communication.

---

## 🌟 Key Features

*   **Interactive Shell:** A dynamic prompt that's more than just a command executor.
*   **`vim`-like Chat Interface:** Navigate and participate in chat rooms using a familiar modal UI (`vim` subcommand).
*   **Standard CLI Commands:** Access essential commands like `ls`, `cd`, `cat`, `mkdir`, `pwd`, `rm`, `mv`, `cp`, `echo`, `grep`, `tail`, `touch` within the chatsh environment.

---

## 🚀 Getting Started

1.  **Clone the repo:**
    ```bash
    git clone https://github.com/ponyo877/chatsh.git
    cd chatsh
    ```
2.  **Run the server:**
    Run server yourself   
    (Default: chatsh-app-1083612487436.asia-northeast1.run.app:443 in ~/.chatsh.yaml)
    ```bash
    go run server/main.go
    ```
3.  **In another terminal, run the client:**
    Build CLI
    ```bash
    go build -o chatsh cli/main.go
    ```
    Run interractive mode
    ```bash
    ./chatsh
    ```
    Or, to jump directly into a chat room:
    ```bash
    ./chatsh cd /home # Go to home directory
    ./chatsh touch test # Create new chat room
    ./chatsh vim test # Login new chat room
    ```

Happy Chatting & Shelling!
