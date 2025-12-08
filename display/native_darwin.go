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

void* createFloatingWindow(int width, int height, bool alwaysOnTop) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindowStyleMask styleMask = NSWindowStyleMaskTitled |
                                      NSWindowStyleMaskClosable |
                                      NSWindowStyleMaskMiniaturizable;

        NSRect frame = NSMakeRect(100, 100, width, height);
        mainWindow = [[NSWindow alloc] initWithContentRect:frame
                                                 styleMask:styleMask
                                                   backing:NSBackingStoreBuffered
                                                     defer:NO];

        [mainWindow setTitle:@"TRMNL Display"];
        [mainWindow setBackgroundColor:[NSColor blackColor]];

        if (alwaysOnTop) {
            [mainWindow setLevel:NSFloatingWindowLevel];
            [mainWindow setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces |
                                               NSWindowCollectionBehaviorFullScreenAuxiliary];
        }

        imageView = [[NSImageView alloc] initWithFrame:frame];
        [imageView setImageScaling:NSImageScaleProportionallyUpOrDown];
        [mainWindow setContentView:imageView];

        [mainWindow makeKeyAndOrderFront:nil];
        [mainWindow center];
        [NSApp activateIgnoringOtherApps:YES];
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

void runNativeApp() {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
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
