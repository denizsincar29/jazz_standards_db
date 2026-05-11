package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestCreateAndUsePassKey(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	sessionToken := registerAndLogin(t, r, "alice", "Alice", "pass")

	rr := doRequest(r, "POST", "/api/users/me/passkeys", map[string]string{"name": "My Script"}, sessionToken)
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)
	rawToken, ok := body["token"].(string)
	if !ok || rawToken == "" {
		t.Fatal("Expected raw token in create response")
	}
	if body["token_hint"] == nil {
		t.Fatal("Expected token_hint in response")
	}

	// Use pass key as bearer token
	rr2 := doRequest(r, "GET", "/api/users/me", nil, rawToken)
	assertStatus(t, rr2, http.StatusOK)
	if parseBody(t, rr2)["username"] != "alice" {
		t.Fatal("Expected alice's profile via pass key")
	}
}

func TestListPassKeysHidesToken(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "carol", "Carol", "pass")
	doRequest(r, "POST", "/api/users/me/passkeys", map[string]string{"name": "k1"}, token)

	rr := doRequest(r, "GET", "/api/users/me/passkeys", nil, token)
	assertStatus(t, rr, http.StatusOK)

	var arr []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&arr); err != nil {
		t.Fatalf("Expected JSON array: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("Expected 1 pass key, got %d", len(arr))
	}
	if arr[0]["token"] != nil {
		t.Fatal("Raw token must not be exposed in list endpoint")
	}
	if arr[0]["token_hint"] == nil {
		t.Fatal("token_hint should be present")
	}
}

func TestDeletePassKeyRevokesAccess(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "dave", "Dave", "pass")

	rr := doRequest(r, "POST", "/api/users/me/passkeys", map[string]string{"name": "Temp"}, token)
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)
	pkID := uint(body["id"].(float64))
	rawToken := body["token"].(string)

	// Confirm it works before deletion
	rr0 := doRequest(r, "GET", "/api/users/me", nil, rawToken)
	assertStatus(t, rr0, http.StatusOK)

	// Delete
	rr2 := doRequest(r, "DELETE", fmt.Sprintf("/api/users/me/passkeys/%d", pkID), nil, token)
	assertStatus(t, rr2, http.StatusOK)

	// Token should no longer work
	rr3 := doRequest(r, "GET", "/api/users/me", nil, rawToken)
	assertStatus(t, rr3, http.StatusUnauthorized)
}

func TestPassKeyRequiresName(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "eve", "Eve", "pass")

	rr := doRequest(r, "POST", "/api/users/me/passkeys", map[string]string{"name": ""}, token)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestDeleteNonExistentPassKey(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "frank", "Frank", "pass")

	rr := doRequest(r, "DELETE", "/api/users/me/passkeys/9999", nil, token)
	assertStatus(t, rr, http.StatusNotFound)
}
