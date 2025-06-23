package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
	"github.com/richard-senior/mcp/pkg/transport"
)

const surl = "https://customsearch.googleapis.com/customsearch/v1"
const searchKey = "AIzaSyBqIgU6NTu8uPnusd4IRvC1tG-CDKaqrgM"
const searchEngineID = "32e99349b2ae84bcd"

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// GoogleSearchTool returns the Google search tool definition
func GoogleSearchTool() protocol.Tool {
	return protocol.Tool{
		Name: "google_search",
		Description: `
		Performs an internet web (google) search for the given text and returns the top 'num' responses.
		For each of the 'num' results the following information is returned:
		- Title: The title of the search result
		- URL: The URL of the search result. This can then be with the html_2_markdown tool to retrieve the content
		- Description: The summary of the contents of the web page
		This tool should be used when:
		- You have no current information about the issue, you can formulate a question that will get you data from the internet
		- the use asks you to find information about..
		- the user suggests that you search the internet for..
		- the user asks you to google..
		- the user asks you to get me information about..
		etc.
		`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"query": {
					Type:        "string",
					Description: "The search term for example 'Ozric Tentacles' ",
				},
				"num": {
					Type:        "integer",
					Description: "The number of results to return, defaults to 3",
				},
			},
			Required: []string{"query"},
		},
	}
}

// HandleGoogleSearchTool handles the Google search tool invocation
func HandleGoogleSearchTool(params any) (any, error) {
	logger.Info("Handling Google search tool invocation")

	// Convert params to map[string]any
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
	results, err := GoogleSearch(query, numResults, false)
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

// googleSearch performs a Google search using the Custom Search API and returns the top results
func GoogleSearch(query string, numResults int, images bool) ([]SearchResult, error) {
	// These would typically be stored in environment variables or configuration

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
	if images {
		params.Add("searchType", "image") // Search for images
		//params.Add("imgSize", "MEDIUM")
	}

	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Get a custom HTTP client with Zscaler support
	client, err := transport.GetCustomHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create a request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Make the HTTP request
	if images {
		logger.Info("Performing Google Image Search for query", query)
	} else {
		logger.Info("Performing Google Search for query", query)
	}
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
