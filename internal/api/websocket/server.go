package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/fortuna/minerva/internal/cache"
	"github.com/fortuna/minerva/internal/publisher"
	"github.com/fortuna/minerva/internal/store"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development (TODO: restrict in production)
	},
}

// Server represents the WebSocket server
type Server struct {
	port      string
	server    *http.Server
	hub       *Hub
	db        *store.Database
	cache     *cache.RedisCache
	publisher *publisher.RedisPublisher
}

// NewServer creates a new WebSocket server
func NewServer(db *store.Database, cache *cache.RedisCache, pub *publisher.RedisPublisher) *Server {
	hub := NewHub()
	
	return &Server{
		hub:       hub,
		db:        db,
		cache:     cache,
		publisher: pub,
	}
}

// Start starts the WebSocket server
func (s *Server) Start(port string) error {
	s.port = port
	
	// Start the hub in a goroutine
	go s.hub.Run()

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/games/live", s.handleLiveGames)
	mux.HandleFunc("/ws/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	log.Printf("WebSocket server listening on :%s", port)
	return s.server.ListenAndServe()
}

// handleLiveGames handles WebSocket connections for live game updates
func (s *Server) handleLiveGames(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		hub:  s.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// handleHealth returns WebSocket server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "clients": %d}`, s.hub.ClientCount())
}

// BroadcastLiveUpdate sends a live game update to all connected clients
func (s *Server) BroadcastLiveUpdate(data []byte) {
	s.hub.Broadcast(data)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
