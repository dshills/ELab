//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>
#include <stdint.h>
extern void elabMenuSelected(uint64_t token, int index);

@interface ELabMenuChoice : NSObject
@property(nonatomic) NSInteger selectedIndex;
- (void)choose:(NSMenuItem *)sender;
@end
@implementation ELabMenuChoice
- (void)choose:(NSMenuItem *)sender { self.selectedIndex = sender.tag; }
@end

// AppKit supplies menu material, rounded corners, selection, scrolling and
// keyboard navigation. No product palette or custom row drawing is applied.
void elabShowChoices(uint64_t token, const char *json, double x, double y) {
    NSString *payload = [[NSString alloc] initWithUTF8String:json];
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            ELabMenuChoice *choice = [[ELabMenuChoice alloc] init];
            choice.selectedIndex = -1;
            NSWindow *window = [NSApp keyWindow];
            if (window == nil || [window isKindOfClass:[NSPanel class]]) {
                for (NSWindow *candidate in [NSApp windows]) {
                    if ([candidate isVisible] && ![candidate isKindOfClass:[NSPanel class]]) {
                        window = candidate;
                        break;
                    }
                }
            }
            NSArray *items = [NSJSONSerialization JSONObjectWithData:[payload dataUsingEncoding:NSUTF8StringEncoding] options:0 error:nil];
            if (window != nil && [items isKindOfClass:[NSArray class]]) {
                [window makeKeyAndOrderFront:nil];
                NSMenu *menu = [[NSMenu alloc] initWithTitle:@""];
                menu.autoenablesItems = NO;
                NSInteger index = 0;
                for (NSDictionary *data in items) {
                    if ([data[@"separator"] boolValue]) {
                        [menu addItem:[NSMenuItem separatorItem]];
                        index++;
                        continue;
                    }
                    NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:data[@"label"] action:@selector(choose:) keyEquivalent:@""];
                    item.target = choice;
                    item.tag = index++;
                    item.enabled = ![data[@"disabled"] boolValue];
                    item.state = [data[@"checked"] boolValue] ? NSControlStateValueOn : NSControlStateValueOff;
                    [menu addItem:item];
                    [item release];
                }
                NSView *view = [window contentView];
                CGFloat scale = [window backingScaleFactor];
                NSPoint point = NSMakePoint(x / scale, [view bounds].size.height - y / scale);
                [menu popUpMenuPositioningItem:nil atLocation:point inView:view];
                [menu release];
            }
            elabMenuSelected(token, (int)choice.selectedIndex);
            [choice release];
            [payload release];
        }
    });
}
