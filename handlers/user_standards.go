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

type AddUserStandardRequest struct {
	CategoryID *uint  `json:"category_id,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type UpdateUserStandardRequest struct {
	CategoryID *uint   `json:"category_id,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

// AddUserStandard adds a standard to user's known list
func AddUserStandard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	standardID, err := strconv.ParseUint(vars["standard_id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req AddUserStandardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is ok
		req = AddUserStandardRequest{}
	}

	// Check if standard exists
	var standard models.JazzStandard
	if err := database.DB.First(&standard, standardID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not found")
		return
	}

	// Check if user already has this standard
	var existing models.UserStandard
	if err := database.DB.Where("user_id = ? AND jazz_standard_id = ?", user.ID, standardID).First(&existing).Error; err == nil {
		utils.RespondError(w, http.StatusConflict, "Standard already in your list")
		return
	}

	// If category specified, verify it belongs to the user
	if req.CategoryID != nil {
		var category models.Category
		if err := database.DB.Where("id = ? AND user_id = ?", *req.CategoryID, user.ID).First(&category).Error; err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid category")
			return
		}
	}

	// Create user standard
	userStandard := models.UserStandard{
		UserID:         user.ID,
		JazzStandardID: uint(standardID),
		CategoryID:     req.CategoryID,
		Notes:          req.Notes,
	}

	if err := database.DB.Create(&userStandard).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to add standard")
		return
	}

	// Load relationships
	database.DB.Preload("JazzStandard").Preload("Category").First(&userStandard, "user_id = ? AND jazz_standard_id = ?", user.ID, standardID)

	utils.RespondJSON(w, http.StatusCreated, userStandard)
}

// ListUserStandards gets all standards user knows, grouped by category
func ListUserStandards(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var userStandards []models.UserStandard
	if err := database.DB.
		Preload("JazzStandard").
		Preload("Category").
		Where("user_id = ?", user.ID).
		Order("jazz_standard_id ASC").
		Find(&userStandards).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch standards")
		return
	}

	// Group by category
	grouped := make(map[string][]models.UserStandard)
	uncategorized := []models.UserStandard{}

	for _, us := range userStandards {
		if us.Category != nil {
			categoryName := us.Category.Name
			grouped[categoryName] = append(grouped[categoryName], us)
		} else {
			uncategorized = append(uncategorized, us)
		}
	}

	if len(uncategorized) > 0 {
		grouped["Uncategorized"] = uncategorized
	}

	response := map[string]interface{}{
		"standards": userStandards,
		"grouped":   grouped,
		"total":     len(userStandards),
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// UpdateUserStandard updates user's notes or category for a standard
func UpdateUserStandard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	standardID, err := strconv.ParseUint(vars["standard_id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req UpdateUserStandardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find user standard
	var userStandard models.UserStandard
	if err := database.DB.Where("user_id = ? AND jazz_standard_id = ?", user.ID, standardID).First(&userStandard).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not in your list")
		return
	}

	// Update fields
	if req.CategoryID != nil {
		// Verify category belongs to user
		var category models.Category
		if err := database.DB.Where("id = ? AND user_id = ?", *req.CategoryID, user.ID).First(&category).Error; err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid category")
			return
		}
		userStandard.CategoryID = req.CategoryID
	}

	if req.Notes != nil {
		userStandard.Notes = *req.Notes
	}

	if err := database.DB.Save(&userStandard).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update standard")
		return
	}

	// Load relationships
	database.DB.Preload("JazzStandard").Preload("Category").First(&userStandard, "user_id = ? AND jazz_standard_id = ?", user.ID, standardID)

	utils.RespondJSON(w, http.StatusOK, userStandard)
}

// DeleteUserStandard removes standard from user's known list
func DeleteUserStandard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	standardID, err := strconv.ParseUint(vars["standard_id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Delete user standard
	result := database.DB.Where("user_id = ? AND jazz_standard_id = ?", user.ID, standardID).Delete(&models.UserStandard{})
	if result.Error != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to remove standard")
		return
	}

	if result.RowsAffected == 0 {
		utils.RespondError(w, http.StatusNotFound, "Standard not in your list")
		return
	}

	utils.RespondSuccess(w, "Standard removed successfully", nil)
}
