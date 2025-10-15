package rooms

import (
	"context"
	"testing"
)

func TestCreateAndGetRoom(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	created, err := svc.Create(ctx, "Test Room")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.Name != "Test Room" {
		t.Fatalf("expected name to be 'Test Room', got %s", created.Name)
	}

	fetched, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("expected id %s, got %s", created.ID, fetched.ID)
	}
}

func TestAddPlayer(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	room, err := svc.Create(ctx, "")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	updated, err := svc.AddPlayer(ctx, room.ID, "alice")
	if err != nil {
		t.Fatalf("AddPlayer returned error: %v", err)
	}
	if len(updated.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(updated.Players))
	}

	// Room should not allow duplicates.
	updated, err = svc.AddPlayer(ctx, room.ID, "alice")
	if err != nil {
		t.Fatalf("AddPlayer returned error for duplicate: %v", err)
	}
	if len(updated.Players) != 1 {
		t.Fatalf("expected 1 player after duplicate add, got %d", len(updated.Players))
	}
}
