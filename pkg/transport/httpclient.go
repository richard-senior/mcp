package transport

import (
	"compress/flate"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/richard-senior/mcp/internal/logger"
)

var httpClient *http.Client

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
func GetCustomHTTPClient() (*http.Client, error) {
	if httpClient != nil {
		return httpClient, nil
	}
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
		// CheckRedirect: nil means use default behavior (follow up to 10 redirects)
		// You can customize this if needed
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects (default behavior)
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
	httpClient = client
	return client, nil
}

// Attempts to get the bytes and filetype of an online image
func GetHtml(htmlUrl string) ([]byte, error) {

	// Get a custom HTTP client with Zscaler support
	client, err := GetCustomHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create a request for the image
	req, err := http.NewRequest("GET", htmlUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create image request: %w", err)
	}

	// Add headers to make the request look more like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Referer", "http://www.google.com/")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Make the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch html: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 200 OK
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request returned error status %d", resp.StatusCode)
	}

	// handle compression (Content-Encoding)
	var reader io.ReadCloser = resp.Body
	contentEncoding := resp.Header.Get("Content-Encoding")
	switch contentEncoding {
	case "gzip":
		logger.Info("Handling gzip compressed content")
		var err error
		reader, err = NewGzipReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()
	case "deflate":
		logger.Info("Handling deflate compressed content")
		var err error
		reader, err = NewDeflateReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create deflate reader: %w", err)
		}
		defer reader.Close()
	case "br":
		logger.Info("Handling brotli compressed content")
		var err error
		reader, err = NewBrotliReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create brotli reader: %w", err)
		}
		defer reader.Close()
	default:
		if contentEncoding != "" {
			logger.Warn("Unknown content encoding:", contentEncoding)
		}
	}

	// Read the decoded content from the appropriate reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	return data, nil
}

// NewGzipReader creates a gzip reader from the provided io.ReadCloser
func NewGzipReader(r io.ReadCloser) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

// NewDeflateReader creates a deflate reader from the provided io.ReadCloser
func NewDeflateReader(r io.ReadCloser) (io.ReadCloser, error) {
	return flate.NewReader(r), nil
}

// NewBrotliReader creates a brotli reader from the provided io.ReadCloser
func NewBrotliReader(r io.ReadCloser) (io.ReadCloser, error) {
	return io.NopCloser(brotli.NewReader(r)), nil
}

// Attempts to get the bytes and filetype of an online image
func GetImage(imageUrl string) ([]byte, string, error) {

	// Get a custom HTTP client with Zscaler support
	client, err := GetCustomHTTPClient()
	if err != nil {
		return nil, "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create a request for the image
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create image request: %w", err)
	}

	// Add headers to make the request look more like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Make the HTTP request for the image
	imgResp, err := client.Do(req)
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

	return imageData, contentType, nil
}
