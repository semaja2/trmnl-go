package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/semaja2/trmnl-go/api"
	"github.com/semaja2/trmnl-go/config"
	"github.com/semaja2/trmnl-go/logging"
	"github.com/semaja2/trmnl-go/metrics"
	"github.com/semaja2/trmnl-go/models"
	"github.com/semaja2/trmnl-go/render"
)

const Version = "1.6.0"

var (
	// Command-line flags
	apiKey       = flag.String("api-key", "", "TRMNL API key (for usetrmnl.com)")
	deviceID     = flag.String("device-id", "", "Device ID (for self-hosted servers)")
	macAddress   = flag.String("mac-address", "", "MAC address to use as Device ID (e.g. AA:BB:CC:DD:EE:FF)")
	netInterface = flag.String("interface", "", "Network interface for MAC address (e.g. en0, eth0)")
	baseURL      = flag.String("base-url", "", "Base URL for TRMNL API")
	model        = flag.String("model", "", "Device model (e.g., TRMNL, virtual-hd, virtual-fhd)")
	listModels   = flag.Bool("list-models", false, "List available device models")
	width        = flag.Int("width", 0, "Window width (overrides model default)")
	height       = flag.Int("height", 0, "Window height (overrides model default)")
	darkMode     = flag.Bool("dark", false, "Enable dark mode (invert colors)")
	alwaysOnTop  = flag.Bool("always-on-top", false, "Keep window always on top (macOS only)")
	fullscreen   = flag.Bool("fullscreen", false, "Enable fullscreen mode")
	rotation     = flag.Int("rotation", 0, "Rotate image (degrees: 0, 90, 180, 270, or -90)")
	mirrorMode   = flag.Bool("mirror", false, "Use mirror mode (show current screen, not device-specific)")
	setup        = flag.Bool("setup", false, "Run setup to retrieve API key via MAC address")
	useFyne          = flag.Bool("use-fyne", false, "Force use of Fyne GUI (default: native window on macOS)")
	verbose          = flag.Bool("verbose", false, "Enable verbose logging")
	logFlushInterval = flag.Int("log-flush-interval", 0, "How often to flush logs to API in seconds (default: 1800/30min, set 60 for dev)")
	showVersion      = flag.Bool("version", false, "Show version information")
	saveConfig       = flag.Bool("save", false, "Save current settings to config file")
)

// DisplayWindow interface for both Fyne and native windows
type DisplayWindow interface {
	Show()
	Close()
	SetOnClosed(func())
	SetOnRefresh(func())
	SetOnRotate(func())
	UpdateImage([]byte) error
	UpdateStatus(string)
	GetApp() interface{}
	SetMenuItemsEnabled(bool)
}

type App struct {
	config         *config.Config
	client         *api.Client
	window         DisplayWindow
	logger         *logging.Logger
	stopCh         chan struct{}
	doneCh         chan struct{}
	refreshCh      chan struct{}
	rotateCh       chan struct{}
	verbose        bool
	needsSetup     bool
	lastImageData  []byte // Store last fetched image for rotation without refresh
	isConnected    bool   // Track if we've successfully connected
}

func isRunningOnMacOS() bool {
	return runtime.GOOS == "darwin"
}

// generateRandomMAC generates a random MAC address
func generateRandomMAC() string {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		// Fallback to timestamp-based if random fails
		return fmt.Sprintf("02:00:00:%02X:%02X:%02X",
			byte(time.Now().Unix()>>16),
			byte(time.Now().Unix()>>8),
			byte(time.Now().Unix()))
	}
	// Set locally administered bit (bit 1 of first byte)
	buf[0] = (buf[0] | 0x02) & 0xFE
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
}

// runGUIApp starts the GUI application
func runGUIApp() {
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("trmnl-go version %s\n", Version)
		os.Exit(0)
	}

	// List models if requested
	if *listModels {
		fmt.Print(models.ListModels())
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override config with command-line flags
	if *apiKey != "" {
		cfg.APIKey = *apiKey
	}
	if *deviceID != "" {
		cfg.DeviceID = *deviceID
	}
	if *macAddress != "" {
		// MAC address flag overrides saved device ID and clears API key
		// This allows testing with the same MAC across platforms
		mac := strings.ToUpper(strings.TrimSpace(*macAddress))
		if len(mac) == 17 && (strings.Count(mac, ":") == 5 || strings.Count(mac, "-") == 5) {
			cfg.DeviceID = mac
			cfg.APIKey = "" // Clear API key to force re-registration
			if *verbose {
				log.Printf("Using manually specified MAC address: %s (API key cleared for re-registration)", cfg.DeviceID)
			}
		} else {
			log.Fatalf("Invalid MAC address format: %s (expected format: AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF)", *macAddress)
		}
	}
	if *baseURL != "" {
		cfg.BaseURL = *baseURL
	}

	// Handle model selection
	if *model != "" {
		cfg.Model = *model
	}

	// Apply model defaults if model is set
	if cfg.Model != "" {
		deviceModel, err := models.GetModel(cfg.Model)
		if err != nil {
			log.Fatalf("Invalid model: %v\nUse -list-models to see available models", err)
		}
		// Set model dimensions as defaults (can be overridden by width/height flags)
		if cfg.WindowWidth == config.DefaultWindowWidth {
			cfg.WindowWidth = deviceModel.Width
		}
		if cfg.WindowHeight == config.DefaultWindowHeight {
			cfg.WindowHeight = deviceModel.Height
		}
	}

	// Override dimensions with explicit width/height flags
	if *width > 0 {
		cfg.WindowWidth = *width
	}
	if *height > 0 {
		cfg.WindowHeight = *height
	}
	if *darkMode {
		cfg.DarkMode = true
	}
	if *alwaysOnTop {
		cfg.AlwaysOnTop = true
	}
	if *fullscreen {
		cfg.Fullscreen = true
	}
	if *rotation != 0 {
		// Normalize -90 to 270
		if *rotation == -90 {
			cfg.Rotation = 270
		} else {
			cfg.Rotation = *rotation
		}
	}
	if *mirrorMode {
		cfg.MirrorMode = true
	}
	if *verbose {
		cfg.Verbose = true
	}
	if *logFlushInterval > 0 {
		cfg.LogFlushInterval = *logFlushInterval
	}

	// Save config if requested
	if *saveConfig {
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Println("Configuration saved successfully")
		os.Exit(0)
	}

	// Auto-detect MAC address as Device ID if not set
	if cfg.DeviceID == "" && cfg.APIKey == "" {
		mac, err := metrics.GetMACAddressForInterface(*netInterface)
		if err != nil {
			log.Printf("Warning: Could not detect MAC address: %v", err)
			log.Println("Generating random MAC address instead")
			cfg.DeviceID = generateRandomMAC()
			if cfg.Verbose {
				log.Printf("Generated random MAC address: %s", cfg.DeviceID)
			}
		} else {
			cfg.DeviceID = mac
			if cfg.Verbose {
				ifaceName := metrics.GetPrimaryInterfaceName()
				if *netInterface != "" {
					ifaceName = *netInterface
				}
				log.Printf("Auto-detected Device ID from %s: %s", ifaceName, mac)
			}
		}
	}

	// Check if setup is needed (will be handled after GUI starts)
	needsSetup := cfg.APIKey == "" || *setup

	// Create application
	app := &App{
		config:     cfg,
		client:     api.NewClient(cfg, cfg.Verbose),
		logger:     logging.NewLogger(cfg.BaseURL, cfg.APIKey, cfg.Verbose),
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
		refreshCh:  make(chan struct{}, 1), // Buffered to avoid blocking
		rotateCh:   make(chan struct{}, 1), // Buffered to avoid blocking
		verbose:    cfg.Verbose,
		needsSetup: needsSetup,
	}

	// Log startup
	mac, _ := metrics.GetMACAddress()
	m := metrics.Collect()

	if app.verbose {
		if cfg.APIKey != "" {
			fmt.Println("[Logger] API logging enabled - logs will be sent to server")
			fmt.Printf("[Logger] Flush interval: %d seconds (%v)\n", cfg.LogFlushInterval, time.Duration(cfg.LogFlushInterval)*time.Second)
		} else {
			fmt.Println("[Logger] API logging disabled (no API key)")
		}
	}

	app.logger.Info("Application started", map[string]any{
		"version":    Version,
		"platform":   runtime.GOOS,
		"arch":       runtime.GOARCH,
		"device_id":  cfg.DeviceID,
		"model":      cfg.Model,
		"resolution": fmt.Sprintf("%dx%d", cfg.WindowWidth, cfg.WindowHeight),
		"mac":        mac,
		"battery":    m.BatteryVoltage,
		"wifi_rssi":  m.RSSI,
	})

	// Print startup info
	if app.verbose {
		fmt.Printf("=== TRMNL Virtual Display v%s ===\n", Version)
		fmt.Printf("Base URL: %s\n", cfg.BaseURL)
		if cfg.APIKey != "" {
			fmt.Printf("Auth: API Key (***%s)\n", cfg.APIKey[len(cfg.APIKey)-4:])
		} else {
			fmt.Printf("Auth: Device ID (%s)\n", cfg.DeviceID)
		}
		if cfg.FriendlyID != "" {
			fmt.Printf("Device Name: %s\n", cfg.FriendlyID)
		}

		// Show MAC address info
		ifaceName := metrics.GetPrimaryInterfaceName()
		if mac != "" {
			fmt.Printf("Network: %s (%s)\n", ifaceName, mac)
		}

		fmt.Printf("Window: %dx%d\n", cfg.WindowWidth, cfg.WindowHeight)
		fmt.Printf("Dark Mode: %v\n", cfg.DarkMode)
		fmt.Printf("Mirror Mode: %v\n", cfg.MirrorMode)
		batteryV := api.PercentageToVoltage(m.BatteryVoltage)
		fmt.Printf("System: Battery %.1f%% (%.2fV), WiFi %d dBm\n", m.BatteryVoltage, batteryV, m.RSSI)
		fmt.Println("=====================================")
	}

	// Create display window (platform-specific logic in app_darwin.go / app_other.go)
	app.window = createWindow(cfg, *useFyne, app.verbose)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Handle window close
	app.window.SetOnClosed(func() {
		if app.verbose {
			fmt.Println("[App] Window closed, shutting down...")
		}
		close(app.stopCh)
	})

	// Handle refresh shortcut (Cmd+R / Ctrl+R)
	app.window.SetOnRefresh(func() {
		if !app.isConnected {
			if app.verbose {
				fmt.Println("[App] Refresh ignored - not yet connected")
			}
			app.window.UpdateStatus("Please wait - connecting...")
			return
		}
		if app.verbose {
			fmt.Println("[App] Manual refresh triggered")
		}
		// Non-blocking send to refresh channel
		select {
		case app.refreshCh <- struct{}{}:
		default:
			// Channel full, refresh already pending
		}
	})

	// Handle rotate shortcut (Cmd+T / Ctrl+T)
	app.window.SetOnRotate(func() {
		if !app.isConnected {
			if app.verbose {
				fmt.Println("[App] Rotate ignored - not yet connected")
			}
			app.window.UpdateStatus("Please wait - connecting...")
			return
		}
		if app.verbose {
			fmt.Println("[App] Manual rotate triggered")
		}
		// Non-blocking send to rotate channel
		select {
		case app.rotateCh <- struct{}{}:
		default:
			// Channel full, rotate already pending
		}
	})

	// Disable menu items until connected
	app.window.SetMenuItemsEnabled(false)

	// Start refresh goroutine
	go app.refreshLoop()

	// Handle signals in goroutine
	go func() {
		<-sigCh
		if app.verbose {
			fmt.Println("[App] Signal received, shutting down...")
		}
		close(app.stopCh)
		app.window.Close()
	}()

	// Show window (blocks until window is closed)
	app.window.Show()

	// Wait for cleanup to complete
	<-app.doneCh

	if app.verbose {
		fmt.Println("[App] Shutdown complete")
	}
}

// refreshLoop continuously fetches and displays images
func (a *App) refreshLoop() {
	defer close(a.doneCh)

	// Wait for window to be ready (NSApp needs time to initialize)
	time.Sleep(500 * time.Millisecond)

	// Show startup screen
	a.showStartupScreen()

	// Keep startup screen visible for a moment
	time.Sleep(2 * time.Second)

	// Handle setup if needed
	if a.needsSetup {
		a.window.UpdateStatus("Registering device...")
		if a.verbose {
			fmt.Println("[App] Running device setup/registration...")
		}

		setupResp, err := a.client.FetchSetup(a.config.DeviceID)
		if err != nil {
			log.Printf("Setup failed: %v", err)
			a.logger.Error("Device setup failed", map[string]any{
				"error":     err.Error(),
				"device_id": a.config.DeviceID,
			})
			a.logger.FlushOnError()
			a.showErrorScreen("Registration Failed", fmt.Sprintf("Device: %s\nError: %v", a.config.DeviceID, err))
			a.window.UpdateStatus("Registration failed - see display for details")

			// Keep window open with error displayed
			// Wait for user to close window or signal
			<-a.stopCh
			if a.verbose {
				fmt.Println("[App] Shutdown after setup failure")
			}
			return
		}

		// Setup successful - update config
		a.config.APIKey = setupResp.APIKey
		a.config.FriendlyID = setupResp.FriendlyID

		// Save the updated config
		if err := a.config.Save(); err != nil {
			log.Printf("Warning: Could not save config: %v", err)
			a.logger.Warn("Failed to save config after setup", map[string]any{
				"error": err.Error(),
			})
		}

		// Update client with new API key
		a.client = api.NewClient(a.config, a.verbose)

		if a.verbose {
			fmt.Printf("[App] Setup successful! Device registered as: %s\n", a.config.FriendlyID)
		}

		a.logger.Info("Device setup successful", map[string]any{
			"friendly_id": a.config.FriendlyID,
			"device_id":   a.config.DeviceID,
		})

		a.window.UpdateStatus(fmt.Sprintf("Registered as %s", a.config.FriendlyID))
		time.Sleep(2 * time.Second) // Show success message briefly
	}

	// Initial status
	a.window.UpdateStatus("Connecting to TRMNL API...")

	// Fetch and display first image
	refreshRate := a.fetchAndDisplay()

	ticker := time.NewTicker(time.Duration(refreshRate) * time.Second)
	defer ticker.Stop()

	// Periodic log flush ticker (configurable, default 30 minutes)
	flushInterval := time.Duration(a.config.LogFlushInterval) * time.Second
	if a.verbose {
		fmt.Printf("[App] Log flush interval: %v\n", flushInterval)
	}
	logFlushTicker := time.NewTicker(flushInterval)
	defer logFlushTicker.Stop()

	for {
		select {
		case <-a.stopCh:
			if a.verbose {
				fmt.Println("[App] Refresh loop stopped")
			}
			// Flush any remaining logs before shutdown
			a.logger.Info("Application shutting down", map[string]any{
				"reason": "user_initiated",
			})
			if err := a.logger.Flush(); err != nil && a.verbose {
				fmt.Printf("[App] Failed to flush logs on shutdown: %v\n", err)
			}
			return

		case <-ticker.C:
			refreshRate = a.fetchAndDisplay()
			ticker.Reset(time.Duration(refreshRate) * time.Second)

		case <-a.refreshCh:
			// Manual refresh triggered by keyboard shortcut
			if a.verbose {
				fmt.Println("[App] Executing manual refresh...")
			}
			refreshRate = a.fetchAndDisplay()
			ticker.Reset(time.Duration(refreshRate) * time.Second)

		case <-a.rotateCh:
			// Manual rotate triggered by keyboard shortcut
			if a.verbose {
				fmt.Println("[App] Executing manual rotate...")
			}
			a.rotateDisplay()
			// Re-render current image with new rotation (don't fetch new image)
			a.reRenderCurrentImage()

		case <-logFlushTicker.C:
			// Periodically flush logs to API (successful operations)
			if err := a.logger.Flush(); err != nil && a.verbose {
				fmt.Printf("[App] Failed to flush logs: %v\n", err)
			}
		}
	}
}

// showStartupScreen displays a startup/splash screen
func (a *App) showStartupScreen() {
	if a.verbose {
		fmt.Println("[App] Showing startup screen...")
	}

	// Use configured Device ID (which may be manually specified MAC)
	mac := a.config.DeviceID
	if mac == "" {
		// Fallback to detecting MAC if not configured
		detectedMAC, err := metrics.GetMACAddress()
		if err != nil || detectedMAC == "" {
			mac = "Unknown"
		} else {
			mac = detectedMAC
		}
	}

	// Build message
	message := "Connecting..."
	if a.config.FriendlyID != "" {
		message = fmt.Sprintf("Device: %s\nMAC: %s", a.config.FriendlyID, mac)
	} else {
		message = fmt.Sprintf("MAC: %s", mac)
	}

	startupImg, err := render.GenerateStartupScreen(
		a.config.WindowWidth,
		a.config.WindowHeight,
		message,
	)
	if err != nil {
		log.Printf("Failed to generate startup screen: %v", err)
		return
	}

	if err := a.window.UpdateImage(startupImg); err != nil {
		log.Printf("Failed to display startup screen: %v", err)
	}
}

// showErrorScreen displays an error message on screen
func (a *App) showErrorScreen(title, message string) {
	if a.verbose {
		fmt.Printf("[App] Showing error screen: %s - %s\n", title, message)
	}

	errorImg, err := render.GenerateErrorScreen(
		a.config.WindowWidth,
		a.config.WindowHeight,
		title,
		message,
	)
	if err != nil {
		log.Printf("Failed to generate error screen: %v", err)
		return
	}

	if err := a.window.UpdateImage(errorImg); err != nil {
		log.Printf("Failed to display error screen: %v", err)
	}
}

// reRenderCurrentImage re-renders the last fetched image with current rotation/dark mode settings
func (a *App) reRenderCurrentImage() {
	if a.lastImageData == nil {
		if a.verbose {
			fmt.Println("[App] No image data to re-render")
		}
		return
	}

	if a.verbose {
		fmt.Println("[App] Re-rendering current image with new rotation...")
	}

	// Update display with stored image data (rotation/dark mode applied in UpdateImage)
	if err := a.window.UpdateImage(a.lastImageData); err != nil {
		log.Printf("Failed to re-render image: %v", err)
		a.window.UpdateStatus(fmt.Sprintf("Error re-rendering: %v", err))
	}
}

// rotateDisplay cycles through rotation angles (0 -> 90 -> 180 -> 270 -> 0)
func (a *App) rotateDisplay() {
	// Cycle through rotation angles
	switch a.config.Rotation {
	case 0:
		a.config.Rotation = 90
	case 90:
		a.config.Rotation = 180
	case 180:
		a.config.Rotation = 270
	case 270:
		a.config.Rotation = 0
	default:
		a.config.Rotation = 0
	}

	if a.verbose {
		fmt.Printf("[App] Rotation set to %d degrees\n", a.config.Rotation)
	}

	// Save the rotation to config
	if err := a.config.Save(); err != nil && a.verbose {
		fmt.Printf("[App] Warning: Failed to save rotation to config: %v\n", err)
	}

	a.logger.Info("Display rotation changed", map[string]any{
		"rotation": a.config.Rotation,
	})
}

// fetchAndDisplay fetches the current display and updates the window
// Returns the refresh rate for the next update
func (a *App) fetchAndDisplay() int {
	if a.verbose {
		if a.config.MirrorMode {
			fmt.Println("[App] Fetching current screen (mirror mode)...")
		} else {
			fmt.Println("[App] Fetching display...")
		}
	}

	// Fetch display info (use mirror mode if enabled)
	var termResp *api.TerminalResponse
	var err error

	if a.config.MirrorMode {
		termResp, err = a.client.FetchCurrentScreen()
	} else {
		termResp, err = a.client.FetchDisplay()
	}

	if err != nil {
		log.Printf("Failed to fetch display: %v", err)
		a.logger.Error("Failed to fetch display", map[string]any{
			"error":       err.Error(),
			"mirror_mode": a.config.MirrorMode,
		})
		a.logger.FlushOnError() // Send logs on error
		a.window.UpdateStatus(fmt.Sprintf("Error: %v", err))
		a.showErrorScreen("Connection Error", fmt.Sprintf("Failed to connect to server: %v", err))
		return 60 // Retry in 60 seconds
	}

	// Check for error response
	if termResp.Error != "" {
		log.Printf("API returned error: %s", termResp.Error)
		a.logger.Error("API error response", map[string]any{
			"error":  termResp.Error,
			"status": termResp.Status,
		})
		a.logger.FlushOnError() // Send logs on error
		a.window.UpdateStatus(fmt.Sprintf("API Error: %s", termResp.Error))
		a.showErrorScreen("API Error", termResp.Error)
		return 60 // Retry in 60 seconds
	}

	// Download image
	imageData, err := a.client.FetchImage(termResp.ImageURL)
	if err != nil {
		log.Printf("Failed to fetch image: %v", err)
		a.logger.Error("Failed to download image", map[string]any{
			"error":     err.Error(),
			"image_url": termResp.ImageURL,
		})
		a.logger.FlushOnError() // Send logs on error
		a.window.UpdateStatus(fmt.Sprintf("Error downloading image: %v", err))
		a.showErrorScreen("Download Error", fmt.Sprintf("Could not download image: %v", err))
		return termResp.RefreshRate
	}

	// Store image data for rotation without refresh
	a.lastImageData = imageData

	// Update display
	if err := a.window.UpdateImage(imageData); err != nil {
		log.Printf("Failed to update display: %v", err)
		a.logger.Error("Failed to render image", map[string]any{
			"error": err.Error(),
		})
		a.logger.FlushOnError() // Send logs on error
		a.window.UpdateStatus(fmt.Sprintf("Error displaying image: %v", err))
		a.showErrorScreen("Display Error", fmt.Sprintf("Could not render image: %v", err))
		return termResp.RefreshRate
	}

	// Mark as connected after first successful display update
	if !a.isConnected {
		a.isConnected = true
		// Enable menu items now that we're connected
		a.window.SetMenuItemsEnabled(true)
		if a.verbose {
			fmt.Println("[App] Successfully connected - shortcuts now enabled")
		}
	}

	// Update status
	nextUpdate := time.Now().Add(time.Duration(termResp.RefreshRate) * time.Second)
	statusMsg := fmt.Sprintf("Last updated: %s | Next: %s",
		time.Now().Format("15:04:05"),
		nextUpdate.Format("15:04:05"))

	if a.config.MirrorMode {
		statusMsg = "[Mirror] " + statusMsg
	}

	a.window.UpdateStatus(statusMsg)

	if a.verbose {
		fmt.Printf("[App] Display updated. Next refresh in %d seconds\n", termResp.RefreshRate)
	}

	// Log successful update (will be buffered and sent periodically or on error)
	a.logger.Info("Display updated successfully", map[string]any{
		"filename":     termResp.Filename,
		"refresh_rate": termResp.RefreshRate,
		"mirror_mode":  a.config.MirrorMode,
		"status":       termResp.Status,
	})

	return termResp.RefreshRate
}
