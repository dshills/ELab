//go:build !darwin || !cgo

package main

import "cogentcore.org/core/core"

func applyNativeWindowMinimum(*core.Scene) {}
