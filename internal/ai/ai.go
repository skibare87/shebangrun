package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

type AIProvider interface {
	GenerateScript(ctx context.Context, prompt string, args []string) (script string, tokens int, err error)
	Name() string
}

type GenerateRequest struct {
	Prompt   string   `json:"prompt"`
	Args     []string `json:"args"`
	Provider string   `json:"provider"`
}

type GenerateResponse struct {
	Script   string `json:"script"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Tokens   int    `json:"tokens"`
}

// BuildSystemPrompt creates the system prompt for script generation
func BuildSystemPrompt(args []string) string {
	argsStr := ""
	if len(args) > 0 {
		argsStr = fmt.Sprintf("\n- Script arguments: %s", strings.Join(args, ", "))
	}
	
	return fmt.Sprintf(`You are a script generation assistant. Generate ONLY executable scripts.

Requirements:
- Include appropriate shebang line (#!/bin/bash, #!/usr/bin/env python3, etc.)
- Add error handling where appropriate
- Use best practices for the language%s
- Keep it minimal and focused
- Output ONLY the script code, no explanations, no markdown blocks

Output format: Raw script only, starting with shebang line.`, argsStr)
}

// ExtractScript cleans up the response to get just the script
func ExtractScript(response string) string {
	// Remove markdown code blocks if present
	response = strings.TrimSpace(response)
	
	// Remove ```bash or ```python blocks
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			// Remove first and last line
			response = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	
	return strings.TrimSpace(response)
}

// ValidateScript checks if the response looks like a valid script
func ValidateScript(script string) error {
	if script == "" {
		return errors.New("empty script generated")
	}
	
	if !strings.HasPrefix(script, "#!") {
		return errors.New("script missing shebang line")
	}
	
	return nil
}

// CountTokens estimates token count (rough approximation)
func CountTokens(text string) int {
	// Rough estimate: 1 token ≈ 4 characters
	return len(text) / 4
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
