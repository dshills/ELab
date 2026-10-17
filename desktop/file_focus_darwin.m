//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>
#include <stdint.h>

// ELab admits one picker at a time. Keep only primitive identity on the main
// queue, without retaining a window or synchronously waiting on AppKit during
// shutdown. Capture is enqueued before the worker asks AppKit to show a panel.
static uint64_t elabFileFocusToken;
static NSInteger elabFileFocusWindowNumber;

void elabCaptureFileDialogFocus(uint64_t token) {
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            NSWindow *window = [NSApp keyWindow];
            if (window == nil || [window isKindOfClass:[NSPanel class]]) {
                window = [NSApp mainWindow];
            }
            elabFileFocusToken = token;
            elabFileFocusWindowNumber = 0;
            if (window.isVisible && !window.isMiniaturized &&
                ![window isKindOfClass:[NSPanel class]]) {
                elabFileFocusWindowNumber = window.windowNumber;
            }
        }
    });
}

void elabRestoreFileDialogFocus(uint64_t token) {
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            if (token != elabFileFocusToken) return;
            NSInteger number = elabFileFocusWindowNumber;
            elabFileFocusToken = 0;
            elabFileFocusWindowNumber = 0;
            if (number <= 0) return;
            NSWindow *key = [NSApp keyWindow];
            // A newly focused window or another live panel takes precedence.
            if (key.isVisible && key.windowNumber != number) return;
            for (NSWindow *window in [NSApp windows]) {
                if (window.windowNumber == number && window.isVisible &&
                    !window.isMiniaturized && window.canBecomeKeyWindow) {
                    // Restore keyboard routing; do not activate the app or
                    // order it in front of another application.
                    [window makeKeyWindow];
                    break;
                }
            }
        }
    });
}
