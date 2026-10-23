// This fixture includes the production helpers so no test-only entry points
// need to ship in the application. It creates menu objects, never a window.
#include "../file_focus_darwin.m"
#include <stdio.h>

#define CHECK(condition, description) do { \
    if (!(condition)) { fprintf(stderr, "%s\n", description); return 1; } \
} while (0)

int main(void) {
    @autoreleasepool {
        NSMenu *root = [[NSMenu alloc] initWithTitle:@"Fixture"];
        [root addItemWithTitle:@"App" action:NULL keyEquivalent:@""];
        [root addItemWithTitle:@"File" action:NULL keyEquivalent:@""];
        [root addItemWithTitle:@"Examples" action:NULL keyEquivalent:@""];
        NSArray *original = [root.itemArray copy];

        elabBeginFileEditing(0, root);
        CHECK([root.itemArray isEqualToArray:original], "zero token modified menu");
        elabBeginFileEditing(1, root);
        CHECK(root.numberOfItems == 4, "missing or duplicate editing group");
        CHECK([root itemAtIndex:2] == elabFileEditingGroup, "editing group order");
        NSMenu *edit = elabFileEditingGroup.submenu;
        CHECK(edit.autoenablesItems && edit.numberOfItems == 8, "invalid editing group");
        NSArray *actions = @[@"undo:", @"redo:", @"", @"cut:", @"copy:", @"paste:", @"", @"selectAll:"];
        NSArray *keys = @[@"z", @"z", @"", @"x", @"c", @"v", @"", @"a"];
        for (NSInteger i = 0; i < edit.numberOfItems; i++) {
            NSMenuItem *item = [edit itemAtIndex:i];
            if (i == 2 || i == 6) {
                CHECK(item.isSeparatorItem, "missing separator");
                continue;
            }
            CHECK(item.action == NSSelectorFromString(actions[i]), "wrong responder action");
            CHECK(item.target == nil, "editing action bypasses responder chain");
            CHECK([item.keyEquivalent isEqualToString:keys[i]], "wrong editing shortcut");
            NSEventModifierFlags expected = NSEventModifierFlagCommand |
                (i == 1 ? NSEventModifierFlagShift : 0);
            CHECK(item.keyEquivalentModifierMask == expected, "wrong shortcut modifiers");
        }
        elabEndFileEditing(2);
        CHECK(root.numberOfItems == 4, "stale completion removed active editing group");
        elabBeginFileEditing(3, root);
        CHECK(root.numberOfItems == 4, "replacement picker duplicated editing group");
        elabEndFileEditing(1);
        CHECK(root.numberOfItems == 4, "old completion removed replacement editing group");
        elabEndFileEditing(3);
        CHECK([root.itemArray isEqualToArray:original], "picker completion changed original menu");
        CHECK(elabFileEditingGroup == nil && elabFileEditingToken == 0, "editing state leaked");
        elabEndFileEditing(3);
        CHECK([root.itemArray isEqualToArray:original], "repeated completion changed menu");

        NSMenuItem *existing = [[NSMenuItem alloc] initWithTitle:@"Edit" action:NULL keyEquivalent:@""];
        NSMenu *existingMenu = [[NSMenu alloc] initWithTitle:@"Edit"];
        [existingMenu addItemWithTitle:@"Select All" action:@selector(selectAll:) keyEquivalent:@"a"];
        existing.submenu = existingMenu;
        [existingMenu release];
        [root addItem:existing];
        [existing release];
        NSArray *withExisting = [root.itemArray copy];
        elabBeginFileEditing(4, root);
        CHECK(elabFileEditingGroup == nil, "duplicated existing native editing commands");
        elabEndFileEditing(4);
        CHECK([root.itemArray isEqualToArray:withExisting], "removed existing editing menu");
        [withExisting release];
        elabBeginFileEditing(5, nil);
        elabEndFileEditing(5);
        CHECK(elabFileEditingGroup == nil && elabFileEditingToken == 0, "nil menu leaked state");
        [original release];
        [root release];
    }
    return 0;
}
