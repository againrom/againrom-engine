package ui

import (
	"slices"
	"testing"

	"againrom/pkg/audio"
)

type settingsPreviewRecorder struct {
	recordingPlayer
	requests []audio.Request
}

func (p *settingsPreviewRecorder) RequestSample(sample audio.Sample, request audio.Request) audio.Voice {
	p.Play(sample, request.Placement)
	p.requests = append(p.requests, request)
	return nil
}

func TestSoundSliderReleaseRequestContracts(t *testing.T) {
	for _, channel := range []audio.Channel{audio.MusicChannel, audio.EffectsChannel, audio.SpeechChannel} {
		t.Run([3]string{"music", "effects", "speech"}[channel], func(t *testing.T) {
			bank := namedStubBank{stubBank: fixedUISoundBank(), names: map[string]audio.Sample{
				"mf_merc/select2.wav": soundSample(777),
			}}
			speech, effects := &settingsPreviewRecorder{}, &settingsPreviewRecorder{}
			a, _ := speechTestApp(t, bank, speech)
			a.SetAudio(effects, bank)
			releaseSlider(t, a, channel)
			wantEffects := []int{2}
			if channel == audio.EffectsChannel {
				wantEffects = append(wantEffects, 100)
			}
			if got := playedUISlots(&effects.recordingPlayer); !slices.Equal(got, wantEffects) {
				t.Fatalf("effects requests = %v, want %v", got, wantEffects)
			}
			wantSpeech := 0
			if channel == audio.SpeechChannel {
				wantSpeech = 1
			}
			if len(speech.plays) != wantSpeech || wantSpeech == 1 && speech.plays[0].Sample.PCM[0] != 777 {
				t.Fatalf("speech requests = %v, want %d fixed voice requests", speech.plays, wantSpeech)
			}
			for _, device := range []struct {
				player *settingsPreviewRecorder
				group  audio.Channel
			}{{effects, audio.EffectsChannel}, {speech, audio.SpeechChannel}} {
				for _, request := range device.player.requests {
					if request.Group != device.group || request.Priority != 220 || request.Repeat || request.Pan != 0 || request.Frequency != 0 ||
						request.Placement != (audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}) {
						t.Fatalf("release request = %+v", request)
					}
				}
			}
		})
	}
}
