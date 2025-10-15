package rooms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Room represents a single game lobby.
type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Players   []string  `json:"players"`
}

// Service manages Rooms in-memory.
type Service struct {
	mu    sync.RWMutex
	rooms map[string]Room
}

// NewService constructs an empty Service.
func NewService() *Service {
	return &Service{
		rooms: make(map[string]Room),
	}
}

// Create creates a new room with the provided name.
func (s *Service) Create(ctx context.Context, name string) (Room, error) {
	if name == "" {
		name = "Quick Match"
	}

	room := Room{
		ID:        generateID(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
		Players:   []string{},
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.ID] = room

	return room, nil
}

func generateID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

// List returns all known rooms.
func (s *Service) List(ctx context.Context) ([]Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		out = append(out, room)
	}

	return out, nil
}

// Get returns a room by ID or an error if it does not exist.
func (s *Service) Get(ctx context.Context, id string) (Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, ok := s.rooms[id]
	if !ok {
		return Room{}, fmt.Errorf("room %s not found", id)
	}
	return room, nil
}

// AddPlayer registers a player in the room if space is available.
func (s *Service) AddPlayer(ctx context.Context, id, player string) (Room, error) {
	if player == "" {
		return Room{}, errors.New("player name must be provided")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[id]
	if !ok {
		return Room{}, fmt.Errorf("room %s not found", id)
	}

	if len(room.Players) >= 4 {
		return Room{}, errors.New("room is full")
	}

	for _, existing := range room.Players {
		if existing == player {
			return room, nil
		}
	}

	room.Players = append(room.Players, player)
	s.rooms[id] = room
	return room, nil
}
