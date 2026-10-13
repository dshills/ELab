//go:build darwin

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>

extern int goELabPointer(int phase, double x, double y, double scroll);
static id elabPointerMonitor = nil;

void elabInstallPointerMonitor(void) {
    void (^install)(void) = ^{
        if (elabPointerMonitor != nil) return;
        NSEventMask mask = NSEventMaskLeftMouseDown | NSEventMaskLeftMouseDragged | NSEventMaskLeftMouseUp | NSEventMaskScrollWheel;
        elabPointerMonitor = [[NSEvent addLocalMonitorForEventsMatchingMask:mask handler:^NSEvent *(NSEvent *event) {
            NSWindow *window = [event window];
            if (window == nil || ![[window title] isEqualToString:@"ELab · Live electronics"]) return event;
            NSView *view = [window contentView];
            NSPoint point = [view convertPoint:[event locationInWindow] fromView:nil];
            CGFloat scale = [window backingScaleFactor];
            int phase = [event type] == NSEventTypeScrollWheel ? 3 : ([event type] == NSEventTypeLeftMouseDown ? 0 : ([event type] == NSEventTypeLeftMouseDragged ? 1 : 2));
            // Use the location carried by this event. GLFW's button callback
            // queries the current hardware cursor instead, which can differ
            // after queued movement or targeted desktop automation.
            int handled = goELabPointer(phase, point.x * scale, ([view bounds].size.height - point.y) * scale, phase == 3 ? [event scrollingDeltaY] : 0);
            return handled ? nil : event;
        }] retain];
    };
    if ([NSThread isMainThread]) install(); else dispatch_async(dispatch_get_main_queue(), install);
}
void elabRemovePointerMonitor(void) {
    void (^remove)(void) = ^{
        if (elabPointerMonitor == nil) return;
        [NSEvent removeMonitor:elabPointerMonitor];
        [elabPointerMonitor release];
        elabPointerMonitor = nil;
    };
    if ([NSThread isMainThread]) remove(); else dispatch_async(dispatch_get_main_queue(), remove);
}
