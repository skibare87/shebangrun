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

type BedrockProvider struct {
	bearerToken string
	modelID     string
	region      string
}

func NewBedrockProvider() *BedrockProvider {
	token := os.Getenv("AWS_BEARER_TOKEN_BEDROCK")
	if token == "" {
		return nil
	}
	
	return &BedrockProvider{
		bearerToken: token,
		modelID:     getEnv("BEDROCK_MODEL_ID", "amazon.nova-lite-v1:0"),
		region:      getEnv("AWS_REGION", "us-east-1"),
	}
}

func (p *BedrockProvider) Name() string {
	return "bedrock-" + p.modelID
}

func (p *BedrockProvider) GenerateScript(ctx context.Context, prompt string, args []string) (string, int, error) {
	systemPrompt := BuildSystemPrompt(args)
	
	reqBody := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        4096,
		"system":            systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	
	body, _ := json.Marshal(reqBody)
	
	url := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke", p.region, p.modelID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.bearerToken)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("bedrock API error: %s", string(body))
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
