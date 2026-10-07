package ui

import "againrom/pkg/audio"

// AudioDeviceState reads settings and actual player gains for executable
// witnesses. It does not start playback or infer physical audibility.
func AudioDeviceState(player any) (audio.Settings, []float64, bool) {
	gain := func(p soundVoicePlayer) []float64 {
		if p != nil && p.IsPlaying() {
			return []float64{p.Volume()}
		}
		return nil
	}
	switch d := player.(type) {
	case *deliveryPlayer:
		return SoundDeviceState(d)
	case *deliveryAmbient:
		if d != nil {
			return sharedDeviceState(d.scope.shared, audio.EffectsChannel)
		}
	case *device:
		return SoundDeviceState(d)
	case *musicDevice:
		if d == nil {
			break
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		if d.pl != nil {
			return d.st, gain(d.pl), true
		}
		return d.st, nil, true
	case *ambientDevice:
		if d == nil {
			break
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		var gains []float64
		for _, loop := range d.loops {
			if loop.player != nil {
				gains = append(gains, gain(loop.player)...)
			}
		}
		return d.st, gains, true
	case *cutsceneAudioDevice:
		if d == nil {
			break
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		if d.s == nil {
			return d.st, nil, true
		}
		d.s.mu.Lock()
		defer d.s.mu.Unlock()
		if d.s.hasPlayed() {
			return d.st, gain(d.s.pl), true
		}
		return d.st, nil, true
	}
	return audio.Settings{}, nil, false
}
