package audioplay

// Player owns one short, explicitly requested preview. It never autoplays.
// A failed Play must leave any previously accepted playback untouched.
type Player interface {
	Play([]byte) error
	Stop()
}
