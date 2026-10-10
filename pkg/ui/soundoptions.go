package ui

import (
	"fmt"
	"image"

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
	bar                  scrollBarInput
}

type soundOptionPointer struct {
	press  buttonLatch
	action gameMenuAction
	value  int
	slider sliderInput
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

func soundOptionHit(p image.Point) (gameMenuAction, bool) {
	for _, action := range []gameMenuAction{gameMenuMusicVolume, gameMenuEffectsVolume, gameMenuSpeechVolume,
		gameMenuAcknowledgments, gameMenuToggleSound, gameMenuVolumeDown, gameMenuVolumeUp, gameMenuTestSound, gameMenuPageReturn,
		gameMenuMusicTracks, gameMenuMusicRandom, gameMenuMusicPlay, gameMenuMusicStop, gameMenuMusicUp, gameMenuMusicDown, gameMenuMusicScroll} {
		r := soundOptionRect(action)
		if channel, slider := soundActionChannel(action); slider {
			r = soundSliderRect(channel)
		}
		if p.In(r) {
			return action, true
		}
	}
	return 0, false
}

func (a *App) stepSoundOptions(in appInput) bool {
	f := a.flow
	if in.Unfocused {
		f.soundPointer = soundOptionPointer{}
		return false
	}
	trackFocused := f.menuRows()[f.menuList.Selection()].Action == gameMenuMusicTracks && f.soundOptions.list != nil
	if in.PaneMode {
		delta := 1
		if in.ShiftHeld {
			delta = -1
		}
		f.menuList.Move(delta)
	} else if trackFocused && (in.Up || in.Down || in.PageUp || in.PageDown || in.Home || in.End) {
		switch {
		case listKey(f.soundOptions.list, in.Up, in.Down, in.PageUp, in.PageDown):
		case in.Home:
			f.soundOptions.list.Select(0)
		case in.End:
			f.soundOptions.list.Select(f.soundOptions.list.Len() - 1)
		}
	} else if in.Up {
		f.menuList.Move(-1)
	} else if in.Down {
		f.menuList.Move(1)
	} else if in.Left || in.Right {
		row := f.menuRows()[f.menuList.Selection()]
		if channel, ok := soundActionChannel(row.Action); ok && row.Enabled {
			step := -soundKeyStep
			if in.Right {
				step = soundKeyStep
			}
			f.stepSoundSlider(channel, step)
		}
	} else if in.Enter {
		a.chooseGameMenu()
	} else if in.Panels && isSoundCheck(f.menuRows()[f.menuList.Selection()].Action) {
		// Focused Space toggles a checkbox (MENU-124).
		a.chooseGameMenu()
	} else {
		for _, r := range in.Typed {
			if f.chooseGameMenuAccelerator(r, a.beforeGameMenuAction) {
				break
			}
		}
	}
	if f.menuPage != gameMenuSoundOptionsPage {
		f.soundPointer = soundOptionPointer{}
		return false
	}
	p, inFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if inFrame && p.In(soundOptionRect(gameMenuMusicTracks)) && f.soundOptions.list != nil && in.WheelY != 0 {
		f.soundOptions.list.Move(-int(in.WheelY) * 3)
	}
	if list := f.soundOptions.list; list != nil {
		if req, pos := f.soundOptions.bar.step(listBar(soundTrackBox(f.menuFont), list), p, inFrame, in); req != barNone {
			listBarRequest(list, req, pos)
			f.focusMusicTracks()
			return false
		}
		if f.soundOptions.bar.active() {
			return false
		}
	}
	action, hit := soundOptionHit(p)
	if in.PrimaryPressed && inFrame && hit && isSoundCheck(action) {
		// A checkbox toggles on the press; its release is a no-op (MENU-124).
		f.soundPointer = soundOptionPointer{}
		for i, row := range f.menuRows() {
			if row.Action == action && row.Enabled {
				f.menuList.Select(i)
				a.chooseGameMenu()
				break
			}
		}
		return false
	}
	if in.PrimaryPressed && !f.soundPointer.press.Holds() {
		f.soundPointer = soundOptionPointer{action: action}
		f.soundPointer.press.Press(int(action), inFrame && hit)
		if action == gameMenuMusicTracks && inFrame {
			// The shared list selects on the press (MENU-120).
			if row := f.soundTrackAt(p); row >= 0 && f.soundOptions.list != nil {
				f.soundOptions.list.Select(row)
				f.focusMusicTracks()
			}
		}
	}
	press := &f.soundPointer
	if channel, slider := soundActionChannel(press.action); press.press.Holds() && slider {
		position := soundPercentSlider(f.soundOptions.Read()[channel])
		if press.slider.active() {
			position = press.value
		}
		if pos, set := press.slider.step(a.soundSlider(channel, position), p, inFrame, in); set {
			press.value = pos
			f.applySoundSlider(channel, soundSliderPercent(pos))
		}
	}
	if !in.PrimaryReleased {
		return false
	}
	selected := *press
	f.soundPointer = soundOptionPointer{}
	if !selected.press.Holds() || !inFrame {
		return false
	}
	if selected.action == gameMenuMusicTracks {
		return false
	}
	for i, row := range f.menuRows() {
		if row.Action != selected.action || !row.Enabled {
			continue
		}
		f.menuList.Select(i)
		if channel, slider := soundActionChannel(selected.action); slider {
			a.playUISound(UISoundCommonControl)
			if channel == audio.EffectsChannel {
				a.playUISound(UISoundOptionsTest)
			} else if channel == audio.SpeechChannel {
				a.playSpeechTest()
			}
		} else if _, activated := selected.press.Release(int(action), hit); activated {
			a.chooseGameMenu()
		}
		break
	}
	return false
}

const speechTestSample = "mf_merc/select2.wav"

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
