package handlers

import (
	"net/http"
	"strconv"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/denizsincar29/jazz_standards_db/utils"
	"github.com/gorilla/mux"
)

// GetMe returns the current authenticated user
func GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	utils.RespondJSON(w, http.StatusOK, user)
}

// ListUsers lists all users (admin only)
func ListUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	utils.RespondJSON(w, http.StatusOK, users)
}

// DeleteUser deletes a user (self or admin)
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if user is deleting themselves or is admin
	if currentUser.ID != uint(userID) && !currentUser.IsAdmin {
		utils.RespondError(w, http.StatusForbidden, "Forbidden")
		return
	}

	// Delete user
	if err := database.DB.Delete(&models.User{}, userID).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	utils.RespondSuccess(w, "User deleted successfully", nil)
}
