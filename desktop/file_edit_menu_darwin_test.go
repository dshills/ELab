//go:build darwin && cgo

package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Exercise the actual Objective-C helpers without opening a window, launching
// ELab, or changing the user's application menu or clipboard.
func TestNativeFileEditingMenuLifetime(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "file-edit-menu")
	compile := exec.CommandContext(ctx, "/usr/bin/clang", "-fblocks", "-Wall", "-Wextra", "-Werror", "-framework", "AppKit", "testdata/file_edit_menu_fixture.m", "-o", binary)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile native menu fixture: %v\n%s", err, output)
	}
	if output, err := exec.CommandContext(ctx, binary).CombinedOutput(); err != nil {
		t.Fatalf("native menu contract: %v\n%s", err, output)
	}
}
