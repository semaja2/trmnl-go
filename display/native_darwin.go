//go:build darwin

package display

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework AppKit
#import <Cocoa/Cocoa.h>
#import <AppKit/AppKit.h>

// Global window reference
static NSWindow* mainWindow = nil;
static NSImageView* imageView = nil;
static volatile bool refreshRequested = false;
static volatile bool rotateRequested = false;

// Menu item references for enabling/disabling
static NSMenuItem* refreshMenuItem = nil;
static NSMenuItem* rotateMenuItem = nil;

// Window delegate to handle close events and menu actions
@interface WindowDelegate : NSObject <NSWindowDelegate>
- (void)refreshAction:(id)sender;
- (void)rotateAction:(id)sender;
@end

@implementation WindowDelegate
- (void)windowWillClose:(NSNotification *)notification {
    [NSApp terminate:nil];
}

- (void)refreshAction:(id)sender {
    refreshRequested = true;
}

- (void)rotateAction:(id)sender {
    rotateRequested = true;
}
@end

static WindowDelegate* windowDelegate = nil;

void* createFloatingWindow(int width, int height, bool alwaysOnTop, bool fullscreen) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindowStyleMask styleMask = NSWindowStyleMaskTitled |
                                      NSWindowStyleMaskClosable |
                                      NSWindowStyleMaskMiniaturizable |
                                      NSWindowStyleMaskResizable;

        NSRect frame = NSMakeRect(100, 100, width, height);
        mainWindow = [[NSWindow alloc] initWithContentRect:frame
                                                 styleMask:styleMask
                                                   backing:NSBackingStoreBuffered
                                                     defer:NO];

        [mainWindow setTitle:@"TRMNL Display"];
        [mainWindow setBackgroundColor:[NSColor blackColor]];

        // Set delegate to handle window close
        windowDelegate = [[WindowDelegate alloc] init];
        [mainWindow setDelegate:windowDelegate];

        // Disable automatic window tabbing (removes "Show Tab Bar" menu)
        if ([NSWindow respondsToSelector:@selector(setAllowsAutomaticWindowTabbing:)]) {
            [NSWindow setAllowsAutomaticWindowTabbing:NO];
        }

        // Always enable fullscreen support, disable tabbing
        [mainWindow setCollectionBehavior:NSWindowCollectionBehaviorFullScreenPrimary];

        if (alwaysOnTop) {
            [mainWindow setLevel:NSFloatingWindowLevel];
            // Note: When always-on-top is enabled, fullscreen may not work as expected
            // due to window level conflicts
        }

        imageView = [[NSImageView alloc] initWithFrame:frame];
        [imageView setImageScaling:NSImageScaleProportionallyUpOrDown];
        [mainWindow setContentView:imageView];

        [mainWindow makeKeyAndOrderFront:nil];
        [mainWindow center];
        [NSApp activateIgnoringOtherApps:YES];

        // Enter fullscreen if requested (with delay to ensure window is ready)
        if (fullscreen) {
            dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.5 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
                [mainWindow toggleFullScreen:nil];
            });
        }
    });

    return (__bridge void*)mainWindow;
}

void updateWindowImage(unsigned char* imageData, int length) {
    if (!imageView) return;

    dispatch_async(dispatch_get_main_queue(), ^{
        NSData* data = [NSData dataWithBytes:imageData length:length];
        NSImage* image = [[NSImage alloc] initWithData:data];
        if (image) {
            [imageView setImage:image];
        }
    });
}

void setupMenuBar() {
    // Create main menu bar
    NSMenu* mainMenu = [[NSMenu alloc] init];

    // App menu
    NSMenu* appMenu = [[NSMenu alloc] init];
    NSMenuItem* appMenuItem = [[NSMenuItem alloc] init];
    [appMenuItem setSubmenu:appMenu];

    // Quit menu item
    NSString* quitTitle = @"Quit TRMNL";
    NSMenuItem* quitItem = [[NSMenuItem alloc] initWithTitle:quitTitle
                                                      action:@selector(terminate:)
                                               keyEquivalent:@"q"];
    [appMenu addItem:quitItem];

    // Add app menu to main menu
    [mainMenu addItem:appMenuItem];

    // View menu
    NSMenu* viewMenu = [[NSMenu alloc] initWithTitle:@"View"];
    NSMenuItem* viewMenuItem = [[NSMenuItem alloc] init];
    [viewMenuItem setSubmenu:viewMenu];

    // Refresh menu item (Cmd+R) - initially disabled
    refreshMenuItem = [[NSMenuItem alloc] initWithTitle:@"Refresh"
                                                 action:@selector(refreshAction:)
                                          keyEquivalent:@"r"];
    [refreshMenuItem setTarget:windowDelegate];
    [refreshMenuItem setEnabled:NO]; // Disabled until connected
    [viewMenu addItem:refreshMenuItem];

    // Rotate menu item (Cmd+T) - initially disabled
    rotateMenuItem = [[NSMenuItem alloc] initWithTitle:@"Rotate Display"
                                                action:@selector(rotateAction:)
                                         keyEquivalent:@"t"];
    [rotateMenuItem setTarget:windowDelegate];
    [rotateMenuItem setEnabled:NO]; // Disabled until connected
    [viewMenu addItem:rotateMenuItem];

    [viewMenu addItem:[NSMenuItem separatorItem]]; // Separator

    // Add Enter/Exit fullscreen menu item
    NSMenuItem* fullscreenItem = [[NSMenuItem alloc] initWithTitle:@"Toggle Full Screen"
                                                            action:@selector(toggleFullScreen:)
                                                     keyEquivalent:@"f"];
    [fullscreenItem setKeyEquivalentModifierMask:NSEventModifierFlagCommand | NSEventModifierFlagControl];
    [viewMenu addItem:fullscreenItem];

    // Add view menu to main menu
    [mainMenu addItem:viewMenuItem];

    // Set the menu bar
    [NSApp setMainMenu:mainMenu];
}

void runNativeApp() {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];

    // Setup menu bar for Cmd+Q support
    setupMenuBar();

    [NSApp run];
}

void stopNativeApp() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
}

// Check if refresh was requested and clear the flag
bool checkAndClearRefreshRequested() {
    if (refreshRequested) {
        refreshRequested = false;
        return true;
    }
    return false;
}

// Check if rotate was requested and clear the flag
bool checkAndClearRotateRequested() {
    if (rotateRequested) {
        rotateRequested = false;
        return true;
    }
    return false;
}

// Enable or disable the action menu items (for connection state)
void setMenuItemsEnabled(bool enabled) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (refreshMenuItem) {
            [refreshMenuItem setEnabled:enabled];
        }
        if (rotateMenuItem) {
            [rotateMenuItem setEnabled:enabled];
        }
    });
}
*/
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"time"
	"unsafe"

	"github.com/semaja2/trmnl-go/config"
)

// NativeWindow represents a native macOS window
type NativeWindow struct {
	windowPtr       unsafe.Pointer
	config          *config.Config
	verbose         bool
	refreshCallback func()
	rotateCallback  func()
}

// NewNativeWindow creates a native macOS window
func NewNativeWindow(cfg *config.Config, verbose bool) *NativeWindow {
	w := &NativeWindow{
		config:  cfg,
		verbose: verbose,
	}

	// Create the window on the main thread
	w.windowPtr = C.createFloatingWindow(
		C.int(cfg.WindowWidth),
		C.int(cfg.WindowHeight),
		C.bool(cfg.AlwaysOnTop),
		C.bool(cfg.Fullscreen),
	)

	return w
}

// Show starts the native app event loop
func (w *NativeWindow) Show() {
	C.runNativeApp()
}

// UpdateImage updates the displayed image
func (w *NativeWindow) UpdateImage(imageData []byte) error {
	if len(imageData) == 0 {
		return nil
	}

	// Apply rotation and/or dark mode if needed
	if w.config.Rotation != 0 || w.config.DarkMode {
		// Decode image
		img, _, err := image.Decode(bytes.NewReader(imageData))
		if err != nil {
			return fmt.Errorf("failed to decode image: %w", err)
		}

		// Apply rotation
		if w.config.Rotation != 0 {
			img = rotateImage(img, w.config.Rotation)
		}

		// Apply dark mode
		if w.config.DarkMode {
			img = invertImage(img)
		}

		// Re-encode image to PNG
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("failed to encode image: %w", err)
		}
		imageData = buf.Bytes()
	}

	// Pass image data to Objective-C
	C.updateWindowImage((*C.uchar)(unsafe.Pointer(&imageData[0])), C.int(len(imageData)))

	return nil
}

// UpdateStatus is a no-op for native window (no status bar)
func (w *NativeWindow) UpdateStatus(status string) {
	// No-op - native window doesn't have a status bar
}

// SetOnClosed sets a callback for window close (not implemented yet)
func (w *NativeWindow) SetOnClosed(callback func()) {
	// TODO: Implement if needed
}

// SetOnRefresh sets the callback for manual refresh (Cmd+R)
func (w *NativeWindow) SetOnRefresh(callback func()) {
	w.refreshCallback = callback
	if callback != nil {
		// Start polling for keyboard shortcut requests from menu actions
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				if bool(C.checkAndClearRefreshRequested()) {
					if w.refreshCallback != nil {
						w.refreshCallback()
					}
				}
				if bool(C.checkAndClearRotateRequested()) {
					if w.rotateCallback != nil {
						w.rotateCallback()
					}
				}
			}
		}()
	}
}

// SetOnRotate sets the callback for manual rotate (Cmd+T)
func (w *NativeWindow) SetOnRotate(callback func()) {
	w.rotateCallback = callback
}

// Close closes the window
func (w *NativeWindow) Close() {
	C.stopNativeApp()
}

// GetApp returns nil for native window
func (w *NativeWindow) GetApp() interface{} {
	return nil
}

// SetMenuItemsEnabled enables or disables the action menu items (Refresh and Rotate)
func (w *NativeWindow) SetMenuItemsEnabled(enabled bool) {
	C.setMenuItemsEnabled(C.bool(enabled))
	if w.verbose {
		if enabled {
			fmt.Println("[Native] Menu items enabled")
		} else {
			fmt.Println("[Native] Menu items disabled")
		}
	}
}
