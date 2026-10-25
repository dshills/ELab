//go:build !darwin || !cgo

package audioplay

import "fmt"

type player struct{}

func New() Player { return &player{} }
func (p *player) Play([]byte) error {
	return fmt.Errorf("audio playback requires the macOS build; export WAV instead")
}
func (p *player) Stop() {}
