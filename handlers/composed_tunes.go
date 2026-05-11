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

type CreateComposedTuneRequest struct {
	Title        string `json:"title"`
	Composer     string `json:"composer"` // defaults to user's name
	Style        string `json:"style"`
	Key          string `json:"key"`
	Description  string `json:"description"`
	IrealProLink string `json:"ireal_pro_link"`
	IsPublic     bool   `json:"is_public"`
}

type UpdateComposedTuneRequest struct {
	Title        *string `json:"title,omitempty"`
	Composer     *string `json:"composer,omitempty"`
	Style        *string `json:"style,omitempty"`
	Key          *string `json:"key,omitempty"`
	Description  *string `json:"description,omitempty"`
	IrealProLink *string `json:"ireal_pro_link,omitempty"`
	IsPublic     *bool   `json:"is_public,omitempty"`
}

// CreateComposedTune – POST /api/users/me/compositions
func CreateComposedTune(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateComposedTuneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Title == "" {
		utils.RespondError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Style != "" && !models.IsValidStyle(req.Style) {
		utils.RespondError(w, http.StatusBadRequest, "Invalid style")
		return
	}

	composer := req.Composer
	if composer == "" {
		composer = user.Name
	}

	shareID, err := utils.GenerateToken(16)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate share ID")
		return
	}

	tune := models.ComposedTune{
		UserID:       user.ID,
		ShareID:      shareID,
		Title:        req.Title,
		Composer:     composer,
		Style:        models.JazzStyle(req.Style),
		Key:          req.Key,
		Description:  req.Description,
		IrealProLink: req.IrealProLink,
		IsPublic:     req.IsPublic,
	}
	if err := database.DB.Create(&tune).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create composition")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, tune)
}

// ListComposedTunes – GET /api/users/me/compositions
func ListComposedTunes(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var tunes []models.ComposedTune
	if err := database.DB.Where("user_id = ?", user.ID).Order("title ASC").Find(&tunes).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch compositions")
		return
	}
	utils.RespondJSON(w, http.StatusOK, tunes)
}

// GetSharedTune – GET /api/shared_tune?id=<shareID>  (public, no auth required)
// The receiving user can then accept it into their personal pieces.
func GetSharedTune(w http.ResponseWriter, r *http.Request) {
	shareID := r.URL.Query().Get("id")
	if shareID == "" {
		utils.RespondError(w, http.StatusBadRequest, "id query param is required")
		return
	}

	var tune models.ComposedTune
	if err := database.DB.Preload("User").
		Where("share_id = ? AND is_public = true", shareID).
		First(&tune).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Tune not found or not public")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"tune":    tune,
		"message": "Accept this tune to add it to your personal pieces list",
	})
}

// AcceptSharedTune – POST /api/shared_tune/accept?id=<shareID>
// Copies the composed tune into the user's personal pieces.
func AcceptSharedTune(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	shareID := r.URL.Query().Get("id")
	if shareID == "" {
		utils.RespondError(w, http.StatusBadRequest, "id query param is required")
		return
	}

	var tune models.ComposedTune
	if err := database.DB.Where("share_id = ? AND is_public = true", shareID).First(&tune).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Tune not found or not public")
		return
	}

	// Check if user already has it
	var existing models.PersonalPiece
	if err := database.DB.Where("user_id = ? AND title = ? AND composer = ?",
		user.ID, tune.Title, tune.Composer).First(&existing).Error; err == nil {
		utils.RespondError(w, http.StatusConflict, "You already have this piece in your list")
		return
	}

	piece := models.PersonalPiece{
		UserID:       user.ID,
		Title:        tune.Title,
		Composer:     tune.Composer,
		Style:        tune.Style,
		Key:          tune.Key,
		Notes:        "Shared composition: " + tune.Description,
		IrealProLink: tune.IrealProLink,
		IsPublic:     false,
	}
	if err := database.DB.Create(&piece).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to accept tune")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"piece":   piece,
		"message": "Tune added to your personal pieces",
	})
}

// UpdateComposedTune – PUT /api/users/me/compositions/{id}
func UpdateComposedTune(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid composition ID")
		return
	}

	var tune models.ComposedTune
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&tune).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Composition not found")
		return
	}

	var req UpdateComposedTuneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Title != nil {
		tune.Title = *req.Title
	}
	if req.Composer != nil {
		tune.Composer = *req.Composer
	}
	if req.Style != nil {
		if *req.Style != "" && !models.IsValidStyle(*req.Style) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid style")
			return
		}
		tune.Style = models.JazzStyle(*req.Style)
	}
	if req.Key != nil {
		tune.Key = *req.Key
	}
	if req.Description != nil {
		tune.Description = *req.Description
	}
	if req.IrealProLink != nil {
		tune.IrealProLink = *req.IrealProLink
	}
	if req.IsPublic != nil {
		tune.IsPublic = *req.IsPublic
	}

	if err := database.DB.Save(&tune).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update composition")
		return
	}
	utils.RespondJSON(w, http.StatusOK, tune)
}

// DeleteComposedTune – DELETE /api/users/me/compositions/{id}
func DeleteComposedTune(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid composition ID")
		return
	}
	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.ComposedTune{})
	if result.Error != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete composition")
		return
	}
	if result.RowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Composition not found")
		return
	}
	utils.RespondSuccess(w, "Composition deleted", nil)
}
