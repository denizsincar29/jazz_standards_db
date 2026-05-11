package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestCreateListUpdateDeleteCategory(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "alice", "Alice", "pass")

	// Create
	rr := doRequest(r, "POST", "/api/users/me/categories", map[string]interface{}{
		"name": "Standards to Learn", "color": "#ff0000",
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)
	catID := uint(body["id"].(float64))
	if body["name"] != "Standards to Learn" {
		t.Fatalf("Expected correct category name, got %v", body["name"])
	}

	// Duplicate
	rr2 := doRequest(r, "POST", "/api/users/me/categories", map[string]interface{}{
		"name": "Standards to Learn",
	}, token)
	assertStatus(t, rr2, http.StatusConflict)

	// List
	rr3 := doRequest(r, "GET", "/api/users/me/categories", nil, token)
	assertStatus(t, rr3, http.StatusOK)
	var cats []map[string]interface{}
	json.NewDecoder(rr3.Body).Decode(&cats)
	if len(cats) != 1 {
		t.Fatalf("Expected 1 category, got %d", len(cats))
	}

	// Update
	rr4 := doRequest(r, "PUT", fmt.Sprintf("/api/users/me/categories/%d", catID), map[string]interface{}{
		"name": "Gigging Standards", "color": "#00ff00",
	}, token)
	assertStatus(t, rr4, http.StatusOK)
	if parseBody(t, rr4)["name"] != "Gigging Standards" {
		t.Fatal("Expected updated category name")
	}

	// Delete
	rr5 := doRequest(r, "DELETE", fmt.Sprintf("/api/users/me/categories/%d", catID), nil, token)
	assertStatus(t, rr5, http.StatusOK)
}

func TestCategoryDefaultColor(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "bob", "Bob", "pass")

	rr := doRequest(r, "POST", "/api/users/me/categories", map[string]interface{}{
		"name": "No Color",
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)
	if body["color"] != "#000000" {
		t.Fatalf("Expected default color #000000, got %v", body["color"])
	}
}

func TestCategoryWithStandards(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "carol", "Carol", "pass")
	stdID := createApprovedStandard(t, "Misty", "Garner", "swing")

	// Create category
	rr := doRequest(r, "POST", "/api/users/me/categories", map[string]interface{}{
		"name": "Ballads",
	}, token)
	catID := uint(parseBody(t, rr)["id"].(float64))

	// Add standard with category
	rr2 := doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), map[string]interface{}{
		"category_id": catID,
	}, token)
	assertStatus(t, rr2, http.StatusCreated)

	// List grouped
	rr3 := doRequest(r, "GET", "/api/users/me/standards", nil, token)
	assertStatus(t, rr3, http.StatusOK)
	body := parseBody(t, rr3)
	grouped := body["grouped"].(map[string]interface{})
	if _, ok := grouped["Ballads"]; !ok {
		t.Fatal("Expected 'Ballads' in grouped standards")
	}
}
