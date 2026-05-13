package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/denizsincar29/jazz_standards_db/utils"
	"github.com/gorilla/mux"
)

type AddUserStandardRequest struct {
	CategoryID  *uint  `json:"category_id,omitempty"`
	Notes       string `json:"notes,omitempty"`
	Proficiency string `json:"proficiency,omitempty"`
}

type UpdateUserStandardRequest struct {
	CategoryID  *uint   `json:"category_id,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	Proficiency *string `json:"proficiency,omitempty"`
}

type LogPracticeRequest struct {
	DurationMin int    `json:"duration_min"`
	Notes       string `json:"notes,omitempty"`
	PracticedAt string `json:"practiced_at,omitempty"` // RFC3339; defaults to now
}

// AddUserStandard – add a standard to the user's list.
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
	json.NewDecoder(r.Body).Decode(&req) //nolint – empty body is fine

	// Proficiency validation
	proficiency := models.ProficiencyLearning
	if req.Proficiency != "" {
		if !models.IsValidProficiency(req.Proficiency) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid proficiency level")
			return
		}
		proficiency = models.Proficiency(req.Proficiency)
	}

	// Check standard exists and is approved
	var standard models.JazzStandard
	if err := database.DB.Where("id = ? AND status = ?", standardID, models.StatusApproved).
		First(&standard).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not found")
		return
	}

	// Duplicate check
	var existing models.UserStandard
	if err := database.DB.Where("user_id = ? AND jazz_standard_id = ?", user.ID, standardID).
		First(&existing).Error; err == nil {
		utils.RespondError(w, http.StatusConflict, "Standard already in your list")
		return
	}

	// Validate category
	if req.CategoryID != nil {
		var cat models.Category
		if err := database.DB.Where("id = ? AND user_id = ?", *req.CategoryID, user.ID).
			First(&cat).Error; err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid category")
			return
		}
	}

	us := models.UserStandard{
		UserID:         user.ID,
		JazzStandardID: uint(standardID),
		CategoryID:     req.CategoryID,
		Notes:          req.Notes,
		Proficiency:    proficiency,
	}
	if err := database.DB.Create(&us).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to add standard")
		return
	}

	database.DB.Preload("JazzStandard").Preload("Category").
		First(&us, "user_id = ? AND jazz_standard_id = ?", user.ID, standardID)

	// Notify followers if proficiency is know_it or master
	if us.Proficiency == models.ProficiencyKnowIt || us.Proficiency == models.ProficiencyMaster {
		actionType := "learned"
		if us.Proficiency == models.ProficiencyMaster {
			actionType = "mastered"
		}
		sid := uint(standardID)
		go CreateFollowerNotifications(user.ID, &sid, actionType)
	}

	utils.RespondJSON(w, http.StatusCreated, us)
}

// ListUserStandards – current user's full list grouped by category.
func ListUserStandards(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	profFilter := r.URL.Query().Get("proficiency")
	q := database.DB.Preload("JazzStandard").Preload("Category").
		Where("user_id = ?", user.ID)
	if profFilter != "" {
		if !models.IsValidProficiency(profFilter) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid proficiency filter")
			return
		}
		q = q.Where("proficiency = ?", profFilter)
	}

	var userStandards []models.UserStandard
	if err := q.Order("jazz_standard_id ASC").Find(&userStandards).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch standards")
		return
	}

	// Group by category
	grouped := make(map[string][]models.UserStandard)
	uncategorized := []models.UserStandard{}
	for _, us := range userStandards {
		if us.Category != nil {
			grouped[us.Category.Name] = append(grouped[us.Category.Name], us)
		} else {
			uncategorized = append(uncategorized, us)
		}
	}
	if len(uncategorized) > 0 {
		grouped["Uncategorized"] = uncategorized
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"standards": userStandards,
		"grouped":   grouped,
		"total":     len(userStandards),
	})
}

// GetPublicUserStandards – view another user's list (only if their profile is public).
func GetPublicUserStandards(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUsername := vars["username"]

	var targetUser models.User
	if err := database.DB.Where("username = ?", targetUsername).First(&targetUser).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "User not found")
		return
	}
	if !targetUser.PublicProfile {
		utils.RespondError(w, http.StatusForbidden, "This user's list is private")
		return
	}

	var userStandards []models.UserStandard
	if err := database.DB.Preload("JazzStandard").Preload("Category").
		Where("user_id = ?", targetUser.ID).
		Order("jazz_standard_id ASC").
		Find(&userStandards).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch standards")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"user":      map[string]interface{}{"id": targetUser.ID, "username": targetUser.Username, "name": targetUser.Name},
		"standards": userStandards,
		"total":     len(userStandards),
	})
}

// UpdateUserStandard – update notes/category/proficiency on a list entry.
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

	var us models.UserStandard
	if err := database.DB.Where("user_id = ? AND jazz_standard_id = ?", user.ID, standardID).
		First(&us).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not in your list")
		return
	}

	if req.CategoryID != nil {
		var cat models.Category
		if err := database.DB.Where("id = ? AND user_id = ?", *req.CategoryID, user.ID).
			First(&cat).Error; err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid category")
			return
		}
		us.CategoryID = req.CategoryID
	}
	if req.Notes != nil {
		us.Notes = *req.Notes
	}
	if req.Proficiency != nil {
		if !models.IsValidProficiency(*req.Proficiency) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid proficiency level")
			return
		}
		us.Proficiency = models.Proficiency(*req.Proficiency)
	}

	if err := database.DB.Save(&us).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update standard")
		return
	}

	// Notify followers if proficiency changed to know_it or master
	if req.Proficiency != nil {
		newProf := models.Proficiency(*req.Proficiency)
		if newProf == models.ProficiencyKnowIt || newProf == models.ProficiencyMaster {
			actionType := "learned"
			if newProf == models.ProficiencyMaster {
				actionType = "mastered"
			}
			sid := uint(standardID)
			go CreateFollowerNotifications(user.ID, &sid, actionType)
		}
	}

	database.DB.Preload("JazzStandard").Preload("Category").
		First(&us, "user_id = ? AND jazz_standard_id = ?", user.ID, standardID)
	utils.RespondJSON(w, http.StatusOK, us)
}

// DeleteUserStandard – remove a standard from the user's list.
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

	result := database.DB.
		Where("user_id = ? AND jazz_standard_id = ?", user.ID, standardID).
		Delete(&models.UserStandard{})
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

// LogPractice – log a practice session for a standard in the user's list.
// POST /api/users/me/standards/{standard_id}/practice
func LogPractice(w http.ResponseWriter, r *http.Request) {
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

	var req LogPracticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.DurationMin < 0 {
		utils.RespondError(w, http.StatusBadRequest, "duration_min must be >= 0")
		return
	}

	// Ensure the standard is in the user's list
	var us models.UserStandard
	if err := database.DB.Where("user_id = ? AND jazz_standard_id = ?", user.ID, standardID).
		First(&us).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not in your list")
		return
	}

	practicedAt := time.Now()
	if req.PracticedAt != "" {
		if t, err := time.Parse(time.RFC3339, req.PracticedAt); err == nil {
			practicedAt = t
		}
	}

	log := models.PracticeLog{
		UserID:         user.ID,
		JazzStandardID: uint(standardID),
		DurationMin:    req.DurationMin,
		Notes:          req.Notes,
		PracticedAt:    practicedAt,
	}
	if err := database.DB.Create(&log).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to log practice")
		return
	}
	database.DB.Preload("JazzStandard").First(&log, log.ID)
	utils.RespondJSON(w, http.StatusCreated, log)
}

// ListPracticeLogs – list practice logs for a user (optionally filtered by standard).
// GET /api/users/me/practice?standard_id=X
func ListPracticeLogs(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	q := database.DB.Preload("JazzStandard").Where("user_id = ?", user.ID)
	if sid := r.URL.Query().Get("standard_id"); sid != "" {
		q = q.Where("jazz_standard_id = ?", sid)
	}

	var logs []models.PracticeLog
	if err := q.Order("practiced_at DESC").Find(&logs).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch logs")
		return
	}

	var totalMin int64
	database.DB.Model(&models.PracticeLog{}).Where("user_id = ?", user.ID).
		Select("COALESCE(SUM(duration_min),0)").Scan(&totalMin)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":              logs,
		"total_entries":     len(logs),
		"total_minutes":     totalMin,
	})
}
