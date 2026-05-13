// cmd/seed/main.go — import iReal Pro jazz standards into the database.
//
// Usage:
//
//	go run cmd/seed/main.go [ireal-file-or-url]
//
// If no argument is supplied the script fetches the well-known "1350 Jazz
// Standards" playlist from irealb.com directly.
//
// The script parses the iReal Pro URI format, maps style names to the app's
// JazzStyle enum, and upserts every song using GORM's FirstOrCreate on the
// unique Title column.  Already-present rows are skipped; new rows are
// inserted with Status = "approved".
//
// Environment: reads .env (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME).
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Inline minimal model (avoids depending on the full app package) ───────────

type JazzStandard struct {
	ID           uint   `gorm:"primaryKey"`
	Title        string `gorm:"uniqueIndex;not null"`
	Composer     string `gorm:"not null"`
	Style        string `gorm:"type:varchar(50);not null"`
	Key          string `gorm:"type:varchar(10);default:''"`
	IrealProLink string `gorm:"type:text;default:''"`
	Status       string `gorm:"type:varchar(20);not null;default:'approved'"`
	CreatedBy    *uint
	ApprovedBy   *uint
}

func (JazzStandard) TableName() string { return "jazz_standards" }

// ── iReal Pro parser ──────────────────────────────────────────────────────────

// iReal playlists arrive in a URI like:
//
//	irealbook://Title=Composer=Style=Key=... ===Title=Composer=...
//
// Songs are delimited by "===".  We only need fields 0 (title), 1 (composer),
// 2 (style), and 3 (key).  Other fields contain the chord chart and are
// skipped.

// cleanTitle strips iReal Pro's special Unicode markers from the title.
func cleanTitle(s string) string {
	// iReal sometimes prefixes titles with a "1" (number) sort key separated by
	// a Unicode non-breaking space (U+00A0) or a regular space.
	s = strings.TrimSpace(s)
	// Remove leading digits + space that iReal uses as a sort index.
	re := regexp.MustCompile(`^\d+\s+`)
	s = re.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// styleMap converts iReal Pro style labels to the app's JazzStyle enum.
var styleMap = map[string]string{
	"swing":          "swing",
	"medium swing":   "swing",
	"med. swing":     "swing",
	"fast swing":     "swing",
	"up swing":       "swing",
	"ballad":         "swing",
	"bebop":          "bebop",
	"bossa nova":     "bossa_nova",
	"bossanova":      "bossa_nova",
	"latin":          "latin",
	"latin swing":    "latin_swing",
	"even 8ths":      "latin",
	"even eighths":   "latin",
	"samba":          "samba",
	"waltz":          "waltz",
	"jazz waltz":     "waltz",
	"3/4":            "waltz",
	"fusion":         "fusion",
	"modal":          "modal",
	"dixieland":      "dixieland",
	"rag":            "ragtime",
	"ragtime":        "ragtime",
	"big band":       "big_band",
	"free":           "free",
}

// mapStyle converts an iReal style string to the app enum, defaulting to "swing".
func mapStyle(irealStyle string) string {
	key := strings.ToLower(strings.TrimSpace(irealStyle))
	if v, ok := styleMap[key]; ok {
		return v
	}
	// Partial match for compound names.
	for k, v := range styleMap {
		if strings.Contains(key, k) {
			return v
		}
	}
	return "swing" // safe default
}

// parseSong parses a single iReal song segment (the part between === delimiters).
// Returns nil if the segment doesn't look like a valid song.
func parseSong(segment string) *JazzStandard {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return nil
	}
	// Fields are separated by "=".  The first four fields are the ones we care
	// about; the rest is the chord chart and may contain "=" characters itself.
	parts := strings.SplitN(segment, "=", 6)
	if len(parts) < 4 {
		return nil
	}
	title := cleanTitle(parts[0])
	composer := strings.TrimSpace(parts[1])
	style := mapStyle(parts[2])
	key := strings.TrimSpace(parts[3])

	if title == "" {
		return nil
	}
	if composer == "" {
		composer = "Unknown"
	}

	return &JazzStandard{
		Title:    title,
		Composer: composer,
		Style:    style,
		Key:      key,
		Status:   "approved",
	}
}

// parseIRealPlaylist extracts all songs from an iReal Pro playlist string or URI.
func parseIRealPlaylist(raw string) []*JazzStandard {
	// Strip the scheme and playlist name header if present.
	// Format: irealbook://PlaylistName===Song1===Song2===...
	if idx := strings.Index(raw, "://"); idx != -1 {
		raw = raw[idx+3:]
	}

	// URL-decode the content.
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		decoded = raw
	}

	// The playlist name appears before the first "==="; skip it.
	segments := strings.Split(decoded, "===")
	if len(segments) > 1 {
		segments = segments[1:] // drop playlist name
	}

	var standards []*JazzStandard
	for _, seg := range segments {
		if s := parseSong(seg); s != nil {
			standards = append(standards, s)
		}
	}
	return standards
}

// ── Fetch helpers ─────────────────────────────────────────────────────────────

const defaultPlaylistURL = "https://irealb.com/%E2%80%9Cjazz-1350%E2%80%9D-1350-jazz-tunes/"

// fetchIRealFromURL fetches a page and extracts ireal:// / irealbook:// URIs.
func fetchIRealFromURL(pageURL string) (string, error) {
	resp, err := http.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("HTTP GET %s: %w", pageURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	content := string(body)

	// Look for an ireal:// or irealbook:// link in the HTML.
	re := regexp.MustCompile(`(?i)(ireal(?:book)?://[^"'\s<>]+)`)
	match := re.FindString(content)
	if match != "" {
		// HTML-decode &amp; entities.
		match = strings.ReplaceAll(match, "&amp;", "&")
		return match, nil
	}

	// Fallback: the page may serve the raw URI directly (plain text).
	if strings.Contains(content, "===") {
		return content, nil
	}

	return "", fmt.Errorf("no iReal Pro playlist found at %s", pageURL)
}

// readIRealFromFile reads a local file (plain text or .html containing ireal:// links).
func readIRealFromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteByte('\n')
	}
	content := sb.String()

	// If it looks like an iReal URI already, return as-is.
	if strings.Contains(strings.ToLower(content[:min(100, len(content))]), "ireal") {
		return strings.TrimSpace(content), nil
	}

	// Extract from HTML.
	re := regexp.MustCompile(`(?i)(ireal(?:book)?://[^"'\s<>]+)`)
	if match := re.FindString(content); match != "" {
		return strings.ReplaceAll(match, "&amp;", "&"), nil
	}

	// Raw playlist.
	if strings.Contains(content, "===") {
		return content, nil
	}

	return "", fmt.Errorf("no iReal Pro playlist data found in %s", path)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── Database ──────────────────────────────────────────────────────────────────

func connectDB() (*gorm.DB, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getenv("DB_HOST", "localhost"),
		getenv("DB_PORT", "5432"),
		getenv("DB_USER", "jazz"),
		getenv("DB_PASSWORD", "jazz"),
		getenv("DB_NAME", "jazz"),
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(0)
	log.Println("=== iReal Pro → Jazz Standards DB Seeder ===")

	// ── 1. Get the raw playlist content ──────────────────────────────────────
	var raw string
	var err error

	if len(os.Args) >= 2 {
		src := os.Args[1]
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			log.Printf("Fetching playlist from URL: %s", src)
			raw, err = fetchIRealFromURL(src)
		} else if strings.HasPrefix(strings.ToLower(src), "ireal") {
			// A raw URI was pasted on the command line.
			raw = src
		} else {
			log.Printf("Reading playlist from file: %s", src)
			raw, err = readIRealFromFile(src)
		}
	} else {
		log.Printf("No argument supplied – fetching default playlist from %s", defaultPlaylistURL)
		raw, err = fetchIRealFromURL(defaultPlaylistURL)
	}

	if err != nil {
		log.Fatalf("Failed to load playlist: %v", err)
	}
	if raw == "" {
		log.Fatal("Playlist is empty")
	}

	// ── 2. Parse ──────────────────────────────────────────────────────────────
	log.Println("Parsing iReal Pro playlist…")
	standards := parseIRealPlaylist(raw)
	log.Printf("Parsed %d songs", len(standards))
	if len(standards) == 0 {
		log.Fatal("No songs found; check the playlist source")
	}

	// ── 3. Connect to DB ─────────────────────────────────────────────────────
	log.Println("Connecting to database…")
	db, err := connectDB()
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}

	// ── 4. Upsert ─────────────────────────────────────────────────────────────
	inserted, skipped := 0, 0
	for _, s := range standards {
		existing := JazzStandard{Title: s.Title}
		result := db.Where("title = ?", s.Title).FirstOrCreate(&existing, s)
		if result.Error != nil {
			log.Printf("WARN: failed to upsert %q: %v", s.Title, result.Error)
			continue
		}
		if result.RowsAffected > 0 {
			inserted++
		} else {
			skipped++
		}
	}

	log.Printf("Done. Inserted: %d  |  Already present (skipped): %d", inserted, skipped)
}
