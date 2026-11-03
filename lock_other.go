//go:build !darwin && !linux

package main

import (
	"os"
	"path/filepath"
)

func lockStore(root string) (func(), error) {
	path := filepath.Join(root, ".lock-exclusive")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return func() { _ = file.Close(); _ = os.Remove(path) }, nil
}
