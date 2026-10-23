//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>
#include <stdint.h>

// ELab admits one picker at a time. Keep only primitive identity on the main
// queue, without retaining a window or synchronously waiting on AppKit during
// shutdown. Capture is enqueued before the worker asks AppKit to show a panel.
static uint64_t elabFileFocusToken;
static NSInteger elabFileFocusWindowNumber;

// AppKit text fields obtain Command-X/C/V/A from responder-chain menu actions.
// ELab's application menu has no Edit group. Supply those actions only while a
// native picker owns input, without routing text edits into schematic commands.
static uint64_t elabFileEditingToken;
static NSMenuItem *elabFileEditingGroup;

static BOOL elabHasNativeSelectAll(NSMenu *menu) {
    for (NSMenuItem *item in menu.itemArray) {
        if (item.action == @selector(selectAll:) ||
            (item.submenu && elabHasNativeSelectAll(item.submenu))) return YES;
    }
    return NO;
}

static void elabRemoveFileEditingGroup(void) {
    if (!elabFileEditingGroup) return;
    NSMenu *parent = elabFileEditingGroup.menu;
    if (parent) [parent removeItem:elabFileEditingGroup];
    [elabFileEditingGroup release];
    elabFileEditingGroup = nil;
}

static void elabBeginFileEditing(uint64_t token, NSMenu *menu) {
    if (!token) return;
    elabRemoveFileEditingGroup();
    elabFileEditingToken = token;
    if (!menu || elabHasNativeSelectAll(menu)) return;
    NSMenu *edit = [[NSMenu alloc] initWithTitle:@"Edit"];
    edit.autoenablesItems = YES;
    NSArray *titles = @[@"Undo", @"Redo", @"", @"Cut", @"Copy", @"Paste", @"", @"Select All"];
    NSArray *actions = @[@"undo:", @"redo:", @"", @"cut:", @"copy:", @"paste:", @"", @"selectAll:"];
    NSArray *keys = @[@"z", @"z", @"", @"x", @"c", @"v", @"", @"a"];
    for (NSUInteger i = 0; i < titles.count; i++) {
        if (![titles[i] length]) {
            [edit addItem:[NSMenuItem separatorItem]];
            continue;
        }
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:titles[i]
            action:NSSelectorFromString(actions[i]) keyEquivalent:keys[i]];
        item.target = nil;
        item.keyEquivalentModifierMask = NSEventModifierFlagCommand |
            (i == 1 ? NSEventModifierFlagShift : 0);
        [edit addItem:item];
        [item release];
    }
    elabFileEditingGroup = [[NSMenuItem alloc] initWithTitle:@"Edit" action:nil keyEquivalent:@""];
    elabFileEditingGroup.submenu = edit;
    [edit release];
    [menu insertItem:elabFileEditingGroup atIndex:MIN((NSInteger)2, menu.numberOfItems)];
}

static void elabEndFileEditing(uint64_t token) {
    if (token != elabFileEditingToken) return;
    elabRemoveFileEditingGroup();
    elabFileEditingToken = 0;
}

void elabCaptureFileDialogFocus(uint64_t token) {
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            NSWindow *window = [NSApp keyWindow];
            if (window == nil || [window isKindOfClass:[NSPanel class]]) {
                window = [NSApp mainWindow];
            }
            elabFileFocusToken = token;
            elabFileFocusWindowNumber = 0;
            elabBeginFileEditing(token, NSApp.mainMenu);
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
            elabEndFileEditing(token);
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
