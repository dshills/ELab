//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework AppKit
#include <stdint.h>
void elabCaptureFileDialogFocus(uint64_t token);
void elabRestoreFileDialogFocus(uint64_t token);
*/
import "C"

import "sync/atomic"

var fileDialogFocusToken atomic.Uint64

// File dialogs run on a worker. Return a presentation-only continuation, called
// by PostWindow after its originating window's lifetime has been checked.
func captureFileDialogFocus() func() {
	token := fileDialogFocusToken.Add(1)
	if token == 0 {
		return func() {}
	}
	C.elabCaptureFileDialogFocus(C.uint64_t(token))
	return func() { C.elabRestoreFileDialogFocus(C.uint64_t(token)) }
}
