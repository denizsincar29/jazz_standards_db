package handlers

// WebAuthn passkey registration and authentication handlers.
//
// Registration flow (requires existing session/JWT):
//   POST /api/users/me/passkeys/begin   → returns PublicKeyCredentialCreationOptions
//   POST /api/users/me/passkeys/finish  → stores credential, returns {id, name}
//
// Authentication flow (no prior auth needed):
//   POST /api/auth/passkey/begin   → accepts {username}, returns PublicKeyCredentialRequestOptions
//   POST /api/auth/passkey/finish  → validates assertion, returns {token, user}
//
// Management (existing session required):
//   GET    /api/users/me/passkeys        → list credentials
//   DELETE /api/users/me/passkeys/{id}   → revoke credential

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/denizsincar29/jazz_standards_db/config"
	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/denizsincar29/jazz_standards_db/utils"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/mux"
)

// ── In-process challenge sessions ────────────────────────────────────────────
// For a single-process server this map is sufficient.  A multi-process
// deployment should move these to Redis or the DB.

var (
	sessionMu    sync.Mutex
	regSessions  = map[uint]*webauthn.SessionData{}   // userID  → session
	authSessions = map[string]*webauthn.SessionData{} // username → session
)

// ── WebAuthn instance ─────────────────────────────────────────────────────────

// getWebAuthn builds a *webauthn.WebAuthn from runtime config.
// RPID must be the plain domain (no scheme, no port).
func getWebAuthn() (*webauthn.WebAuthn, error) {
	rpid := config.AppConfig.RPID
	if rpid == "" {
		rpid = "localhost"
	}
	origins := config.AppConfig.RPOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:8000", "https://" + rpid}
	}
	return webauthn.New(&webauthn.Config{
		RPDisplayName: "Jazz Standards DB",
		RPID:          rpid,
		RPOrigins:     origins,
	})
}

// ── webauthn.User adapter ────────────────────────────────────────────────────

type webAuthnUser struct {
	user  models.User
	creds []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	// Encode the uint user ID as 8 big-endian bytes.
	id := make([]byte, 8)
	uid := u.user.ID
	for i := 7; i >= 0; i-- {
		id[i] = byte(uid & 0xff)
		uid >>= 8
	}
	return id
}
func (u *webAuthnUser) WebAuthnName() string                        { return u.user.Username }
func (u *webAuthnUser) WebAuthnDisplayName() string                 { return u.user.Name }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential  { return u.creds }

// loadWebAuthnUser loads a User and all its stored WebAuthn credentials.
func loadWebAuthnUser(user models.User) (*webAuthnUser, error) {
	var keys []models.PassKey
	err := database.DB.
		Where("user_id = ? AND octet_length(credential_id) > 0", user.ID).
		Find(&keys).Error
	if err != nil {
		return nil, err
	}
	creds := make([]webauthn.Credential, 0, len(keys))
	for _, k := range keys {
		var transports []protocol.AuthenticatorTransport
		for _, t := range strings.Split(k.Transports, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				transports = append(transports, protocol.AuthenticatorTransport(t))
			}
		}
		creds = append(creds, webauthn.Credential{
			ID:        k.CredentialID,
			PublicKey: k.PublicKey,
			Transport: transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    k.AAGUID,
				SignCount: k.SignCount,
			},
		})
	}
	return &webAuthnUser{user: user, creds: creds}, nil
}

// ── Tiny helper ───────────────────────────────────────────────────────────────

// newJSONRequest wraps a JSON byte slice in an *http.Request so the
// webauthn library's ParseCredential* helpers can read it.
func newJSONRequest(b []byte) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// bytesReadCloser is an io.ReadCloser backed by a []byte.
type bytesReadCloser struct{ *bytes.Reader }

func (bytesReadCloser) Close() error { return nil }

func newBytesReadCloser(b []byte) io.ReadCloser {
	return bytesReadCloser{bytes.NewReader(b)}
}

// ── Registration ─────────────────────────────────────────────────────────────

// BeginRegistration starts the WebAuthn credential creation ceremony.
// POST /api/users/me/passkeys/begin
func BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	wa, err := getWebAuthn()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "WebAuthn configuration error")
		return
	}

	wau, err := loadWebAuthnUser(*user)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to load credentials")
		return
	}

	options, session, err := wa.BeginRegistration(wau)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to begin registration: "+err.Error())
		return
	}

	sessionMu.Lock()
	regSessions[user.ID] = session
	sessionMu.Unlock()

	utils.RespondJSON(w, http.StatusOK, options)
}

// FinishRegistration completes registration and persists the new credential.
// POST /api/users/me/passkeys/finish
//
// Request body (JSON):
//
//	{
//	  "name": "My MacBook Touch ID",   // optional label
//	  "id": "...",                      // PublicKeyCredential fields
//	  "rawId": "...",
//	  "type": "public-key",
//	  "response": { ... }
//	}
func FinishRegistration(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessionMu.Lock()
	session, ok := regSessions[user.ID]
	sessionMu.Unlock()
	if !ok {
		utils.RespondError(w, http.StatusBadRequest, "No pending registration session; call /begin first")
		return
	}

	// Buffer the body so we can extract "name" and also feed bytes to the library.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	// Extract optional "name" field.
	var wrapper map[string]json.RawMessage
	keyName := "My Passkey"
	if err := json.Unmarshal(bodyBytes, &wrapper); err == nil {
		if nameJSON, ok := wrapper["name"]; ok {
			var n string
			if err := json.Unmarshal(nameJSON, &n); err == nil && n != "" {
				keyName = n
			}
		}
	}

	wa, err := getWebAuthn()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "WebAuthn configuration error")
		return
	}

	wau, err := loadWebAuthnUser(*user)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to load credentials")
		return
	}

	// Feed the original body bytes to the library.
	credential, err := wa.FinishRegistration(wau, *session, newJSONRequest(bodyBytes))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Registration verification failed: "+err.Error())
		return
	}

	// Clear the pending registration session.
	sessionMu.Lock()
	delete(regSessions, user.ID)
	sessionMu.Unlock()

	// Serialize transports list.
	var transportStrs []string
	for _, t := range credential.Transport {
		transportStrs = append(transportStrs, string(t))
	}

	pk := models.PassKey{
		UserID:       user.ID,
		Name:         keyName,
		CredentialID: credential.ID,
		PublicKey:    credential.PublicKey,
		AAGUID:       credential.Authenticator.AAGUID,
		SignCount:    credential.Authenticator.SignCount,
		Transports:   strings.Join(transportStrs, ","),
	}
	if err := database.DB.Create(&pk).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to save credential")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         pk.ID,
		"name":       pk.Name,
		"created_at": pk.CreatedAt,
	})
}

// ── Authentication ────────────────────────────────────────────────────────────

type beginAuthRequest struct {
	Username string `json:"username"`
}

// BeginAuthentication starts the WebAuthn assertion ceremony.
// POST /api/auth/passkey/begin  body: {"username":"..."}
func BeginAuthentication(w http.ResponseWriter, r *http.Request) {
	var req beginAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		utils.RespondError(w, http.StatusBadRequest, "username is required")
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		// Avoid username enumeration: return same error as "no passkeys".
		utils.RespondError(w, http.StatusBadRequest, "No passkeys found for this user")
		return
	}

	wa, err := getWebAuthn()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "WebAuthn configuration error")
		return
	}

	wau, err := loadWebAuthnUser(user)
	if err != nil || len(wau.creds) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "No passkeys registered for this user")
		return
	}

	options, session, err := wa.BeginLogin(wau)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to begin authentication: "+err.Error())
		return
	}

	sessionMu.Lock()
	authSessions[req.Username] = session
	sessionMu.Unlock()

	utils.RespondJSON(w, http.StatusOK, options)
}

// FinishAuthentication completes assertion verification and issues a session token.
// POST /api/auth/passkey/finish?username=<username>
func FinishAuthentication(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		utils.RespondError(w, http.StatusBadRequest, "username query parameter is required")
		return
	}

	sessionMu.Lock()
	session, ok := authSessions[username]
	sessionMu.Unlock()
	if !ok {
		utils.RespondError(w, http.StatusBadRequest, "No pending authentication session; call /begin first")
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "User not found")
		return
	}

	wa, err := getWebAuthn()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "WebAuthn configuration error")
		return
	}

	wau, err := loadWebAuthnUser(user)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to load credentials")
		return
	}

	credential, err := wa.FinishLogin(wau, *session, r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "Authentication failed: "+err.Error())
		return
	}

	// Clear authentication session.
	sessionMu.Lock()
	delete(authSessions, username)
	sessionMu.Unlock()

	// Persist updated sign count and last-used timestamp.
	now := time.Now()
	database.DB.Model(&models.PassKey{}).
		Where("user_id = ? AND credential_id = ?", user.ID, credential.ID).
		Updates(map[string]interface{}{
			"sign_count": credential.Authenticator.SignCount,
			"last_used":  now,
		})

	// Issue a new session token (same mechanism as password login).
	token, err := utils.GenerateToken(32)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	user.Token = &token
	if err := database.DB.Save(&user).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     cookiePath(),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// ── Management ────────────────────────────────────────────────────────────────

// ListPassKeys lists the current user's passkeys (no secrets exposed).
// GET /api/users/me/passkeys
func ListPassKeys(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var keys []models.PassKey
	if err := database.DB.Where("user_id = ?", user.ID).Order("created_at DESC").
		Find(&keys).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch passkeys")
		return
	}
	utils.RespondJSON(w, http.StatusOK, keys)
}

// DeletePassKey revokes a passkey by ID.
// DELETE /api/users/me/passkeys/{id}
func DeletePassKey(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid passkey ID")
		return
	}
	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.PassKey{})
	if result.Error != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete passkey")
		return
	}
	if result.RowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Passkey not found")
		return
	}
	utils.RespondSuccess(w, "Passkey revoked", nil)
}
