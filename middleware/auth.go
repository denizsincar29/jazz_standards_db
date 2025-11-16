package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/denizsincar29/jazz_standards_db/utils"
)

type contextKey string

const UserContextKey contextKey = "user"

// RequireAuth middleware ensures the user is authenticated
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getUserFromRequest(r)
		if err != nil {
			utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireAdmin middleware ensures the user is an admin
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

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// getUserFromRequest extracts user from authorization header or cookie
func getUserFromRequest(r *http.Request) (*models.User, error) {
	var token string

	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Check for Bearer token
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if strings.HasPrefix(authHeader, "Basic ") {
			// TODO: Implement Basic auth if needed
			return nil, http.ErrNoCookie
		}
	}

	// If no Authorization header, check cookie
	if token == "" {
		cookie, err := r.Cookie("token")
		if err != nil {
			return nil, err
		}
		token = cookie.Value
	}

	// Look up user by token
	var user models.User
	if err := database.DB.Where("token = ?", token).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
