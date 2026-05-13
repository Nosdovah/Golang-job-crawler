package analyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type AIResponse struct {
	Requirements []string `json:"requirements"`
}

// ExtractRequirementsWithAI uses an LLM (Gemma/Qwen/Llama) to intelligently extract skills.
func ExtractRequirementsWithAI(description string) ([]string, error) {
	apiKey := os.Getenv("AI_API_KEY")
	apiURL := os.Getenv("AI_API_URL")
	model := os.Getenv("AI_MODEL")

	if apiURL == "" {
		return nil, fmt.Errorf("AI_API_URL not set")
	}

	// Prepare the prompt for the AI
	prompt := `Task: Extract technical job requirements from the following job description.
Requirements should include programming languages, frameworks, databases, and cloud tools.
Return the result ONLY as a JSON object with a single key "requirements" containing an array of strings.
Description: ` + description

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a professional technical recruiter assistant."},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.1,
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Note: Standard OpenAI-compatible response parsing
	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	if openAIResp.Error.Message != "" {
		return nil, fmt.Errorf("AI API Error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("AI returned no results")
	}

	// Parse the inner JSON string returned by the AI
	var extracted AIResponse
	cleanContent := strings.TrimSpace(openAIResp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(cleanContent), &extracted); err != nil {
		return nil, fmt.Errorf("AI did not return valid JSON requirements: %v", err)
	}

	return extracted.Requirements, nil
}
