package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type ClaudeProvider struct {
	apiKey string
	model  string
}

func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{
		apiKey: os.Getenv("CLAUDE_API_KEY"),
		model:  getEnv("CLAUDE_MODEL", "claude-3-5-sonnet-20241022"),
	}
}

func (p *ClaudeProvider) Name() string {
	return "claude-" + p.model
}

func (p *ClaudeProvider) GenerateScript(ctx context.Context, prompt string, args []string) (string, int, error) {
	if p.apiKey == "" {
		return "", 0, fmt.Errorf("CLAUDE_API_KEY not configured")
	}
	
	systemPrompt := BuildSystemPrompt(args)
	
	reqBody := map[string]interface{}{
		"model":      p.model,
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("claude API error: %s", string(body))
	}
	
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}
	
	if len(result.Content) == 0 {
		return "", 0, fmt.Errorf("no content in response")
	}
	
	script := ExtractScript(result.Content[0].Text)
	if err := ValidateScript(script); err != nil {
		return "", 0, err
	}
	
	tokens := result.Usage.InputTokens + result.Usage.OutputTokens
	return script, tokens, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
