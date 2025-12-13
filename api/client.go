package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/semaja2/trmnl-go/config"
	"github.com/semaja2/trmnl-go/metrics"
)

const (
	DisplayEndpoint       = "/api/display"
	SetupEndpoint         = "/api/setup"
	CurrentScreenEndpoint = "/api/current_screen"
	ModelsEndpoint        = "/api/models"
	UserAgent             = "trmnl-go-virtual/1.0.0"
	FirmwareVersion       = "1.6.9"
	DefaultTimeout        = 30 * time.Second
	DefaultDeviceModel    = "virtual"
	MinBatteryVoltage     = 3.0
	MaxBatteryVoltage     = 4.08
)

// SetupResponse represents the response from /api/setup
type SetupResponse struct {
	Status     int    `json:"status,omitempty"`
	APIKey     string `json:"api_key,omitempty"`
	ImageURL   string `json:"image_url,omitempty"`
	Message    string `json:"message,omitempty"`
	FriendlyID string `json:"friendly_id,omitempty"`
}

// TerminalResponse represents the response from the TRMNL API
type TerminalResponse struct {
	ImageURL    string `json:"image_url"`
	Filename    string `json:"filename"`
	RefreshRate int    `json:"refresh_rate"` // in seconds
	Status      int    `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
}

// DeviceModel represents a TRMNL device model from the API
type DeviceModel struct {
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Colors      int     `json:"colors"`
	BitDepth    int     `json:"bit_depth"`
	ScaleFactor float64 `json:"scale_factor"`
	Rotation    int     `json:"rotation"`
	MimeType    string  `json:"mime_type"`
	OffsetX     int     `json:"offset_x"`
	OffsetY     int     `json:"offset_y"`
	PublishedAt string  `json:"published_at"`
}

// ModelsResponse represents the response from /api/models
type ModelsResponse struct {
	Data []DeviceModel `json:"data"`
}

// Client handles communication with the TRMNL API
type Client struct {
	config      *config.Config
	httpClient  *http.Client
	verbose     bool
	refreshRate int // Last known refresh rate
}

// PercentageToVoltage converts battery percentage (0-100) to voltage (3.0-4.08V)
// using a realistic Li-ion battery discharge curve
func PercentageToVoltage(percentage float64) float64 {
	// Clamp edge cases
	if percentage >= 100 {
		return MaxBatteryVoltage // 4.08V = full charge
	}
	if percentage >= 95 {
		return 4.06 // midpoint of 95-100% block
	}
	if percentage >= 90 {
		return 4.02 // midpoint of 90-95% block
	}
	if percentage <= 1 {
		return MinBatteryVoltage // 3.0V = empty
	}

	// Linear band: 1-83%
	// Formula: V = 3.0 + pct * 0.012
	if percentage > 1 && percentage <= 83 {
		return MinBatteryVoltage + percentage*0.012
	}

	// 83-90% band
	if percentage > 83 && percentage < 90 {
		return 3.996 + (percentage-83)*0.003429 // ~4.02V at 90%
	}

	return MinBatteryVoltage
}

// NewClient creates a new TRMNL API client
func NewClient(cfg *config.Config, verbose bool) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		verbose:     verbose,
		refreshRate: 60, // Default refresh rate
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

	// Set authentication headers (matches firmware exactly)
	authHeader, authValue := c.config.GetAuthHeader()
	req.Header.Set(authHeader, authValue)

	// Set device metrics headers
	systemMetrics := metrics.Collect()
	batteryPercent := systemMetrics.BatteryVoltage // This is actually percentage (0-100)
	batteryVoltage := PercentageToVoltage(batteryPercent)

	req.Header.Set("percent_charged", fmt.Sprintf("%.2f", batteryPercent))
	req.Header.Set("Battery-Voltage", fmt.Sprintf("%.2f", batteryVoltage))
	req.Header.Set("RSSI", fmt.Sprintf("%d", systemMetrics.RSSI))

	// Set firmware/version info
	req.Header.Set("FW-Version", FirmwareVersion)

	// Use configured model name if set, otherwise use default
	modelName := c.config.Model
	if modelName == "" {
		modelName = DefaultDeviceModel
	}
	req.Header.Set("Model", modelName)

	// Set display dimensions (matches firmware Width/Height headers)
	req.Header.Set("Width", fmt.Sprintf("%d", c.config.WindowWidth))
	req.Header.Set("Height", fmt.Sprintf("%d", c.config.WindowHeight))

	// Set current refresh rate
	req.Header.Set("Refresh-Rate", fmt.Sprintf("%d", c.refreshRate))

	// Set content type
	req.Header.Set("Content-Type", "application/json")

	if c.verbose {
		if authHeader == "access-token" {
			fmt.Printf("[API] Access-Token: %s\n", authValue)
		} else {
			fmt.Printf("[API] ID: %s\n", authValue)
		}
		fmt.Printf("[API] Battery: %.2f%% (%.2fV), RSSI: %d dBm\n", batteryPercent, batteryVoltage, systemMetrics.RSSI)
		fmt.Printf("[API] Model: %s, FW-Version: %s\n", modelName, FirmwareVersion)
		fmt.Printf("[API] Dimensions: %dx%d, Refresh-Rate: %d\n", c.config.WindowWidth, c.config.WindowHeight, c.refreshRate)
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

	// Save refresh rate for next request
	c.refreshRate = termResp.RefreshRate

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

// FetchSetup performs device registration/setup using MAC address
// Returns API key, friendly ID, and initial image URL
func (c *Client) FetchSetup(macAddress string) (*SetupResponse, error) {
	url := c.config.BaseURL + SetupEndpoint

	if c.verbose {
		fmt.Printf("[API] Fetching setup from: %s\n", url)
		fmt.Printf("[API] Device MAC: %s\n", macAddress)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create setup request: %w", err)
	}

	// Send MAC address in ID header
	req.Header.Set("ID", macAddress)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("setup request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("setup API returned status %d: %s", resp.StatusCode, string(body))
	}

	var setupResp SetupResponse
	if err := json.NewDecoder(resp.Body).Decode(&setupResp); err != nil {
		return nil, fmt.Errorf("failed to decode setup response: %w", err)
	}

	// Check application-level status
	if setupResp.Status != 0 && setupResp.Status != 200 {
		return nil, fmt.Errorf("setup failed: %s (status %d)", setupResp.Message, setupResp.Status)
	}

	if c.verbose {
		fmt.Printf("[API] Setup successful: APIKey=%s, FriendlyID=%s\n",
			setupResp.APIKey, setupResp.FriendlyID)
	}

	return &setupResp, nil
}

// FetchCurrentScreen retrieves the current screen for mirror mode
func (c *Client) FetchCurrentScreen() (*TerminalResponse, error) {
	url := c.config.BaseURL + CurrentScreenEndpoint

	if c.verbose {
		fmt.Printf("[API] Fetching current screen (mirror mode) from: %s\n", url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create current screen request: %w", err)
	}

	// Set authentication headers
	authHeader, authValue := c.config.GetAuthHeader()
	req.Header.Set(authHeader, authValue)

	// Set device metrics headers (same as display)
	systemMetrics := metrics.Collect()
	batteryPercent := systemMetrics.BatteryVoltage
	batteryVoltage := PercentageToVoltage(batteryPercent)

	req.Header.Set("percent_charged", fmt.Sprintf("%.2f", batteryPercent))
	req.Header.Set("Battery-Voltage", fmt.Sprintf("%.2f", batteryVoltage))
	req.Header.Set("RSSI", fmt.Sprintf("%d", systemMetrics.RSSI))
	req.Header.Set("FW-Version", FirmwareVersion)

	modelName := c.config.Model
	if modelName == "" {
		modelName = DefaultDeviceModel
	}
	req.Header.Set("Model", modelName)
	req.Header.Set("Width", fmt.Sprintf("%d", c.config.WindowWidth))
	req.Header.Set("Height", fmt.Sprintf("%d", c.config.WindowHeight))
	req.Header.Set("Refresh-Rate", fmt.Sprintf("%d", c.refreshRate))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("current screen request failed: %w", err)
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
		fmt.Printf("[API] Mirror response: ImageURL=%s, Filename=%s, RefreshRate=%ds\n",
			termResp.ImageURL, termResp.Filename, termResp.RefreshRate)
	}

	if termResp.RefreshRate == 0 {
		termResp.RefreshRate = 60
	}

	c.refreshRate = termResp.RefreshRate

	return &termResp, nil
}