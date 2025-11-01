package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type ClientStatus struct {
	ID         string    `json:"id"`
	Active     bool      `json:"active"`
	LastSeen   time.Time `json:"last_seen"`
	Uploading  bool      `json:"uploading"`
	LastUpload string    `json:"last_upload,omitempty"`
}

var (
	mu      sync.Mutex
	clients = map[string]*ClientStatus{}
)

const clientsFile = "./clients.json"

func saveClients() {
	data, err := json.MarshalIndent(clients, "", "  ")
	if err != nil {
		fmt.Println("[Server] JSON marshal error:", err)
		return
	}
	if err := os.WriteFile(clientsFile, data, 0644); err != nil {
		fmt.Println("[Server] File write error:", err)
		return
	}
}

func loadClients() {
	if data, err := os.ReadFile(clientsFile); err == nil {
		_ = json.Unmarshal(data, &clients)
		fmt.Printf("Loaded %d clients from %s\n", len(clients), clientsFile)
	} else {
		fmt.Println("No existing clients.json found, starting fresh.")
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id required", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if c, exists := clients[clientID]; !exists {
		clients[clientID] = &ClientStatus{
			ID:       clientID,
			Active:   true,
			LastSeen: time.Now(),
		}
		fmt.Printf("[Server] Registered new client: %s\n", clientID)
	} else {
		c.Active = true
		c.LastSeen = time.Now()
		fmt.Printf("[Server] Refreshed client: %s\n", clientID)
	}

	saveClients()
}

func clientsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(clientsFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read clients.json: %v", err), http.StatusInternalServerError)
		return
	}

	if len(data) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id required", http.StatusBadRequest)
		return
	}

	loadClients()

	mu.Lock()
	client, exists := clients[clientID]
	mu.Unlock()

	if !exists {
		http.Error(w, fmt.Sprintf("client %s not registered", clientID), http.StatusForbidden)
		return
	}

	if !client.Active {
		http.Error(w, fmt.Sprintf("%s is inactive", clientID), http.StatusForbidden)
		return
	}

	os.MkdirAll("downloads", 0755)
	savePath := fmt.Sprintf("./downloads/%s_file.txt", clientID)

	out, err := os.Create(savePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mu.Lock()
	client.LastUpload = time.Now().Format(time.RFC3339)
	client.Uploading = false
	client.LastSeen = time.Now()
	mu.Unlock()

	saveClients()

	fmt.Printf("[Server] File from %s saved to %s\n", clientID, savePath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("upload successful"))
}

func pollHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("client_id")

	mu.Lock()
	client, ok := clients[id]
	if !ok {
		mu.Unlock()
		http.Error(w, "client not registered", http.StatusNotFound)
		return
	}

	client.LastSeen = time.Now()
	client.Active = true

	if client.Uploading {
		fmt.Fprint(w, "download")
	} else {
		fmt.Fprint(w, "ok")
	}

	mu.Unlock()
	saveClients()
}

func triggerHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id required", http.StatusBadRequest)
		return
	}
	triggerDownload(clientID)
	w.Write([]byte("triggered download for " + clientID))
}

func checkClientActivity() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	changed := false

	for id, client := range clients {
		if now.Sub(client.LastSeen) > 2*time.Minute && client.Active {
			fmt.Printf("[Server] %s is now inactive (last seen %s)\n", id, client.LastSeen.Format(time.RFC3339))
			client.Active = false
			changed = true
		}
	}

	if changed {
		saveClients()
	}
}

func triggerDownload(clientID string) {
	mu.Lock()
	defer mu.Unlock()
	if c, ok := clients[clientID]; ok && c.Active {
		c.Uploading = true
		fmt.Printf("[Server] Download triggered for %s, wait for client to receive the command\n", clientID)
		saveClients()
	} else {
		fmt.Printf("[Server] Cannot trigger download: %s inactive or not found\n", clientID)
	}
}

func main() {
	loadClients()

	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/clients", clientsHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/poll", pollHandler)
	http.HandleFunc("/trigger", triggerHandler)

	go func() {
		for range time.Tick(1 * time.Minute) {
			checkClientActivity()
		}
	}()

	fmt.Println("[Server] Listening on :8080")
	http.ListenAndServe(":8080", nil)
}
