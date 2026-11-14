package ui

import (
	"fmt"

	"againrom/pkg/audio"
	"againrom/pkg/words"
)

type SoundOptionWords struct {
	Title, OK, Master, Enabled, Disabled, Test, NotSaved string
	Acknowledgments                                      string
	Tracks, RandomOrder, Play, Stop                      string
	Unavailable                                          string
	Labels                                               [audio.ChannelCount]string
}

func DefaultSoundOptionWords() SoundOptionWords {
	var english words.Book
	return SoundOptionWords{Title: "Sound Options", OK: "OK", Master: english.Text("sound.master"),
		Enabled: english.Text("sound.enabled"), Disabled: english.Text("sound.disabled"), Test: english.Text("sound.test"),
		NotSaved: english.Text("sound.not_saved"),
		Labels:   [audio.ChannelCount]string{"Music volume", "SFX volume", "Speech volume"}, Acknowledgments: "Acknowledgments",
		Tracks: "Tracks", RandomOrder: "Random Order", Play: "Play", Stop: "Stop", Unavailable: english.Text("sound.unavailable")}
}

type SoundOptionControls struct {
	Read                 func() audio.ChannelVolumes
	Write                func(audio.Channel, int) error
	Words                SoundOptionWords
	ReadAcknowledgments  func() bool
	WriteAcknowledgments func(bool) error
	acknowledgments      bool
	ReadPlayback         func() MusicPreferences
	WritePlayback        func(MusicPreferences) error
	TrackTitle           func(string) string
	music                func() *MusicController
	list                 *Picker
}

func (a *App) SetSoundOptionControls(c SoundOptionControls) {
	if a == nil || a.flow == nil {
		return
	}
	// The install states none of these six; they are the engine's own words in
	// the install's language.
	f := a.flow
	c.Words.Master = f.menuWord("sound.master")
	c.Words.Enabled = f.menuWord("sound.enabled")
	c.Words.Disabled = f.menuWord("sound.disabled")
	c.Words.Test = f.menuWord("sound.test")
	c.Words.NotSaved = f.word("sound.not_saved")
	c.Words.Unavailable = f.word("sound.unavailable")
	a.flow.soundOptions = c
	a.flow.soundOptions.music = func() *MusicController { return a.music }
	if a.music != nil && c.ReadPlayback != nil {
		a.music.SetPreferences(c.ReadPlayback())
	}
}

func soundChannelAction(channel audio.Channel) gameMenuAction {
	return gameMenuMusicVolume + gameMenuAction(channel)
}

func soundActionChannel(action gameMenuAction) (audio.Channel, bool) {
	if action < gameMenuMusicVolume || action > gameMenuSpeechVolume {
		return 0, false
	}
	return audio.Channel(action - gameMenuMusicVolume), true
}

func (f *flow) soundOptionRows() []gameMenuRow {
	words := f.soundOptions.Words
	volumes := f.soundOptions.Read()
	enabled, master, available := f.readMenuSound()
	var rows []gameMenuRow
	for channel := audio.Channel(0); channel < audio.ChannelCount; channel++ {
		rows = append(rows, gameMenuRow{Label: fmt.Sprintf("%s: %d%%", words.Labels[channel], volumes[channel]),
			Action: soundChannelAction(channel), Enabled: available && f.soundOptions.Write != nil,
			Fallback: [3]byte{'M', 'F', 'V'}[channel]})
	}
	if f.soundOptions.ReadPlayback != nil {
		present := f.soundOptions.list != nil && f.soundOptions.list.Len() > 0
		editable := f.soundOptions.WritePlayback != nil
		rows = append(rows,
			gameMenuRow{Label: words.Tracks, Fallback: 'L', Action: gameMenuMusicTracks, Enabled: present},
			gameMenuRow{Label: words.RandomOrder, Fallback: 'R', Action: gameMenuMusicRandom, Enabled: editable},
			gameMenuRow{Label: words.Play, Fallback: 'P', Action: gameMenuMusicPlay, Enabled: present && editable},
			gameMenuRow{Label: words.Stop, Fallback: 'S', Action: gameMenuMusicStop, Enabled: editable})
	}
	state := words.Disabled
	if f.soundOptions.ReadAcknowledgments != nil {
		rows = append(rows, gameMenuRow{Label: words.Acknowledgments, Fallback: 'A',
			Action: gameMenuAcknowledgments, Enabled: f.soundOptions.WriteAcknowledgments != nil})
	}
	if enabled {
		state = words.Enabled
	}
	return append(rows,
		gameMenuRow{Label: state, Fallback: 'E', Action: gameMenuToggleSound, Enabled: available && f.setMenuSound != nil},
		gameMenuRow{Label: words.Master + " -", Fallback: 'D', Action: gameMenuVolumeDown, Enabled: available && f.setMenuSound != nil && master > 0},
		gameMenuRow{Label: words.Master + " +", Fallback: 'U', Action: gameMenuVolumeUp, Enabled: available && f.setMenuSound != nil && master < 100},
		gameMenuRow{Label: words.Test, Fallback: 'T', Action: gameMenuTestSound, Enabled: available},
		gameMenuRow{Label: words.OK, Fallback: 'O', Action: gameMenuPageReturn, Enabled: true})
}

func (f *flow) setSoundOption(channel audio.Channel, value int) {
	if f.soundOptions.Write == nil || channel >= audio.ChannelCount {
		return
	}
	err := f.soundOptions.Write(channel, max(0, min(100, value)))
	selection := f.menuList.Selection()
	f.rebuildGameMenu(gameMenuSoundOptionsPage, selection)
	if err != nil {
		f.msg = f.soundOptions.Words.NotSaved + err.Error()
	}
}

// soundKeyStep is one keyboard or endcap step of a slider, in slider
// positions: max(trunc(N/16),1) (MENU-118).
const soundKeyStep = soundSliderRange / 16

// applySoundSlider applies a slider's channel percentage at the control event,
// without leaving the page and without ending a drag.
func (f *flow) applySoundSlider(channel audio.Channel, percent int) {
	if f.soundOptions.Write == nil || channel >= audio.ChannelCount {
		return
	}
	percent = max(0, min(100, percent))
	if f.soundOptions.Read()[channel] == percent {
		return
	}
	if err := f.soundOptions.Write(channel, percent); err != nil {
		f.msg = f.soundOptions.Words.NotSaved + err.Error()
	}
}

// stepSoundSlider moves a channel by whole slider positions and always changes
// the stored percentage when the slider has room to move.
func (f *flow) stepSoundSlider(channel audio.Channel, step int) {
	current := f.soundOptions.Read()[channel]
	next := soundSliderPercent(soundPercentSlider(current) + step)
	if next == current {
		next += step / soundKeyStep
	}
	f.applySoundSlider(channel, next)
	message := f.msg
	f.rebuildGameMenu(gameMenuSoundOptionsPage, f.menuList.Selection())
	f.msg = message
}

const speechTestSample = "mf_merc/select2.wav" // VIDEO-SFX-085, DIV-2122

func (a *App) playSpeechTest() {
	bank := a.namedSounds()
	if a.speechPlayer == nil || bank == nil {
		return
	}
	sample, ok := bank.NamedSample(speechTestSample)
	if !ok {
		return
	}
	audio.Dispatch(a.speechPlayer, sample, audio.FixedRequest("fixed-interface", "voice:"+speechTestSample,
		audio.SpeechChannel, 220, false, audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
}

// isSoundCheck reports whether a Sound Options row is a checkbox.
func isSoundCheck(a gameMenuAction) bool {
	return a == gameMenuAcknowledgments || a == gameMenuMusicRandom
}
