package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddAndListUserStandards(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "alice", "Alice", "pass")
	stdID := createApprovedStandard(t, "Autumn Leaves", "Kosma", "swing")

	rr := doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), map[string]interface{}{
		"proficiency": "learning", "notes": "working on it",
	}, token)
	assertStatus(t, rr, http.StatusCreated)

	// Duplicate
	rr2 := doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, token)
	assertStatus(t, rr2, http.StatusConflict)

	// List
	rr3 := doRequest(r, "GET", "/api/users/me/standards", nil, token)
	assertStatus(t, rr3, http.StatusOK)
	body := parseBody(t, rr3)
	if body["total"].(float64) != 1 {
		t.Fatalf("Expected 1 standard in list, got %v", body["total"])
	}
}

func TestUpdateUserStandard(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "bob", "Bob", "pass")
	stdID := createApprovedStandard(t, "Blue Monk", "Monk", "bebop")
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, token)

	rr := doRequest(r, "PUT", fmt.Sprintf("/api/users/me/standards/%d", stdID), map[string]interface{}{
		"proficiency": "master", "notes": "can play it in any key",
	}, token)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["proficiency"] != "master" {
		t.Fatalf("Expected proficiency=master, got %v", body["proficiency"])
	}
}

func TestDeleteUserStandard(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "carol", "Carol", "pass")
	stdID := createApprovedStandard(t, "Misty", "Garner", "swing")
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, token)

	rr := doRequest(r, "DELETE", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, token)
	assertStatus(t, rr, http.StatusOK)

	rr2 := doRequest(r, "DELETE", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, token)
	assertStatus(t, rr2, http.StatusNotFound)
}

func TestProficiencyFilter(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "dave", "Dave", "pass")
	s1 := createApprovedStandard(t, "So What", "Davis", "modal")
	s2 := createApprovedStandard(t, "Giant Steps", "Coltrane", "bebop")
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", s1), map[string]interface{}{"proficiency": "master"}, token)
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", s2), map[string]interface{}{"proficiency": "beginner"}, token)

	rr := doRequest(r, "GET", "/api/users/me/standards?proficiency=master", nil, token)
	assertStatus(t, rr, http.StatusOK)
	if parseBody(t, rr)["total"].(float64) != 1 {
		t.Fatal("Expected 1 master standard")
	}
}

func TestPracticeLog(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "eve", "Eve", "pass")
	stdID := createApprovedStandard(t, "Stella by Starlight", "Young", "swing")
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, token)

	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d/practice", stdID), map[string]interface{}{"duration_min": 30, "notes": "changes"}, token)
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d/practice", stdID), map[string]interface{}{"duration_min": 20}, token)

	rr := doRequest(r, "GET", "/api/users/me/practice", nil, token)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["total_entries"].(float64) != 2 {
		t.Fatalf("Expected 2 entries, got %v", body["total_entries"])
	}
	if body["total_minutes"].(float64) != 50 {
		t.Fatalf("Expected 50 total minutes, got %v", body["total_minutes"])
	}
}

func TestCannotLogPracticeForUnlistedStandard(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "frank", "Frank", "pass")
	stdID := createApprovedStandard(t, "Footprints", "Shorter", "modal")

	rr := doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d/practice", stdID), map[string]interface{}{"duration_min": 15}, token)
	assertStatus(t, rr, http.StatusNotFound)
}

func TestExportStandardsJSON(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "grace", "Grace", "pass")
	stdID := createApprovedStandard(t, "All of Me", "Marks", "swing")
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, token)

	rr := doRequest(r, "GET", "/api/users/me/standards/export", nil, token)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["exported"].(float64) != 1 {
		t.Fatalf("Expected 1 exported standard, got %v", body["exported"])
	}
}

func TestExportStandardsCSV(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "grace2", "Grace2", "pass")
	stdID := createApprovedStandard(t, "Body and Soul", "Green", "swing")
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, token)

	req := httptest.NewRequest("GET", "/api/users/me/standards/export?format=csv", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	newRouter().ServeHTTP(rr, req)
	assertStatus(t, rr, http.StatusOK)
	if ct := rr.Header().Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("Expected text/csv Content-Type, got %s", ct)
	}
}

func TestPublicProfile(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	ownerToken := registerAndLogin(t, r, "henry", "Henry", "pass")
	viewerToken := registerAndLogin(t, r, "iris", "Iris", "pass")
	stdID := createApprovedStandard(t, "Waltz for Debby", "Evans", "waltz")
	doRequest(r, "POST", fmt.Sprintf("/api/users/me/standards/%d", stdID), nil, ownerToken)

	// Private by default → 403
	rr := doRequest(r, "GET", "/api/users/henry/standards", nil, viewerToken)
	assertStatus(t, rr, http.StatusForbidden)

	// Make public
	doRequest(r, "PATCH", "/api/users/me", map[string]interface{}{"public_profile": true}, ownerToken)

	// Now visible
	rr2 := doRequest(r, "GET", "/api/users/henry/standards", nil, viewerToken)
	assertStatus(t, rr2, http.StatusOK)
	if parseBody(t, rr2)["total"].(float64) != 1 {
		t.Fatal("Expected 1 standard in public profile")
	}
}
