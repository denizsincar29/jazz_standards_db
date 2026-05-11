package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/denizsincar29/jazz_standards_db/utils"
)

type contextKey string

const UserContextKey contextKey = "user"

// RequireAuth ensures the request carries a valid session token or pass key.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getUserFromRequest(r)
		if err != nil {
			utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireAdmin ensures the user is authenticated AND is an admin.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getUserFromRequest(r)
		if err != nil {
			utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		if !user.IsAdmin {
			utils.RespondError(w, http.StatusForbidden, "Admin access required")
			return
		}
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// getUserFromRequest resolves the authenticated user from:
//  1. Authorization: Bearer <token>  – session token OR pass key token
//  2. Cookie: token=<session-token>
func getUserFromRequest(r *http.Request) (*models.User, error) {
	var token string

	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}

	if token == "" {
		if c, err := r.Cookie("token"); err == nil {
			token = c.Value
		}
	}

	if token == "" {
		return nil, http.ErrNoCookie
	}

	// 1. Try session token on the users table
	var user models.User
	if err := database.DB.Where("token = ?", token).First(&user).Error; err == nil {
		return &user, nil
	}

	// 2. Try pass key
	var pk models.PassKey
	if err := database.DB.Where("token = ?", token).First(&pk).Error; err != nil {
		return nil, err
	}

	// Update last_used timestamp (best-effort)
	now := time.Now()
	database.DB.Model(&pk).Update("last_used", now)

	if err := database.DB.First(&user, pk.UserID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserFromContext retrieves the user from the request context.
func GetUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
