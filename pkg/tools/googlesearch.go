package tools

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// GoogleSearchTool returns the Google search tool definition
func GoogleSearchTool() protocol.Tool {
	return protocol.Tool{
		Name:        "google_search",
		Description: "Performs a google search for the given text and returns the top 'num' responses",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"query": {
					Type:        "string",
					Description: "The search string to be entered into google search",
				},
				"num": {
					Type:        "integer",
					Description: "The number of results to return",
				},
			},
			Required: []string{"query"},
		},
	}
}

// HandleGoogleSearchTool handles the Google search tool invocation
func HandleGoogleSearchTool(params any) (any, error) {
	logger.Info("Handling Google search tool invocation")

	// Parse parameters
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	query, ok := paramsMap["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required and must be a string")
	}

	// Get number of results (default to 5)
	numResults := 5
	if numParam, ok := paramsMap["num"]; ok {
		if numFloat, ok := numParam.(float64); ok {
			numResults = int(numFloat)
		}
	}

	// Validate number of results
	if numResults <= 0 || numResults > 10 {
		numResults = 5 // Reset to default if invalid
	}

	// Perform the search
	results, err := googleSearch(query, numResults)
	if err != nil {
		return nil, err
	}

	// Return the results
	return map[string]any{
		"results": results,
		"query":   query,
		"count":   len(results),
	}, nil
}

// getZScalerBundle returns the Zscaler CA bundle if available
func getZScalerBundle() ([]byte, error) {
	// Path to Zscaler CA bundle
	bundlePath := filepath.Join(os.Getenv("HOME"), ".ssh/zscaler_ca_bundle.pem")

	// Load Zscaler CA bundle
	caCert, err := os.ReadFile(bundlePath)
	if err != nil {
		logger.Warn("Failed to read Zscaler CA bundle", err)
		return nil, err
	}

	return caCert, nil
}

// getCustomHTTPClient returns an HTTP client with custom TLS configuration
func getCustomHTTPClient() (*http.Client, error) {
	// Create a custom certificate pool
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		logger.Warn("Failed to get system cert pool", err)
		rootCAs = x509.NewCertPool()
	}

	// Get the Zscaler bundle
	zscalerCert, err := getZScalerBundle()
	if err != nil {
		logger.Warn("Proceeding without Zscaler certificate", err)
	} else {
		// Append the Zscaler certificate to the root CAs
		if ok := rootCAs.AppendCertsFromPEM(zscalerCert); !ok {
			logger.Warn("Failed to append Zscaler CA certificate")
		} else {
			logger.Info("Added Zscaler certificate to root CAs")
		}
	}

	// Create custom transport with the certificate pool
	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: rootCAs,
		},
		Proxy: http.ProxyFromEnvironment,
	}

	// Create a custom client with the transport
	client := &http.Client{
		Transport: customTransport,
		Timeout:   30 * time.Second,
	}

	return client, nil
}

// googleSearch performs a Google search using the Custom Search API and returns the top results
func googleSearch(query string, numResults int) ([]SearchResult, error) {
	// These would typically be stored in environment variables or configuration
	const searchKey = "YOUR_API_KEY"               // Replace with actual API key in production
	const searchEngineID = "YOUR_SEARCH_ENGINE_ID" // Replace with actual search engine ID in production

	if numResults <= 0 {
		numResults = 5 // Default to 5 results if not specified or invalid
	}

	// Google Custom Search API endpoint
	baseURL := "https://www.googleapis.com/customsearch/v1"

	// Create URL parameters
	params := url.Values{}
	params.Add("q", query)                           // Search query
	params.Add("key", searchKey)                     // API key
	params.Add("cx", searchEngineID)                 // Search engine ID
	params.Add("num", fmt.Sprintf("%d", numResults)) // Number of results

	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Get a custom HTTP client with Zscaler support
	client, err := getCustomHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create a request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Make the HTTP request
	logger.Info("Performing Google Custom Search for query", query)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to search API: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 200 OK
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search API returned error status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %w", err)
	}

	// Parse the JSON response
	var searchResponse struct {
		Items []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Snippet     string `json:"snippet"`
			DisplayLink string `json:"displayLink"`
		} `json:"items"`
	}

	err = json.Unmarshal(body, &searchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Convert the API response to our SearchResult format
	var results []SearchResult
	for _, item := range searchResponse.Items {
		results = append(results, SearchResult{
			Title:       item.Title,
			URL:         item.Link,
			Description: item.Snippet,
		})
	}

	// Return the results, which may be an empty array if no results were found
	return results, nil
}
