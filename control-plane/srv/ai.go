package srv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// AIClient handles AI API interactions
type AIClient struct {
	Provider   string // "openai" or "anthropic"
	APIKey     string
	Model      string
	SystemPrompt string
}

// NewAIClient creates a new AI client based on available API keys
func NewAIClient() *AIClient {
	// Try Ollama (local, free) first
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		return &AIClient{
			Provider:     "ollama",
			APIKey:       host, // Store host URL in APIKey field
			Model:        getEnvOrDefault("OLLAMA_MODEL", "qwen2.5:1.5b"),
			SystemPrompt: akiguraSystemPrompt,
		}
	}
	// Default Ollama on localhost
	if _, err := http.Get("http://localhost:11434/api/tags"); err == nil {
		return &AIClient{
			Provider:     "ollama",
			APIKey:       "http://localhost:11434",
			Model:        getEnvOrDefault("OLLAMA_MODEL", "qwen2.5:1.5b"),
			SystemPrompt: akiguraSystemPrompt,
		}
	}
	// Try Anthropic (Claude)
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return &AIClient{
			Provider:     "anthropic",
			APIKey:       key,
			Model:        getEnvOrDefault("ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
			SystemPrompt: akiguraSystemPrompt,
		}
	}
	// Try OpenAI
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return &AIClient{
			Provider:     "openai",
			APIKey:       key,
			Model:        getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
			SystemPrompt: akiguraSystemPrompt,
		}
	}
	return nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

const akiguraSystemPrompt = `あなたはAkiGura（アキグラ）のAIサポートアシスタントです。
AkiGuraは草野球チーム向けのグラウンド空き枠監視・通知サービスです。

## サービス概要
- ユーザーが監視条件（施設、曜日、時間帯）を登録
- 定期的に自治体の予約サイトをスクレイピング
- 条件にマッチする空き枠が見つかったらメール/LINE/Slackで通知

## 料金プラン
- Free: 無料、1施設まで監視可能
- Personal: ¥500/月、5施設まで
- Pro: ¥2,000/月、20施設まで
- Organization: ¥10,000/月、無制限

## 対応地域
- 横浜市、綾瀬市、平塚市、神奈川県、鎌倉市、藤沢市
- 他の地域は順次追加予定

## よくある質問への回答方針
- 料金について聞かれたら、プラン一覧を提示
- 使い方を聞かれたら、監視条件の登録方法を説明
- 対応施設を聞かれたら、対応地域を案内
- 解約について聞かれたら、設定画面から可能と案内
- 技術的な問題は、詳細を聞いてサポートチケットを作成するよう案内

丁寧で親しみやすい日本語で回答してください。回答は簡潔に。`

// Chat sends a message to the AI and returns the response
func (c *AIClient) Chat(ctx context.Context, userMessage string, conversationHistory []ChatMessage) (string, error) {
	switch c.Provider {
	case "ollama":
		return c.chatOllama(ctx, userMessage, conversationHistory)
	case "anthropic":
		return c.chatAnthropic(ctx, userMessage, conversationHistory)
	case "openai":
		return c.chatOpenAI(ctx, userMessage, conversationHistory)
	default:
		return "", fmt.Errorf("unknown provider: %s", c.Provider)
	}
}

// Ollama API (local LLM)
func (c *AIClient) chatOllama(ctx context.Context, userMessage string, history []ChatMessage) (string, error) {
	messages := make([]map[string]string, 0, len(history)+2)
	messages = append(messages, map[string]string{"role": "system", "content": c.SystemPrompt})
	for _, m := range history {
		role := m.Role
		if role == "ai" {
			role = "assistant"
		}
		messages = append(messages, map[string]string{"role": role, "content": m.Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": userMessage})

	reqBody := map[string]interface{}{
		"model":    c.Model,
		"messages": messages,
		"stream":   false,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.APIKey+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ollama API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Message.Content, nil
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Anthropic (Claude) API
func (c *AIClient) chatAnthropic(ctx context.Context, userMessage string, history []ChatMessage) (string, error) {
	messages := make([]map[string]string, 0, len(history)+1)
	for _, m := range history {
		role := m.Role
		if role == "ai" {
			role = "assistant"
		}
		messages = append(messages, map[string]string{"role": role, "content": m.Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": userMessage})

	reqBody := map[string]interface{}{
		"model":      c.Model,
		"max_tokens": 1024,
		"system":     c.SystemPrompt,
		"messages":   messages,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	return "", fmt.Errorf("no content in response")
}

// OpenAI API
func (c *AIClient) chatOpenAI(ctx context.Context, userMessage string, history []ChatMessage) (string, error) {
	messages := make([]map[string]string, 0, len(history)+2)
	messages = append(messages, map[string]string{"role": "system", "content": c.SystemPrompt})
	for _, m := range history {
		role := m.Role
		if role == "ai" {
			role = "assistant"
		}
		messages = append(messages, map[string]string{"role": role, "content": m.Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": userMessage})

	reqBody := map[string]interface{}{
		"model":      c.Model,
		"max_tokens": 1024,
		"messages":   messages,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("openai API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no choices in response")
}

// IsConfigured returns true if AI is available
func (c *AIClient) IsConfigured() bool {
	return c != nil && c.APIKey != ""
}

// AnalyzeTicket uses AI to analyze a support ticket
func (c *AIClient) AnalyzeTicket(ctx context.Context, subject, message string) (*TicketAnalysis, error) {
	prompt := fmt.Sprintf(`以下のサポート問い合わせを分析してください。

件名: %s
内容: %s

以下のJSON形式で回答してください:
{
  "priority": "low|normal|high|urgent",
  "category": "billing|technical|general|feature_request",
  "suggested_response": "推奨する回答",
  "needs_human": true/false,
  "reason": "判断理由"
}`, subject, message)

	response, err := c.Chat(ctx, prompt, nil)
	if err != nil {
		return nil, err
	}

	// Extract JSON from response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}") + 1
	if start >= 0 && end > start {
		response = response[start:end]
	}

	var analysis TicketAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		// If JSON parsing fails, return default
		return &TicketAnalysis{
			Priority:          "normal",
			Category:          "general",
			SuggestedResponse: response,
			NeedsHuman:        true,
			Reason:            "AI分析結果のパースに失敗",
		}, nil
	}
	return &analysis, nil
}

type TicketAnalysis struct {
	Priority          string `json:"priority"`
	Category          string `json:"category"`
	SuggestedResponse string `json:"suggested_response"`
	NeedsHuman        bool   `json:"needs_human"`
	Reason            string `json:"reason"`
}
