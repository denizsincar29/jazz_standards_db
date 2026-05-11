package tests

import (
	"net/http"
	"testing"
)

func TestRegisterAndLogin(t *testing.T) {
	setupTestDB(t)
	r := newRouter()

	// Register
	rr := doRequest(r, "POST", "/api/register", map[string]string{
		"username": "alice", "name": "Alice", "password": "secret123",
	}, "")
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)
	if body["token"] == nil {
		t.Fatal("Expected token in response")
	}

	// Duplicate username
	rr2 := doRequest(r, "POST", "/api/register", map[string]string{
		"username": "alice", "name": "Alice2", "password": "other",
	}, "")
	assertStatus(t, rr2, http.StatusConflict)

	// Login
	rr3 := doRequest(r, "POST", "/api/login", map[string]string{
		"username": "alice", "password": "secret123",
	}, "")
	assertStatus(t, rr3, http.StatusOK)
	if parseBody(t, rr3)["token"] == nil {
		t.Fatal("Expected token on login")
	}

	// Wrong password
	rr4 := doRequest(r, "POST", "/api/login", map[string]string{
		"username": "alice", "password": "wrong",
	}, "")
	assertStatus(t, rr4, http.StatusUnauthorized)
}

func TestGetMe(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "bob", "Bob", "pass")

	rr := doRequest(r, "GET", "/api/users/me", nil, token)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["username"] != "bob" {
		t.Fatalf("Expected username=bob, got %v", body["username"])
	}
}

func TestUpdateProfile(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "carol", "Carol", "pass")

	rr := doRequest(r, "PATCH", "/api/users/me", map[string]interface{}{
		"name":           "Carol Updated",
		"public_profile": true,
	}, token)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["name"] != "Carol Updated" {
		t.Fatalf("Expected name update, got %v", body["name"])
	}
	if body["public_profile"] != true {
		t.Fatalf("Expected public_profile=true, got %v", body["public_profile"])
	}
}

func TestUnauthenticated(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	rr := doRequest(r, "GET", "/api/users/me", nil, "")
	assertStatus(t, rr, http.StatusUnauthorized)
}

func TestLogout(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "dave", "Dave", "pass")
	rr := doRequest(r, "POST", "/api/logout", nil, token)
	assertStatus(t, rr, http.StatusOK)
}
