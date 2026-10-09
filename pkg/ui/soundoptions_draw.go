package ui

import (
	"fmt"
	"image"

	"againrom/pkg/audio"
	"againrom/pkg/render/frame"
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

// soundSlider is a channel's volume control: the shared slider over
// positions 0..soundSliderRange (MENU-118).
func (a *App) soundSlider(channel audio.Channel, position int) hSlider {
	p, ok := a.pointerFrame()
	return hSlider{Rect: soundSliderRect(channel), Pos: position, Max: soundSliderRange}.withPointer(p, ok)
}

func (a *App) soundOptionsPicture() *image.RGBA {
	f := a.flow
	g := soundOptionsDialog
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	drawSnappedDialog(dst, f.menuArt, g)
	font, words := f.menuFont, f.soundOptions.Words
	title := gameMenuLabelText(words.Title)
	font.Draw(dst, title, g.Min.X+(g.W()-font.Advance(title))/2, g.Min.Y+20, gameMenuText)
	values := f.soundOptions.Read()
	pointer, pointerOK := a.pointerFrame()
	for i, row := range f.menuRows() {
		r := soundOptionRect(row.Action)
		focused := f.menuList.Selection() == i
		since := markCapture()
		if row.Action == gameMenuMusicTracks {
			a.drawSoundTracks(dst, focused)
		} else if channel, slider := soundActionChannel(row.Action); slider {
			tone := gameMenuText
			if focused {
				tone = loadSelectedText
			}
			label := gameMenuLabelText(words.Labels[channel])
			font.Draw(dst.SubImage(r).(*image.RGBA), label, r.Min.X, r.Min.Y, tone)
			position := soundPercentSlider(values[channel])
			if f.soundPointer.slider.active() && f.soundPointer.action == row.Action {
				position = f.soundPointer.value
			}
			s := a.soundSlider(channel, position)
			s.Disabled = !row.Enabled
			drawHSlider(dst, a.media.scroll, s)
			if !row.Enabled {
				dimDisabledRow(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, s.Rect.Min.Y), since)
			}
			continue
		} else if row.Action == gameMenuAcknowledgments || row.Action == gameMenuMusicRandom {
			on := f.soundOptions.acknowledgments
			if row.Action == gameMenuMusicRandom {
				on = f.soundOptions.ReadPlayback().RandomOrder
			}
			a.drawGameOption(dst, r, row.Label, on, focused, !row.Enabled)
			continue
		} else {
			label, literal := row.Label, row.Literal
			if row.Action == gameMenuVolumeDown {
				label, literal = "-", true
			} else if row.Action == gameMenuVolumeUp {
				label, literal = "+", true
			}
			inside := pointerOK && pointer.In(r)
			drawPushButton(dst, font, pushButton{Rect: r, Label: label, Literal: literal, Hover: inside, Inside: inside, Focus: focused,
				Pressed: f.soundPointer.pressed && f.soundPointer.action == row.Action, Disabled: !row.Enabled})
			continue
		}
		if !row.Enabled {
			dimDisabledRow(dst, r, since)
		}
	}
	_, master, _ := f.readMenuSound()
	label := fmt.Sprintf("%d%%", master)
	mid := soundOptionRect(gameMenuVolumeDown)
	font.Draw(dst, label, (mid.Max.X+soundOptionRect(gameMenuVolumeUp).Min.X)/2-font.Advance(label)/2,
		mid.Min.Y+(mid.Dy()-font.Height())/2, gameMenuText)
	message := f.menuDisplayText(f.msg)
	for i, line := range NoticeLines(font, message, g.W()-80) {
		if i == 2 {
			break
		}
		font.Draw(dst, string(line), g.Min.X+40, g.Min.Y+44+i*(font.Height()+1), gameMenuText)
	}
	return dst
}

// soundTrackBar is the track list's bar, for geometry only.
func soundTrackBar(f *text.Font) vScrollBar { return vScrollBar{Rect: soundTrackBox(f).Bar()} }

func (a *App) drawSoundTracks(dst *image.RGBA, focused bool) {
	f := a.flow
	g := soundOptionsDialog
	r := soundOptionRect(gameMenuMusicTracks)
	f.menuFont.Draw(dst, gameMenuLabelText(f.soundOptions.Words.Tracks), r.Min.X, g.Min.Y+60, loadSelectedText)
	list := f.soundOptions.list
	pointer, pointerOK := a.pointerFrame()
	drawListBox(dst, f.menuFont, a.media.scroll, soundTrackBox(f.menuFont), list, func(row, _ int) string {
		return list.Rows()[row].Text
	}, pointer, pointerOK)
}
