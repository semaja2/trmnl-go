package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"crypto/rand"

	"github.com/semaja2/trmnl-go/api"
	"github.com/semaja2/trmnl-go/config"
	"github.com/semaja2/trmnl-go/display"
	"github.com/semaja2/trmnl-go/metrics"
	"github.com/semaja2/trmnl-go/models"
	"github.com/semaja2/trmnl-go/render"
)

const Version = "1.0.0"

var (
	// Command-line flags
	apiKey       = flag.String("api-key", "", "TRMNL API key (for usetrmnl.com)")
	deviceID     = flag.String("device-id", "", "Device ID (for self-hosted servers)")
	netInterface = flag.String("interface", "", "Network interface for MAC address (e.g. en0, eth0)")
	baseURL      = flag.String("base-url", "", "Base URL for TRMNL API")
	model        = flag.String("model", "", "Device model (e.g., TRMNL, virtual-hd, virtual-fhd)")
	listModels   = flag.Bool("list-models", false, "List available device models")
	width        = flag.Int("width", 0, "Window width (overrides model default)")
	height       = flag.Int("height", 0, "Window height (overrides model default)")
	darkMode     = flag.Bool("dark", false, "Enable dark mode (invert colors)")
	alwaysOnTop  = flag.Bool("always-on-top", false, "Keep window always on top (macOS only)")
	mirrorMode   = flag.Bool("mirror", false, "Use mirror mode (show current screen, not device-specific)")
	setup        = flag.Bool("setup", false, "Run setup to retrieve API key via MAC address")
	useFyne      = flag.Bool("use-fyne", false, "Force use of Fyne GUI (default: native window on macOS)")
	verbose      = flag.Bool("verbose", false, "Enable verbose logging")
	showVersion  = flag.Bool("version", false, "Show version information")
	saveConfig   = flag.Bool("save", false, "Save current settings to config file")
)

// DisplayWindow interface for both Fyne and native windows
type DisplayWindow interface {
	Show()
	Close()
	SetOnClosed(func())
	UpdateImage([]byte) error
	UpdateStatus(string)
	GetApp() interface{}
}

type App struct {
	config  *config.Config
	client  *api.Client
	window  DisplayWindow
	stopCh  chan struct{}
	doneCh  chan struct{}
	verbose bool
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
	if *mirrorMode {
		cfg.MirrorMode = true
	}
	if *verbose {
		cfg.Verbose = true
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

	// Auto-setup if no credentials are configured
	needsSetup := cfg.APIKey == "" && *setup == false
	if needsSetup {
		fmt.Println("No API key found. Running automatic setup...")
		*setup = true
	}

	// Handle setup mode - retrieve API key via MAC address
	if *setup {
		if cfg.DeviceID == "" {
			log.Fatal("Cannot run setup: Device ID (MAC address) is required")
		}

		client := api.NewClient(cfg, cfg.Verbose)
		fmt.Printf("Running setup for device: %s\n", cfg.DeviceID)
		fmt.Printf("Contacting server: %s\n", cfg.BaseURL)

		setupResp, err := client.FetchSetup(cfg.DeviceID)
		if err != nil {
			log.Fatalf("Setup failed: %v\n\nIf you're using a self-hosted server, use:\n  ./trmnl-go -api-key YOUR_KEY\nor:\n  ./trmnl-go -device-id YOUR_DEVICE_ID -base-url https://your-server.com", err)
		}

		cfg.APIKey = setupResp.APIKey
		cfg.FriendlyID = setupResp.FriendlyID

		fmt.Printf("\n✓ Setup successful!\n")
		fmt.Printf("API Key: %s\n", cfg.APIKey)
		if cfg.FriendlyID != "" {
			fmt.Printf("Friendly ID: %s\n", cfg.FriendlyID)
		}

		// Save configuration
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save configuration: %v", err)
		}
		fmt.Println("\nConfiguration saved successfully!")

		// If this was automatic setup, continue running instead of exiting
		if needsSetup {
			fmt.Println("Starting application...")
		} else {
			fmt.Println("You can now run the application without the -setup flag.")
			os.Exit(0)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create application
	app := &App{
		config:  cfg,
		client:  api.NewClient(cfg, cfg.Verbose),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		verbose: cfg.Verbose,
	}

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
		mac, _ := metrics.GetMACAddress()
		ifaceName := metrics.GetPrimaryInterfaceName()
		if mac != "" {
			fmt.Printf("Network: %s (%s)\n", ifaceName, mac)
		}

		fmt.Printf("Window: %dx%d\n", cfg.WindowWidth, cfg.WindowHeight)
		fmt.Printf("Dark Mode: %v\n", cfg.DarkMode)
		fmt.Printf("Mirror Mode: %v\n", cfg.MirrorMode)
		m := metrics.Collect()
		batteryV := api.PercentageToVoltage(m.BatteryVoltage)
		fmt.Printf("System: Battery %.1f%% (%.2fV), WiFi %d dBm\n", m.BatteryVoltage, batteryV, m.RSSI)
		fmt.Println("=====================================")
	}

	// Create display window
	// Use native window on macOS by default (unless -use-fyne flag is set)
	if isRunningOnMacOS() && !*useFyne {
		if app.verbose {
			fmt.Println("[App] Using native macOS window")
		}
		app.window = display.NewNativeWindow(cfg, app.verbose)
	} else {
		if app.verbose && isRunningOnMacOS() {
			fmt.Println("[App] Using Fyne window (forced via -use-fyne flag)")
		}
		app.window = display.NewWindow(cfg, app.verbose)
	}

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

	// Initial status
	a.window.UpdateStatus("Connecting to TRMNL API...")

	// Fetch and display first image
	refreshRate := a.fetchAndDisplay()

	ticker := time.NewTicker(time.Duration(refreshRate) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			if a.verbose {
				fmt.Println("[App] Refresh loop stopped")
			}
			return

		case <-ticker.C:
			refreshRate = a.fetchAndDisplay()
			ticker.Reset(time.Duration(refreshRate) * time.Second)
		}
	}
}

// showStartupScreen displays a startup/splash screen
func (a *App) showStartupScreen() {
	if a.verbose {
		fmt.Println("[App] Showing startup screen...")
	}

	// Get MAC address
	mac, err := metrics.GetMACAddress()
	if err != nil || mac == "" {
		mac = "Unknown"
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
		a.window.UpdateStatus(fmt.Sprintf("Error: %v", err))
		a.showErrorScreen("Connection Error", fmt.Sprintf("Failed to connect to server: %v", err))
		return 60 // Retry in 60 seconds
	}

	// Check for error response
	if termResp.Error != "" {
		log.Printf("API returned error: %s", termResp.Error)
		a.window.UpdateStatus(fmt.Sprintf("API Error: %s", termResp.Error))
		a.showErrorScreen("API Error", termResp.Error)
		return 60 // Retry in 60 seconds
	}

	// Download image
	imageData, err := a.client.FetchImage(termResp.ImageURL)
	if err != nil {
		log.Printf("Failed to fetch image: %v", err)
		a.window.UpdateStatus(fmt.Sprintf("Error downloading image: %v", err))
		a.showErrorScreen("Download Error", fmt.Sprintf("Could not download image: %v", err))
		return termResp.RefreshRate
	}

	// Update display
	if err := a.window.UpdateImage(imageData); err != nil {
		log.Printf("Failed to update display: %v", err)
		a.window.UpdateStatus(fmt.Sprintf("Error displaying image: %v", err))
		a.showErrorScreen("Display Error", fmt.Sprintf("Could not render image: %v", err))
		return termResp.RefreshRate
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

	return termResp.RefreshRate
}
