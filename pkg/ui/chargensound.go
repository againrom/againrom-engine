package ui

import (
	"image"

	"againrom/pkg/audio"
)

// NamedSoundBank answers a sample by its sfx.res member path. pkg/game's
// SoundBank implements it beside the numbered-slot lookup.
type NamedSoundBank interface {
	NamedSample(path string) (audio.Sample, bool)
}

// SFXVoices holds one surface's own instances, at most one per member. Its
// zero value is ready. The instances are presentation state and never enter
// saves or hashes.
type SFXVoices struct {
	members []string
	voices  []audio.Voice
	scope   *AudioScope
}

// Restart stops a playing instance of member and requests the member again
// from its first sample: the stop, rewind and request of VIDEO-SFX-058 and
// VIDEO-SFX-059.
func (s *SFXVoices) Restart(p audio.Player, b NamedSoundBank, member string) {
	s.RestartFor("character-generator", p, b, member)
}

func (s *SFXVoices) RestartFor(source string, p audio.Player, b NamedSoundBank, member string) {
	if i := s.index(member); i >= 0 && s.voices[i] != nil {
		audio.StopReset(s.voices[i])
	}
	s.start(source, p, b, member)
}

// Request requests member unless its instance is playing.
func (s *SFXVoices) Request(p audio.Player, b NamedSoundBank, member string) {
	s.RequestFor("character-generator", p, b, member)
}

func (s *SFXVoices) RequestFor(source string, p audio.Player, b NamedSoundBank, member string) {
	if i := s.index(member); i >= 0 && s.voices[i] != nil && s.voices[i].Playing() {
		return
	}
	s.start(source, p, b, member)
}

// Stop stops every instance the surface requested and forgets them, leaving
// the zero value.
func (s *SFXVoices) Stop() {
	for _, v := range s.voices {
		if v != nil {
			audio.StopReset(v)
		}
	}
	s.scope.Destroy()
	*s = SFXVoices{}
}

func (s *SFXVoices) index(member string) int {
	for i, m := range s.members {
		if m == member {
			return i
		}
	}
	return -1
}

// start makes one request: centred (pan 0), one shot, at the effects device's
// own channel volume. Without a voice device or a named bank the request is
// silent, as every other retained interface sound here is.
func (s *SFXVoices) start(source string, p audio.Player, b NamedSoundBank, member string) {
	if p == nil || b == nil || member == "" {
		return
	}
	sample, ok := b.NamedSample(member)
	if !ok {
		return
	}
	if s.scope != nil && s.scope.Owner() != DeliveryOwner(p) {
		s.scope.Destroy()
		s.scope = nil
	}
	if s.scope == nil {
		s.scope = NewAudioScope(p)
	}
	if s.scope != nil {
		p = s.scope.Player(audio.EffectsChannel)
	}
	v := audio.Dispatch(p, sample, audio.FixedRequest(source, member, audio.EffectsChannel, 128, false,
		audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
	if i := s.index(member); i >= 0 {
		s.voices[i] = v
		return
	}
	s.members = append(s.members, member)
	s.voices = append(s.voices, v)
}

// The held statistic repeat in App ticks (DIV-1493). VIDEO-SFX-059 posts the
// first repeat on the first cursor tick more than 150 ms after the last mouse
// message and each later one on the first tick more than 66 ms after the
// previous repeat. Its cursor tick rate is Unknown; at the App's 60 ticks per
// second those are the 10th and the 4th tick.
const (
	chargenRepeatDelayTicks    = 10
	chargenRepeatIntervalTicks = 4
)

// chargenRepeat counts App ticks on the detailed page while the left button
// stays down. Its zero value has seen no tick, so the first tick counts as a
// mouse message. A double click's second press leaves the left-button flag
// clear, so a button held after it posts no repeat until the next press
// (MENU-146).
type chargenRepeat struct {
	seen        bool
	cursor      image.Point
	quiet       int
	posted      bool
	afterDouble bool
}

// tick reports whether this tick posts a repeat. A press or release of either
// button, wheel movement or a changed cursor position is a mouse message and
// restores the first delay.
func (r *chargenRepeat) tick(in appInput) bool {
	return r.tickEvery(in, chargenRepeatDelayTicks, chargenRepeatIntervalTicks)
}

// tickEvery is tick with its own first delay and interval.
func (r *chargenRepeat) tickEvery(in appInput, delay, interval int) bool {
	cursor := image.Pt(in.CursorX, in.CursorY)
	message := !r.seen || cursor != r.cursor || in.PrimaryPressed || in.PrimaryReleased ||
		in.SecondaryPressed || in.SecondaryReleased || in.WheelY != 0
	r.seen, r.cursor = true, cursor
	if in.PrimaryPressed {
		r.afterDouble = in.PrimaryDouble
	}
	if message || !in.Viewer.PrimaryDown || r.afterDouble {
		r.quiet, r.posted = 0, false
		return false
	}
	r.quiet++
	if !r.posted && r.quiet >= delay || r.posted && r.quiet >= interval {
		r.quiet, r.posted = 0, true
		return true
	}
	return false
}
