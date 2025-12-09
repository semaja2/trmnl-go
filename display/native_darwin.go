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

// Window delegate to handle close events
@interface WindowDelegate : NSObject <NSWindowDelegate>
@end

@implementation WindowDelegate
- (void)windowWillClose:(NSNotification *)notification {
    [NSApp terminate:nil];
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

        // Always enable fullscreen support
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
*/
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"unsafe"

	"github.com/semaja2/trmnl-go/config"
)

// NativeWindow represents a native macOS window
type NativeWindow struct {
	windowPtr unsafe.Pointer
	config    *config.Config
	verbose   bool
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

// Close closes the window
func (w *NativeWindow) Close() {
	C.stopNativeApp()
}

// GetApp returns nil for native window
func (w *NativeWindow) GetApp() interface{} {
	return nil
}
