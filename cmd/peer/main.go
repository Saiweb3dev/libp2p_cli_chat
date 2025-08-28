package main

import (
    "flag"
    "log"

    "github.com/Saiweb3dev/distributed-compute-network/pkg/cli"
    "github.com/Saiweb3dev/distributed-compute-network/pkg/p2p"
)

func main() {
    // Parse command-line flags
    port := flag.Int("port", 0, "listening port (0 for random)")
    name := flag.String("name", "", "display name for this peer")
    peerAddr := flag.String("peer", "", "target peer multiaddr to connect to (optional)")
    flag.Parse()

    // Validate required parameters
    if *name == "" {
        log.Fatal("❌ Please provide a --name for this peer")
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

    // Display host information
    log.Printf("✅ Peer started! Name: %s", *name)
    connectionManager.PrintHostInfo()

    // Connect to initial peer if specified
    if *peerAddr != "" {
        if err := connectionManager.ConnectToPeer(*peerAddr); err != nil {
            log.Printf("⚠️  Failed to connect to initial peer: %v", err)
        }
    }

    // Start the CLI interface
    chatCLI := cli.NewChatCLI(chatService, connectionManager)
    chatCLI.Start()
}