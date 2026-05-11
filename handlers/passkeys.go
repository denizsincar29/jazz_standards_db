package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/denizsincar29/jazz_standards_db/utils"
	"github.com/gorilla/mux"
)

type CreatePassKeyRequest struct {
	Name string `json:"name"`
}

// CreatePassKey – create a new named API pass key for the current user.
// POST /api/users/me/passkeys
// The raw token is returned ONLY in this response.
func CreatePassKey(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreatePassKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		utils.RespondError(w, http.StatusBadRequest, "name is required")
		return
	}

	token, err := utils.GenerateToken(32)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	pk := models.PassKey{
		UserID:    user.ID,
		Name:      req.Name,
		Token:     token,
		TokenHint: token[len(token)-4:],
	}
	if err := database.DB.Create(&pk).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create pass key")
		return
	}

	// Return the full token only once
	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         pk.ID,
		"name":       pk.Name,
		"token":      token, // shown once
		"token_hint": pk.TokenHint,
		"created_at": pk.CreatedAt,
		"message":    "Save this token – it will not be shown again",
	})
}

// ListPassKeys – list the current user's pass keys (tokens are hidden).
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
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch pass keys")
		return
	}
	utils.RespondJSON(w, http.StatusOK, keys)
}

// DeletePassKey – revoke a pass key by ID.
// DELETE /api/users/me/passkeys/{id}
func DeletePassKey(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid pass key ID")
		return
	}

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.PassKey{})
	if result.Error != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete pass key")
		return
	}
	if result.RowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Pass key not found")
		return
	}
	utils.RespondSuccess(w, "Pass key revoked", nil)
}
