//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>

// Vellum owns installation, callback validation, and shutdown of the menu.
// This bridge only presents that same AppKit menu below its in-window button.
void elabShowMenu(const char *group, double x, double y) {
    NSString *name = [[NSString alloc] initWithUTF8String:group];
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            NSWindow *window = [NSApp keyWindow];
            if (window == nil) {
                for (NSWindow *candidate in [NSApp windows]) {
                    if ([candidate isVisible] && ![candidate isKindOfClass:[NSPanel class]]) {
                        window = candidate;
                        break;
                    }
                }
            }
            if (window != nil && ![window isKindOfClass:[NSPanel class]]) {
                [window makeKeyAndOrderFront:nil];
                NSMenu *menu = [[NSApp.mainMenu itemWithTitle:name] submenu];
                NSView *view = [window contentView];
                CGFloat scale = [window backingScaleFactor];
                NSPoint point = NSMakePoint(x / scale, [view bounds].size.height - y / scale);
                if (menu != nil) [menu popUpMenuPositioningItem:nil atLocation:point inView:view];
            }
            [name release];
        }
    });
}
