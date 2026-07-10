package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config holds OpenAI-compatible LLM settings.
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

// Client calls chat completion APIs.
type Client struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Client {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.deepseek.com"
	}
	model := cfg.Model
	if model == "" {
		model = "deepseek-chat"
	}
	return &Client{
		cfg:    Config{APIKey: cfg.APIKey, BaseURL: base, Model: model},
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func ConfigFromEnv() Config {
	key := os.Getenv("BIGBASE_LLM_API_KEY")
	if key == "" {
		key = os.Getenv("DEEPSEEK_API_KEY")
	}
	base := os.Getenv("BIGBASE_LLM_BASE_URL")
	if base == "" {
		base = "https://api.deepseek.com"
	}
	model := os.Getenv("BIGBASE_LLM_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}
	return Config{APIKey: key, BaseURL: base, Model: model}
}

func (c *Client) Enabled() bool {
	return c != nil && c.cfg.APIKey != ""
}

// Complete sends a one-shot user prompt and returns assistant text.
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	if c == nil || c.cfg.APIKey == "" {
		return "", fmt.Errorf("llm not configured")
	}
	prompt = StripSecretLines(prompt)
	body, _ := json.Marshal(map[string]any{
		"model": c.cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	url := c.cfg.BaseURL + "/chat/completions"
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt == 0 {
			time.Sleep(500 * time.Millisecond)
			lastErr = fmt.Errorf("rate limited")
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("llm status %d: %s", resp.StatusCode, string(respBody))
		}
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return "", err
		}
		if len(parsed.Choices) == 0 {
			return "", fmt.Errorf("empty llm response")
		}
		return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("llm request failed")
}

var secretPatterns = []string{"API_KEY", "SECRET", "TOKEN", "PASSWORD", "PRIVATE_KEY"}

// StripSecretLines removes lines that likely contain secrets before LLM prompts.
func StripSecretLines(text string) string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		upper := strings.ToUpper(line)
		skip := false
		for _, p := range secretPatterns {
			if strings.Contains(upper, p+"=") || strings.Contains(upper, p+":") {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
