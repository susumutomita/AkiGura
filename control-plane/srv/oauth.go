package srv

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"srv.exe.dev/db/dbgen"
)

// OAuthConfig holds OAuth settings
type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	BaseURL            string
}

func getOAuthConfig() OAuthConfig {
	return OAuthConfig{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		BaseURL:            getEnvOrDefault("BASE_URL", "http://localhost:8001"),
	}
}

// generateOAuthState creates a secure random state for CSRF protection
func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HandleGoogleLogin initiates Google OAuth flow
func (s *Server) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	config := getOAuthConfig()

	if config.GoogleClientID == "" {
		s.jsonError(w, "Google OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	state, err := generateOAuthState()
	if err != nil {
		slog.Error("generate oauth state", "error", err)
		s.jsonError(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	// Store state in cookie for verification
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   strings.HasPrefix(config.BaseURL, "https"),
		SameSite: http.SameSiteLaxMode,
	})

	redirectURI := config.BaseURL + "/auth/google/callback"

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&access_type=offline&prompt=select_account",
		url.QueryEscape(config.GoogleClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape("openid email profile"),
		url.QueryEscape(state),
	)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleGoogleCallback handles the OAuth callback from Google
func (s *Server) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	config := getOAuthConfig()

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		slog.Warn("oauth state cookie missing")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		slog.Warn("oauth state mismatch")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Check for error from Google
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		slog.Warn("oauth error from google", "error", errParam)
		http.Redirect(w, r, "/user?error=oauth_failed", http.StatusTemporaryRedirect)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	tokenResp, err := exchangeGoogleCode(config, code)
	if err != nil {
		slog.Error("exchange google code", "error", err)
		http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
		return
	}

	// Get user info
	userInfo, err := getGoogleUserInfo(tokenResp.AccessToken)
	if err != nil {
		slog.Error("get google user info", "error", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	if userInfo.Email == "" {
		http.Error(w, "Email not provided by Google", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Find or create team
	team, err := s.Queries.GetTeamByEmail(ctx, userInfo.Email)
	if err == sql.ErrNoRows {
		// Create new team
		teamName := userInfo.Name
		if teamName == "" {
			teamName = "My Team"
		}
		team, err = s.Queries.CreateTeam(ctx, dbgen.CreateTeamParams{
			ID:    uuid.New().String(),
			Name:  teamName,
			Email: userInfo.Email,
			Plan:  "free",
		})
		if err != nil {
			slog.Error("create team", "error", err)
			http.Error(w, "Failed to create team", http.StatusInternalServerError)
			return
		}
		slog.Info("created team via google oauth", "email", userInfo.Email, "name", teamName)
	} else if err != nil {
		slog.Error("get team", "error", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	slog.Info("google oauth login", "email", userInfo.Email, "team_id", team.ID)

	// Return HTML that stores the session and redirects (same as magic link)
	teamJSON := teamToJSON(team)
	teamBase64 := base64.StdEncoding.EncodeToString([]byte(teamJSON))

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>ログイン完了 - AkiGura</title>
</head>
<body>
    <script>
        var teamData = JSON.parse(atob('%s'));
        localStorage.setItem('akigura_team', JSON.stringify(teamData));
        window.location.href = '/user';
    </script>
    <noscript>
        <p>ログインが完了しました。<a href="/user">こちら</a>をクリックしてダッシュボードに移動してください。</p>
    </noscript>
</body>
</html>`, teamBase64)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent))
}

// GoogleTokenResponse represents the token response from Google
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

// GoogleUserInfo represents user info from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

func exchangeGoogleCode(config OAuthConfig, code string) (*GoogleTokenResponse, error) {
	redirectURI := config.BaseURL + "/auth/google/callback"

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", config.GoogleClientID)
	data.Set("client_secret", config.GoogleClientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.Post(
		"https://oauth2.googleapis.com/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// HandleOAuthConfig returns OAuth configuration for the frontend
func (s *Server) HandleOAuthConfig(w http.ResponseWriter, r *http.Request) {
	config := getOAuthConfig()
	s.jsonResponse(w, map[string]interface{}{
		"google_enabled": config.GoogleClientID != "",
	})
}
