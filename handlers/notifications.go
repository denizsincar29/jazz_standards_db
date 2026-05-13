package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/denizsincar29/jazz_standards_db/models"
)

// GetNotifications returns notifications for the current user (unread first, limit 50).
// GET /api/users/me/notifications
func GetNotifications(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID

	var notifications []models.Notification
	database.DB.
		Preload("Actor").
		Preload("JazzStandard").
		Where("recipient_id = ?", currentUserID).
		Order("read ASC, created_at DESC").
		Limit(50).
		Find(&notifications)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// MarkNotificationsRead marks all notifications for the current user as read.
// POST /api/users/me/notifications/read
func MarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID

	database.DB.Model(&models.Notification{}).
		Where("recipient_id = ? AND read = false", currentUserID).
		Update("read", true)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "All notifications marked as read"})
}

// CreateFollowerNotifications inserts a notification for every follower of actorID.
func CreateFollowerNotifications(actorID uint, standardID *uint, actionType string) {
	var follows []models.Follow
	database.DB.Where("following_id = ?", actorID).Find(&follows)

	for _, f := range follows {
		notif := models.Notification{
			RecipientID:    f.FollowerID,
			ActorID:        actorID,
			ActionType:     actionType,
			JazzStandardID: standardID,
		}
		database.DB.Create(&notif)
	}
}
