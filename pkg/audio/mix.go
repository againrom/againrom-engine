package audio

import (
	"encoding/binary"
	"math"
)

// MasterUnit is the fixed-point unit Settings.Master is expressed over: a
// Master of MasterUnit is full volume and 0 is silent.
//
// THIS NUMBER IS OURS. Nothing in spec.md ties a master volume to the
// original's own arithmetic — the -volume flag is a new control surface
// this story adds, not a decoded one — so 100 is chosen for the ordinary
// reason a percentage is: it is the unit cmd/againrom's 0..100 -volume flag
// reads as without translation (plan T3).
const MasterUnit = 100

// Settings carries a gain and a mute. Retained devices apply these settings
// to active players; Stereo can also apply them to an offline PCM buffer.
type Settings struct {
	// Master is the volume, over MasterUnit: MasterUnit is full and 0 is
	// silent. Both Stereo and the retained devices clamp the applied gain.
	Master int
	// Muted overrides Master. Retained players keep advancing at zero gain;
	// offline Stereo returns no buffer.
	Muted bool
}

// DefaultSettings is unmuted at full volume: the state a FrontEnd starts in
// before cmd/againrom's two flags are applied, and the state every test in
// this package that is not itself exercising the volume or the mute uses.
var DefaultSettings = Settings{Master: MasterUnit, Muted: false}

// DeviceRate is the one sample rate every device this tree opens plays at:
// 22050 Hz.
//
// THE MEASUREMENT IS DECODED AND THE CHOICE TO STANDARDISE ON IT IS OURS.
// Picking that one is what makes resample's identity path (see DecodeWAV)
// the common case rather than the exception, leaving only 2 of 290 leaves
// doing any resampling arithmetic at all.
const DeviceRate = 22050

// Player is whatever a Sample and a Placement can be handed to (spec Terms).
// A device that discards everything is a LAWFUL Player and is the state of
// every headless run: pkg/ui's playSlotAt calls Play on whatever it holds,
// and a nil Player is refused before this interface is ever consulted,
// rather than by an implementation that has to do nothing on purpose.
//
// It takes a Sample and a Placement and NOT a pre-built buffer, because the
// one concrete Player this tree ships — pkg/ui's ebiten-backed device — needs
// a Settings value to build one via Stereo, and Settings are the CALLER's to
// hold, not this interface's to thread through on every call (see Settings'
// own comment).
type Player interface {
	Play(Sample, Placement)
}

// Voice is one retained optional sound. Stop releases its resources and is
// idempotent. Playing reports actual player status, not an estimated duration.
type Voice interface {
	Playing() bool
	Stop()
}

// VoicePlayer is an optional capability for UI effects with status/cancel
// contracts. Callers must not emulate it with unlimited Player.Play calls.
type VoicePlayer interface {
	StartVoice(Sample, Placement) Voice
}

// Stereo renders one play as an interleaved 16-bit little-endian stereo
// buffer: sample i's left channel at byte offset 4*i and its right channel
// at 4*i+2.
//
// The scaling is two SEQUENTIAL divisions — the Placement gain first, then
// the master — rather than one product of three terms, which is what keeps
// every intermediate value inside a comfortable 32-bit range for any Sample
// this package can decode: a 16-bit PCM value is at most 32767, scaled by a
// gain of at most GainUnit (10000) is at most ~3.3e8, and that result scaled
// again by an ordinary master leaves ample headroom below int32's ~2.1e9
// limit, where multiplying all three terms together first would not. The
// final value is still clamped to the int16 range rather than trusted to
// stay there, because nothing in Settings' own type stops a caller from
// asking for a master above MasterUnit — AC-10 exercises the ordinary range,
// and the clamp is what keeps an unusual one saturating instead of wrapping.
//
// Stereo returns nil, not a buffer of zeroes, when both channels' scaled
// gain comes out at zero — muted, a zero master, or a Placement a caller
// built directly with both sides at zero — because nothing downstream
// should have to inspect a buffer's contents to learn that nothing is meant
// to be heard (AC-10).
func Stereo(s Sample, p Placement, st Settings) []byte {
	if st.Muted {
		return nil
	}
	left := scaleGain(p.Left, st.Master)
	right := scaleGain(p.Right, st.Master)
	if left == 0 && right == 0 {
		return nil
	}

	buf := make([]byte, 4*len(s.PCM))
	for i, v := range s.PCM {
		l := clamp16(scaleSample(int32(v), left))
		r := clamp16(scaleSample(int32(v), right))
		binary.LittleEndian.PutUint16(buf[4*i:], uint16(l))
		binary.LittleEndian.PutUint16(buf[4*i+2:], uint16(r))
	}
	return buf
}

// scaleGain folds one channel's Placement gain (over GainUnit) and the
// master volume (over MasterUnit) into a single factor still expressed over
// GainUnit, so scaleSample below has one unit to divide by regardless of
// whether the master ever changed it. It is not defensive tidying -- the two
// ends are wrong in different ways and neither is hypothetical, because the
// master arrives from a command-line integer with no range of its own:
//
//   - a NEGATIVE master inverts every sample's sign. That is not quieter, it
//     is the same loudness in opposite phase, so a player who typed a minus by
//     accident gets exactly the noise he was trying to stop, and gets it with
//     nothing in the tree reporting a problem;
//   - a master ABOVE MasterUnit amplifies into clamp16's saturation, which is
//     audible clipping rather than a louder game.
func scaleGain(gain, master int) int32 {
	switch {
	case master < 0:
		master = 0
	case master > MasterUnit:
		master = MasterUnit
	}
	return int32(gain) * int32(master) / int32(MasterUnit)
}

// scaleSample applies one channel's folded gain (over GainUnit, from
// scaleGain) to one PCM value. It is int32 throughout; see Stereo's own
// comment for why that is safe rather than merely usual.
func scaleSample(v, gain int32) int32 {
	return v * gain / int32(GainUnit)
}

// clamp16 saturates v into the signed 16-bit range rather than letting it
// wrap, which is what a caller asking Settings for a master above MasterUnit
// would otherwise do to the buffer Stereo hands a device.
func clamp16(v int32) int16 {
	switch {
	case v > math.MaxInt16:
		return math.MaxInt16
	case v < math.MinInt16:
		return math.MinInt16
	default:
		return int16(v)
	}
}
