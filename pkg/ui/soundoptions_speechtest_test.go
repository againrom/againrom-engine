package ui

import (
	"image"
	"testing"

	"againrom/pkg/audio"
)

type namedStubBank struct {
	stubBank
	names map[string]audio.Sample
}

func (b namedStubBank) NamedSample(path string) (audio.Sample, bool) {
	s, ok := b.names[path]
	return s, ok
}

func speechTestApp(t *testing.T, bank SoundBank, speech audio.Player) (*App, *recordingPlayer) {
	t.Helper()
	a := NewApp("speech test", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen, a.flow.menuFont = fiViewer(t), ScreenMap, gameMenuTestFont()
	effects := &recordingPlayer{}
	a.SetAudio(effects, bank)
	a.SetSpeechAudio(speech)
	a.SetGameMenuSettings(nil, nil, func() (bool, int, bool) { return true, 100, true }, func(bool, int) error { return nil })
	volumes := audio.ChannelVolumes{50, 50, 50}
	a.SetSoundOptionControls(SoundOptionControls{Words: DefaultSoundOptionWords(),
		Read:  func() audio.ChannelVolumes { return volumes },
		Write: func(c audio.Channel, v int) error { volumes[c] = v; return nil }})
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	return a, effects
}

func releaseSlider(t *testing.T, a *App, channel audio.Channel) {
	t.Helper()
	track := soundSliderRect(channel)
	p := image.Pt(track.Min.X+track.Dx()/2, track.Min.Y+5)
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSpeechSliderReleasePlaysTheVoiceTestOnTheSpeechDevice(t *testing.T) {
	voice := audio.Sample{Rate: audio.DeviceRate, PCM: []int16{777}}
	bank := namedStubBank{stubBank: fixedUISoundBank(), names: map[string]audio.Sample{"mf_merc/select2.wav": voice}}
	speech := &recordingPlayer{}
	a, effects := speechTestApp(t, bank, speech)

	releaseSlider(t, a, audio.SpeechChannel)
	if len(speech.plays) != 1 || speech.plays[0].Sample.PCM[0] != 777 {
		t.Fatalf("speech release plays = %v, want the voice test once", speech.plays)
	}
	for _, play := range effects.plays {
		if play.Sample.PCM[0] == 777 || play.Sample.PCM[0] == int16(UISoundOptionsTest) {
			t.Fatalf("speech release played %d on the effects device", play.Sample.PCM[0])
		}
	}

	speech.plays = nil
	releaseSlider(t, a, audio.MusicChannel)
	releaseSlider(t, a, audio.EffectsChannel)
	if len(speech.plays) != 0 {
		t.Fatalf("music or effects release played speech: %v", speech.plays)
	}
}

func TestSpeechSliderReleaseIsSilentWithoutDeviceOrRecording(t *testing.T) {
	voice := audio.Sample{Rate: audio.DeviceRate, PCM: []int16{777}}
	bank := namedStubBank{stubBank: fixedUISoundBank(), names: map[string]audio.Sample{"mf_merc/select2.wav": voice}}
	a, _ := speechTestApp(t, bank, nil)
	releaseSlider(t, a, audio.SpeechChannel)

	speech := &recordingPlayer{}
	a, _ = speechTestApp(t, namedStubBank{stubBank: fixedUISoundBank()}, speech)
	releaseSlider(t, a, audio.SpeechChannel)
	if len(speech.plays) != 0 {
		t.Fatalf("absent recording played %v", speech.plays)
	}
}
