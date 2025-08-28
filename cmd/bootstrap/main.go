package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Peer struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

var (
	peers = make(map[string]Peer)
	mu    sync.Mutex
)

func registerPeer(w http.ResponseWriter, r *http.Request) {
	var p Peer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	peers[p.ID] = p
	mu.Unlock()

	fmt.Printf("✅ Registered peer: %s at %s\n", p.ID, p.Addr)

	// Return all peers except the caller
	mu.Lock()
	var list []Peer
	for id, peer := range peers {
		if id != p.ID {
			list = append(list, peer)
		}
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func main() {
	http.HandleFunc("/register", registerPeer)
	fmt.Println("🚀 Bootstrap server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
