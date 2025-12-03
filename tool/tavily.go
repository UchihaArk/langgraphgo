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

// TavilySearch is a tool that uses the Tavily API to search the web.
type TavilySearch struct {
	APIKey      string
	BaseURL     string
	SearchDepth string
}

type TavilyOption func(*TavilySearch)

// WithTavilyBaseURL sets the base URL for the Tavily API.
func WithTavilyBaseURL(url string) TavilyOption {
	return func(t *TavilySearch) {
		t.BaseURL = url
	}
}

// WithTavilySearchDepth sets the search depth for the Tavily API.
// Valid values are "basic" and "advanced".
func WithTavilySearchDepth(depth string) TavilyOption {
	return func(t *TavilySearch) {
		t.SearchDepth = depth
	}
}

// NewTavilySearch creates a new TavilySearch tool.
// If apiKey is empty, it tries to read from TAVILY_API_KEY environment variable.
func NewTavilySearch(apiKey string, opts ...TavilyOption) (*TavilySearch, error) {
	if apiKey == "" {
		apiKey = os.Getenv("TAVILY_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("TAVILY_API_KEY not set")
	}

	t := &TavilySearch{
		APIKey:      apiKey,
		BaseURL:     "https://api.tavily.com",
		SearchDepth: "basic",
	}

	for _, opt := range opts {
		opt(t)
	}

	return t, nil
}

// Name returns the name of the tool.
func (t *TavilySearch) Name() string {
	return "Tavily_Search"
}

// Description returns the description of the tool.
func (t *TavilySearch) Description() string {
	return "A search engine optimized for comprehensive, accurate, and trusted results. " +
		"Useful for when you need to answer questions about current events. " +
		"Input should be a search query."
}

// Call executes the search.
func (t *TavilySearch) Call(ctx context.Context, input string) (string, error) {
	reqBody := map[string]interface{}{
		"query":        input,
		"api_key":      t.APIKey,
		"search_depth": t.SearchDepth,
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tavily api returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Format the output
	var sb strings.Builder
	if results, ok := result["results"].([]interface{}); ok {
		for _, r := range results {
			if item, ok := r.(map[string]interface{}); ok {
				title, _ := item["title"].(string)
				url, _ := item["url"].(string)
				content, _ := item["content"].(string)
				sb.WriteString(fmt.Sprintf("Title: %s\nURL: %s\nContent: %s\n\n", title, url, content))
			}
		}
	}

	return sb.String(), nil
}

// SearchResult represents a single search result with images
type SearchResult struct {
	Text   string
	Images []string
}

// CallWithImages executes the search and returns both text and images.
func (t *TavilySearch) CallWithImages(ctx context.Context, input string) (*SearchResult, error) {
	reqBody := map[string]interface{}{
		"query":          input,
		"api_key":        t.APIKey,
		"search_depth":   t.SearchDepth,
		"include_images": true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.BaseURL+"/search", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily api returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	searchResult := &SearchResult{
		Images: []string{},
	}

	// Format the text output
	var sb strings.Builder
	if results, ok := result["results"].([]interface{}); ok {
		for _, r := range results {
			if item, ok := r.(map[string]interface{}); ok {
				title, _ := item["title"].(string)
				url, _ := item["url"].(string)
				content, _ := item["content"].(string)
				sb.WriteString(fmt.Sprintf("Title: %s\nURL: %s\nContent: %s\n\n", title, url, content))
			}
		}
	}
	searchResult.Text = sb.String()

	// Extract images
	if images, ok := result["images"].([]interface{}); ok {
		for _, img := range images {
			if imgURL, ok := img.(string); ok {
				searchResult.Images = append(searchResult.Images, imgURL)
			}
		}
	}

	return searchResult, nil
}
