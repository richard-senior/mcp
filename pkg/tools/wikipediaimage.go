package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
	"github.com/richard-senior/mcp/pkg/transport"
)

// WikipediaImageTool returns the Wikipedia image search tool definition
func WikipediaImageTool() protocol.Tool {
	return protocol.Tool{
		Name: "get_image",
		Description: `
		Finds an image (gif, jpeg etc.) that matches the given query string and downloads it to the given location at the given image size
		This tool should be used when the user asks for an image of something.
		Outputs the downloaded image location
		`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"query": {
					Type:        "string",
					Description: "The search string to be entered into google search",
				},
				"location": {
					Type: "string",
					Description: `
						the directory into which the image should be downloaded, defaults to the present working directory
					`,
				},
				"size": {
					Type:        "integer",
					Description: "The image width of the image to be downloaded, default is 500",
				},
			},
			Required: []string{"query"},
		},
	}
}

// HandleWikipediaImageTool handles the Wikipedia image save tool invocation
func HandleWikipediaImageTool(params any) (any, error) {
	logger.Info("Handling Wikipedia image save tool invocation")

	// Parse parameters
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	query, ok := paramsMap["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required and must be a string")
	}

	// Get image size (default to 500)
	imageSize := 500
	if sizeParam, ok := paramsMap["size"]; ok {
		if sizeFloat, ok := sizeParam.(float64); ok {
			imageSize = int(sizeFloat)
		}
	}

	// Get output path (default to empty string, will be generated based on query)
	outputPath := ""
	if pathParam, ok := paramsMap["output_path"]; ok {
		if pathStr, ok := pathParam.(string); ok {
			outputPath = pathStr
		}
	}

	// Save the image
	ret, err := SaveWikipediaImage(query, imageSize, outputPath)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// wikipediaImageSearch searches for an image on Wikipedia and returns the image bytes if found
func WikipediaImageSearch(query string, imageSize int) ([]byte, string, error) {
	// Default image size if not specified or invalid
	if imageSize <= 0 {
		imageSize = 500
	}

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

	logger.Info("Wikipedia returned nothing.. Calling Google Image Search")
	ret, err := GoogleSearch(query, 1, true)
	if err != nil || ret == nil {
		return nil, "No image found for any variation of query, and google search failed", err
	}

	// Just get the first image that is returned
	for _, i := range ret {
		if i.URL == "" {
			continue
		}
		ib, t, err := transport.GetImage(i.URL)
		if err != nil {
			continue
		}
		return ib, t, nil
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
	client, err := transport.GetCustomHTTPClient()
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

	imageData, contentType, err := transport.GetImage(imageURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch image: %w", err)
	}

	logger.Info("Successfully retrieved image for", query, "with size:", len(imageData), "bytes")

	return imageData, contentType, nil
}

// saveWikipediaImage saves an image from Wikipedia to disk with the correct file extension
func SaveWikipediaImage(query string, imageSize int, outputPath string) (any, error) {
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
		return nil, fmt.Errorf("failed to get image: %w", err)
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
	} else if strings.Contains(contentType, "svg") {
		extension = "svg"
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
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write the image data to disk
	err = os.WriteFile(outputPath, imageData, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write image to disk: %w", err)
	}

	logger.Info("Image saved to", outputPath)

	return map[string]any{
		"location": outputPath,
	}, nil
}
