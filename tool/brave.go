package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// BraveSearch is a tool that uses the Brave Search API to search the web.
type BraveSearch struct {
	APIKey  string
	BaseURL string
	Count   int
	Country string
	Lang    string
}

type BraveOption func(*BraveSearch)

// WithBraveBaseURL sets the base URL for the Brave Search API.
func WithBraveBaseURL(baseURL string) BraveOption {
	return func(b *BraveSearch) {
		b.BaseURL = baseURL
	}
}

// WithBraveCount sets the number of results to return (1-20).
func WithBraveCount(count int) BraveOption {
	return func(b *BraveSearch) {
		if count < 1 {
			count = 1
		}
		if count > 20 {
			count = 20
		}
		b.Count = count
	}
}

// WithBraveCountry sets the country code for search results (e.g., "US", "CN").
func WithBraveCountry(country string) BraveOption {
	return func(b *BraveSearch) {
		b.Country = country
	}
}

// WithBraveLang sets the language code for search results (e.g., "en", "zh").
func WithBraveLang(lang string) BraveOption {
	return func(b *BraveSearch) {
		b.Lang = lang
	}
}

// NewBraveSearch creates a new BraveSearch tool.
// If apiKey is empty, it tries to read from BRAVE_API_KEY environment variable.
func NewBraveSearch(apiKey string, opts ...BraveOption) (*BraveSearch, error) {
	if apiKey == "" {
		apiKey = os.Getenv("BRAVE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("BRAVE_API_KEY not set")
	}

	b := &BraveSearch{
		APIKey:  apiKey,
		BaseURL: "https://api.search.brave.com/res/v1/web/search",
		Count:   10,
		Country: "US",
		Lang:    "en",
	}

	for _, opt := range opts {
		opt(b)
	}

	return b, nil
}

// Name returns the name of the tool.
func (b *BraveSearch) Name() string {
	return "Brave_Search"
}

// Description returns the description of the tool.
func (b *BraveSearch) Description() string {
	return "A privacy-focused search engine powered by Brave. " +
		"Useful for finding current information and answering questions. " +
		"Input should be a search query."
}

// Call executes the search.
func (b *BraveSearch) Call(ctx context.Context, input string) (string, error) {
	// Build query parameters
	params := url.Values{}
	params.Set("q", input)
	params.Set("count", fmt.Sprintf("%d", b.Count))
	if b.Country != "" {
		params.Set("country", b.Country)
	}
	if b.Lang != "" {
		params.Set("search_lang", b.Lang)
	}

	// Create request URL
	reqURL := fmt.Sprintf("%s?%s", b.BaseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", b.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("brave api returned status: %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Format the output
	var sb strings.Builder

	// Extract web results
	if web, ok := result["web"].(map[string]any); ok {
		if results, ok := web["results"].([]any); ok {
			for i, r := range results {
				if item, ok := r.(map[string]any); ok {
					title, _ := item["title"].(string)
					url, _ := item["url"].(string)
					description, _ := item["description"].(string)

					sb.WriteString(fmt.Sprintf("%d. Title: %s\nURL: %s\nDescription: %s\n\n",
						i+1, title, url, description))
				}
			}
		}
	}

	if sb.Len() == 0 {
		return "No results found", nil
	}

	return sb.String(), nil
}
