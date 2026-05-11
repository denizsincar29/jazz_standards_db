package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestCreateAndListComposedTunes(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "alice", "Alice", "pass")

	rr := doRequest(r, "POST", "/api/users/me/compositions", map[string]interface{}{
		"title":       "Midnight in Ankara",
		"style":       "modal",
		"key":         "D-",
		"description": "A moody piece inspired by late-night walks",
		"is_public":   true,
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)

	if body["title"] != "Midnight in Ankara" {
		t.Fatalf("Expected correct title, got %v", body["title"])
	}
	if body["share_id"] == nil || body["share_id"] == "" {
		t.Fatal("Expected non-empty share_id")
	}
	// Composer should default to user's name
	if body["composer"] != "Alice" {
		t.Fatalf("Expected composer=Alice, got %v", body["composer"])
	}

	shareID := body["share_id"].(string)
	tuneID := uint(body["id"].(float64))

	// List
	rr2 := doRequest(r, "GET", "/api/users/me/compositions", nil, token)
	assertStatus(t, rr2, http.StatusOK)
	var tunes []map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&tunes)
	if len(tunes) != 1 {
		t.Fatalf("Expected 1 composition, got %d", len(tunes))
	}

	// View shared (unauthenticated)
	rr3 := doRequest(r, "GET", fmt.Sprintf("/api/shared_tune?id=%s", shareID), nil, "")
	assertStatus(t, rr3, http.StatusOK)
	b3 := parseBody(t, rr3)
	tuneObj := b3["tune"].(map[string]interface{})
	if tuneObj["title"] != "Midnight in Ankara" {
		t.Fatalf("Expected shared tune title, got %v", tuneObj["title"])
	}

	// Accept into personal pieces (different user)
	t2 := registerAndLogin(t, r, "bob", "Bob", "pass")
	rr4 := doRequest(r, "POST", fmt.Sprintf("/api/shared_tune/accept?id=%s", shareID), nil, t2)
	assertStatus(t, rr4, http.StatusCreated)
	b4 := parseBody(t, rr4)
	if b4["piece"] == nil {
		t.Fatal("Expected piece in accept response")
	}

	// Cannot accept same tune twice
	rr5 := doRequest(r, "POST", fmt.Sprintf("/api/shared_tune/accept?id=%s", shareID), nil, t2)
	assertStatus(t, rr5, http.StatusConflict)

	// Update the tune
	rr6 := doRequest(r, "PUT", fmt.Sprintf("/api/users/me/compositions/%d", tuneID), map[string]interface{}{
		"description": "Updated description",
		"is_public":   false,
	}, token)
	assertStatus(t, rr6, http.StatusOK)
	if parseBody(t, rr6)["description"] != "Updated description" {
		t.Fatal("Expected updated description")
	}

	// Shared link no longer works (is_public=false)
	rr7 := doRequest(r, "GET", fmt.Sprintf("/api/shared_tune?id=%s", shareID), nil, "")
	assertStatus(t, rr7, http.StatusNotFound)

	// Delete
	rr8 := doRequest(r, "DELETE", fmt.Sprintf("/api/users/me/compositions/%d", tuneID), nil, token)
	assertStatus(t, rr8, http.StatusOK)
}

func TestSharedTuneNotFoundForPrivate(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "carol", "Carol", "pass")

	rr := doRequest(r, "POST", "/api/users/me/compositions", map[string]interface{}{
		"title":     "Secret Tune",
		"is_public": false,
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	shareID := parseBody(t, rr)["share_id"].(string)

	rr2 := doRequest(r, "GET", fmt.Sprintf("/api/shared_tune?id=%s", shareID), nil, "")
	assertStatus(t, rr2, http.StatusNotFound)
}

func TestInvalidShareID(t *testing.T) {
	setupTestDB(t)
	r := newRouter()

	rr := doRequest(r, "GET", "/api/shared_tune?id=doesnotexist12345", nil, "")
	assertStatus(t, rr, http.StatusNotFound)
}

func TestComposedTuneTitleRequired(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "dave", "Dave", "pass")

	rr := doRequest(r, "POST", "/api/users/me/compositions", map[string]interface{}{
		"style": "swing",
	}, token)
	assertStatus(t, rr, http.StatusBadRequest)
}
