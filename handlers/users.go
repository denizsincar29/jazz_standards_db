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

type UpdateProfileRequest struct {
	Name          *string `json:"name,omitempty"`
	PublicProfile *bool   `json:"public_profile,omitempty"`
}

// GetMe returns the current authenticated user.
func GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	utils.RespondJSON(w, http.StatusOK, user)
}

// UpdateMe – update profile name or privacy settings.
// PATCH /api/users/me
func UpdateMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name != nil && *req.Name != "" {
		user.Name = *req.Name
	}
	if req.PublicProfile != nil {
		user.PublicProfile = *req.PublicProfile
	}

	if err := database.DB.Save(user).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	utils.RespondJSON(w, http.StatusOK, user)
}

// ListUsers – admin only.
func ListUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	utils.RespondJSON(w, http.StatusOK, users)
}

// DeleteUser – self or admin.
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	current := middleware.GetUserFromContext(r)
	if current == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if current.ID != uint(userID) && !current.IsAdmin {
		utils.RespondError(w, http.StatusForbidden, "Forbidden")
		return
	}

	if err := database.DB.Delete(&models.User{}, userID).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}
	utils.RespondSuccess(w, "User deleted successfully", nil)
}
