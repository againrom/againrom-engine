package ui

import (
	"againrom/pkg/audio"
	"againrom/pkg/video"
)

// CutsceneAudioDevice owns exactly one streaming session tied to the current
// movie's own decoded track. Start opens a retained player against the
// movie's own rate and channel count; Push delivers one video frame's own
// PCM chunk, in wire order, and is silently dropped by an implementation
// that has not started or that refuses the declared shape -- exactly
// Track/Sample's own "unplayable input is silence" rule (pkg/audio doc.go)
// applied to a stream instead of a whole decoded buffer. A device this
// project mixes and mutes on its OWN policy, not the decoder's (DIV-1255):
// ROM1's own decoder arms its sound driver once setup succeeds (VIDEO-034);
// this project's device is armed by App once the movie's first frame is
// known instead, and volume/mute come from the SAME sound preferences every
// other retained stream already honours (SetSettings).
type CutsceneAudioDevice interface {
	Start(rate int, channels int)
	Push(pcm []byte)
	Stop()
	SetSettings(audio.Settings)
}

// SetCutsceneAudio installs the optional streaming device. A nil device
// leaves cutscene playback exactly as silent as it always was; Push and
// Start calls on a nil CutsceneAudioDevice are never made (see
// pumpCutsceneAudio).
func (a *App) SetCutsceneAudio(device CutsceneAudioDevice) {
	if a == nil {
		return
	}
	if a.cutsceneAudio != nil {
		a.cutsceneAudio.Stop()
	}
	a.cutsceneAudio = device
}

// pumpCutsceneAudio forwards the audio that arrived with the movie's most
// recently advanced frame, once per frame change: Start fires exactly once
// per opened movie, the instant its format is known, and Push forwards only
// a chunk this specific frame carried, never a replay of the last one. A
// frame with no audio at all (a video-only stream, or a silent frame in one
// that has a track) pushes nothing.
func (a *App) pumpCutsceneAudio() {
	if a == nil || a.cutscene == nil || a.cutsceneAudio == nil {
		return
	}
	n := a.cutscene.FrameNumber()
	if n == a.cutsceneAudioFrame {
		return
	}
	a.cutsceneAudioFrame = n
	if !a.cutsceneAudioStarted {
		a.cutsceneAudioStarted = true
		if rate, channels := a.cutscene.AudioFormat(); rate != 0 {
			a.cutsceneAudio.Start(int(rate), int(channels))
			// The device's played position then paces the frames (VIDEO-081); a
			// device with no clock, or one whose session did not open, leaves
			// the timer wait in charge.
			if clock, ok := a.cutsceneAudio.(video.SoundClock); ok {
				a.cutscene.SetSoundClock(clock)
			}
		}
	}
	if chunk := a.cutscene.AudioChunk(); len(chunk) > 0 {
		a.cutsceneAudio.Push(chunk)
	}
}

// stopCutsceneAudio tears down the streaming session and resets the
// per-movie latches closeCutscene owns. It is a no-op with no device
// installed, exactly like every other optional retained stream in this file.
func (a *App) stopCutsceneAudio() {
	if a == nil {
		return
	}
	a.haltCutsceneAudio()
	a.cutsceneAudioFrame = 0
}

func (a *App) haltCutsceneAudio() {
	if a.cutsceneAudioStarted && a.cutsceneAudio != nil {
		a.cutsceneAudio.Stop()
	}
	a.cutsceneAudioStarted = false
}
