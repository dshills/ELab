//go:build !darwin || !cgo

package main

import "github.com/dshills/vellum/ui"

func (s *screen) platformMenu(node ui.Node, _ string) ui.Node { return node }
