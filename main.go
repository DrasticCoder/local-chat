package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

// Client represents a connected user
type Client struct {
	ID   string
	Conn *websocket.Conn
}

// Server holds connected clients and broadcasts messages
type Server struct {
	mu      sync.Mutex
	clients map[string]*Client
}

func NewServer() *Server {
	return &Server{
		clients: make(map[string]*Client),
	}
}

func (s *Server) AddClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client.ID] = client
	fmt.Printf("Client connected: %s\n", client.ID)
}

func (s *Server) RemoveClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, clientID)
	fmt.Printf("Client disconnected: %s\n", clientID)
}

func (s *Server) Broadcast(senderID, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, client := range s.clients {
		if id != senderID { // Skip sender
			err := client.Conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s: %s", senderID, message)))
			if err != nil {
				fmt.Printf("Error sending message to %s: %v\n", id, err)
			}
		}
	}
}

// ServeWS handles WebSocket connections
func (s *Server) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade connection: %v\n", err)
		return
	}

	// Assign a unique ID for the client
	clientID := r.RemoteAddr
	client := &Client{ID: clientID, Conn: conn}
	s.AddClient(client)

	// Listen for messages from the client
	go func() {
		defer func() {
			conn.Close()
			s.RemoveClient(clientID)
		}()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Error reading message from %s: %v\n", clientID, err)
				break
			}
			fmt.Printf("Message from %s: %s\n", clientID, string(msg))
			s.Broadcast(clientID, string(msg))
		}
	}()
}

func main() {
	server := NewServer()

	http.HandleFunc("/ws", server.ServeWS)

	// Serve static files (HTML client)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
