//go:build !darwin || !cgo

package main

func captureFileDialogFocus() func() { return func() {} }
