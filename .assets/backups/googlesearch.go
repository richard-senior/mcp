package processor

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
	"regexp"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/richard-senior/mcp/internal/logger"
)

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// GoogleSearchResponse represents the response from the Google search
type GoogleSearchResponse struct {
	Results []SearchResult `json:"results"`
	Query   string         `json:"query"`
}

const searchKey = "AIzaSyBqIgU6NTu8uPnusd4IRvC1tG-CDKaqrgM" // Replace with actual API key in production
const searchEngineID = "32e99349b2ae84bcd"                  // Replace with actual search engine ID in production

func GetZScalerBundle() ([]byte, error) {
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

func GetCustomHTTPClient() (*http.Client, error) {
	// Create a custom certificate pool
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		logger.Warn("Failed to get system cert pool", err)
		rootCAs = x509.NewCertPool()
	}

	// Get the Zscaler bundle
	zscalerCert, err := GetZScalerBundle()
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

// WikipediaImageSearch searches for an image on Wikipedia and returns the image bytes if found
func WikipediaImageSearch(query string, imageSize int) ([]byte, string, error) {
	// Default image size if not specified or invalid
	if imageSize <= 0 {
		imageSize = 500
	}

	// Trim leading and trailing spaces from the query
	query = strings.TrimSpace(query)

	// Trim leading and trailing spaces from the query
	query = strings.TrimSpace(query)

	// Create an array of search term variations to try
	variations := []string{
		query,                                                // Original query
		strings.ToLower(query),                               // Lowercase
		strings.ReplaceAll(query, " ", "_"),                  // Replace spaces with underscores
		strings.ReplaceAll(query, " ", "-"),                  // Replace spaces with hyphens
		strings.Title(strings.ToLower(query)),                // Title case
		strings.ReplaceAll(strings.ToLower(query), " ", "_"), // Lowercase with underscores
		strings.ReplaceAll(strings.ToLower(query), " ", "-"), // Lowercase with hyphens
		strings.ReplaceAll(strings.ToLower(query), " ", "-"), // Lowercase with hyphens
	}

	// Remove duplicates from variations
	uniqueVariations := []string{}
	seen := make(map[string]bool)
	for _, variation := range variations {
		if !seen[variation] {
			seen[variation] = true
			uniqueVariations = append(uniqueVariations, variation)
		}
	}

	// Try each variation until we find an image
	for _, searchTerm := range uniqueVariations {
		imageData, contentType, err := tryWikipediaImageSearch(searchTerm, imageSize)
		if err == nil {
			// Success! Return the image data
			return imageData, contentType, nil
		}
		logger.Info("Search failed for variation:", searchTerm, "- trying next variation")
	}

	// If we get here, all variations failed
	return nil, "", fmt.Errorf("no image found for any variation of query: %s", query)
}

// tryWikipediaImageSearch attempts to find an image on Wikipedia for a specific search term
func tryWikipediaImageSearch(query string, imageSize int) ([]byte, string, error) {
	// Wikipedia API endpoint for searching images
	baseURL := "https://en.wikipedia.org/w/api.php"

	// Create URL parameters
	params := url.Values{}
	params.Add("action", "query")
	params.Add("titles", query)
	params.Add("prop", "pageimages")
	params.Add("format", "json")
	params.Add("pithumbsize", fmt.Sprintf("%d", imageSize))

	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Get a custom HTTP client with Zscaler support
	client, err := GetCustomHTTPClient()
	if err != nil {
		return nil, "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create a request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to make the request look more like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Make the HTTP request
	logger.Info("Performing Wikipedia image search for query:", query)
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to Wikipedia API: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 200 OK
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("Wikipedia API returned error status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read Wikipedia API response: %w", err)
	}

	// Parse the JSON response
	var apiResponse struct {
		Query struct {
			Pages map[string]struct {
				Thumbnail struct {
					Source string `json:"source"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"thumbnail"`
				PageImage string `json:"pageimage"`
				Title     string `json:"title"`
			} `json:"pages"`
		} `json:"query"`
	}

	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse Wikipedia API response: %w", err)
	}

	// Check if we got any pages with images
	var imageURL string
	for _, page := range apiResponse.Query.Pages {
		if page.Thumbnail.Source != "" {
			imageURL = page.Thumbnail.Source
			break
		}
	}

	if imageURL == "" {
		return nil, "", fmt.Errorf("no image found for query: %s", query)
	}

	// Now fetch the actual image
	logger.Info("Found image for", query, "at URL:", imageURL)

	// Create a request for the image
	imgReq, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create image request: %w", err)
	}

	// Make the HTTP request for the image
	imgResp, err := client.Do(imgReq)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch image: %w", err)
	}
	defer imgResp.Body.Close()

	// Check if the response status code is not 200 OK
	if imgResp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("image request returned error status %d", imgResp.StatusCode)
	}

	// Check if the response is actually an image
	contentType := imgResp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("response is not an image, content type: %s", contentType)
	}

	// Read the image data
	imageData, err := io.ReadAll(imgResp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %w", err)
	}

	logger.Info("Successfully retrieved image for", query, "with size:", len(imageData), "bytes")

	return imageData, contentType, nil
}

// GoogleSearch performs a Google search using the Custom Search API and returns the top results
func GoogleSearch(query string, numResults int) ([]SearchResult, error) {
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
	client, err := GetCustomHTTPClient()
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

// ProcessGoogleSearchRequest handles Google search requests
func ProcessGoogleSearchRequest(query string, requestID string) ([]byte, error) {
	// Extract the search query
	// Format: googlesearch <query> [num_results]

	// Handle quoted queries
	var searchQuery string
	var numResults int = 5 // Default to 5 results

	if strings.Contains(query, "\"") {
		// Extract the quoted part
		startQuote := strings.Index(query, "\"")
		endQuote := strings.LastIndex(query, "\"")

		if startQuote != -1 && endQuote != -1 && startQuote != endQuote {
			searchQuery = query[startQuote+1 : endQuote]

			// Check if there's a number after the quoted part
			if endQuote+1 < len(query) {
				fmt.Sscanf(query[endQuote+1:], "%d", &numResults)
			}
		} else {
			// Fallback to simple parsing
			parts := strings.SplitN(query, " ", 3)
			if len(parts) < 2 {
				return CreateErrorResponse("google_search_error", "Invalid googlesearch command format", requestID)
			}
			searchQuery = parts[1]

			// Check if number of results is specified
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &numResults)
			}
		}
	} else {
		// Simple parsing for non-quoted queries
		parts := strings.SplitN(query, " ", 3)
		if len(parts) < 2 {
			return CreateErrorResponse("google_search_error", "Invalid googlesearch command format", requestID)
		}
		searchQuery = parts[1]

		// Check if number of results is specified
		if len(parts) >= 3 {
			fmt.Sscanf(parts[2], "%d", &numResults)
		}
	}

	// Validate number of results
	if numResults <= 0 || numResults > 10 {
		numResults = 5 // Reset to default if invalid
	}

	// Perform the search
	results, err := GoogleSearch(searchQuery, numResults)
	if err != nil {
		logger.Error("Failed to perform Google search", err)
		return CreateErrorResponse("google_search_error", err.Error(), requestID)
	}

	// Create a response with the search results
	response := MCPResponse{
		RequestID: requestID,
		Context: map[string]interface{}{
			"results": results,
			"query":   searchQuery,
			"count":   len(results),
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	// Marshal the response to JSON
	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal response to JSON", err)
		return CreateErrorResponse("internal_error", "Failed to create response", requestID)
	}

	return jsonResult, nil
}

// ProcessWikipediaImageRequest handles Wikipedia image search requests
func ProcessWikipediaImageRequest(query string, requestID string) ([]byte, error) {
	// Extract the search query
	// Format: wikipediaimage <query> [size]

	// Handle quoted queries
	var searchQuery string
	var imageSize int = 500 // Default to 500px if not specified

	if strings.Contains(query, "\"") {
		// Extract the quoted part
		startQuote := strings.Index(query, "\"")
		endQuote := strings.LastIndex(query, "\"")

		if startQuote != -1 && endQuote != -1 && startQuote != endQuote {
			searchQuery = query[startQuote+1 : endQuote]

			// Check if there's a number after the quoted part
			if endQuote+1 < len(query) {
				fmt.Sscanf(query[endQuote+1:], "%d", &imageSize)
			}
		} else {
			// Fallback to simple parsing
			parts := strings.SplitN(query, " ", 3)
			if len(parts) < 2 {
				return CreateErrorResponse("wikipedia_image_error", "Invalid wikipediaimage command format", requestID)
			}
			searchQuery = parts[1]

			// Check if image size is specified
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &imageSize)
			}
		}
	} else {
		// Simple parsing for non-quoted queries
		parts := strings.SplitN(query, " ", 3)
		if len(parts) < 2 {
			return CreateErrorResponse("wikipedia_image_error", "Invalid wikipediaimage command format", requestID)
		}
		searchQuery = parts[1]

		// Check if image size is specified
		if len(parts) >= 3 {
			fmt.Sscanf(parts[2], "%d", &imageSize)
		}
	}

	// Validate image size
	if imageSize <= 0 {
		imageSize = 500 // Reset to default if invalid
	}

	// Perform the image search
	imageData, contentType, err := WikipediaImageSearch(searchQuery, imageSize)
	if err != nil {
		logger.Error("Failed to perform Wikipedia image search", err)
		return CreateErrorResponse("wikipedia_image_error", err.Error(), requestID)
	}

	// Create a response with the image data
	response := MCPResponse{
		RequestID: requestID,
		Context: map[string]interface{}{
			"image":         imageData,
			"contentType":   contentType,
			"query":         searchQuery,
			"size":          len(imageData),
			"requestedSize": imageSize,
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	// Marshal the response to JSON
	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal response to JSON", err)
		return CreateErrorResponse("internal_error", "Failed to create response", requestID)
	}

	return jsonResult, nil
}

// ProcessWebPage2MarkdownRequest handles requests to convert a webpage to Markdown
func ProcessWebPage2MarkdownRequest(query string, requestID string) ([]byte, error) {
	// Extract the URL
	// Format: webpage2markdown <url> [summarize=true/false]

	// Handle quoted URLs
	var urlStr string
	var summarize bool = false

	if strings.Contains(query, "\"") {
		// Extract the quoted part
		startQuote := strings.Index(query, "\"")
		endQuote := strings.LastIndex(query, "\"")

		if startQuote != -1 && endQuote != -1 && startQuote != endQuote {
			urlStr = query[startQuote+1 : endQuote]
			
			// Check if there's a summarize parameter after the quoted URL
			if strings.Contains(query[endQuote+1:], "summarize=true") {
				summarize = true
			}
		} else {
			// Fallback to simple parsing
			parts := strings.SplitN(query, " ", 2)
			if len(parts) < 2 {
				return CreateErrorResponse("webpage2markdown_error", "Invalid webpage2markdown command format", requestID)
			}
			urlStr = parts[1]
			
			// Check if there's a summarize parameter
			if strings.Contains(urlStr, "summarize=true") {
				urlStr = strings.Replace(urlStr, "summarize=true", "", 1)
				urlStr = strings.TrimSpace(urlStr)
				summarize = true
			}
		}
	} else {
		// Simple parsing for non-quoted URLs
		parts := strings.SplitN(query, " ", 2)
		if len(parts) < 2 {
			return CreateErrorResponse("webpage2markdown_error", "Invalid webpage2markdown command format", requestID)
		}
		urlStr = parts[1]
		
		// Check if there's a summarize parameter
		if strings.Contains(urlStr, "summarize=true") {
			urlStr = strings.Replace(urlStr, "summarize=true", "", 1)
			urlStr = strings.TrimSpace(urlStr)
			summarize = true
		}
	}

	// Validate URL
	_, err := url.ParseRequestURI(urlStr)
	if err != nil {
		logger.Error("Invalid URL", err)
		return CreateErrorResponse("webpage2markdown_error", "Invalid URL: "+err.Error(), requestID)
	}

	// Convert the webpage to Markdown
	markdown, err := WebPage2Markdown(urlStr)
	if err != nil {
		logger.Error("Failed to convert webpage to Markdown", err)
		return CreateErrorResponse("webpage2markdown_error", err.Error(), requestID)
	}

	// Create a response with the Markdown content
	response := MCPResponse{
		RequestID: requestID,
		Context: map[string]interface{}{
			"markdown":  markdown,
			"url":       urlStr,
			"length":    len(markdown),
			"summarize": summarize,
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	// If summarize is requested, add a summary to the response
	if summarize {
		// For now, we'll create a simple summary by taking the first few paragraphs
		// In a real implementation, you might want to use an LLM or a more sophisticated summarization algorithm
		summary := createSimpleSummary(markdown)
		response.Context["summary"] = summary
	}

	// Marshal the response to JSON
	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal response to JSON", err)
		return CreateErrorResponse("internal_error", "Failed to create response", requestID)
	}

	return jsonResult, nil
}

// createSimpleSummary creates a simple summary of the markdown content
func createSimpleSummary(markdown string) string {
	// Split the markdown into paragraphs
	paragraphs := strings.Split(markdown, "\n\n")
	
	// Filter out paragraphs that are likely not content (links, headers, etc.)
	var contentParagraphs []string
	for _, p := range paragraphs {
		// Skip empty paragraphs
		if strings.TrimSpace(p) == "" {
			continue
		}
		
		// Skip paragraphs that are likely navigation or headers
		if strings.HasPrefix(p, "#") || strings.HasPrefix(p, "-") || strings.HasPrefix(p, "*") {
			continue
		}
		
		// Skip paragraphs that are mostly links
		if strings.Count(p, "](") > len(p)/20 {
			continue
		}
		
		contentParagraphs = append(contentParagraphs, p)
	}
	
	// Take the first few paragraphs as a summary
	var summaryParagraphs []string
	maxParagraphs := 3
	if len(contentParagraphs) < maxParagraphs {
		maxParagraphs = len(contentParagraphs)
	}
	
	for i := 0; i < maxParagraphs; i++ {
		summaryParagraphs = append(summaryParagraphs, contentParagraphs[i])
	}
	
	// Join the paragraphs and add an ellipsis if we truncated
	summary := strings.Join(summaryParagraphs, "\n\n")
	if len(contentParagraphs) > maxParagraphs {
		summary += "\n\n..."
	}
	
	return summary
}

// SaveWikipediaImage saves an image from Wikipedia to disk with the correct file extension
func SaveWikipediaImage(query string, imageSize int, outputPath string) error {
	// Trim leading and trailing spaces from the query
	query = strings.TrimSpace(query)

	// If no output path is provided, create one based on the query
	if outputPath == "" {
		// Replace spaces with underscores and remove special characters
		sanitizedQuery := strings.ReplaceAll(query, " ", "_")
		sanitizedQuery = regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(sanitizedQuery, "")
		outputPath = sanitizedQuery + ".jpg" // Default to jpg, will be updated based on content type
	} else {
		// Trim leading and trailing spaces from the output path
		outputPath = strings.TrimSpace(outputPath)
	}

	// Get the image data and content type
	imageData, contentType, err := WikipediaImageSearch(query, imageSize)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	// Determine the file extension based on content type
	extension := "jpg" // Default extension
	if strings.Contains(contentType, "png") {
		extension = "png"
	} else if strings.Contains(contentType, "gif") {
		extension = "gif"
	} else if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		extension = "jpg"
	} else if strings.Contains(contentType, "webp") {
		extension = "webp"
	}

	// If the output path doesn't have an extension, add one
	if !strings.Contains(filepath.Base(outputPath), ".") {
		outputPath = outputPath + "." + extension
	} else {
		// Replace the existing extension with the correct one
		outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "." + extension
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write the image data to disk
	err = os.WriteFile(outputPath, imageData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write image to disk: %w", err)
	}

	logger.Info("Image saved to", outputPath)
	return nil
}

// webPage2Markdown fetches the HTML content of a URL and converts it to Markdown
func WebPage2Markdown(urlStr string) (string, error) {
	// Get a custom HTTP client with Zscaler support
	client, err := GetCustomHTTPClient()
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create a request
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to make the request look more like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Make the HTTP request
	logger.Info("Fetching web page content from:", urlStr)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch web page: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 200 OK
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web page returned error status %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read web page content: %w", err)
	}

	// Convert HTML to Markdown
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	return markdown, nil
}

// ProcessWikipediaImageSaveRequest handles requests to save Wikipedia images to disk
func ProcessWikipediaImageSaveRequest(query string, requestID string) ([]byte, error) {
	// Extract the search query and output path
	// Format: wikipediaimagesave <query> [size] [output_path]

	// Handle quoted queries
	var searchQuery string
	var imageSize int = 500    // Default to 500px if not specified
	var outputPath string = "" // Default to empty string, will be generated based on query

	// Check if this is a command line invocation
	if strings.HasPrefix(query, "wikipediaimagesave ") && !strings.Contains(query, "\"") {
		// This is likely a command line invocation
		// Format: wikipediaimagesave Crab Nebula 800 /path/to/output.jpg

		// Remove the command prefix
		queryWithoutPrefix := strings.TrimPrefix(query, "wikipediaimagesave ")

		// Split the remaining string by spaces
		parts := strings.Fields(queryWithoutPrefix)

		if len(parts) < 1 {
			return CreateErrorResponse("wikipedia_image_save_error", "Missing search query", requestID)
		}

		// Determine which parts are the query, size, and path
		if len(parts) == 1 {
			// Only query provided
			searchQuery = parts[0]
		} else if len(parts) == 2 {
			// Could be "query1 query2" or "query size"
			// Try to parse the second part as a number
			var size int
			if _, err := fmt.Sscanf(parts[1], "%d", &size); err == nil {
				// Second part is a number, so it's the size
				searchQuery = parts[0]
				imageSize = size
			} else {
				// Second part is not a number, so it's part of the query
				searchQuery = parts[0] + " " + parts[1]
			}
		} else if len(parts) >= 3 {
			// Could be "query1 query2 size" or "query size path"
			// Try to parse the third part as a number
			var size int
			if _, err := fmt.Sscanf(parts[2], "%d", &size); err == nil {
				// Third part is a number, so it's the size
				searchQuery = parts[0] + " " + parts[1]
				imageSize = size

				// If there are more parts, they form the path
				if len(parts) > 3 {
					outputPath = strings.Join(parts[3:], " ")
				}
			} else {
				// Third part is not a number, so the second part might be the size
				if _, err := fmt.Sscanf(parts[1], "%d", &size); err == nil {
					// Second part is a number, so it's the size
					searchQuery = parts[0]
					imageSize = size

					// The rest is the path
					outputPath = strings.Join(parts[2:], " ")
				} else {
					// Neither second nor third part is a number, so assume the first two parts are the query
					searchQuery = parts[0] + " " + parts[1]

					// Try to parse the third part as a size
					if _, err := fmt.Sscanf(parts[2], "%d", &size); err == nil {
						imageSize = size

						// If there are more parts, they form the path
						if len(parts) > 3 {
							outputPath = strings.Join(parts[3:], " ")
						}
					} else {
						// Third part is not a size, so it must be part of the path
						outputPath = strings.Join(parts[2:], " ")
					}
				}
			}
		}
	} else if strings.Contains(query, "\"") {
		// Extract the quoted part
		startQuote := strings.Index(query, "\"")
		endQuote := strings.LastIndex(query, "\"")

		if startQuote != -1 && endQuote != -1 && startQuote != endQuote {
			searchQuery = query[startQuote+1 : endQuote]

			// Check if there are parameters after the quoted part
			if endQuote+1 < len(query) {
				// Split the remaining part by spaces
				params := strings.Fields(query[endQuote+1:])

				// First parameter might be the image size
				if len(params) > 0 {
					fmt.Sscanf(params[0], "%d", &imageSize)

					// Second parameter might be the output path
					if len(params) > 1 {
						// Join the remaining parameters as the output path
						outputPath = strings.Join(params[1:], " ")
					}
				}
			}
		} else {
			// Fallback to simple parsing
			parts := strings.SplitN(query, " ", 4)
			if len(parts) < 2 {
				return CreateErrorResponse("wikipedia_image_save_error", "Invalid wikipediaimagesave command format", requestID)
			}
			searchQuery = parts[1]

			// Check if image size is specified
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &imageSize)
			}

			// Check if output path is specified
			if len(parts) >= 4 {
				outputPath = parts[3]
			}
		}
	} else {
		// Simple parsing for non-quoted queries
		parts := strings.SplitN(query, " ", 4)
		if len(parts) < 2 {
			return CreateErrorResponse("wikipedia_image_save_error", "Invalid wikipediaimagesave command format", requestID)
		}
		searchQuery = parts[1]

		// Check if image size is specified
		if len(parts) >= 3 {
			fmt.Sscanf(parts[2], "%d", &imageSize)
		}

		// Check if output path is specified
		if len(parts) >= 4 {
			outputPath = parts[3]
		}
	}

	// Validate image size
	if imageSize <= 0 {
		imageSize = 500 // Reset to default if invalid
	}

	// Save the image to disk
	err := SaveWikipediaImage(searchQuery, imageSize, outputPath)
	if err != nil {
		logger.Error("Failed to save Wikipedia image", err)
		return CreateErrorResponse("wikipedia_image_save_error", err.Error(), requestID)
	}

	// Create a response with the image path
	response := MCPResponse{
		RequestID: requestID,
		Context: map[string]interface{}{
			"query":      searchQuery,
			"size":       imageSize,
			"outputPath": outputPath,
			"success":    true,
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	// Marshal the response to JSON
	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal response to JSON", err)
		return CreateErrorResponse("internal_error", "Failed to create response", requestID)
	}

	return jsonResult, nil
}
