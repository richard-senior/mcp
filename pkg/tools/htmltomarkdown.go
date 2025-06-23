package tools

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
	"github.com/richard-senior/mcp/pkg/transport"
)

func HTMLToMarkdownTool() protocol.Tool {
	return protocol.Tool{
		Name: "html_2_markdown",
		Description: `
		Assumes that the url returns HTML content and converts it to Markdown format for comsumption by LLM clients.
		This allows the content to be more easily consumed by LLMs.
		This tool should be used when:
		- More information about a previous use of the google_search tool is required
		- The user asks for a Precis or summary of the content of a web page
		etc.
		`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"url": {
					Type:        "string",
					Description: "The URL of of the html to convert to markdown ie. https://www.richardsenior.net/",
				},
			},
			Required: []string{"url"},
		},
	}
}

// ConvertURLToMarkdown converts HTML content from a URL to markdown
func HandleURLToMarkdown(params any) (any, error) {
	// Parse parameters
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	url, ok := paramsMap["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("no url was passed")
	}
	// Get a custom HTTP client with Zscaler support
	client, err := transport.GetCustomHTTPClient()
	if err != nil {
		return nil, err
	}

	// Create a request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers to make the request look more like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")

	// Make the HTTP request
	logger.Info("Getting HTML from:", url)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body:", err)
		return nil, err
	}

	// Get base URL for converting relative links to absolute
	// Extract domain from URL
	domain, err := extractDomain(url)

	if err != nil {
		logger.Warn("Failed to extract domain from URL:", err)
		domain = "unknown"
	}

	markdown, err := htmltomarkdown.ConvertString(
		string(body),
		converter.WithDomain(domain),
	)
	if err != nil {
		logger.Error("Failed to convert HTML to Markdown:", err)
		return nil, err
	}

	// Limit the size of the markdown if it's too large
	const maxLength = 10000
	if len(markdown) > maxLength {
		markdown = markdown[:maxLength] + "\n\n... (content truncated due to size)"
	}

	return map[string]any{
		"markdown": markdown,
		"url":      url,
		"title":    extractTitle(string(body)),
		"domain":   domain,
	}, nil
}

// extractTitle attempts to extract the title from HTML content
func extractTitle(html string) string {
	titleStart := strings.Index(html, "<title>")
	if titleStart == -1 {
		return "No title found"
	}

	titleStart += 7 // Length of "<title>"
	titleEnd := strings.Index(html[titleStart:], "</title>")
	if titleEnd == -1 {
		return "No title found"
	}

	return strings.TrimSpace(html[titleStart : titleStart+titleEnd])
}

// extractDomain extracts the domain portion from a URL string
func extractDomain(urlString string) (string, error) {
	// Add http:// prefix if not present for proper URL parsing
	if !strings.HasPrefix(urlString, "http://") && !strings.HasPrefix(urlString, "https://") {
		urlString = "https://" + urlString
	}

	// Parse the URL
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %v", err)
	}

	if strings.HasPrefix(urlString, "http://") {
		return "http://" + parsedURL.Hostname(), nil
	} else {
		return "https://" + parsedURL.Hostname(), nil
	}
}
