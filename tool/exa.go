package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// ExaSearch is a tool that uses the Exa API to search the web.
type ExaSearch struct {
	APIKey     string
	BaseURL    string
	NumResults int
}

type ExaOption func(*ExaSearch)

// WithExaBaseURL sets the base URL for the Exa API.
func WithExaBaseURL(url string) ExaOption {
	return func(t *ExaSearch) {
		t.BaseURL = url
	}
}

// WithExaNumResults sets the number of results to return.
func WithExaNumResults(num int) ExaOption {
	return func(t *ExaSearch) {
		t.NumResults = num
	}
}

// NewExaSearch creates a new ExaSearch tool.
// If apiKey is empty, it tries to read from EXA_API_KEY environment variable.
func NewExaSearch(apiKey string, opts ...ExaOption) (*ExaSearch, error) {
	if apiKey == "" {
		apiKey = os.Getenv("EXA_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("EXA_API_KEY not set")
	}

	t := &ExaSearch{
		APIKey:     apiKey,
		BaseURL:    "https://api.exa.ai",
		NumResults: 5,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t, nil
}

// Name returns the name of the tool.
func (t *ExaSearch) Name() string {
	return "Exa_Search"
}

// Description returns the description of the tool.
func (t *ExaSearch) Description() string {
	return "A search engine optimized for LLMs. " +
		"Useful for finding high-quality content and answering questions. " +
		"Input should be a search query."
}

// Call executes the search.
func (t *ExaSearch) Call(ctx context.Context, input string) (string, error) {
	reqBody := map[string]any{
		"query":      input,
		"numResults": t.NumResults,
		"contents": map[string]any{
			"text": true,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.BaseURL+"/search", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", t.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("exa api returned status: %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Format the output
	var sb strings.Builder
	if results, ok := result["results"].([]any); ok {
		for _, r := range results {
			if item, ok := r.(map[string]any); ok {
				title, _ := item["title"].(string)
				url, _ := item["url"].(string)
				text, _ := item["text"].(string)
				// Truncate text if it's too long
				if len(text) > 500 {
					text = text[:500] + "..."
				}
				sb.WriteString(fmt.Sprintf("Title: %s\nURL: %s\nContent: %s\n\n", title, url, text))
			}
		}
	}

	return sb.String(), nil
}
