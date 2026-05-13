package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/gorilla/mux"
)

// FollowUser follows a user by username.
// POST /api/users/{username}/follow
func FollowUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID
	username := mux.Vars(r)["username"]

	var target models.User
	if err := database.DB.Where("username = ?", username).First(&target).Error; err != nil {
		http.Error(w, `{"message":"User not found"}`, http.StatusNotFound)
		return
	}

	if target.ID == currentUserID {
		http.Error(w, `{"message":"Cannot follow yourself"}`, http.StatusBadRequest)
		return
	}

	follow := models.Follow{FollowerID: currentUserID, FollowingID: target.ID}
	database.DB.FirstOrCreate(&follow, follow)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Following"})
}

// UnfollowUser unfollows a user by username.
// DELETE /api/users/{username}/follow
func UnfollowUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID
	username := mux.Vars(r)["username"]

	var target models.User
	if err := database.DB.Where("username = ?", username).First(&target).Error; err != nil {
		http.Error(w, `{"message":"User not found"}`, http.StatusNotFound)
		return
	}

	database.DB.Delete(&models.Follow{}, "follower_id = ? AND following_id = ?", currentUserID, target.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Unfollowed"})
}

// GetFollowing returns the list of users the current user follows.
// GET /api/users/me/following
func GetFollowing(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID

	var follows []models.Follow
	database.DB.Preload("Following").Where("follower_id = ?", currentUserID).Find(&follows)

	users := make([]models.User, 0, len(follows))
	for _, f := range follows {
		if f.Following != nil {
			users = append(users, *f.Following)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetFollowers returns the list of users who follow the current user.
// GET /api/users/me/followers
func GetFollowers(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID

	var follows []models.Follow
	database.DB.Preload("Follower").Where("following_id = ?", currentUserID).Find(&follows)

	users := make([]models.User, 0, len(follows))
	for _, f := range follows {
		if f.Follower != nil {
			users = append(users, *f.Follower)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// IsFollowing returns whether the current user follows a given username.
// GET /api/users/{username}/follow
func IsFollowing(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID
	username := mux.Vars(r)["username"]

	var target models.User
	if err := database.DB.Where("username = ?", username).First(&target).Error; err != nil {
		http.Error(w, `{"message":"User not found"}`, http.StatusNotFound)
		return
	}

	var count int64
	database.DB.Model(&models.Follow{}).
		Where("follower_id = ? AND following_id = ?", currentUserID, target.ID).
		Count(&count)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"following": count > 0})
}
