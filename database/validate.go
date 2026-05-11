package database

import (
	"fmt"
	"strings"

	"github.com/denizsincar29/jazz_standards_db/models"
)

type IntegrityIssue struct {
	Check   string
	Count   int64
	Details string
}

type IntegrityReport struct {
	Issues []IntegrityIssue
}

func (r *IntegrityReport) OK() bool {
	return r != nil && len(r.Issues) == 0
}

func (r *IntegrityReport) String() string {
	if r == nil {
		return ""
	}
	if len(r.Issues) == 0 {
		return "no integrity issues found"
	}

	var builder strings.Builder
	for _, issue := range r.Issues {
		builder.WriteString(fmt.Sprintf("- %s: %d", issue.Check, issue.Count))
		if issue.Details != "" {
			builder.WriteString(" (")
			builder.WriteString(issue.Details)
			builder.WriteString(")")
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func ValidateIntegrity() (*IntegrityReport, error) {
	if DB == nil {
		return nil, fmt.Errorf("database is not connected")
	}

	report := &IntegrityReport{}

	checks := []struct {
		name    string
		query   string
		details string
	}{
		{name: "user_jazz_standards.user_id references users.id", query: `SELECT COUNT(*) FROM user_jazz_standards ujs LEFT JOIN users u ON u.id = ujs.user_id WHERE u.id IS NULL`, details: "orphan user-standard rows"},
		{name: "user_jazz_standards.jazz_standard_id references jazz_standards.id", query: `SELECT COUNT(*) FROM user_jazz_standards ujs LEFT JOIN jazz_standards js ON js.id = ujs.jazz_standard_id WHERE js.id IS NULL`, details: "orphan user-standard rows"},
		{name: "user_jazz_standards.category_id references user_categories.id", query: `SELECT COUNT(*) FROM user_jazz_standards ujs LEFT JOIN user_categories c ON c.id = ujs.category_id WHERE ujs.category_id IS NOT NULL AND c.id IS NULL`, details: "stale category references"},
		{name: "user_categories.user_id references users.id", query: `SELECT COUNT(*) FROM user_categories c LEFT JOIN users u ON u.id = c.user_id WHERE u.id IS NULL`, details: "orphan category rows"},
		{name: "pass_keys.user_id references users.id", query: `SELECT COUNT(*) FROM pass_keys p LEFT JOIN users u ON u.id = p.user_id WHERE u.id IS NULL`, details: "orphan pass keys"},
		{name: "practice_logs.user_id references users.id", query: `SELECT COUNT(*) FROM practice_logs p LEFT JOIN users u ON u.id = p.user_id WHERE u.id IS NULL`, details: "orphan practice logs"},
		{name: "practice_logs.jazz_standard_id references jazz_standards.id", query: `SELECT COUNT(*) FROM practice_logs p LEFT JOIN jazz_standards js ON js.id = p.jazz_standard_id WHERE js.id IS NULL`, details: "orphan practice logs"},
		{name: "personal_pieces.user_id references users.id", query: `SELECT COUNT(*) FROM personal_pieces p LEFT JOIN users u ON u.id = p.user_id WHERE u.id IS NULL`, details: "orphan personal pieces"},
		{name: "composed_tunes.user_id references users.id", query: `SELECT COUNT(*) FROM composed_tunes c LEFT JOIN users u ON u.id = c.user_id WHERE u.id IS NULL`, details: "orphan compositions"},
		{name: "jazz_standards.created_by references users.id", query: `SELECT COUNT(*) FROM jazz_standards js LEFT JOIN users u ON u.id = js.created_by WHERE js.created_by IS NOT NULL AND u.id IS NULL`, details: "orphan creator references"},
		{name: "jazz_standards.approved_by references users.id", query: `SELECT COUNT(*) FROM jazz_standards js LEFT JOIN users u ON u.id = js.approved_by WHERE js.approved_by IS NOT NULL AND u.id IS NULL`, details: "orphan approver references"},
		{name: "user_categories unique name per user", query: `SELECT COUNT(*) FROM (SELECT user_id, name, COUNT(*) AS c FROM user_categories GROUP BY user_id, name HAVING COUNT(*) > 1) duplicates`, details: "duplicate category names"},
	}

	for _, check := range checks {
		var count int64
		if err := DB.Raw(check.query).Scan(&count).Error; err != nil {
			return nil, fmt.Errorf("%s: %w", check.name, err)
		}
		if count > 0 {
			report.Issues = append(report.Issues, IntegrityIssue{
				Check:   check.name,
				Count:   count,
				Details: check.details,
			})
		}
	}

	for _, table := range []struct {
		name  string
		model any
	}{
		{name: models.User{}.TableName(), model: &models.User{}},
		{name: models.PassKey{}.TableName(), model: &models.PassKey{}},
		{name: models.JazzStandard{}.TableName(), model: &models.JazzStandard{}},
		{name: models.Category{}.TableName(), model: &models.Category{}},
		{name: models.UserStandard{}.TableName(), model: &models.UserStandard{}},
		{name: models.PracticeLog{}.TableName(), model: &models.PracticeLog{}},
		{name: models.PersonalPiece{}.TableName(), model: &models.PersonalPiece{}},
		{name: models.ComposedTune{}.TableName(), model: &models.ComposedTune{}},
	} {
		if !DB.Migrator().HasTable(table.model) {
			report.Issues = append(report.Issues, IntegrityIssue{
				Check:   fmt.Sprintf("table exists: %s", table.name),
				Count:   1,
				Details: "missing table",
			})
		}
	}

	return report, nil
}