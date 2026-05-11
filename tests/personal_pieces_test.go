package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestCreateListUpdateDeletePersonalPiece(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "alice", "Alice", "pass")

	// Create
	rr := doRequest(r, "POST", "/api/users/me/pieces", map[string]interface{}{
		"title":    "My Village Blues",
		"composer": "Alice",
		"style":    "bebop",
		"key":      "F",
		"notes":    "Local tune we play at the Sunday jam",
		"is_public": false,
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)
	pieceID := uint(body["id"].(float64))
	if body["title"] != "My Village Blues" {
		t.Fatalf("Expected correct title, got %v", body["title"])
	}

	// List
	rr2 := doRequest(r, "GET", "/api/users/me/pieces", nil, token)
	assertStatus(t, rr2, http.StatusOK)
	var pieces []map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&pieces)
	if len(pieces) != 1 {
		t.Fatalf("Expected 1 piece, got %d", len(pieces))
	}

	// Update – make public
	rr3 := doRequest(r, "PUT", fmt.Sprintf("/api/users/me/pieces/%d", pieceID), map[string]interface{}{
		"is_public": true,
		"notes":     "Updated notes",
	}, token)
	assertStatus(t, rr3, http.StatusOK)
	b3 := parseBody(t, rr3)
	if b3["is_public"] != true {
		t.Fatal("Expected is_public=true after update")
	}

	// Delete
	rr4 := doRequest(r, "DELETE", fmt.Sprintf("/api/users/me/pieces/%d", pieceID), nil, token)
	assertStatus(t, rr4, http.StatusOK)

	// Confirm gone
	rr5 := doRequest(r, "GET", "/api/users/me/pieces", nil, token)
	assertStatus(t, rr5, http.StatusOK)
	var pieces2 []map[string]interface{}
	json.NewDecoder(rr5.Body).Decode(&pieces2)
	if len(pieces2) != 0 {
		t.Fatal("Expected empty list after delete")
	}
}

func TestPersonalPieceTitleRequired(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "bob", "Bob", "pass")

	rr := doRequest(r, "POST", "/api/users/me/pieces", map[string]interface{}{
		"composer": "Bob",
		"style":    "swing",
	}, token)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestPersonalPieceInvalidStyle(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "carol", "Carol", "pass")

	rr := doRequest(r, "POST", "/api/users/me/pieces", map[string]interface{}{
		"title": "Test", "style": "not_a_style",
	}, token)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestCannotAccessOtherUserPieces(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	t1 := registerAndLogin(t, r, "dave", "Dave", "pass")
	t2 := registerAndLogin(t, r, "eve", "Eve", "pass")

	// Dave creates a piece
	rr := doRequest(r, "POST", "/api/users/me/pieces", map[string]interface{}{
		"title": "Dave's Secret Tune",
	}, t1)
	assertStatus(t, rr, http.StatusCreated)
	pieceID := uint(parseBody(t, rr)["id"].(float64))

	// Eve cannot update it
	rr2 := doRequest(r, "PUT", fmt.Sprintf("/api/users/me/pieces/%d", pieceID), map[string]interface{}{
		"title": "Hacked",
	}, t2)
	assertStatus(t, rr2, http.StatusNotFound)

	// Eve cannot delete it
	rr3 := doRequest(r, "DELETE", fmt.Sprintf("/api/users/me/pieces/%d", pieceID), nil, t2)
	assertStatus(t, rr3, http.StatusNotFound)
}
