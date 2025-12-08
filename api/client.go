package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/semaja2/trmnl-go/config"
	"github.com/semaja2/trmnl-go/metrics"
)

const (
	DisplayEndpoint = "/api/display"
	UserAgent       = "trmnl-go-virtual/1.0.0"
	DefaultTimeout  = 30 * time.Second
)

// TerminalResponse represents the response from the TRMNL API
type TerminalResponse struct {
	ImageURL    string `json:"image_url"`
	Filename    string `json:"filename"`
	RefreshRate int    `json:"refresh_rate"` // in seconds
}

// Client handles communication with the TRMNL API
type Client struct {
	config     *config.Config
	httpClient *http.Client
	verbose    bool
}

// NewClient creates a new TRMNL API client
func NewClient(cfg *config.Config, verbose bool) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		verbose: verbose,
	}
}

// FetchDisplay retrieves the current display information from the API
func (c *Client) FetchDisplay() (*TerminalResponse, error) {
	url := c.config.BaseURL + DisplayEndpoint

	if c.verbose {
		fmt.Printf("[API] Fetching display from: %s\n", url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header
	authHeader, authValue := c.config.GetAuthHeader()
	req.Header.Set(authHeader, authValue)

	// Set device metrics headers
	systemMetrics := metrics.Collect()
	req.Header.Set("battery-voltage", fmt.Sprintf("%.2f", systemMetrics.BatteryVoltage))
	req.Header.Set("rssi", fmt.Sprintf("%d", systemMetrics.RSSI))
	req.Header.Set("User-Agent", UserAgent)

	// Add additional system info
	req.Header.Set("X-Device-Type", "virtual")
	req.Header.Set("X-OS", runtime.GOOS)
	req.Header.Set("X-Arch", runtime.GOARCH)

	if c.verbose {
		fmt.Printf("[API] Auth: %s=%s\n", authHeader, authValue)
		fmt.Printf("[API] Battery: %.2f, RSSI: %d\n", systemMetrics.BatteryVoltage, systemMetrics.RSSI)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var termResp TerminalResponse
	if err := json.NewDecoder(resp.Body).Decode(&termResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if c.verbose {
		fmt.Printf("[API] Response: ImageURL=%s, Filename=%s, RefreshRate=%ds\n",
			termResp.ImageURL, termResp.Filename, termResp.RefreshRate)
	}

	// Default refresh rate if not provided
	if termResp.RefreshRate == 0 {
		termResp.RefreshRate = 60
	}

	return &termResp, nil
}

// FetchImage downloads the image from the provided URL
func (c *Client) FetchImage(imageURL string) ([]byte, error) {
	if c.verbose {
		fmt.Printf("[API] Downloading image: %s\n", imageURL)
	}

	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create image request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("image download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	if c.verbose {
		fmt.Printf("[API] Downloaded %d bytes\n", len(data))
	}

	return data, nil
}
