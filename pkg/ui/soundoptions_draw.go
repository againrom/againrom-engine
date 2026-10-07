package ui

import (
	"fmt"
	"image"

	"againrom/pkg/audio"
	"againrom/pkg/render/frame"
)

// soundTrackRows is how many track titles the list shows at once, and
// soundTrackRowH the height of one.
const (
	soundTrackRows = 4
	soundTrackRowH = 22
)

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
		return g.Rect(40, 80, w-64, 170)
	case gameMenuMusicUp:
		return g.Rect(w-64, 80, w-40, 104)
	case gameMenuMusicScroll:
		return g.Rect(w-64, 104, w-40, 146)
	case gameMenuMusicDown:
		return g.Rect(w-64, 146, w-40, 170)
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

// soundSliderValue is the slider position, 0..soundSliderRange, under frame x.
func soundSliderValue(channel audio.Channel, x int) int {
	r := soundSliderRect(channel)
	return min(max((x-r.Min.X)*soundSliderRange/max(1, r.Dx()-1), 0), soundSliderRange)
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
			if f.soundPointer.pressed && f.soundPointer.action == row.Action {
				position = f.soundPointer.value
			}
			drawSlider(dst, soundSliderRect(channel), position, soundSliderRange, focused)
		} else if row.Action == gameMenuAcknowledgments || row.Action == gameMenuMusicRandom {
			if focused {
				drawMovieBox(dst, r, true)
			}
			on := f.soundOptions.acknowledgments
			if row.Action == gameMenuMusicRandom {
				on = f.soundOptions.ReadPlayback().RandomOrder
			}
			a.drawGameOption(dst, r, row.Label, on, false)
		} else {
			drawMovieBox(dst, r, focused || f.soundPointer.pressed && f.soundPointer.action == row.Action)
			label := row.text()
			if row.Action == gameMenuVolumeDown {
				label = "-"
			} else if row.Action == gameMenuVolumeUp {
				label = "+"
			}
			font.Draw(dst.SubImage(r).(*image.RGBA), label, r.Min.X+(r.Dx()-font.Advance(label))/2,
				r.Min.Y+(r.Dy()-font.Height())/2, gameMenuText)
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

func (a *App) drawSoundTracks(dst *image.RGBA, focused bool) {
	f := a.flow
	g := soundOptionsDialog
	r := soundOptionRect(gameMenuMusicTracks)
	f.menuFont.Draw(dst, gameMenuLabelText(f.soundOptions.Words.Tracks), r.Min.X, g.Min.Y+60, loadSelectedText)
	list := f.soundOptions.list
	top, count := 0, 0
	if list != nil {
		top, count = list.Visible()
	}
	for i := 0; i < soundTrackRows; i++ {
		row := image.Rect(r.Min.X, r.Min.Y+i*soundTrackRowH, r.Max.X, r.Min.Y+(i+1)*soundTrackRowH)
		selected := i < count && list.Selection() == top+i
		drawMovieBox(dst, row, selected && focused)
		if i < count {
			tone := gameMenuText
			if selected {
				tone = loadSelectedText
			}
			label := list.Rows()[top+i].Text
			f.menuFont.Draw(dst.SubImage(row).(*image.RGBA), label, row.Min.X+3, row.Min.Y+2, tone)
		}
	}
	track := soundOptionRect(gameMenuMusicScroll)
	if len(a.media.scroll) < 26 {
		for _, action := range []gameMenuAction{gameMenuMusicUp, gameMenuMusicDown, gameMenuMusicScroll} {
			drawMovieBox(dst, soundOptionRect(action), false)
		}
		return
	}
	for y := track.Min.Y; y < track.Max.Y; y += 24 {
		copyNativeOver(dst, a.media.scroll[19], image.Pt(track.Min.X, y), track)
	}
	copyNativeOver(dst, a.media.scroll[18], soundOptionRect(gameMenuMusicUp).Min, dst.Bounds())
	copyNativeOver(dst, a.media.scroll[20], soundOptionRect(gameMenuMusicDown).Min, dst.Bounds())
	y := track.Min.Y
	if list != nil && list.Len() > 1 {
		y += list.Selection() * (track.Dy() - 24) / (list.Len() - 1)
	}
	copyNativeOver(dst, a.media.scroll[16], image.Pt(track.Min.X, y), dst.Bounds())
}
