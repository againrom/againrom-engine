package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/text"
)

const cheatChatLimit = 256

type cheatInputState struct {
	chat    func(string)
	alt     func(byte)
	viewer  *Viewer
	open    bool
	line    string
	image   *ebiten.Image
	overlay textOverlay
}

var altLetterKeys = [...]ebiten.Key{
	ebiten.KeyA, ebiten.KeyB, ebiten.KeyC, ebiten.KeyD, ebiten.KeyE, ebiten.KeyF,
	ebiten.KeyG, ebiten.KeyH, ebiten.KeyI, ebiten.KeyJ, ebiten.KeyK, ebiten.KeyL,
	ebiten.KeyM, ebiten.KeyN, ebiten.KeyO, ebiten.KeyP, ebiten.KeyQ, ebiten.KeyR,
	ebiten.KeyS, ebiten.KeyT, ebiten.KeyU, ebiten.KeyV, ebiten.KeyW, ebiten.KeyX,
	ebiten.KeyY, ebiten.KeyZ,
}

func readAltLetter(alt bool, pressed func(ebiten.Key) bool) byte {
	if alt {
		for i, key := range altLetterKeys {
			if pressed(key) {
				return byte('A' + i)
			}
		}
	}
	return 0
}

// SetCheatCommands installs mission chat and debug-command destinations.
func (a *App) SetCheatCommands(chat func(string), alt func(byte)) {
	a.cheatInput.chat, a.cheatInput.alt = chat, alt
	if chat == nil {
		a.cheatInput.open, a.cheatInput.line = false, ""
	}
}

func (a *App) dispatchCheatAlt(letter byte) {
	if letter == 'S' {
		a.requestScreenshot()
		return
	}
	if letter >= 'B' && letter <= 'Y' && letter != 'S' && a.cheatInput.alt != nil {
		a.cheatInput.alt(letter)
	}
}

func (f *flow) missionCheatInput() bool {
	if f.viewer == nil {
		return false
	}
	switch f.screen {
	case ScreenMap:
		return true
	case ScreenGameMenu:
		return f.menuBack == ScreenMap
	case ScreenSave:
		return f.saveDialog != nil && f.saveDialog.request.OnMap
	case ScreenLoad:
		return f.loadBack == ScreenMap || f.loadBack == ScreenGameMenu && f.menuBack == ScreenMap
	}
	return false
}

func (a *App) stepCheatChat(in appInput) appInput {
	c := &a.cheatInput
	v := a.flow.viewer
	if a.flow.screen != ScreenMap || v == nil || a.flow.popupOpen() && !v.HelpOpen() || v.goldModalOpen() || c.viewer != v {
		c.open, c.line = false, ""
	}
	if v != nil && v.HelpOpen() {
		return in
	}
	if !c.open {
		if c.chat == nil || in.Unfocused || !in.Enter || a.flow.screen != ScreenMap || v == nil || a.flow.popupOpen() || v.goldModalOpen() {
			return in
		}
		c.open, c.line, c.viewer = true, "", v
	} else if !in.Unfocused {
		switch {
		case in.Escape:
			c.open, c.line = false, ""
		case in.Enter:
			line := c.line
			c.open, c.line = false, ""
			if line != "" && c.chat != nil {
				c.chat(line)
			}
		default:
			if !in.Viewer.Alt && in.Backspace && c.line != "" {
				_, size := utf8.DecodeLastRuneInString(c.line)
				c.line = c.line[:len(c.line)-size]
			}
			if !in.Viewer.Alt {
				for _, r := range in.Typed {
					if !unicode.IsControl(r) && len(c.line)+utf8.RuneLen(r) <= cheatChatLimit {
						c.line += string(r)
					}
				}
			}
		}
	}
	return appInput{CursorX: in.CursorX, CursorY: in.CursorY, Help: in.Help, AltLetter: in.AltLetter,
		Viewer: Input{CursorX: in.Viewer.CursorX, CursorY: in.Viewer.CursorY, Alt: in.Viewer.Alt, Unfocused: in.Unfocused}, Unfocused: in.Unfocused}
}

func (a *App) cheatChatPicture() (*image.RGBA, image.Point) {
	v := a.flow.viewer
	if !a.cheatInput.open || a.flow.screen != ScreenMap || v == nil || v.popupOpen() || a.cheatInput.viewer != v || v.font == nil {
		return nil, image.Point{}
	}
	line := make([]byte, 0, len(a.cheatInput.line))
	for _, r := range a.cheatInput.line {
		b, ok := byte(r), r >= 32 && r < 127
		if a.flow.encodeMenuKey != nil {
			b, ok = a.flow.encodeMenuKey(r)
		}
		if !ok {
			b = '?'
		}
		line = append(line, b)
	}
	width := max(1, v.cam.ViewW-2*messageOriginX)
	for len(line) > 0 && v.font.Advance("> "+string(line)+"_")+8 > width {
		line = line[1:]
	}
	if v.font.Height() <= 0 {
		return nil, image.Point{}
	}
	pic := image.NewRGBA(image.Rect(0, 0, width, v.font.Height()+messageShadow+8))
	draw.Draw(pic, pic.Bounds(), &image.Uniform{C: color.RGBA{R: 8, G: 8, B: 8, A: 230}}, image.Point{}, draw.Src)
	caption := "> " + string(line) + "_"
	v.font.DrawFlat(pic, caption, 4+messageShadow, 4+messageShadow, messageShadowColor)
	v.font.Draw(pic, caption, 4, 4, messageInkColor(MessageWhite))
	return pic, image.Pt(messageOriginX, max(messageOriginY, v.cam.ViewH-pic.Bounds().Dy()-messageOriginY))
}

func (a *App) drawCheatChat(screen *ebiten.Image) {
	v := a.flow.viewer
	if v == nil || !v.place.Valid() {
		return
	}
	var at image.Point
	pic, blit, calls := glyphPicture(true, func() *image.RGBA {
		pic, point := a.cheatChatPicture()
		at = point
		return pic
	})
	if pic == nil {
		return
	}
	c := &a.cheatInput
	if c.image == nil || c.image.Bounds().Size() != pic.Bounds().Size() {
		if c.image != nil {
			c.image.Dispose()
		}
		c.image = ebiten.NewImage(pic.Bounds().Dx(), pic.Bounds().Dy())
	}
	if !v.textSmoothingEnabled {
		blit, calls = pic, nil
	}
	c.image.WritePixels(blit.Pix)
	ox, oy := v.place.Origin()
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(v.place.Scale(), v.place.Scale())
	op.GeoM.Translate(ox+float64(at.X)*v.place.Scale(), oy+float64(at.Y)*v.place.Scale())
	screen.DrawImage(c.image, &op)
	if len(calls) > 0 {
		c.overlay.draw(screen, calls, v.place.Scale(), ox+float64(at.X)*v.place.Scale(), oy+float64(at.Y)*v.place.Scale())
	}
}

func (a *App) HeadlessChatState() (string, bool) {
	return a.cheatInput.line, a.cheatInput.open
}

func (a *App) HeadlessChatFrame() (*image.RGBA, error) {
	var pic *image.RGBA
	text.Record(func() { pic, _ = a.cheatChatPicture() })
	if pic == nil {
		return nil, fmt.Errorf("headless chat: no open chat with a mission font")
	}
	return pic, nil
}
