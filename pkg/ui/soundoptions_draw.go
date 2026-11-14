package ui

import (
	"fmt"
	"image"

	"againrom/pkg/audio"
	"againrom/pkg/render/text"
)

// soundTrackRows is how many track titles the list shows at once.
const soundTrackRows = 4

// soundTrackBox is the track list: the shared list at the list box's
// argument rectangle, with its bar at the right edge (MENU-121).
func soundTrackBox(f *text.Font) listBox {
	return newListBox(soundOptionsDialog.Rect(40, 80, soundOptionsDialog.W()-64, 170), soundTrackRows, f)
}

// soundOptionRect is a control's rectangle in frame coordinates, placed from
// the Sound Options dialog's snapped origin (MENU-075, MENU-077). The
// rectangles are Medium. The sound switch, master volume and test button are
// authored controls; they share the strip under OK.
func soundOptionRect(action gameMenuAction) image.Rectangle {
	g := soundOptionsDialog
	w := g.W()
	if channel, ok := soundActionChannel(action); ok {
		top := [...]int{175, 224, 275}[channel]
		return g.Rect(258, top, w-40, top+39)
	}
	switch action {
	case gameMenuMusicTracks:
		return soundTrackBox(nil).Rect
	case gameMenuMusicUp:
		return soundTrackBar(nil).top()
	case gameMenuMusicScroll:
		b := soundTrackBar(nil)
		return image.Rect(b.Rect.Min.X, b.top().Max.Y, b.Rect.Max.X, b.bottom().Min.Y)
	case gameMenuMusicDown:
		return soundTrackBar(nil).bottom()
	case gameMenuMusicRandom:
		return g.Rect(40, 190, 252, 214)
	case gameMenuAcknowledgments:
		return g.Rect(40, 223, 252, 247)
	case gameMenuMusicPlay:
		return g.Rect(40, 256, 140, 280)
	case gameMenuMusicStop:
		return g.Rect(150, 256, 252, 280)
	case gameMenuPageReturn:
		return g.Rect(40, 290, 252, 314)
	case gameMenuToggleSound:
		return g.Rect(52, 318, 180, 336)
	case gameMenuVolumeDown:
		return g.Rect(184, 318, 208, 336)
	case gameMenuVolumeUp:
		return g.Rect(252, 318, 276, 336)
	case gameMenuTestSound:
		return g.Rect(284, 318, w-60, 336)
	}
	return image.Rectangle{}
}

func soundSliderRect(channel audio.Channel) image.Rectangle {
	r := soundOptionRect(soundChannelAction(channel))
	return image.Rect(r.Min.X, r.Min.Y+15, r.Max.X, r.Max.Y)
}

func soundTrackBar(f *text.Font) vScrollBar { return vScrollBar{Rect: soundTrackBox(f).Bar()} }

// soundOptionsPage is the Sound Options page; the page's message wraps
// under the title.
func soundOptionsPage() settingsPage {
	g := soundOptionsDialog
	return settingsPage{dialog: g, titleSpan: g.W(), title: func(f *flow) string { return f.soundOptions.Words.Title },
		control: (*App).soundOptionsControl, extras: (*App).soundOptionsExtras,
		notice: image.Rect(g.Min.X+40, g.Min.Y+44, g.Max.X-40, g.Min.Y+80)}
}

// soundOptionsControl is a Sound Options row's control: each channel a
// slider over positions 0..soundSliderRange (MENU-118), the tracks the
// shared list (MENU-120), two checkboxes and the push buttons.
func (a *App) soundOptionsControl(row gameMenuRow) settingsControl {
	f := a.flow
	r := soundOptionRect(row.Action)
	c := settingsControl{kind: settingsPush, rect: r, label: row.Label, literal: row.Literal, disabled: !row.Enabled}
	if channel, ok := soundActionChannel(row.Action); ok {
		c.kind, c.label, c.focusing = settingsSlider, f.soundOptions.Words.Labels[channel], true
		c.slider = hSlider{Rect: soundSliderRect(channel), Pos: soundPercentSlider(f.soundOptions.Read()[channel]), Max: soundSliderRange}
		c.set = func(pos int) int { f.applySoundSlider(channel, soundSliderPercent(pos)); return pos }
		c.released = func() {
			if channel == audio.EffectsChannel {
				a.playUISound(UISoundOptionsTest)
			} else if channel == audio.SpeechChannel {
				a.playSpeechTest()
			}
		}
		if row.Enabled {
			c.adjust = func(delta int) { f.stepSoundSlider(channel, delta*soundKeyStep) }
		}
		return c
	}
	switch {
	case row.Action == gameMenuMusicTracks:
		list := f.soundOptions.list
		c.kind, c.label, c.list, c.box, c.wheel = settingsList, f.soundOptions.Words.Tracks, list, soundTrackBox(f.menuFont), settingsPageList
		c.labelAt = image.Pt(r.Min.X, soundOptionsDialog.Min.Y+60)
		c.text = func(row, _ int) string { return list.Rows()[row].Text }
	case isSoundCheck(row.Action):
		c.kind, c.on = settingsCheck, f.soundOptions.acknowledgments
		if row.Action == gameMenuMusicRandom {
			c.on = f.soundOptions.ReadPlayback().RandomOrder
		}
	case row.Action == gameMenuVolumeDown:
		c.label, c.literal = "-", true
	case row.Action == gameMenuVolumeUp:
		c.label, c.literal = "+", true
	}
	return c
}

// soundOptionsExtras is the master volume between its two buttons.
func (a *App) soundOptionsExtras() []settingsControl {
	_, master, _ := a.flow.readMenuSound()
	down, up := soundOptionRect(gameMenuVolumeDown), soundOptionRect(gameMenuVolumeUp)
	return []settingsControl{{kind: settingsText, rect: image.Rect(down.Max.X, down.Min.Y, up.Min.X, down.Max.Y), label: fmt.Sprintf("%d%%", master)}}
}
