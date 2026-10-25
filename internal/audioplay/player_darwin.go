//go:build darwin && cgo

package audioplay

/*
#cgo LDFLAGS: -framework Foundation -framework AVFoundation
#include <stdlib.h>
void *elabAudioPlay(const void *,int);
void elabAudioStop(void *);
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type player struct{ handle unsafe.Pointer }

func New() Player { return &player{} }
func (p *player) Play(wav []byte) error {
	if len(wav) == 0 {
		return fmt.Errorf("empty audio")
	}
	handle := C.elabAudioPlay(unsafe.Pointer(&wav[0]), C.int(len(wav)))
	if handle == nil {
		return fmt.Errorf("macOS could not start the audio preview")
	}
	p.Stop()
	p.handle = handle
	return nil
}
func (p *player) Stop() {
	if p.handle != nil {
		C.elabAudioStop(p.handle)
		p.handle = nil
	}
}
