package cli

import (
    "bufio"
    "fmt"
    "os"
    "strings"
	"strconv"
	"time"

    "github.com/libp2p/go-libp2p/core/peer"
    "libp2p_compute/pkg/p2p"
)

// ChatCLI handles the command-line interface for the chat application
type ChatCLI struct {
    chatService       *p2p.ChatService
    connectionManager *p2p.ConnectionManager
	taskService       *p2p.TaskService
    reader           *bufio.Reader
}

// NewChatCLI creates a new CLI instance
func NewChatCLI(cs *p2p.ChatService, cm *p2p.ConnectionManager , ts *p2p.TaskService) *ChatCLI {
    return &ChatCLI{
        chatService:       cs,
        connectionManager: cm,
		taskService:       ts,
        reader:           bufio.NewReader(os.Stdin),
    }
}

// Start begins the interactive CLI loop
func (cli *ChatCLI) Start() {
    fmt.Println("\n🚀 P2P Chat & Task Processing CLI started!")
    fmt.Println("Chat Commands:")
    fmt.Println("  <name|peerID> <message>     - Send message to peer")
    fmt.Println("\nTask Commands:")
    fmt.Println("  /task prime <number>        - Check if number is prime")
    fmt.Println("  /task factorial <number>    - Calculate factorial")
    fmt.Println("  /task fibonacci <number>    - Calculate fibonacci")
    fmt.Println("  /task sum <n1,n2,n3...>     - Calculate sum of numbers")
    fmt.Println("\nSystem Commands:")
    fmt.Println("  /peers                      - List connected peers")
    fmt.Println("  /connect <multiaddr>        - Connect to a peer")
    fmt.Println("  /queue                      - Show task queue status")
    fmt.Println("  /pending                    - Show pending tasks")
    fmt.Println("  /help                       - Show this help")
    fmt.Println("  /quit                       - Exit the application")
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

// handleSpecialCommand processes commands like /peers, /connect, /task, etc.
func (cli *ChatCLI) handleSpecialCommand(line string) error {
    parts := strings.SplitN(line, " ", 3)
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
        
    case "/queue":
        cli.showQueueStatus()
        
    case "/pending":
        cli.showPendingTasks()
        
    case "/task":
        if len(parts) < 3 {
            return fmt.Errorf("usage: /task <type> <parameters>")
        }
        return cli.handleTaskCommand(parts[1], parts[2])
        
    default:
        return fmt.Errorf("unknown command: %s (use /help for available commands)", command)
    }
    
    return nil
}

// handleTaskCommand processes task submission commands
func (cli *ChatCLI) handleTaskCommand(taskType, params string) error {
    switch taskType {
    case "prime":
        number, err := strconv.ParseInt(params, 10, 64)
        if err != nil {
            return fmt.Errorf("invalid number: %s", params)
        }
        taskID, err := cli.taskService.SubmitTask(p2p.TaskTypePrime, number, nil, 1)
        if err != nil {
            return err
        }
        fmt.Printf("📋 Submitted prime check task: %s\n", taskID)
        
    case "factorial":
        number, err := strconv.ParseInt(params, 10, 64)
        if err != nil {
            return fmt.Errorf("invalid number: %s", params)
        }
        taskID, err := cli.taskService.SubmitTask(p2p.TaskTypeFactorial, number, nil, 1)
        if err != nil {
            return err
        }
        fmt.Printf("📋 Submitted factorial task: %s\n", taskID)
        
    case "fibonacci":
        number, err := strconv.ParseInt(params, 10, 64)
        if err != nil {
            return fmt.Errorf("invalid number: %s", params)
        }
        taskID, err := cli.taskService.SubmitTask(p2p.TaskTypeFibonacci, number, nil, 1)
        if err != nil {
            return err
        }
        fmt.Printf("📋 Submitted fibonacci task: %s\n", taskID)
        
    case "sum":
        numberStrs := strings.Split(params, ",")
        numbers := make([]int64, len(numberStrs))
        for i, numStr := range numberStrs {
            num, err := strconv.ParseInt(strings.TrimSpace(numStr), 10, 64)
            if err != nil {
                return fmt.Errorf("invalid number: %s", numStr)
            }
            numbers[i] = num
        }
        taskID, err := cli.taskService.SubmitTask(p2p.TaskTypeSum, 0, numbers, 1)
        if err != nil {
            return err
        }
        fmt.Printf("📋 Submitted sum task: %s\n", taskID)
        
    default:
        return fmt.Errorf("unknown task type: %s", taskType)
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
    fmt.Println("\nChat Commands:")
    fmt.Println("  <name|peerID> <message>     - Send message to a peer")
    fmt.Println("\nTask Commands:")
    fmt.Println("  /task prime <number>        - Check if number is prime")
    fmt.Println("  /task factorial <number>    - Calculate factorial")
    fmt.Println("  /task fibonacci <number>    - Calculate fibonacci") 
    fmt.Println("  /task sum <n1,n2,n3...>     - Calculate sum of numbers")
    fmt.Println("\nSystem Commands:")
    fmt.Println("  /peers                      - List all connected peers")
    fmt.Println("  /connect <multiaddr>        - Connect to a new peer")
    fmt.Println("  /queue                      - Show task queue status")
    fmt.Println("  /pending                    - Show pending tasks")
    fmt.Println("  /info                       - Show this host's information")
    fmt.Println("  /help                       - Show this help message")
    fmt.Println("  /quit                       - Exit the application")
    fmt.Println()
}

// showQueueStatus displays current task queue status
func (cli *ChatCLI) showQueueStatus() {
    count, tasks := cli.taskService.GetQueueStatus()
    
    fmt.Printf("\n📋 Task Queue Status: %d tasks\n", count)
    if count == 0 {
        fmt.Println("   Queue is empty")
        return
    }
    
    for i, task := range tasks {
        fmt.Printf("   %d. [%s] %s (Priority: %d)\n", 
            i+1, task.Type, task.ID, task.Priority)
    }
    fmt.Println()
}

// showPendingTasks displays pending tasks for coordinators
func (cli *ChatCLI) showPendingTasks() {
    pending := cli.taskService.GetPendingTasks()
    
    fmt.Printf("\n⏳ Pending Tasks: %d\n", len(pending))
    if len(pending) == 0 {
        fmt.Println("   No pending tasks")
        return
    }
    
    for taskID, task := range pending {
        elapsed := time.Since(task.StartTime)
        fmt.Printf("   %s [%s] - %v elapsed\n", 
            taskID, task.Task.Type, elapsed.Round(time.Millisecond))
    }
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