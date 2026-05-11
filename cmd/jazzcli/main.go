package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

type Session struct {
	APIURL   string    `json:"api_url"`
	Token    string    `json:"token"`
	Username string    `json:"username,omitempty"`
	SavedAt  time.Time `json:"saved_at,omitempty"`
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type User struct {
	ID            uint   `json:"id"`
	Username      string `json:"username"`
	Name          string `json:"name"`
	IsAdmin       bool   `json:"is_admin"`
	PublicProfile bool   `json:"public_profile"`
	Token         string `json:"token,omitempty"`
}

type Standard struct {
	ID             uint   `json:"id"`
	Title          string `json:"title"`
	Composer       string `json:"composer"`
	Style          string `json:"style"`
	Key            string `json:"key,omitempty"`
	AdditionalNote string `json:"additional_note,omitempty"`
	Status         string `json:"status,omitempty"`
	PopularityCount int64  `json:"popularity_count,omitempty"`
}

type Category struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type UserStandard struct {
	UserID         uint       `json:"user_id"`
	JazzStandardID uint       `json:"jazz_standard_id"`
	CategoryID     *uint      `json:"category_id,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	Proficiency    string     `json:"proficiency"`
	Category       *Category  `json:"category,omitempty"`
	JazzStandard   *Standard  `json:"jazz_standard,omitempty"`
}

type PersonalPiece struct {
	ID           uint     `json:"id"`
	UserID       uint     `json:"user_id"`
	Title        string   `json:"title"`
	Composer     string   `json:"composer,omitempty"`
	Style        string   `json:"style,omitempty"`
	Key          string   `json:"key,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	IrealProLink string   `json:"ireal_pro_link,omitempty"`
	IsPublic     bool     `json:"is_public"`
	User         *User    `json:"user,omitempty"`
}

type ComposedTune struct {
	ID           uint     `json:"id"`
	UserID       uint     `json:"user_id"`
	ShareID      string   `json:"share_id"`
	Title        string   `json:"title"`
	Composer     string   `json:"composer"`
	Style        string   `json:"style,omitempty"`
	Key          string   `json:"key,omitempty"`
	Description  string   `json:"description,omitempty"`
	IrealProLink string   `json:"ireal_pro_link,omitempty"`
	IsPublic     bool     `json:"is_public"`
	User         *User    `json:"user,omitempty"`
}

type SharedTuneResponse struct {
	Tune    ComposedTune `json:"tune"`
	Message string       `json:"message"`
}

type StandardsPage struct {
	Standards []Standard `json:"standards"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
	Total     int64      `json:"total"`
}

type MyStandardsResponse struct {
	Standards []UserStandard              `json:"standards"`
	Grouped   map[string][]UserStandard   `json:"grouped"`
	Total     int64                      `json:"total"`
}

type PendingResponse struct {
	Standards []Standard `json:"standards"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
	Total     int64      `json:"total"`
}

type PracticeLog struct {
	ID          uint      `json:"id"`
	DurationMin int       `json:"duration_min"`
	Notes       string    `json:"notes,omitempty"`
	PracticedAt time.Time `json:"practiced_at"`
	JazzStandard *Standard `json:"jazz_standard,omitempty"`
}

var inputReader = bufio.NewReader(os.Stdin)

func main() {
	apiURLFlag := flag.String("api-url", "", "API base URL")
	flag.Parse()

	session, err := loadSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not load saved session:", err)
		session = &Session{}
	}
	if *apiURLFlag != "" {
		session.APIURL = normalizeBaseURL(*apiURLFlag)
	}
	if session.APIURL == "" {
		session.APIURL = normalizeBaseURL(promptString("API URL [http://localhost:8000]: ", "http://localhost:8000"))
	}

	client := NewClient(session.APIURL, session.Token)
	user, err := client.Me()
	if err != nil {
		if session.Token != "" {
			fmt.Println("Saved login is no longer valid, please sign in again.")
		}
		session.Token = ""
		session.Username = ""
		_ = saveSession(session)
		user, session = authenticate(client, session)
	}

	mainLoop(client, session, user)
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: normalizeBaseURL(baseURL),
		token:   token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) setToken(token string) {
	c.token = token
}

func (c *Client) request(method, path string, body any, out any) error {
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) requestBytes(method, path string, body any) ([]byte, string, error) {
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		payload = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, payload)
	if err != nil {
		return nil, "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header.Get("Content-Type"), decodeAPIErrorFromBytes(resp.StatusCode, data)
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func decodeAPIError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	return decodeAPIErrorFromBytes(resp.StatusCode, data)
}

func decodeAPIErrorFromBytes(statusCode int, data []byte) error {
	var payload struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(data, &payload); err == nil {
		if payload.Message != "" {
			return errors.New(payload.Message)
		}
		if payload.Error != "" {
			return errors.New(payload.Error)
		}
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return fmt.Errorf("request failed with status %d", statusCode)
	}
	return errors.New(trimmed)
}

func (c *Client) Register(username, name, password string) (*User, error) {
	var user User
	if err := c.request(http.MethodPost, "/register", map[string]string{"username": username, "name": name, "password": password}, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) Login(username, password string) (*User, error) {
	var user User
	if err := c.request(http.MethodPost, "/login", map[string]string{"username": username, "password": password}, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) Logout() error {
	return c.request(http.MethodPost, "/logout", nil, nil)
}

func (c *Client) Me() (*User, error) {
	var user User
	if err := c.request(http.MethodGet, "/users/me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) UpdateMe(name *string, publicProfile *bool) (*User, error) {
	body := map[string]any{}
	if name != nil {
		body["name"] = *name
	}
	if publicProfile != nil {
		body["public_profile"] = *publicProfile
	}
	var user User
	if err := c.request(http.MethodPatch, "/users/me", body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) ListStandards(page, limit int, search, style string) (*StandardsPage, error) {
	query := []string{fmt.Sprintf("page=%d", page), fmt.Sprintf("limit=%d", limit)}
	if search != "" {
		query = append(query, "search="+urlQuery(search))
	}
	if style != "" {
		query = append(query, "style="+urlQuery(style))
	}
	var pageResp StandardsPage
	if err := c.request(http.MethodGet, "/jazz_standards?"+strings.Join(query, "&"), nil, &pageResp); err != nil {
		return nil, err
	}
	return &pageResp, nil
}

func (c *Client) CreateStandard(title, composer, style, note, key, ireal string) error {
	body := map[string]any{
		"title":            title,
		"composer":         composer,
		"style":            style,
		"additional_note":  note,
		"key":              key,
		"ireal_pro_link":   ireal,
	}
	data, _, err := c.requestBytes(http.MethodPost, "/jazz_standards", body)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err == nil {
		if message, ok := payload["message"].(string); ok && message != "" {
			fmt.Println(message)
			return nil
		}
		if standard, ok := payload["standard"].(map[string]any); ok {
			fmt.Printf("Created standard: %v\n", standard["title"])
			return nil
		}
	}
	fmt.Println("Standard created")
	return nil
}

func (c *Client) GetMyStandards() (*MyStandardsResponse, error) {
	var resp MyStandardsResponse
	if err := c.request(http.MethodGet, "/users/me/standards", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) AddMyStandard(id uint, categoryID *uint, notes, proficiency string) error {
	body := map[string]any{}
	if categoryID != nil {
		body["category_id"] = *categoryID
	}
	if notes != "" {
		body["notes"] = notes
	}
	if proficiency != "" {
		body["proficiency"] = proficiency
	}
	return c.request(http.MethodPost, fmt.Sprintf("/users/me/standards/%d", id), body, nil)
}

func (c *Client) UpdateMyStandard(id uint, categoryID *uint, notes, proficiency string) error {
	body := map[string]any{}
	if categoryID != nil {
		body["category_id"] = *categoryID
	}
	if notes != "" {
		body["notes"] = notes
	}
	if proficiency != "" {
		body["proficiency"] = proficiency
	}
	return c.request(http.MethodPut, fmt.Sprintf("/users/me/standards/%d", id), body, nil)
}

func (c *Client) RemoveMyStandard(id uint) error {
	return c.request(http.MethodDelete, fmt.Sprintf("/users/me/standards/%d", id), nil, nil)
}

func (c *Client) ListCategories() ([]Category, error) {
	var categories []Category
	if err := c.request(http.MethodGet, "/users/me/categories", nil, &categories); err != nil {
		return nil, err
	}
	return categories, nil
}

func (c *Client) CreateCategory(name, color string) error {
	return c.request(http.MethodPost, "/users/me/categories", map[string]string{"name": name, "color": color}, nil)
}

func (c *Client) UpdateCategory(id uint, name, color *string) error {
	body := map[string]any{}
	if name != nil {
		body["name"] = *name
	}
	if color != nil {
		body["color"] = *color
	}
	return c.request(http.MethodPut, fmt.Sprintf("/users/me/categories/%d", id), body, nil)
}

func (c *Client) DeleteCategory(id uint) error {
	return c.request(http.MethodDelete, fmt.Sprintf("/users/me/categories/%d", id), nil, nil)
}

func (c *Client) ListPracticeLogs(standardID *uint) ([]PracticeLog, error) {
	path := "/users/me/practice"
	if standardID != nil {
		path += "?standard_id=" + strconv.Itoa(int(*standardID))
	}
	var payload struct {
		Logs []PracticeLog `json:"logs"`
	}
	if err := c.request(http.MethodGet, path, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Logs, nil
}

func (c *Client) LogPractice(standardID uint, duration int, notes, practicedAt string) error {
	body := map[string]any{"duration_min": duration}
	if notes != "" {
		body["notes"] = notes
	}
	if practicedAt != "" {
		body["practiced_at"] = practicedAt
	}
	return c.request(http.MethodPost, fmt.Sprintf("/users/me/standards/%d/practice", standardID), body, nil)
}

func (c *Client) PendingStandards(page, limit int) (*PendingResponse, error) {
	path := fmt.Sprintf("/jazz_standards/pending?page=%d&limit=%d", page, limit)
	var resp PendingResponse
	if err := c.request(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ApproveStandard(id uint) error {
	return c.request(http.MethodPost, fmt.Sprintf("/jazz_standards/%d/approve", id), nil, nil)
}

func (c *Client) RejectStandard(id uint) error {
	return c.request(http.MethodPost, fmt.Sprintf("/jazz_standards/%d/reject", id), nil, nil)
}

func (c *Client) ExportMyStandards(format string) ([]byte, string, error) {
	path := "/users/me/standards/export"
	if format != "" {
		path += "?format=" + urlQuery(format)
	}
	return c.requestBytes(http.MethodGet, path, nil)
}

func (c *Client) ListPersonalPieces() ([]PersonalPiece, error) {
	var pieces []PersonalPiece
	if err := c.request(http.MethodGet, "/users/me/pieces", nil, &pieces); err != nil {
		return nil, err
	}
	return pieces, nil
}

func (c *Client) CreatePersonalPiece(title, composer, style, key, notes, ireal string, isPublic bool) error {
	body := map[string]any{
		"title":          title,
		"composer":       composer,
		"style":          style,
		"key":            key,
		"notes":          notes,
		"ireal_pro_link": ireal,
		"is_public":      isPublic,
	}
	return c.request(http.MethodPost, "/users/me/pieces", body, nil)
}

func (c *Client) UpdatePersonalPiece(id uint, title, composer, style, key, notes, ireal *string, isPublic *bool) error {
	body := map[string]any{}
	if title != nil {
		body["title"] = *title
	}
	if composer != nil {
		body["composer"] = *composer
	}
	if style != nil {
		body["style"] = *style
	}
	if key != nil {
		body["key"] = *key
	}
	if notes != nil {
		body["notes"] = *notes
	}
	if ireal != nil {
		body["ireal_pro_link"] = *ireal
	}
	if isPublic != nil {
		body["is_public"] = *isPublic
	}
	return c.request(http.MethodPut, fmt.Sprintf("/users/me/pieces/%d", id), body, nil)
}

func (c *Client) DeletePersonalPiece(id uint) error {
	return c.request(http.MethodDelete, fmt.Sprintf("/users/me/pieces/%d", id), nil, nil)
}

func (c *Client) ListComposedTunes() ([]ComposedTune, error) {
	var tunes []ComposedTune
	if err := c.request(http.MethodGet, "/users/me/compositions", nil, &tunes); err != nil {
		return nil, err
	}
	return tunes, nil
}

func (c *Client) CreateComposedTune(title, composer, style, key, description, ireal string, isPublic bool) error {
	body := map[string]any{
		"title":          title,
		"composer":       composer,
		"style":          style,
		"key":            key,
		"description":    description,
		"ireal_pro_link": ireal,
		"is_public":      isPublic,
	}
	return c.request(http.MethodPost, "/users/me/compositions", body, nil)
}

func (c *Client) UpdateComposedTune(id uint, title, composer, style, key, description, ireal *string, isPublic *bool) error {
	body := map[string]any{}
	if title != nil {
		body["title"] = *title
	}
	if composer != nil {
		body["composer"] = *composer
	}
	if style != nil {
		body["style"] = *style
	}
	if key != nil {
		body["key"] = *key
	}
	if description != nil {
		body["description"] = *description
	}
	if ireal != nil {
		body["ireal_pro_link"] = *ireal
	}
	if isPublic != nil {
		body["is_public"] = *isPublic
	}
	return c.request(http.MethodPut, fmt.Sprintf("/users/me/compositions/%d", id), body, nil)
}

func (c *Client) DeleteComposedTune(id uint) error {
	return c.request(http.MethodDelete, fmt.Sprintf("/users/me/compositions/%d", id), nil, nil)
}

func (c *Client) GetSharedTune(shareID string) (*SharedTuneResponse, error) {
	var resp SharedTuneResponse
	if err := c.request(http.MethodGet, "/shared_tune?id="+urlQuery(shareID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) AcceptSharedTune(shareID string) error {
	return c.request(http.MethodPost, "/shared_tune/accept?id="+urlQuery(shareID), nil, nil)
}

func authenticate(client *Client, session *Session) (*User, *Session) {
	for {
		fmt.Println()
		fmt.Println("Sign in to the Jazz Standards CLI")
		fmt.Println("1) Login")
		fmt.Println("2) Register")
		fmt.Println("3) Change API URL")
		fmt.Println("0) Exit")
		choice := promptString("Select an option: ", "")

		switch choice {
		case "1":
			username := promptString("Username: ", "")
			password := promptPassword("Password: ")
			user, err := client.Login(username, password)
			if err != nil {
				fmt.Println("Login failed:", err)
				continue
			}
			session.Token = user.Token
			session.Username = user.Username
			session.SavedAt = time.Now().UTC()
			client.setToken(user.Token)
			_ = saveSession(session)
			return user, session
		case "2":
			username := promptString("Username: ", "")
			name := promptString("Full name: ", "")
			password := promptPassword("Password: ")
			user, err := client.Register(username, name, password)
			if err != nil {
				fmt.Println("Registration failed:", err)
				continue
			}
			session.Token = user.Token
			session.Username = user.Username
			session.SavedAt = time.Now().UTC()
			client.setToken(user.Token)
			_ = saveSession(session)
			return user, session
		case "3":
			session.APIURL = normalizeBaseURL(promptString("API URL: ", session.APIURL))
			client.baseURL = session.APIURL
			_ = saveSession(session)
		case "0":
			os.Exit(0)
		default:
			fmt.Println("Unknown option")
		}
	}
}

func mainLoop(client *Client, session *Session, user *User) {
	for {
		fmt.Println()
		fmt.Printf("Logged in as %s (%s)\n", user.Name, user.Username)
		fmt.Println("1) Profile")
		fmt.Println("2) List all standards")
		fmt.Println("3) Create standard")
		fmt.Println("4) My standards")
		fmt.Println("5) Categories")
		fmt.Println("6) Personal pieces")
		fmt.Println("7) Compositions")
		fmt.Println("8) Practice logs")
		if user.IsAdmin {
			fmt.Println("9) Pending approvals")
		}
		if user.IsAdmin {
			fmt.Println("10) Logout")
		} else {
			fmt.Println("9) Logout")
		}
		fmt.Println("0) Exit")

		choice := promptString("Select an option: ", "")
		switch choice {
		case "1":
			profileMenu(client, user)
		case "2":
			standardsMenu(client)
		case "3":
			createStandardMenu(client)
		case "4":
			myStandardsMenu(client)
		case "5":
			categoriesMenu(client)
		case "6":
			personalPiecesMenu(client)
		case "7":
			compositionsMenu(client)
		case "8":
			practiceMenu(client)
		case "9":
			if user.IsAdmin {
				pendingMenu(client)
			} else {
				if err := client.Logout(); err != nil {
					fmt.Println("Logout request failed:", err)
				}
				client.setToken("")
				session.Token = ""
				session.Username = ""
				_ = saveSession(session)
				fmt.Println("Logged out")
				user, session = authenticate(client, session)
			}
		case "10":
			if user.IsAdmin {
				if err := client.Logout(); err != nil {
					fmt.Println("Logout request failed:", err)
				}
				client.setToken("")
				session.Token = ""
				session.Username = ""
				_ = saveSession(session)
				fmt.Println("Logged out")
				user, session = authenticate(client, session)
			}
		case "0":
			return
		default:
			fmt.Println("Unknown option")
		}
	}
}

func profileMenu(client *Client, user *User) {
	fmt.Printf("\nID: %d\nUsername: %s\nName: %s\nAdmin: %t\nPublic profile: %t\n", user.ID, user.Username, user.Name, user.IsAdmin, user.PublicProfile)
	choice := promptString("Update profile? (y/n): ", "n")
	if strings.ToLower(choice) != "y" {
		return
	}
	name := promptString("New name (blank keeps current): ", "")
	publicValue := promptString("Public profile (true/false, blank keeps current): ", "")
	var namePtr *string
	if name != "" {
		namePtr = &name
	}
	var publicPtr *bool
	if publicValue != "" {
		value, err := strconv.ParseBool(publicValue)
		if err != nil {
			fmt.Println("Invalid public profile value")
			return
		}
		publicPtr = &value
	}
	updated, err := client.UpdateMe(namePtr, publicPtr)
	if err != nil {
		fmt.Println("Update failed:", err)
		return
	}
	*user = *updated
	fmt.Println("Profile updated")
}

func standardsMenu(client *Client) {
	page := promptInt("Page [1]: ", 1)
	limit := promptInt("Limit [20]: ", 20)
	search := promptString("Search (blank for none): ", "")
	style := promptString("Style (blank for none): ", "")
	resp, err := client.ListStandards(page, limit, search, style)
	if err != nil {
		fmt.Println("Failed to list standards:", err)
		return
	}
	fmt.Printf("\nTotal: %d\n", resp.Total)
	for _, std := range resp.Standards {
		fmt.Printf("%d | %s | %s | %s\n", std.ID, std.Title, std.Composer, std.Style)
	}
}

func createStandardMenu(client *Client) {
	title := promptString("Title: ", "")
	composer := promptString("Composer: ", "")
	style := promptString("Style: ", "")
	note := promptString("Additional note (blank for none): ", "")
	key := promptString("Key (blank for none): ", "")
	ireal := promptString("iReal Pro link (blank for none): ", "")
	if err := client.CreateStandard(title, composer, style, note, key, ireal); err != nil {
		fmt.Println("Create failed:", err)
		return
	}
	fmt.Println("Standard submitted")
}

func myStandardsMenu(client *Client) {
	resp, err := client.GetMyStandards()
	if err != nil {
		fmt.Println("Failed to load my standards:", err)
		return
	}
	fmt.Printf("\nTotal: %d\n", resp.Total)
	if len(resp.Grouped) == 0 {
		fmt.Println("No standards yet")
	} else {
		for category, items := range resp.Grouped {
			fmt.Printf("\n[%s]\n", category)
			for _, item := range items {
				name := ""
				if item.JazzStandard != nil {
					name = item.JazzStandard.Title
				}
				fmt.Printf("%d | %s | %s | %s\n", item.JazzStandardID, name, item.Proficiency, item.Notes)
			}
		}
	}
	fmt.Println()
	fmt.Println("1) Add standard")
	fmt.Println("2) Update entry")
	fmt.Println("3) Remove entry")
	fmt.Println("4) Export JSON")
	fmt.Println("5) Export CSV")
	fmt.Println("0) Back")
	choice := promptString("Select an option: ", "0")
	switch choice {
	case "1":
		standardID := promptUint("Standard ID: ")
		var categoryID *uint
		categoryRaw := promptString("Category ID (blank for none): ", "")
		if categoryRaw != "" {
			value, err := strconv.ParseUint(categoryRaw, 10, 32)
			if err != nil {
				fmt.Println("Invalid category ID")
				return
			}
			converted := uint(value)
			categoryID = &converted
		}
		notes := promptString("Notes (blank for none): ", "")
		proficiency := promptString("Proficiency (beginner/learning/know_it/master, blank keeps default): ", "")
		if err := client.AddMyStandard(standardID, categoryID, notes, proficiency); err != nil {
			fmt.Println("Add failed:", err)
		}
	case "2":
		standardID := promptUint("Standard ID: ")
		var categoryID *uint
		categoryRaw := promptString("Category ID (blank for none): ", "")
		if categoryRaw != "" {
			value, err := strconv.ParseUint(categoryRaw, 10, 32)
			if err != nil {
				fmt.Println("Invalid category ID")
				return
			}
			converted := uint(value)
			categoryID = &converted
		}
		notes := promptString("Notes (blank for none): ", "")
		proficiency := promptString("Proficiency (blank keeps current): ", "")
		if err := client.UpdateMyStandard(standardID, categoryID, notes, proficiency); err != nil {
			fmt.Println("Update failed:", err)
		}
	case "3":
		standardID := promptUint("Standard ID: ")
		if err := client.RemoveMyStandard(standardID); err != nil {
			fmt.Println("Remove failed:", err)
		}
	case "4":
		data, contentType, err := client.ExportMyStandards("json")
		if err != nil {
			fmt.Println("Export failed:", err)
			return
		}
		fmt.Println("Content-Type:", contentType)
		fmt.Println(string(data))
	case "5":
		data, _, err := client.ExportMyStandards("csv")
		if err != nil {
			fmt.Println("Export failed:", err)
			return
		}
		r := csv.NewReader(bytes.NewReader(data))
		rows, err := r.ReadAll()
		if err != nil {
			fmt.Println(string(data))
			return
		}
		for _, row := range rows {
			fmt.Println(strings.Join(row, " | "))
		}
	}
}

func categoriesMenu(client *Client) {
	categories, err := client.ListCategories()
	if err != nil {
		fmt.Println("Failed to load categories:", err)
		return
	}
	fmt.Printf("\nTotal: %d\n", len(categories))
	for _, cat := range categories {
		fmt.Printf("%d | %s | %s\n", cat.ID, cat.Name, cat.Color)
	}
	fmt.Println("1) Create category")
	fmt.Println("2) Update category")
	fmt.Println("3) Delete category")
	fmt.Println("0) Back")
	choice := promptString("Select an option: ", "0")
	switch choice {
	case "1":
		name := promptString("Name: ", "")
		color := promptString("Color [#000000]: ", "#000000")
		if err := client.CreateCategory(name, color); err != nil {
			fmt.Println("Create failed:", err)
		}
	case "2":
		id := promptUint("Category ID: ")
		name := promptString("New name (blank keeps current): ", "")
		color := promptString("New color (blank keeps current): ", "")
		var namePtr, colorPtr *string
		if name != "" {
			namePtr = &name
		}
		if color != "" {
			colorPtr = &color
		}
		if err := client.UpdateCategory(id, namePtr, colorPtr); err != nil {
			fmt.Println("Update failed:", err)
		}
	case "3":
		id := promptUint("Category ID: ")
		if err := client.DeleteCategory(id); err != nil {
			fmt.Println("Delete failed:", err)
		}
	}
}

func personalPiecesMenu(client *Client) {
	for {
		pieces, err := client.ListPersonalPieces()
		if err != nil {
			fmt.Println("Failed to load personal pieces:", err)
			return
		}
		fmt.Printf("\nPersonal pieces: %d\n", len(pieces))
		for _, piece := range pieces {
			fmt.Printf("%d | %s | %s | public=%t\n", piece.ID, piece.Title, piece.Composer, piece.IsPublic)
		}
		fmt.Println("1) Create piece")
		fmt.Println("2) Update piece")
		fmt.Println("3) Delete piece")
		fmt.Println("0) Back")
		choice := promptString("Select an option: ", "0")
		switch choice {
		case "1":
			title := promptString("Title: ", "")
			composer := promptString("Composer (blank for none): ", "")
			style := promptString("Style (blank for none): ", "")
			key := promptString("Key (blank for none): ", "")
			notes := promptString("Notes (blank for none): ", "")
			ireal := promptString("iReal Pro link (blank for none): ", "")
			isPublicValue := promptString("Public? (true/false) [false]: ", "false")
			isPublic, err := strconv.ParseBool(isPublicValue)
			if err != nil {
				fmt.Println("Invalid public value")
				continue
			}
			if err := client.CreatePersonalPiece(title, composer, style, key, notes, ireal, isPublic); err != nil {
				fmt.Println("Create failed:", err)
			}
		case "2":
			id := promptUint("Piece ID: ")
			title := promptOptionalString("Title (blank keeps current): ")
			composer := promptOptionalString("Composer (blank keeps current): ")
			style := promptOptionalString("Style (blank keeps current): ")
			key := promptOptionalString("Key (blank keeps current): ")
			notes := promptOptionalString("Notes (blank keeps current): ")
			ireal := promptOptionalString("iReal Pro link (blank keeps current): ")
			publicValue := promptOptionalString("Public? (true/false, blank keeps current): ")
			var titlePtr, composerPtr, stylePtr, keyPtr, notesPtr, irealPtr *string
			var publicPtr *bool
			if title != nil {
				titlePtr = title
			}
			if composer != nil {
				composerPtr = composer
			}
			if style != nil {
				stylePtr = style
			}
			if key != nil {
				keyPtr = key
			}
			if notes != nil {
				notesPtr = notes
			}
			if ireal != nil {
				irealPtr = ireal
			}
			if publicValue != nil {
				parsed, err := strconv.ParseBool(*publicValue)
				if err != nil {
					fmt.Println("Invalid public value")
					continue
				}
				publicPtr = &parsed
			}
			if err := client.UpdatePersonalPiece(id, titlePtr, composerPtr, stylePtr, keyPtr, notesPtr, irealPtr, publicPtr); err != nil {
				fmt.Println("Update failed:", err)
			}
		case "3":
			id := promptUint("Piece ID: ")
			if err := client.DeletePersonalPiece(id); err != nil {
				fmt.Println("Delete failed:", err)
			}
		case "0":
			return
		default:
			fmt.Println("Unknown option")
		}
	}
}

func compositionsMenu(client *Client) {
	for {
		tunes, err := client.ListComposedTunes()
		if err != nil {
			fmt.Println("Failed to load compositions:", err)
			return
		}
		fmt.Printf("\nCompositions: %d\n", len(tunes))
		for _, tune := range tunes {
			fmt.Printf("%d | %s | %s | public=%t | share=%s\n", tune.ID, tune.Title, tune.Composer, tune.IsPublic, tune.ShareID)
		}
		fmt.Println("1) Create composition")
		fmt.Println("2) Update composition")
		fmt.Println("3) Delete composition")
		fmt.Println("4) View shared tune")
		fmt.Println("5) Accept shared tune")
		fmt.Println("0) Back")
		choice := promptString("Select an option: ", "0")
		switch choice {
		case "1":
			title := promptString("Title: ", "")
			composer := promptString("Composer (blank uses your name): ", "")
			style := promptString("Style (blank for none): ", "")
			key := promptString("Key (blank for none): ", "")
			description := promptString("Description (blank for none): ", "")
			ireal := promptString("iReal Pro link (blank for none): ", "")
			isPublicValue := promptString("Public? (true/false) [true]: ", "true")
			isPublic, err := strconv.ParseBool(isPublicValue)
			if err != nil {
				fmt.Println("Invalid public value")
				continue
			}
			if err := client.CreateComposedTune(title, composer, style, key, description, ireal, isPublic); err != nil {
				fmt.Println("Create failed:", err)
			}
		case "2":
			id := promptUint("Composition ID: ")
			title := promptOptionalString("Title (blank keeps current): ")
			composer := promptOptionalString("Composer (blank keeps current): ")
			style := promptOptionalString("Style (blank keeps current): ")
			key := promptOptionalString("Key (blank keeps current): ")
			description := promptOptionalString("Description (blank keeps current): ")
			ireal := promptOptionalString("iReal Pro link (blank keeps current): ")
			publicValue := promptOptionalString("Public? (true/false, blank keeps current): ")
			var titlePtr, composerPtr, stylePtr, keyPtr, descPtr, irealPtr *string
			var publicPtr *bool
			if title != nil {
				titlePtr = title
			}
			if composer != nil {
				composerPtr = composer
			}
			if style != nil {
				stylePtr = style
			}
			if key != nil {
				keyPtr = key
			}
			if description != nil {
				descPtr = description
			}
			if ireal != nil {
				irealPtr = ireal
			}
			if publicValue != nil {
				parsed, err := strconv.ParseBool(*publicValue)
				if err != nil {
					fmt.Println("Invalid public value")
					continue
				}
				publicPtr = &parsed
			}
			if err := client.UpdateComposedTune(id, titlePtr, composerPtr, stylePtr, keyPtr, descPtr, irealPtr, publicPtr); err != nil {
				fmt.Println("Update failed:", err)
			}
		case "3":
			id := promptUint("Composition ID: ")
			if err := client.DeleteComposedTune(id); err != nil {
				fmt.Println("Delete failed:", err)
			}
		case "4":
			shareID := promptString("Share ID: ", "")
			resp, err := client.GetSharedTune(shareID)
			if err != nil {
				fmt.Println("View failed:", err)
				continue
			}
			fmt.Printf("Tune: %s | %s | %s | public=%t\n", resp.Tune.Title, resp.Tune.Composer, resp.Tune.Style, resp.Tune.IsPublic)
			fmt.Println(resp.Message)
		case "5":
			shareID := promptString("Share ID: ", "")
			if err := client.AcceptSharedTune(shareID); err != nil {
				fmt.Println("Accept failed:", err)
			}
		case "0":
			return
		default:
			fmt.Println("Unknown option")
		}
	}
}

func promptOptionalString(label string) *string {
	value := promptString(label, "")
	if value == "" {
		return nil
	}
	return &value
}

func practiceMenu(client *Client) {
	logs, err := client.ListPracticeLogs(nil)
	if err != nil {
		fmt.Println("Failed to load practice logs:", err)
		return
	}
	fmt.Printf("\nTotal entries: %d\n", len(logs))
	for _, log := range logs {
		title := ""
		if log.JazzStandard != nil {
			title = log.JazzStandard.Title
		}
		fmt.Printf("%d | %s | %d min | %s\n", log.ID, title, log.DurationMin, log.PracticedAt.Format(time.RFC3339))
	}
	fmt.Println("1) Log practice")
	fmt.Println("0) Back")
	choice := promptString("Select an option: ", "0")
	if choice != "1" {
		return
	}
	standardID := promptUint("Standard ID: ")
	duration := promptInt("Duration minutes: ", 30)
	notes := promptString("Notes (blank for none): ", "")
	practicedAt := promptString("Practiced at RFC3339 (blank for now): ", "")
	if err := client.LogPractice(standardID, duration, notes, practicedAt); err != nil {
		fmt.Println("Log failed:", err)
		return
	}
	fmt.Println("Practice logged")
}

func pendingMenu(client *Client) {
	page := promptInt("Page [1]: ", 1)
	limit := promptInt("Limit [20]: ", 20)
	resp, err := client.PendingStandards(page, limit)
	if err != nil {
		fmt.Println("Failed to load pending standards:", err)
		return
	}
	fmt.Printf("\nTotal pending: %d\n", resp.Total)
	for _, std := range resp.Standards {
		fmt.Printf("%d | %s | %s | %s\n", std.ID, std.Title, std.Composer, std.Style)
	}
	fmt.Println("1) Approve")
	fmt.Println("2) Reject")
	fmt.Println("0) Back")
	choice := promptString("Select an option: ", "0")
	switch choice {
	case "1":
		id := promptUint("Standard ID: ")
		if err := client.ApproveStandard(id); err != nil {
			fmt.Println("Approve failed:", err)
		}
	case "2":
		id := promptUint("Standard ID: ")
		if err := client.RejectStandard(id); err != nil {
			fmt.Println("Reject failed:", err)
		}
	}
}

func loadSession() (*Session, error) {
	path, err := sessionFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Session{}, nil
		}
		return nil, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func saveSession(session *Session) error {
	path, err := sessionFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	session.SavedAt = time.Now().UTC()
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func sessionFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "jazz_standards_db", "cli_session.json"), nil
}

func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return "http://localhost:8000"
	}
	return raw
}

func promptString(label, defaultValue string) string {
	fmt.Print(label)
	text, _ := inputReader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue
	}
	return text
}

func promptPassword(label string) string {
	fmt.Print(label)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(password))
}

func promptInt(label string, defaultValue int) int {
	text := promptString(label, "")
	if text == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return defaultValue
	}
	return value
}

func promptUint(label string) uint {
	for {
		text := promptString(label, "")
		value, err := strconv.ParseUint(text, 10, 32)
		if err == nil {
			return uint(value)
		}
		fmt.Println("Please enter a valid number")
	}
}

func urlQuery(value string) string {
	return url.QueryEscape(value)
}
