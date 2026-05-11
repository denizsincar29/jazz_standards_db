package tests

import (
	"fmt"
	"net/http"
	"testing"
)

func TestCreateAndListStandards(t *testing.T) {
	setupTestDB(t)
	r := newRouter()

	// Admin can create + auto-approve
	adminToken := registerAndLogin(t, r, "admin", "Admin", "pass")
	makeAdmin(t, "admin")

	rr := doRequest(r, "POST", "/api/jazz_standards", map[string]interface{}{
		"title": "Autumn Leaves", "composer": "Joseph Kosma",
		"style": "swing", "key": "G-",
	}, adminToken)
	assertStatus(t, rr, http.StatusCreated)
	body := parseBody(t, rr)
	if body["status"] != "approved" {
		t.Fatalf("Admin creation should auto-approve, got status=%v", body["status"])
	}

	// Non-admin submission goes to pending
	userToken := registerAndLogin(t, r, "user1", "User One", "pass")
	rr2 := doRequest(r, "POST", "/api/jazz_standards", map[string]interface{}{
		"title": "Blue Bossa", "composer": "Kenny Dorham", "style": "bossa_nova",
	}, userToken)
	assertStatus(t, rr2, http.StatusCreated)
	b2 := parseBody(t, rr2)
	if b2["standard"] == nil {
		t.Fatalf("Expected pending standard response with 'standard' key")
	}

	// List – user sees only approved
	rr3 := doRequest(r, "GET", "/api/jazz_standards", nil, userToken)
	assertStatus(t, rr3, http.StatusOK)
	b3 := parseBody(t, rr3)
	total := b3["total"].(float64)
	if total < 1 {
		t.Fatalf("Expected at least 1 approved standard, got %v", total)
	}
}

func TestSearch(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	adminToken := registerAndLogin(t, r, "admin", "Admin", "pass")
	makeAdmin(t, "admin")

	createApprovedStandard(t, "So What", "Miles Davis", "modal")
	createApprovedStandard(t, "Giant Steps", "John Coltrane", "bebop")

	rr := doRequest(r, "GET", "/api/jazz_standards?search=Giant", nil, adminToken)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["total"].(float64) != 1 {
		t.Fatalf("Expected 1 result for 'Giant', got %v", body["total"])
	}
}

func TestStyleFilter(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "admin", "Admin", "pass")
	makeAdmin(t, "admin")

	createApprovedStandard(t, "Girl from Ipanema", "Jobim", "bossa_nova")
	createApprovedStandard(t, "Round Midnight", "Monk", "bebop")

	rr := doRequest(r, "GET", "/api/jazz_standards?style=bossa_nova", nil, token)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["total"].(float64) != 1 {
		t.Fatalf("Expected 1 bossa_nova standard, got %v", body["total"])
	}
}

func TestApproveReject(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	adminToken := registerAndLogin(t, r, "admin", "Admin", "pass")
	makeAdmin(t, "admin")
	userToken := registerAndLogin(t, r, "user1", "User One", "pass")

	// Submit
	rr := doRequest(r, "POST", "/api/jazz_standards", map[string]interface{}{
		"title": "Confirmation", "composer": "Charlie Parker", "style": "bebop",
	}, userToken)
	assertStatus(t, rr, http.StatusCreated)
	stdID := uint(parseBody(t, rr)["standard"].(map[string]interface{})["id"].(float64))

	// Approve
	rr2 := doRequest(r, "POST", fmt.Sprintf("/api/jazz_standards/%d/approve", stdID), nil, adminToken)
	assertStatus(t, rr2, http.StatusOK)
	if parseBody(t, rr2)["status"] != "approved" {
		t.Fatal("Expected approved status")
	}

	// Submit another for rejection
	rr3 := doRequest(r, "POST", "/api/jazz_standards", map[string]interface{}{
		"title": "Unknown Tune XYZ", "composer": "Nobody", "style": "free",
	}, userToken)
	assertStatus(t, rr3, http.StatusCreated)
	stdID2 := uint(parseBody(t, rr3)["standard"].(map[string]interface{})["id"].(float64))

	rr4 := doRequest(r, "POST", fmt.Sprintf("/api/jazz_standards/%d/reject", stdID2), nil, adminToken)
	assertStatus(t, rr4, http.StatusOK)
	if parseBody(t, rr4)["status"] != "rejected" {
		t.Fatal("Expected rejected status")
	}
}

func TestUpdateAndDeleteStandard(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	adminToken := registerAndLogin(t, r, "admin", "Admin", "pass")
	makeAdmin(t, "admin")

	stdID := createApprovedStandard(t, "Misty", "Erroll Garner", "swing")

	// Update
	rr := doRequest(r, "PUT", fmt.Sprintf("/api/jazz_standards/%d", stdID), map[string]interface{}{
		"key": "Eb", "ireal_pro_link": "irealbook://Misty=...",
	}, adminToken)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["key"] != "Eb" {
		t.Fatalf("Expected key=Eb, got %v", body["key"])
	}

	// Delete
	rr2 := doRequest(r, "DELETE", fmt.Sprintf("/api/jazz_standards/%d", stdID), nil, adminToken)
	assertStatus(t, rr2, http.StatusOK)
}

func TestRandomStandard(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	token := registerAndLogin(t, r, "admin", "Admin", "pass")
	makeAdmin(t, "admin")

	createApprovedStandard(t, "Blue Monk", "Monk", "bebop")
	createApprovedStandard(t, "Maiden Voyage", "Herbie Hancock", "modal")

	rr := doRequest(r, "GET", "/api/jazz_standards/random", nil, token)
	assertStatus(t, rr, http.StatusOK)
	body := parseBody(t, rr)
	if body["title"] == nil {
		t.Fatal("Expected a standard title in random response")
	}
}

func TestBulkImport(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	adminToken := registerAndLogin(t, r, "admin", "Admin", "pass")
	makeAdmin(t, "admin")
	userToken := registerAndLogin(t, r, "user1", "User", "pass")

	payload := []map[string]interface{}{
		{"title": "Take Five", "composer": "Paul Desmond", "style": "bebop", "key": "Eb-"},
		{"title": "So What", "composer": "Miles Davis", "style": "modal", "key": "D-"},
		{"title": "", "composer": "Nobody", "style": "swing"}, // should fail
	}

	// Non-admin cannot bulk import
	rr := doRequest(r, "POST", "/api/jazz_standards/bulk_import", payload, userToken)
	assertStatus(t, rr, http.StatusForbidden)

	// Admin can
	rr2 := doRequest(r, "POST", "/api/jazz_standards/bulk_import", payload, adminToken)
	assertStatus(t, rr2, http.StatusOK)
	body := parseBody(t, rr2)
	if body["imported"].(float64) != 2 {
		t.Fatalf("Expected 2 imported, got %v", body["imported"])
	}
	if body["failed"].(float64) != 1 {
		t.Fatalf("Expected 1 failed, got %v", body["failed"])
	}
}

func TestAdminStats(t *testing.T) {
	setupTestDB(t)
	r := newRouter()
	adminToken := registerAndLogin(t, r, "admin", "Admin", "pass")
	makeAdmin(t, "admin")
	userToken := registerAndLogin(t, r, "user1", "User", "pass")

	// Non-admin gets 403
	rr := doRequest(r, "GET", "/api/admin/stats", nil, userToken)
	assertStatus(t, rr, http.StatusForbidden)

	rr2 := doRequest(r, "GET", "/api/admin/stats", nil, adminToken)
	assertStatus(t, rr2, http.StatusOK)
	body := parseBody(t, rr2)
	if body["total_users"] == nil {
		t.Fatal("Expected total_users in stats")
	}
}
