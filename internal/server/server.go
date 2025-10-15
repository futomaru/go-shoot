package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go-shoot/internal/config"
	"go-shoot/internal/lobby"
	"go-shoot/internal/rooms"
	ws "go-shoot/internal/websocket"
)

// Server represents the HTTP server for the Go Shoot backend services.
type Server struct {
	cfg      config.Config
	rooms    *rooms.Service
	lobbyHub *lobby.Hub
}

// New constructs a Server with default routes.
func New(cfg config.Config) *Server {
	return &Server{
		cfg:      cfg,
		rooms:    rooms.NewService(),
		lobbyHub: lobby.NewHub(),
	}
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	handler := s.routes()
	server := &http.Server{
		Addr:              s.cfg.Address(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Go Shoot API listening on %s", s.cfg.Address())
	return server.ListenAndServe()
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/v1/rooms", s.handleRooms)
	mux.HandleFunc("/api/v1/rooms/", s.handleRoomByID)
	mux.HandleFunc("/ws/lobby/", s.handleLobbyWebSocket)

	mux.Handle("/docs/", http.StripPrefix("/docs", http.FileServer(http.Dir("docs"))))
	mux.Handle("/", http.FileServer(http.Dir("web")))

	return s.loggingMiddleware(mux)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rooms, err := s.rooms.List(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
		respondJSON(w, http.StatusOK, rooms)
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		room, err := s.rooms.Create(r.Context(), req.Name)
		if err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		respondJSON(w, http.StatusCreated, room)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRoomByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/rooms/")
	segments := strings.Split(strings.Trim(path, "/"), "/")

	if len(segments) == 0 || segments[0] == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	id := segments[0]

	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		room, err := s.rooms.Get(r.Context(), id)
		if err != nil {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondJSON(w, http.StatusOK, room)
		return
	}

	if len(segments) == 2 && segments[1] == "join" {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Player string `json:"player"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		room, err := s.rooms.AddPlayer(r.Context(), id, req.Player)
		if err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		respondJSON(w, http.StatusOK, room)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) handleLobbyWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/lobby/")
	roomID = strings.Trim(roomID, "/")
	if roomID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	conn, err := ws.Upgrade(w, r)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	cleanup := s.lobbyHub.Register(roomID, conn)
	defer cleanup()

	welcome := map[string]any{
		"type":    "welcome",
		"roomID":  roomID,
		"message": "You are connected to the Go Shoot lobby.",
	}
	payload, _ := json.Marshal(welcome)
	if err := conn.WriteMessage(1, payload); err != nil {
		return
	}

	for {
		opcode, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if opcode == 1 { // text frame
			s.lobbyHub.Broadcast(roomID, payload, conn)
		}
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to encode json response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, err error) {
	respondJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}
