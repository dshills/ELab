//go:build !darwin || !cgo

package main

import "github.com/dshills/vellum/ui"

func (s *screen) platformSelectors(_ *ui.Node, _ uint64) {}
