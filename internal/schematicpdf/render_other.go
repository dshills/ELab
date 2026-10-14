//go:build !darwin || !cgo

package schematicpdf

import "fmt"

func Supported() bool { return false }
func newPainter(Page, float64, float64, float64) (painter, error) {
	return nil, fmt.Errorf("schematic PDF requires the macOS CoreGraphics build")
}
