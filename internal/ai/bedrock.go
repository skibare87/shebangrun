package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type BedrockProvider struct {
	client  *bedrockruntime.Client
	modelID string
}

func NewBedrockProvider(ctx context.Context) (*BedrockProvider, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	
	return &BedrockProvider{
		client:  bedrockruntime.NewFromConfig(cfg),
		modelID: getEnv("BEDROCK_MODEL_ID", "anthropic.claude-3-5-sonnet-20241022-v2:0"),
	}, nil
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
	
	output, err := p.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(p.modelID),
		Body:        body,
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return "", 0, err
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
	
	if err := json.Unmarshal(output.Body, &result); err != nil {
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
