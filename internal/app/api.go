package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (a *App) StartServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/reset", a.handleReset)
	mux.HandleFunc("/stats", a.handleStats)

	fmt.Printf("Starting API server on port %s\n", a.cfg.ApiPort)
	go func() {
		if err := http.ListenAndServe(":"+a.cfg.ApiPort, mux); err != nil {
			fmt.Printf("API server failed: %v\n", err)
		}
	}()
}

func (a *App) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "Missing ip parameter", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	var targetClient *Client
	for _, client := range a.stats.Clients {
		if client.IP == ip {
			targetClient = client
			break
		}
	}

	if targetClient == nil {
		http.Error(w, "Client not found in stats", http.StatusNotFound)
		return
	}

	// Unblock in Pi-hole
	fmt.Printf("API: Unblocking client %s\n", ip)
	err := a.client.UnblockDomainsForClient(context.Background(), ip)
	if err != nil {
		fmt.Printf("API: Failed to unblock client %s: %v\n", ip, err)
		// We still reset stats even if Pi-hole fails?
		// Actually, if it fails to unblock, maybe we shouldn't reset stats so it tries again or remains blocked.
		// But the user specifically asked to reset stats AND unblock.
		// If unblock fails, the client is still blocked in Pi-hole.
		http.Error(w, fmt.Sprintf("Failed to unblock in Pi-hole: %v", err), http.StatusInternalServerError)
		return
	}

	// Reset stats
	resetClientStats(targetClient)
	targetClient.Blocked = false

	fmt.Fprintf(w, "Successfully reset stats and unblocked client %s\n", ip)
}

func (a *App) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(a.stats); err != nil {
		fmt.Printf("API: Failed to encode stats: %v\n", err)
	}
}
