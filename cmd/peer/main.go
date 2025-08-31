package main

import (
    "fmt"
    "flag"
    "log"
    "time"

    "libp2p_compute/pkg/cli"
    "libp2p_compute/pkg/p2p"
)

func main() {
    // Parse command-line flags
    port := flag.Int("port", 0, "listening port (0 for random)")
    name := flag.String("name", "", "display name for this peer")
    peerAddr := flag.String("peer", "", "target peer multiaddr to connect to (optional)")
    worker := flag.Bool("worker", false, "enable worker mode (processes tasks)")
    coordinator := flag.Bool("coordinator", false, "enable coordinator mode (submits tasks)")
    bootstrap := flag.String("bootstrap", "", "bootstrap server URL (e.g., http://bootstrap:8080)")
    autoDemo := flag.Bool("auto-demo", false, "automatically run demo tasks (coordinator only)")
    flag.Parse()

    // Auto-assign roles based on name if not specified
    if !*worker && !*coordinator {
        if *name == "coordinator" {
            *coordinator = true
        } else {
            *worker = true // Default to worker
        }
    }

    // Auto-assign name if not provided
    if *name == "" {
        if *coordinator {
            *name = "coordinator"
        } else {
            *name = fmt.Sprintf("worker-%d", *port)
        }
    }

    // Validate required parameters
    if !*worker && !*coordinator {
        log.Fatal("❌ Please specify at least one mode: --worker or --coordinator")
    }

    // Create the libp2p host
    host, cancel, err := p2p.NewHost(*port)
    if err != nil {
        log.Fatalf("❌ Failed to create host: %v", err)
    }
    defer cancel()

    // Initialize services
    chatService := p2p.NewChatService(host, *name)
    connectionManager := p2p.NewConnectionManager(host, chatService)
    taskService := p2p.NewTaskService(host, chatService, *worker, *coordinator)

    // Determine role string for display
    roleStr := ""
    role := ""
    if *worker && *coordinator {
        roleStr = " (Worker & Coordinator)"
        role = "hybrid"
    } else if *worker {
        roleStr = " (Worker)"
        role = "worker"
    } else if *coordinator {
        roleStr = " (Coordinator)"
        role = "coordinator"
    }

    log.Printf("✅ Peer started! Name: %s%s", *name, roleStr)
    connectionManager.PrintHostInfo()

    // Bootstrap discovery if URL provided
    if *bootstrap != "" {
        bootstrapClient := p2p.NewBootstrapClient(*bootstrap, host, *name, role)
        
        // Wait a moment for the bootstrap server to be ready
        time.Sleep(2 * time.Second)
        
        // Register with bootstrap
        if err := bootstrapClient.RegisterWithBootstrap(); err != nil {
            log.Printf("⚠️  Failed to register with bootstrap: %v", err)
        }
        
        // Wait a moment for other peers to register
        time.Sleep(3 * time.Second)
        
        // Connect to known peers
        if err := bootstrapClient.ConnectToKnownPeers(); err != nil {
            log.Printf("⚠️  Failed to connect to peers: %v", err)
        }
        
        // Exchange names with connected peers
        connectedPeers := connectionManager.GetConnectedPeers()
        for _, peerID := range connectedPeers {
            if err := chatService.SendName(peerID); err != nil {
                log.Printf("⚠️  Failed to exchange names with %s: %v", peerID, err)
            }
        }
        
        // Start periodic registration
        bootstrapClient.StartPeriodicRegistration()
    }

    // Connect to initial peer if specified (manual override)
    if *peerAddr != "" {
        if err := connectionManager.ConnectToPeer(*peerAddr); err != nil {
            log.Printf("⚠️  Failed to connect to initial peer: %v", err)
        }
    }

    // Auto-demo mode for coordinators
    if *autoDemo && *coordinator {
        go runAutoDemo(taskService, *name)
    }

    // Start the CLI interface
    chatCLI := cli.NewChatCLI(chatService, connectionManager, taskService)
    chatCLI.Start()
}

// runAutoDemo automatically submits tasks for demonstration
func runAutoDemo(taskService *p2p.TaskService, peerName string) {
    // Wait for workers to connect
    time.Sleep(10 * time.Second)
    
    log.Printf("🤖 Starting auto-demo mode for %s", peerName)
    
    demoTasks := []struct {
        taskType p2p.TaskType
        number   int64
        numbers  []int64
        desc     string
    }{
        {p2p.TaskTypePrime, 97, nil, "Check if 97 is prime"},
        {p2p.TaskTypeFactorial, 10, nil, "Calculate factorial of 10"},
        {p2p.TaskTypeFibonacci, 15, nil, "Calculate 15th Fibonacci number"},
        {p2p.TaskTypeSum, 0, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, "Sum of 1-10"},
        {p2p.TaskTypePrime, 1009, nil, "Check if 1009 is prime"},
        {p2p.TaskTypeFactorial, 12, nil, "Calculate factorial of 12"},
        {p2p.TaskTypeFibonacci, 20, nil, "Calculate 20th Fibonacci number"},
        {p2p.TaskTypeSum, 0, []int64{100, 200, 300, 400, 500}, "Sum of large numbers"},
    }
    
    for i, task := range demoTasks {
        time.Sleep(5 * time.Second) // Wait between tasks
        
        log.Printf("🎯 Demo Task %d: %s", i+1, task.desc)
        
        taskID, err := taskService.SubmitTask(task.taskType, task.number, task.numbers, 1)
        if err != nil {
            log.Printf("❌ Failed to submit demo task: %v", err)
            continue
        }
        
        log.Printf("📋 Submitted demo task: %s", taskID)
    }
    
    log.Printf("🏁 Auto-demo completed!")
}