package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/denizsincar29/jazz_standards_db/utils"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register creates a new user account
func Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Name == "" || req.Password == "" {
		utils.RespondError(w, http.StatusBadRequest, "Username, name, and password are required")
		return
	}

	// Check if username exists
	var existingUser models.User
	if err := database.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		utils.RespondError(w, http.StatusConflict, "Username already exists")
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	// Generate token
	token, err := utils.GenerateToken(32)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Create user
	user := models.User{
		Username:     req.Username,
		Name:         req.Name,
		PasswordHash: hashedPassword,
		IsAdmin:      false,
		Token:        &token,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	utils.RespondJSON(w, http.StatusCreated, user)
}

// Login authenticates a user and returns a token
func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find user
	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check password
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		utils.RespondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate new token
	token, err := utils.GenerateToken(32)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Update user token
	user.Token = &token
	if err := database.DB.Save(&user).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update token")
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	utils.RespondJSON(w, http.StatusOK, user)
}

// CreateAdmin creates the first admin user (only works if no users exist)
func CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if any users exist
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		utils.RespondError(w, http.StatusForbidden, "Admin already exists")
		return
	}

	// Validate input
	if req.Username == "" || req.Name == "" || req.Password == "" {
		utils.RespondError(w, http.StatusBadRequest, "Username, name, and password are required")
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	// Generate token
	token, err := utils.GenerateToken(32)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Create admin user
	user := models.User{
		Username:     req.Username,
		Name:         req.Name,
		PasswordHash: hashedPassword,
		IsAdmin:      true,
		Token:        &token,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create admin")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, user)
}

// Logout clears the user's token
func Logout(w http.ResponseWriter, r *http.Request) {
	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	utils.RespondSuccess(w, "Logged out successfully", nil)
}
