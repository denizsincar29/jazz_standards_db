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

type CreatePersonalPieceRequest struct {
	Title        string `json:"title"`
	Composer     string `json:"composer"`
	Style        string `json:"style"`
	Key          string `json:"key"`
	Notes        string `json:"notes"`
	IrealProLink string `json:"ireal_pro_link"`
	IsPublic     bool   `json:"is_public"`
}

type UpdatePersonalPieceRequest struct {
	Title        *string `json:"title,omitempty"`
	Composer     *string `json:"composer,omitempty"`
	Style        *string `json:"style,omitempty"`
	Key          *string `json:"key,omitempty"`
	Notes        *string `json:"notes,omitempty"`
	IrealProLink *string `json:"ireal_pro_link,omitempty"`
	IsPublic     *bool   `json:"is_public,omitempty"`
}

// CreatePersonalPiece – POST /api/users/me/pieces
func CreatePersonalPiece(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreatePersonalPieceRequest
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

	piece := models.PersonalPiece{
		UserID:       user.ID,
		Title:        req.Title,
		Composer:     req.Composer,
		Style:        models.JazzStyle(req.Style),
		Key:          req.Key,
		Notes:        req.Notes,
		IrealProLink: req.IrealProLink,
		IsPublic:     req.IsPublic,
	}
	if err := database.DB.Create(&piece).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create piece")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, piece)
}

// ListPersonalPieces – GET /api/users/me/pieces
func ListPersonalPieces(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var pieces []models.PersonalPiece
	if err := database.DB.Where("user_id = ?", user.ID).
		Order("title ASC").Find(&pieces).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch pieces")
		return
	}
	utils.RespondJSON(w, http.StatusOK, pieces)
}

// UpdatePersonalPiece – PUT /api/users/me/pieces/{id}
func UpdatePersonalPiece(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid piece ID")
		return
	}

	var piece models.PersonalPiece
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&piece).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Piece not found")
		return
	}

	var req UpdatePersonalPieceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Title != nil {
		piece.Title = *req.Title
	}
	if req.Composer != nil {
		piece.Composer = *req.Composer
	}
	if req.Style != nil {
		if *req.Style != "" && !models.IsValidStyle(*req.Style) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid style")
			return
		}
		piece.Style = models.JazzStyle(*req.Style)
	}
	if req.Key != nil {
		piece.Key = *req.Key
	}
	if req.Notes != nil {
		piece.Notes = *req.Notes
	}
	if req.IrealProLink != nil {
		piece.IrealProLink = *req.IrealProLink
	}
	if req.IsPublic != nil {
		piece.IsPublic = *req.IsPublic
	}

	if err := database.DB.Save(&piece).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update piece")
		return
	}
	utils.RespondJSON(w, http.StatusOK, piece)
}

// DeletePersonalPiece – DELETE /api/users/me/pieces/{id}
func DeletePersonalPiece(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid piece ID")
		return
	}
	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.PersonalPiece{})
	if result.Error != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete piece")
		return
	}
	if result.RowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Piece not found")
		return
	}
	utils.RespondSuccess(w, "Piece deleted", nil)
}
