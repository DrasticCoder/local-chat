package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

type Message struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	ID     string
	Device string
	Conn   *websocket.Conn
}

type Server struct {
	mu                 sync.Mutex
	clients            map[string]*Client
	deviceNameToClient map[string]*Client
	db                 *sql.DB
}

func NewServer() *Server {
	// Initialize SQLite database
	db, err := sql.Open("sqlite3", "./messages.db")
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	// Create messages table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender TEXT,
			receiver TEXT,
			content TEXT,
			timestamp TEXT
		)
	`)
	if err != nil {
		panic(fmt.Sprintf("Failed to create messages table: %v", err))
	}

	return &Server{
		clients:            make(map[string]*Client),
		deviceNameToClient: make(map[string]*Client),
		db:                 db,
	}
}

func (s *Server) AddClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure that the nickname is unique
	for _, existingClient := range s.clients {
		if existingClient.Device == client.Device {
			fmt.Printf("Device with name %s already connected\n", client.Device)
			return
		}
	}

	s.clients[client.ID] = client
	s.deviceNameToClient[client.Device] = client
	fmt.Printf("Client connected: %s (%s)\n", client.ID, client.Device)
	s.BroadcastDeviceList()
}

func (s *Server) RemoveClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.clients[clientID]; exists {
		delete(s.deviceNameToClient, client.Device)
	}
	delete(s.clients, clientID)
	fmt.Printf("Client disconnected: %s\n", clientID)
	s.BroadcastDeviceList()
}

func (s *Server) BroadcastDeviceList() {
	deviceList := []string{"ALL"} // Include "ALL" group
	for _, client := range s.clients {
		deviceList = append(deviceList, client.Device)
	}
	payload, _ := json.Marshal(struct {
		Type    string   `json:"type"`
		Devices []string `json:"devices"`
	}{
		Type:    "device_list",
		Devices: deviceList,
	})
	for _, client := range s.clients {
		client.Conn.WriteMessage(websocket.TextMessage, payload)
	}
}

// SaveMessage inserts a new message into the database
func (s *Server) SaveMessage(msg Message) {
	now := time.Now().Format(time.RFC3339)

	// Insert message into the database, SQLite will generate the messageId automatically
	_, err := s.db.Exec(`
        INSERT INTO messages (sender, receiver, content, timestamp)
        VALUES (?, ?, ?, ?)
    `, msg.From, msg.To, msg.Content, now)
	if err != nil {
		fmt.Printf("Failed to save message: %v\n", err)
	}
}

func (s *Server) BroadcastMessage(senderDevice, targetDevice, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	message := Message{
		From:      senderDevice,
		To:        targetDevice,
		Content:   content,
		Timestamp: now,
	}
	s.SaveMessage(message)

	payload, _ := json.Marshal(message)

	if targetDevice == "ALL" {
		for _, client := range s.clients {
			client.Conn.WriteMessage(websocket.TextMessage, payload)
		}
	} else if client, exists := s.deviceNameToClient[targetDevice]; exists {
		client.Conn.WriteMessage(websocket.TextMessage, payload)
	}
	if sender, exists := s.deviceNameToClient[senderDevice]; exists {
		sender.Conn.WriteMessage(websocket.TextMessage, payload) // Echo to sender
	}
}

func (s *Server) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade connection: %v\n", err)
		return
	}

	// Assign a random nickname if not present in localStorage
	clientAddr := r.RemoteAddr
	// No need to split host and port, use clientAddr directly

	// Generate a random name for the device
	names := []string{"User A", "User B", "User C", "User D", "Charlie", "Tommy", "Spidey"}
	rand.Seed(time.Now().UnixNano())
	deviceName := names[rand.Intn(len(names))]

	client := &Client{ID: clientAddr, Device: deviceName, Conn: conn}
	s.AddClient(client)

	go func() {
		defer func() {
			conn.Close()
			s.RemoveClient(clientAddr)
		}()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Error reading message from %s: %v\n", clientAddr, err)
				break
			}

			var received Message
			if err := json.Unmarshal(msg, &received); err != nil {
				fmt.Printf("Invalid message from %s: %v\n", clientAddr, err)
				continue
			}

			// Use plain text message
			s.BroadcastMessage(received.From, received.To, received.Content)
		}
	}()
}

func main() {
	server := NewServer()

	http.HandleFunc("/ws", server.ServeWS)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
