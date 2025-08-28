package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p"
	protocol "github.com/libp2p/go-libp2p/core/protocol"
	host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const ProtocolID = "/compute/1.0.0"
const NameProtocolID = "/name/1.0.0"

// Global map of peerID -> shortName
var (
	peerNames   = make(map[peer.ID]string)
	peerNamesMu sync.RWMutex
)

// ----------------- Handlers -----------------

// handleStream handles /compute/1.0.0 message streams
func handleStream(s network.Stream) {
	log.Println("📩 New stream opened by:", s.Conn().RemotePeer())

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go func() {
		for {
			str, err := rw.ReadString('\n')
			if err != nil {
				return
			}
			if str != "" {
				peerNamesMu.RLock()
				name := peerNames[s.Conn().RemotePeer()]
				peerNamesMu.RUnlock()
				if name == "" {
					name = s.Conn().RemotePeer().String()
				}
				fmt.Printf("[From %s]: %s", name, str)
			}
		}
	}()
}

// handleNameStream handles /name/1.0.0 protocol for name exchange
func handleNameStream(s network.Stream) {
    r := bufio.NewReader(s)
    name, _ := r.ReadString('\n')
    name = strings.TrimSpace(name)

    peerID := s.Conn().RemotePeer()

    peerNamesMu.Lock()
    peerNames[peerID] = name
    peerNamesMu.Unlock()

    log.Printf("✅ Registered peer %s as '%s'\n", peerID, name)

    // 🔑 Send our own name back (so both sides know each other)
    myName := peerNames[s.Conn().LocalPeer()]
    _, _ = s.Write([]byte(myName + "\n"))
    log.Printf("📤 Sent my name '%s' to %s\n", myName, peerID)
}

// sendName sends our name to the connected peer
func sendName(h host.Host, peerID peer.ID, name string) {
    s, err := h.NewStream(context.Background(), peerID, protocol.ID(NameProtocolID))
    if err != nil {
        log.Printf("Error opening name stream: %v\n", err)
        return
    }
    defer s.Close()

    // Send my name
    _, _ = s.Write([]byte(name + "\n"))
    log.Printf("📤 Sent my name '%s' to %s\n", name, peerID)

    // Wait for their name
    r := bufio.NewReader(s)
    theirName, _ := r.ReadString('\n')
    theirName = strings.TrimSpace(theirName)

    if theirName != "" {
        peerNamesMu.Lock()
        peerNames[peerID] = theirName
        peerNamesMu.Unlock()
        log.Printf("✅ Registered peer %s as '%s'\n", peerID, theirName)
    }
}


// ----------------- Messaging -----------------

// sendMessage sends a chat message to target peer
func sendMessage(h host.Host, target peer.ID, fromName string, message string) {
	s, err := h.NewStream(context.Background(), target, ProtocolID)
	if err != nil {
		fmt.Println("Error opening stream:", err)
		return
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	rw.WriteString(fmt.Sprintf("[%s]: %s\n", fromName, message))
	rw.Flush()
}

// ----------------- Main -----------------

func main() {
	// CLI flags
	port := flag.Int("port", 0, "listening port")
	name := flag.String("name", "", "short name of peer")
	peerAddr := flag.String("peer", "", "target peer multiaddr (optional)")
	flag.Parse()

	if *name == "" {
		log.Fatal("❌ Please provide a --name for this peer")
	}

	// Create host
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)),
	)
	if err != nil {
		panic(err)
	}

	// Save own name
	peerNamesMu.Lock()
	peerNames[h.ID()] = *name
	peerNamesMu.Unlock()

	// Register stream handlers
	h.SetStreamHandler(ProtocolID, handleStream)
	h.SetStreamHandler(NameProtocolID, handleNameStream)

	// Print peer info
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID().String()))
	for _, addr := range h.Addrs() {
		fmt.Println("Listening on:", addr.Encapsulate(hostAddr))
	}
	log.Printf("✅ Peer started! Name: %s, ID: %s", *name, h.ID().String())

	// If --peer was passed, connect
	if *peerAddr != "" {
		maddr, err := ma.NewMultiaddr(*peerAddr)
		if err != nil {
			log.Fatal(err)
		}
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Fatal(err)
		}

		if err := h.Connect(context.Background(), *info); err != nil {
			log.Fatal("❌ Connection failed:", err)
		} else {
			log.Println("✅ Connected to", info.ID.String())
			// Immediately send our name to the peer
			sendName(h, info.ID, *name)
		}
	}

	// CLI REPL
	stdReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(">> ")
		line, _ := stdReader.ReadString('\n')
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			fmt.Println("Usage: <peerName|peerID> <message>")
			continue
		}
		targetStr, message := parts[0], parts[1]

		var targetID peer.ID
		var ok bool

		peerNamesMu.RLock()
		for pid, pname := range peerNames {
			if pname == targetStr {
				targetID = pid
				ok = true
				break
			}
		}
		peerNamesMu.RUnlock()

		if !ok {
			// maybe user typed a peerID
			tid, err := peer.Decode(targetStr)
			if err != nil {
				fmt.Println("❌ Unknown name or invalid peerID")
				continue
			}
			targetID = tid
		}

		sendMessage(h, targetID, *name, message)
	}
}
