package srv

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"srv.exe.dev/db/dbgen"
)

// AuthConfig holds authentication settings
type AuthConfig struct {
	TokenExpiry   time.Duration
	BaseURL       string
	SendGridKey   string
	EmailFrom     string
	EmailFromName string
}

func getAuthConfig() AuthConfig {
	return AuthConfig{
		TokenExpiry:   15 * time.Minute,
		BaseURL:       getEnvOrDefault("BASE_URL", "http://localhost:8000"),
		SendGridKey:   os.Getenv("SENDGRID_API_KEY"),
		EmailFrom:     getEnvOrDefault("SENDGRID_FROM", "noreply@akigura.jp"),
		EmailFromName: getEnvOrDefault("SENDGRID_FROM_NAME", "AkiGura"),
	}
}

// generateToken creates a secure random token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HandleRequestMagicLink sends a magic link to the user's email
func (s *Server) HandleRequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"` // for registration
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		s.jsonError(w, "email required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Find or create team
	team, err := s.Queries.GetTeamByEmail(ctx, req.Email)
	if err == sql.ErrNoRows {
		// New user - create team
		if req.Name == "" {
			req.Name = "My Team"
		}
		team, err = s.Queries.CreateTeam(ctx, dbgen.CreateTeamParams{
			ID:    uuid.New().String(),
			Name:  req.Name,
			Email: req.Email,
			Plan:  "free",
		})
		if err != nil {
			slog.Error("create team", "error", err)
			s.jsonError(w, "failed to create team", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		slog.Error("get team", "error", err)
		s.jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	// Generate token
	token, err := generateToken()
	if err != nil {
		slog.Error("generate token", "error", err)
		s.jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	config := getAuthConfig()
	expiresAt := time.Now().Add(config.TokenExpiry)

	// Save token
	_, err = s.Queries.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		ID:        uuid.New().String(),
		TeamID:    team.ID,
		Token:     token,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		slog.Error("create auth token", "error", err)
		s.jsonError(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	// Send magic link email
	magicLink := fmt.Sprintf("%s/auth/verify?token=%s", config.BaseURL, token)

	if err := sendMagicLinkEmail(config, req.Email, team.Name, magicLink); err != nil {
		slog.Warn("send magic link email", "error", err, "email", req.Email)
		// Don't fail the request - show the link in dev mode
		if os.Getenv("DEBUG") == "true" {
			s.jsonResponse(w, map[string]string{
				"message":    "Magic link created (email sending disabled)",
				"debug_link": magicLink,
			})
			return
		}
	}

	s.jsonResponse(w, map[string]string{
		"message": "認証メールを送信しました。メールをご確認ください。",
	})
}

// HandleVerifyMagicLink verifies the magic link token
func (s *Server) HandleVerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Find token
	authToken, err := s.Queries.GetAuthTokenByToken(ctx, token)
	if err == sql.ErrNoRows {
		http.Error(w, "リンクが無効または期限切れです。再度ログインしてください。", http.StatusBadRequest)
		return
	} else if err != nil {
		slog.Error("get auth token", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Mark token as used
	if err := s.Queries.MarkAuthTokenUsed(ctx, authToken.ID); err != nil {
		slog.Error("mark token used", "error", err)
	}

	// Get team
	team, err := s.Queries.GetTeam(ctx, authToken.TeamID)
	if err != nil {
		slog.Error("get team", "error", err)
		http.Error(w, "Team not found", http.StatusInternalServerError)
		return
	}

	// Return HTML that stores the session and redirects
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>認証完了 - AkiGura</title>
</head>
<body>
    <script>
        localStorage.setItem('akigura_team', JSON.stringify(%s));
        window.location.href = '/user';
    </script>
    <noscript>
        <p>認証が完了しました。<a href="/user">こちら</a>をクリックしてダッシュボードに移動してください。</p>
    </noscript>
</body>
</html>`, teamToJSON(team))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func teamToJSON(team dbgen.Team) string {
	data, _ := json.Marshal(map[string]interface{}{
		"id":         team.ID,
		"name":       team.Name,
		"email":      team.Email,
		"plan":       team.Plan,
		"status":     team.Status,
		"created_at": team.CreatedAt,
		"updated_at": team.UpdatedAt,
	})
	return string(data)
}

// sendMagicLinkEmail sends the magic link via SendGrid
func sendMagicLinkEmail(config AuthConfig, email, teamName, magicLink string) error {
	if config.SendGridKey == "" {
		slog.Info("Magic link (SendGrid not configured)", "email", email, "link", magicLink)
		return nil
	}

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="background: #4F46E5; color: white; padding: 20px; border-radius: 8px 8px 0 0;">
    <h1 style="margin: 0;">⚾ AkiGura</h1>
  </div>
  <div style="border: 1px solid #e5e7eb; border-top: none; padding: 20px; border-radius: 0 0 8px 8px;">
    <p>%s 様</p>
    <p>以下のボタンをクリックしてログインしてください。このリンクは15分間有効です。</p>
    <p style="text-align: center; margin: 30px 0;">
      <a href="%s" style="display: inline-block; background: #4F46E5; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: bold;">ログインする</a>
    </p>
    <p style="color: #6b7280; font-size: 12px;">このメールに心当たりがない場合は、無視してください。</p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
    <p style="color: #9ca3af; font-size: 11px;">リンクが機能しない場合は、以下のURLをブラウザに貼り付けてください：<br>%s</p>
  </div>
</body>
</html>`, teamName, magicLink, magicLink)

	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{"email": email},
				},
				"subject": "【AkiGura】ログインリンク",
			},
		},
		"from": map[string]string{
			"email": config.EmailFrom,
			"name":  config.EmailFromName,
		},
		"content": []map[string]string{
			{
				"type":  "text/html",
				"value": htmlContent,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.SendGridKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: %d", resp.StatusCode)
	}
	return nil
}
