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

type OpenAIProvider struct {
	apiKey string
	model  string
}

func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		model:  getEnv("OPENAI_MODEL", "gpt-4"),
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai-" + p.model
}

func (p *OpenAIProvider) GenerateScript(ctx context.Context, prompt string, args []string) (string, int, error) {
	if p.apiKey == "" {
		return "", 0, fmt.Errorf("OPENAI_API_KEY not configured")
	}
	
	systemPrompt := BuildSystemPrompt(args)
	
	reqBody := map[string]interface{}{
		"model":       p.model,
		"max_tokens":  4096,
		"temperature": 0.7,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
	}
	
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("openai API error: %s", string(body))
	}
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}
	
	if len(result.Choices) == 0 {
		return "", 0, fmt.Errorf("no choices in response")
	}
	
	script := ExtractScript(result.Choices[0].Message.Content)
	if err := ValidateScript(script); err != nil {
		return "", 0, err
	}
	
	return script, result.Usage.TotalTokens, nil
}
