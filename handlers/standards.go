package handlers

import (
	"encoding/csv"
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/denizsincar29/jazz_standards_db/database"
	"github.com/denizsincar29/jazz_standards_db/middleware"
	"github.com/denizsincar29/jazz_standards_db/models"
	"github.com/denizsincar29/jazz_standards_db/utils"
	"github.com/gorilla/mux"
)

// ── Request/response types ────────────────────────────────────────────────────

type CreateStandardRequest struct {
	Title          string `json:"title"`
	Composer       string `json:"composer"`
	AdditionalNote string `json:"additional_note"`
	Style          string `json:"style"`
	Key            string `json:"key"`
	IrealProLink   string `json:"ireal_pro_link"`
}

type UpdateStandardRequest struct {
	Title          *string `json:"title,omitempty"`
	Composer       *string `json:"composer,omitempty"`
	AdditionalNote *string `json:"additional_note,omitempty"`
	Style          *string `json:"style,omitempty"`
	Key            *string `json:"key,omitempty"`
	IrealProLink   *string `json:"ireal_pro_link,omitempty"`
}

// BulkImportItem is a single entry in a bulk-import JSON array.
type BulkImportItem struct {
	Title          string `json:"title"`
	Composer       string `json:"composer"`
	Style          string `json:"style"`
	Key            string `json:"key"`
	AdditionalNote string `json:"additional_note"`
	IrealProLink   string `json:"ireal_pro_link"`
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// CreateStandard – any authenticated user can submit; admins auto-approve.
func CreateStandard(w http.ResponseWriter, r *http.Request) {
	var req CreateStandardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" || req.Composer == "" || req.Style == "" {
		utils.RespondError(w, http.StatusBadRequest, "title, composer, and style are required")
		return
	}
	if !models.IsValidStyle(req.Style) {
		utils.RespondError(w, http.StatusBadRequest, "Invalid style")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check for duplicate
	var existing models.JazzStandard
	if err := database.DB.Where("title = ? AND status IN ?", req.Title,
		[]string{"approved", "pending"}).First(&existing).Error; err == nil {
		if existing.Status == models.StatusApproved {
			utils.RespondError(w, http.StatusConflict, "Standard already exists")
		} else {
			utils.RespondError(w, http.StatusConflict, "Standard is pending approval")
		}
		return
	}

	status := models.StatusPending
	var approvedBy *uint
	if user.IsAdmin {
		status = models.StatusApproved
		approvedBy = &user.ID
	}

	standard := models.JazzStandard{
		Title:          req.Title,
		Composer:       req.Composer,
		AdditionalNote: req.AdditionalNote,
		Style:          models.JazzStyle(req.Style),
		Key:            req.Key,
		IrealProLink:   req.IrealProLink,
		Status:         status,
		CreatedBy:      &user.ID,
		ApprovedBy:     approvedBy,
	}

	if err := database.DB.Create(&standard).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create standard")
		return
	}

	if status == models.StatusPending {
		// Fire ntfy notification to admins
		utils.NotifyAdminPendingStandard(standard.Title, user.Username)
		utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
			"standard": standard,
			"message":  "Standard submitted for approval",
		})
	} else {
		utils.RespondJSON(w, http.StatusCreated, standard)
	}
}

// ListStandards – paginated, searchable, filterable.
func ListStandards(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 100
	}

	search := r.URL.Query().Get("search")
	style := r.URL.Query().Get("style")
	key := r.URL.Query().Get("key")
	status := r.URL.Query().Get("status")
	offset := (page - 1) * limit

	query := database.DB.Model(&models.JazzStandard{})

	user := middleware.GetUserFromContext(r)
	isAdmin := user != nil && user.IsAdmin

	if !isAdmin {
		query = query.Where("status = ?", models.StatusApproved)
	} else if status != "" {
		if !models.IsValidStatus(status) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid status")
			return
		}
		query = query.Where("status = ?", status)
	}

	if search != "" {
		query = query.Where("title ILIKE ? OR composer ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if style != "" {
		if !models.IsValidStyle(style) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid style")
			return
		}
		query = query.Where("style = ?", style)
	}
	if key != "" {
		query = query.Where("key = ?", key)
	}

	var total int64
	query.Count(&total)

	var standards []models.JazzStandard
	if err := query.Preload("Creator").Preload("Approver").
		Offset(offset).Limit(limit).Order("title ASC").
		Find(&standards).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch standards")
		return
	}

	// Attach popularity counts in one query
	attachPopularity(standards)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"standards": standards,
		"page":      page,
		"limit":     limit,
		"total":     total,
	})
}

// GetStandard returns a single standard by ID.
func GetStandard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}

	user := middleware.GetUserFromContext(r)
	var standard models.JazzStandard
	q := database.DB.Preload("Creator").Preload("Approver")
	if user == nil || !user.IsAdmin {
		q = q.Where("status = ?", models.StatusApproved)
	}
	if err := q.First(&standard, id).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not found")
		return
	}

	// Popularity
	var cnt int64
	database.DB.Model(&models.UserStandard{}).Where("jazz_standard_id = ?", id).Count(&cnt)
	standard.PopularityCount = cnt

	utils.RespondJSON(w, http.StatusOK, standard)
}

// UpdateStandard – admin only.
func UpdateStandard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}
	var req UpdateStandardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var standard models.JazzStandard
	if err := database.DB.First(&standard, id).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not found")
		return
	}

	if req.Title != nil {
		standard.Title = *req.Title
	}
	if req.Composer != nil {
		standard.Composer = *req.Composer
	}
	if req.AdditionalNote != nil {
		standard.AdditionalNote = *req.AdditionalNote
	}
	if req.Style != nil {
		if !models.IsValidStyle(*req.Style) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid style")
			return
		}
		standard.Style = models.JazzStyle(*req.Style)
	}
	if req.Key != nil {
		standard.Key = *req.Key
	}
	if req.IrealProLink != nil {
		standard.IrealProLink = *req.IrealProLink
	}

	if err := database.DB.Save(&standard).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update standard")
		return
	}
	utils.RespondJSON(w, http.StatusOK, standard)
}

// DeleteStandard – admin only.
func DeleteStandard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}
	if err := database.DB.Delete(&models.JazzStandard{}, id).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete standard")
		return
	}
	utils.RespondSuccess(w, "Standard deleted successfully", nil)
}

// ApproveStandard – admin only.
func ApproveStandard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}
	user := middleware.GetUserFromContext(r)

	var standard models.JazzStandard
	if err := database.DB.First(&standard, id).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not found")
		return
	}
	if standard.Status == models.StatusApproved {
		utils.RespondError(w, http.StatusBadRequest, "Standard is already approved")
		return
	}

	standard.Status = models.StatusApproved
	standard.ApprovedBy = &user.ID
	if err := database.DB.Save(&standard).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to approve standard")
		return
	}
	database.DB.Preload("Creator").Preload("Approver").First(&standard, id)
	utils.RespondJSON(w, http.StatusOK, standard)
}

// RejectStandard – admin only.
func RejectStandard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid standard ID")
		return
	}
	user := middleware.GetUserFromContext(r)

	var standard models.JazzStandard
	if err := database.DB.First(&standard, id).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Standard not found")
		return
	}
	if standard.Status == models.StatusApproved {
		utils.RespondError(w, http.StatusBadRequest, "Cannot reject an approved standard")
		return
	}
	if standard.Status == models.StatusRejected {
		utils.RespondError(w, http.StatusBadRequest, "Standard is already rejected")
		return
	}

	standard.Status = models.StatusRejected
	standard.ApprovedBy = &user.ID
	if err := database.DB.Save(&standard).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to reject standard")
		return
	}
	database.DB.Preload("Creator").Preload("Approver").First(&standard, id)
	utils.RespondJSON(w, http.StatusOK, standard)
}

// ListPendingStandards – admin only.
func ListPendingStandards(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int64
	database.DB.Model(&models.JazzStandard{}).Where("status = ?", models.StatusPending).Count(&total)

	var standards []models.JazzStandard
	if err := database.DB.Where("status = ?", models.StatusPending).
		Preload("Creator").Offset(offset).Limit(limit).
		Order("created_at ASC").Find(&standards).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch pending standards")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"standards": standards,
		"page":      page,
		"limit":     limit,
		"total":     total,
	})
}

// RandomStandard returns a random approved standard.
// Optional query params: style, key
func RandomStandard(w http.ResponseWriter, r *http.Request) {
	query := database.DB.Model(&models.JazzStandard{}).Where("status = ?", models.StatusApproved)

	if style := r.URL.Query().Get("style"); style != "" {
		if !models.IsValidStyle(style) {
			utils.RespondError(w, http.StatusBadRequest, "Invalid style")
			return
		}
		query = query.Where("style = ?", style)
	}
	if key := r.URL.Query().Get("key"); key != "" {
		query = query.Where("key = ?", key)
	}

	var ids []uint
	database.DB.Model(&models.JazzStandard{}).Where("status = ?", models.StatusApproved).Pluck("id", &ids)

	// Apply filters to the id list
	var filteredIDs []uint
	if style := r.URL.Query().Get("style"); style != "" {
		database.DB.Model(&models.JazzStandard{}).
			Where("status = ? AND style = ?", models.StatusApproved, style).
			Pluck("id", &filteredIDs)
	} else if key := r.URL.Query().Get("key"); key != "" {
		database.DB.Model(&models.JazzStandard{}).
			Where("status = ? AND key = ?", models.StatusApproved, key).
			Pluck("id", &filteredIDs)
	} else {
		filteredIDs = ids
	}

	if len(filteredIDs) == 0 {
		utils.RespondError(w, http.StatusNotFound, "No standards found with the given filters")
		return
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomID := filteredIDs[rand.Intn(len(filteredIDs))]

	var standard models.JazzStandard
	database.DB.Preload("Creator").First(&standard, randomID)
	utils.RespondJSON(w, http.StatusOK, standard)
}

// BulkImportStandards – admin only.  POST /api/jazz_standards/bulk_import
// Body: JSON array of BulkImportItem
func BulkImportStandards(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)

	var items []BulkImportItem
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid JSON array")
		return
	}

	type result struct {
		Title  string `json:"title"`
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	}
	results := make([]result, 0, len(items))
	imported, skipped, failed := 0, 0, 0

	for _, item := range items {
		if item.Title == "" || item.Composer == "" || item.Style == "" {
			results = append(results, result{item.Title, "failed", "title, composer and style required"})
			failed++
			continue
		}
		if !models.IsValidStyle(item.Style) {
			results = append(results, result{item.Title, "failed", "invalid style"})
			failed++
			continue
		}

		// Skip duplicates
		var existing models.JazzStandard
		if err := database.DB.Where("title = ?", item.Title).First(&existing).Error; err == nil {
			results = append(results, result{item.Title, "skipped", "already exists"})
			skipped++
			continue
		}

		s := models.JazzStandard{
			Title:          item.Title,
			Composer:       item.Composer,
			Style:          models.JazzStyle(item.Style),
			Key:            item.Key,
			AdditionalNote: item.AdditionalNote,
			IrealProLink:   item.IrealProLink,
			Status:         models.StatusApproved,
			CreatedBy:      &user.ID,
			ApprovedBy:     &user.ID,
		}
		if err := database.DB.Create(&s).Error; err != nil {
			results = append(results, result{item.Title, "failed", err.Error()})
			failed++
			continue
		}
		results = append(results, result{item.Title, "imported", ""})
		imported++
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"imported": imported,
		"skipped":  skipped,
		"failed":   failed,
		"results":  results,
	})
}

// ExportMyStandards – exports the current user's list as JSON or CSV.
// Query param: format=json|csv  (default: json)
func ExportMyStandards(w http.ResponseWriter, r *http.Request) {
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

	format := r.URL.Query().Get("format")
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="my_standards.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"Title", "Composer", "Style", "Key", "Category", "Proficiency", "Notes"})
		for _, us := range userStandards {
			if us.JazzStandard == nil {
				continue
			}
			cat := ""
			if us.Category != nil {
				cat = us.Category.Name
			}
			_ = cw.Write([]string{
				us.JazzStandard.Title,
				us.JazzStandard.Composer,
				string(us.JazzStandard.Style),
				us.JazzStandard.Key,
				cat,
				string(us.Proficiency),
				us.Notes,
			})
		}
		cw.Flush()
		return
	}

	// Default: JSON
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"user":      user.Username,
		"exported":  len(userStandards),
		"standards": userStandards,
	})
}

// AdminStats – admin-only overview.
func AdminStats(w http.ResponseWriter, r *http.Request) {
	var totalStandards, pendingStandards, totalUsers, totalPracticeMin int64
	database.DB.Model(&models.JazzStandard{}).Where("status = ?", models.StatusApproved).Count(&totalStandards)
	database.DB.Model(&models.JazzStandard{}).Where("status = ?", models.StatusPending).Count(&pendingStandards)
	database.DB.Model(&models.User{}).Count(&totalUsers)
	database.DB.Model(&models.PracticeLog{}).Select("COALESCE(SUM(duration_min),0)").Scan(&totalPracticeMin)

	// Top 10 most popular standards
	type popularRow struct {
		JazzStandardID uint   `json:"id"`
		Title          string `json:"title"`
		Count          int64  `json:"count"`
	}
	var popular []popularRow
	database.DB.Raw(`
		SELECT us.jazz_standard_id, js.title, COUNT(*) as count
		FROM user_jazz_standards us
		JOIN jazz_standards js ON js.id = us.jazz_standard_id
		GROUP BY us.jazz_standard_id, js.title
		ORDER BY count DESC
		LIMIT 10
	`).Scan(&popular)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"total_approved_standards": totalStandards,
		"pending_standards":        pendingStandards,
		"total_users":              totalUsers,
		"total_practice_minutes":   totalPracticeMin,
		"top_standards":            popular,
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseID(r *http.Request, name string) (uint64, error) {
	return strconv.ParseUint(mux.Vars(r)[name], 10, 32)
}

// attachPopularity fills the PopularityCount field on a slice of standards.
func attachPopularity(standards []models.JazzStandard) {
	if len(standards) == 0 {
		return
	}
	ids := make([]uint, len(standards))
	for i, s := range standards {
		ids[i] = s.ID
	}
	type row struct {
		JazzStandardID uint
		Count          int64
	}
	var rows []row
	database.DB.Model(&models.UserStandard{}).
		Select("jazz_standard_id, COUNT(*) as count").
		Where("jazz_standard_id IN ?", ids).
		Group("jazz_standard_id").
		Scan(&rows)

	m := make(map[uint]int64, len(rows))
	for _, r := range rows {
		m[r.JazzStandardID] = r.Count
	}
	for i := range standards {
		standards[i].PopularityCount = m[standards[i].ID]
	}
}
