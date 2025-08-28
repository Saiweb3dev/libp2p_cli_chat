package cli

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/Saiweb3dev/distributed-compute-network/pkg/p2p"
)

// ChatCLI handles the command-line interface for the chat application
type ChatCLI struct {
    chatService       *p2p.ChatService
    connectionManager *p2p.ConnectionManager
    reader           *bufio.Reader
}

// NewChatCLI creates a new CLI instance
func NewChatCLI(cs *p2p.ChatService, cm *p2p.ConnectionManager) *ChatCLI {
    return &ChatCLI{
        chatService:       cs,
        connectionManager: cm,
        reader:           bufio.NewReader(os.Stdin),
    }
}

// Start begins the interactive CLI loop
func (cli *ChatCLI) Start() {
    fmt.Println("\n🚀 Chat CLI started!")
    fmt.Println("Commands:")
    fmt.Println("  <name|peerID> <message>  - Send message to peer")
    fmt.Println("  /peers                   - List connected peers")
    fmt.Println("  /connect <multiaddr>     - Connect to a peer")
    fmt.Println("  /help                    - Show this help")
    fmt.Println("  /quit                    - Exit the application")
    fmt.Println()
    
    for {
        fmt.Print(">> ")
        line, err := cli.reader.ReadString('\n')
        if err != nil {
            fmt.Printf("Error reading input: %v\n", err)
            continue
        }
        
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        
        if err := cli.processCommand(line); err != nil {
            fmt.Printf("❌ Error: %v\n", err)
        }
    }
}

// processCommand handles different types of user commands
func (cli *ChatCLI) processCommand(line string) error {
    // Handle special commands that start with /
    if strings.HasPrefix(line, "/") {
        return cli.handleSpecialCommand(line)
    }
    
    // Handle regular chat messages
    return cli.handleChatMessage(line)
}

// handleSpecialCommand processes commands like /peers, /connect, etc.
func (cli *ChatCLI) handleSpecialCommand(line string) error {
    parts := strings.SplitN(line, " ", 2)
    command := parts[0]
    
    switch command {
    case "/help":
        cli.showHelp()
        
    case "/quit", "/exit":
        fmt.Println("👋 Goodbye!")
        os.Exit(0)
        
    case "/peers":
        cli.listPeers()
        
    case "/connect":
        if len(parts) < 2 {
            return fmt.Errorf("usage: /connect <multiaddr>")
        }
        return cli.connectionManager.ConnectToPeer(parts[1])
        
    case "/info":
        cli.connectionManager.PrintHostInfo()
        
    default:
        return fmt.Errorf("unknown command: %s (use /help for available commands)", command)
    }
    
    return nil
}

// handleChatMessage processes regular chat messages
func (cli *ChatCLI) handleChatMessage(line string) error {
    parts := strings.SplitN(line, " ", 2)
    if len(parts) != 2 {
        return fmt.Errorf("usage: <peerName|peerID> <message>")
    }
    
    targetStr, message := parts[0], parts[1]
    
    // Try to find peer by name first
    targetID, found := cli.chatService.FindPeerByName(targetStr)
    
    // If not found by name, try to decode as peer ID
    if !found {
        tid, err := peer.Decode(targetStr)
        if err != nil {
            return fmt.Errorf("unknown name or invalid peerID: %s", targetStr)
        }
        targetID = tid
    }
    
    // Send the message
    if err := cli.chatService.SendMessage(targetID, message); err != nil {
        return fmt.Errorf("failed to send message: %v", err)
    }
    
    // Show confirmation
    targetName := cli.chatService.GetPeerName(targetID)
    fmt.Printf("📤 Message sent to %s\n", targetName)
    
    return nil
}

// showHelp displays available commands
func (cli *ChatCLI) showHelp() {
    fmt.Println("\n📚 Available Commands:")
    fmt.Println("  <name|peerID> <message>  - Send message to a peer")
    fmt.Println("  /peers                   - List all connected peers")
    fmt.Println("  /connect <multiaddr>     - Connect to a new peer")
    fmt.Println("  /info                    - Show this host's information")
    fmt.Println("  /help                    - Show this help message")
    fmt.Println("  /quit                    - Exit the application")
    fmt.Println()
}

// listPeers shows all connected peers and their names
func (cli *ChatCLI) listPeers() {
    peers := cli.chatService.GetAllPeers()
    connectedPeers := cli.connectionManager.GetConnectedPeers()
    
    fmt.Println("\n👥 Known Peers:")
    if len(peers) <= 1 { // Only ourselves
        fmt.Println("   No other peers known yet")
        return
    }
    
    // Create a set of connected peer IDs for quick lookup
    connectedSet := make(map[peer.ID]bool)
    for _, peerID := range connectedPeers {
        connectedSet[peerID] = true
    }
    
    // Get our own peer ID using the getter method
    myPeerID := cli.connectionManager.GetHostID()
    
    for peerID, name := range peers {
        // Skip ourselves
        if peerID == myPeerID {
            continue
        }
        
        status := "❌ Disconnected"
        if connectedSet[peerID] {
            status = "✅ Connected"
        }
        
        fmt.Printf("   %s - %s (%s)\n", name, peerID.String(), status)
    }
    fmt.Println()
}