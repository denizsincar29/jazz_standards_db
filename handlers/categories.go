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

type CreateCategoryRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type UpdateCategoryRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}

// CreateCategory creates a new category
func CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Validate input
	if req.Name == "" {
		utils.RespondError(w, http.StatusBadRequest, "Category name is required")
		return
	}

	// Set default color if not provided
	if req.Color == "" {
		req.Color = "#000000"
	}

	// Check if category already exists for this user
	var existing models.Category
	if err := database.DB.Where("user_id = ? AND name = ?", user.ID, req.Name).First(&existing).Error; err == nil {
		utils.RespondError(w, http.StatusConflict, "Category already exists")
		return
	}

	// Create category
	category := models.Category{
		UserID: user.ID,
		Name:   req.Name,
		Color:  req.Color,
	}

	if err := database.DB.Create(&category).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create category")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, category)
}

// ListCategories lists user's categories
func ListCategories(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var categories []models.Category
	if err := database.DB.Where("user_id = ?", user.ID).Order("name ASC").Find(&categories).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch categories")
		return
	}

	utils.RespondJSON(w, http.StatusOK, categories)
}

// UpdateCategory updates a category
func UpdateCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	categoryID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid category ID")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find category
	var category models.Category
	if err := database.DB.Where("id = ? AND user_id = ?", categoryID, user.ID).First(&category).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Category not found")
		return
	}

	// Update fields
	if req.Name != nil {
		// Check for duplicate name
		var existing models.Category
		if err := database.DB.Where("user_id = ? AND name = ? AND id != ?", user.ID, *req.Name, categoryID).First(&existing).Error; err == nil {
			utils.RespondError(w, http.StatusConflict, "Category with that name already exists")
			return
		}
		category.Name = *req.Name
	}

	if req.Color != nil {
		category.Color = *req.Color
	}

	if err := database.DB.Save(&category).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update category")
		return
	}

	utils.RespondJSON(w, http.StatusOK, category)
}

// DeleteCategory deletes a category (sets standards to no category)
func DeleteCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	categoryID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid category ID")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Verify category belongs to user
	var category models.Category
	if err := database.DB.Where("id = ? AND user_id = ?", categoryID, user.ID).First(&category).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Category not found")
		return
	}

	// Delete category (cascade will set user_standards.category_id to NULL due to ON DELETE SET NULL)
	if err := database.DB.Delete(&category).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete category")
		return
	}

	utils.RespondSuccess(w, "Category deleted successfully", nil)
}
