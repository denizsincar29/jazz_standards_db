package utils

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/denizsincar29/jazz_standards_db/config"
)

// SendNtfy posts a push notification to an ntfy.sh topic.
// It is a best-effort fire-and-forget call: errors are logged but not returned.
//
// priority: one of min/low/default/high/urgent (empty = "default")
// tags:     optional emoji/tag strings (comma-separated), e.g. "bell,jazz"
func SendNtfy(title, message, priority, tags string) {
	cfg := config.AppConfig
	if !cfg.NtfyEnabled() {
		return
	}

	url := strings.TrimRight(cfg.NtfyURL, "/") + "/" + cfg.NtfyTopic

	go func() {
		req, err := http.NewRequest("POST", url, strings.NewReader(message))
		if err != nil {
			log.Printf("ntfy: failed to create request: %v", err)
			return
		}

		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		if title != "" {
			req.Header.Set("Title", title)
		}
		if priority != "" {
			req.Header.Set("Priority", priority)
		}
		if tags != "" {
			req.Header.Set("Tags", tags)
		}
		if cfg.NtfyToken != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.NtfyToken)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("ntfy: request failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			log.Printf("ntfy: unexpected status %d for topic %s", resp.StatusCode, cfg.NtfyTopic)
		}
	}()
}

// NotifyAdminPendingStandard sends an ntfy notification when a user
// submits a new standard that needs admin approval.
func NotifyAdminPendingStandard(standardTitle, submittedBy string) {
	SendNtfy(
		"New standard awaiting approval",
		fmt.Sprintf("🎷 %q submitted by %q needs your review.", standardTitle, submittedBy),
		"high",
		"bell,musical_note",
	)
}
