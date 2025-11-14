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

type CreateStandardRequest struct {
	Title          string `json:"title"`
	Composer       string `json:"composer"`
	AdditionalNote string `json:"additional_note"`
	Style          string `json:"style"`
}

type UpdateStandardRequest struct {
	Title          *string `json:"title,omitempty"`
	Composer       *string `json:"composer,omitempty"`
	AdditionalNote *string `json:"additional_note,omitempty"`
	Style          *string `json:"style,omitempty"`
}

// CreateStandard creates a new jazz standard (admin only)
func CreateStandard(w http.ResponseWriter, r *http.Request) {
	var req CreateStandardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Title == "" || req.Composer == "" || req.Style == "" {
		utils.RespondError(w, http.StatusBadRequest, "Title, composer, and style are required")
		return
	}

	// Validate style
	if !models.IsValidStyle(req.Style) {
		utils.RespondError(w, http.StatusBadRequest, "Invalid style")
		return
	}

	// Get current user
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if standard already exists
	var existing models.JazzStandard
	if err := database.DB.Where("title = ?", req.Title).First(&existing).Error; err == nil {
		utils.RespondError(w, http.StatusConflict, "Standard already exists")
		return
	}

	// Create standard
	standard := models.JazzStandard{
		Title:          req.Title,
		Composer:       req.Composer,
		AdditionalNote: req.AdditionalNote,
		Style:          models.JazzStyle(req.Style),
		CreatedBy:      &user.ID,
	}

	if err := database.DB.Create(&standard).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create standard")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, standard)
}

// ListStandards lists all jazz standards with pagination and search
func ListStandards(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 100
	}

	search := r.URL.Query().Get("search")
	style := r.URL.Query().Get("style")

	offset := (page - 1) * limit

	query := database.DB.Model(&models.JazzStandard{})

	// Apply filters
	if search != "" {
		query = query.Where("title ILIKE ? OR composer ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if style != "" {
		if !models.IsValidStyle(style) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid style")
			return
		}
		query = query.Where("style = ?", style)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get standards
	var standards []models.JazzStandard
	if err := query.Offset(offset).Limit(limit).Order("title ASC").Find(&standards).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch standards")
		return
	}

	response := map[string]interface{}{
		"standards": standards,
		"page":      page,
		"limit":     limit,
		"total":     total,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// GetStandard returns a specific jazz standard
func GetStandard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	standardID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}

	var standard models.JazzStandard
	if err := database.DB.Preload("Creator").First(&standard, standardID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not found")
		return
	}

	utils.RespondJSON(w, http.StatusOK, standard)
}

// UpdateStandard updates a jazz standard (admin only)
func UpdateStandard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	standardID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}

	var req UpdateStandardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find standard
	var standard models.JazzStandard
	if err := database.DB.First(&standard, standardID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not found")
		return
	}

	// Update fields
	if req.Title != nil {
		standard.Title = *req.Title
	}
	if req.Composer != nil {
		standard.Composer = *req.Composer
	}
	if req.AdditionalNote != nil {
		standard.AdditionalNote = *req.AdditionalNote
	}
	if req.Style != nil {
		if !models.IsValidStyle(*req.Style) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid style")
			return
		}
		standard.Style = models.JazzStyle(*req.Style)
	}

	if err := database.DB.Save(&standard).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update standard")
		return
	}

	utils.RespondJSON(w, http.StatusOK, standard)
}

// DeleteStandard deletes a jazz standard (admin only)
func DeleteStandard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	standardID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}

	if err := database.DB.Delete(&models.JazzStandard{}, standardID).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete standard")
		return
	}

	utils.RespondSuccess(w, "Standard deleted successfully", nil)
}
