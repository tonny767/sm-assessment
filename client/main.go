package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	Active   bool   `json:"active"`
	LastSeen string `json:"last_seen"`
}

const clientsFile = "../server/clients.json"

func main() {
	clientID := os.Getenv("CLIENT_ID")
	serverURL := os.Getenv("SERVER_URL")
	if clientID == "" || serverURL == "" {
		fmt.Println("Please set CLIENT_ID and SERVER_URL")
		return
	}

	if !ensureClientActive(clientsFile, serverURL, clientID) {
		fmt.Println("Registering client...")
		registerClient(serverURL, clientID)
	} else {
		fmt.Printf("[%s] Already registered with server\n", clientID)
	}

	go func() {
		for range time.Tick(10 * time.Second) {
			resp, err := http.Get(fmt.Sprintf("%s/poll?client_id=%s", serverURL, clientID))
			if err != nil {
				fmt.Printf("[%s] Poll error: %v\n", clientID, err)
				continue
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if string(body) == "download" {
				fmt.Printf("[%s] Server requested download, sending file...\n", clientID)
				sendFile(clientID, serverURL, "../file_to_download.txt")
			} else {
				fmt.Printf("[%s] Poll OK at %v\n", clientID, time.Now().Format(time.RFC3339))
			}
		}
	}()

	select {}
}

func ensureClientActive(filePath, serverURL, clientID string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading clients.json:", err)
		return false
	}

	var clients map[string]Client
	if err := json.Unmarshal(data, &clients); err != nil {
		fmt.Println("Error parsing clients.json:", err)
		return false
	}

	client, exists := clients[clientID]
	if !exists {
		return false
	}

	if !client.Active {
		http.Get(fmt.Sprintf("%s/poll?client_id=%s", serverURL, clientID))
		fmt.Println("Client pinged server to reactivate")
		return true
	}

	fmt.Printf("[%s] Already active (LastSeen: %v)\n", clientID, client.LastSeen)
	return true
}

func registerClient(serverURL, clientID string) {
	resp, err := http.Get(fmt.Sprintf("%s/register?client_id=%s", serverURL, clientID))
	if err != nil {
		fmt.Println("Error registering client:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[%s] Registered with server\n", clientID)
}

func sendFile(clientID, serverURL, path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("File open error:", err)
		return
	}
	defer f.Close()

	resp, err := http.Post(fmt.Sprintf("%s/upload?client_id=%s", serverURL, clientID),
		"application/octet-stream", f) // if i dont set content-type, server may reject
	if err != nil {
		fmt.Println("Upload error:", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	fmt.Printf("[%s] File sent successfully\n", clientID)
}
