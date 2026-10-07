package ui

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"time"

	"againrom/pkg/render/text"
)

// MessageInk names the colour ramp a post chose (MISSION-MSGPOST-058). A
// ramp holds one colour per glyph level: the white ramp 17 per channel per
// level and the grey ramp 14, which text.Shade gives for the inks below.
type MessageInk uint8

const (
	MessageWhite MessageInk = iota
	MessageGrey
)

// messageInkColor is the colour whose text.Shade ramp is ink's: level k of
// white is 17k in every channel and of grey 14k (MISSION-MSGLINE-056).
func messageInkColor(ink MessageInk) color.RGBA {
	if ink == MessageGrey {
		return color.RGBA{R: 210, G: 210, B: 210, A: 255}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}

const (
	// messageOriginX and messageOriginY are where the oldest line's text
	// starts, in frame pixels: every line starts at the same x
	// (MISSION-MSGLINE-056).
	messageOriginX = 8
	messageOriginY = 8
	// messageLinePad is added to the font's height for the pitch between two
	// lines, and messageShadow is the offset of the shadow behind a line.
	messageLinePad = 2
	messageShadow  = 1
	// messageViewSnap is the row height the view's bottom snaps down to
	// before the capacity is taken from it (MISSION-MSGLINE-057).
	messageViewSnap = 32
)

// messageShadowColor is the flat shadow: the shadow ramp holds 8 in every
// channel of every entry (MISSION-MSGLINE-056).
var messageShadowColor = color.RGBA{R: 8, G: 8, B: 8, A: 255}

// MessageLine is one line of the map message line: its text in the font's own
// alphabet, the ramp it draws in, and how long it lives once it is the oldest
// line standing (MISSION-MSGLINE-057).
type MessageLine struct {
	Text string
	Ink  MessageInk
	Life time.Duration
}

// messageList is the message line's state: its lines oldest first, the time of
// the last restart or tick, the time counted against the oldest line, and the
// latest clock reading a tick delivered. serial changes whenever the lines do.
type messageList struct {
	lines   []MessageLine
	stamp   time.Time
	elapsed time.Duration
	clock   time.Time
	serial  int
}

// post appends pieces, each with the post's ink and life. When the list then
// holds one line, the clock restarts at the latest reading. While the list
// holds more than capacity it drops its oldest line and leaves the time
// counted unchanged (MISSION-MSGLINE-057).
func (m *messageList) post(pieces []string, ink MessageInk, life time.Duration, capacity int) {
	if len(pieces) == 0 {
		return
	}
	for _, p := range pieces {
		m.lines = append(m.lines, MessageLine{Text: p, Ink: ink, Life: life})
	}
	if len(m.lines) == 1 {
		m.stamp, m.elapsed = m.clock, 0
	}
	for len(m.lines) > max(capacity, 0) {
		m.lines = slices.Delete(m.lines, 0, 1)
	}
	m.serial++
}

// clear empties the line's text, ink and life entries and leaves the clock and
// elapsed time as they were (MENU-061, DIV-2078).
func (m *messageList) clear() {
	m.lines = nil
	m.serial++
}

// ClearMessages empties the map message line.
func (v *Viewer) ClearMessages() { v.messages.clear() }

// tick counts the time since the last restart or tick against the oldest
// line. When that exceeds its life, the time counted restarts and the line
// goes: one line per tick, so lifetimes follow one another. A tick with no
// line standing changes no time (DIV-1499).
func (m *messageList) tick(now time.Time) {
	m.clock = now
	if len(m.lines) == 0 {
		return
	}
	if m.stamp.IsZero() {
		m.stamp = now
	}
	if d := now.Sub(m.stamp); d > 0 {
		m.elapsed += d
	}
	m.stamp = now
	if m.elapsed > m.lines[0].Life {
		m.elapsed = 0
		m.lines = slices.Delete(m.lines, 0, 1)
		m.serial++
	}
}

// messageCapacity is how many lines the list keeps for a view viewH pixels
// high once its bottom is snapped down to whole rows: half the lines that
// fit at the font's pitch (MISSION-MSGLINE-057).
func messageCapacity(f *text.Font, viewH int) int {
	return ((viewH &^ (messageViewSnap - 1)) / (f.Height() + messageLinePad)) / 2
}

// wrapMessage splits s into the lines that fit width: a line breaks at a
// space, and holds text narrower than width. A word at least as wide as width
// stands alone on its line (DIV-1500). A nil font or no width leaves s whole.
func wrapMessage(f *text.Font, s string, width int) []string {
	if s == "" {
		return nil
	}
	if f == nil || width <= 0 || f.Advance(s) < width {
		return []string{s}
	}
	var out []string
	var line string
	for i, word := range strings.Split(s, " ") {
		if i == 0 {
			line = word
			continue
		}
		if candidate := line + " " + word; f.Advance(candidate) < width {
			line = candidate
			continue
		}
		out = append(out, line)
		line = word
	}
	return append(out, line)
}

// PostMessage puts text on the map message line in ink for life. The text
// wraps against the view's width and every piece takes the post's ink and life.
// A carriage return becomes a space (MISSION-MSGLINE-056).
func (v *Viewer) PostMessage(s string, ink MessageInk, life time.Duration) {
	view := viewportSize(v.frameW, v.frameH)
	pieces := wrapMessage(v.font, strings.ReplaceAll(s, "\r", " "), view.X)
	v.messages.post(pieces, ink, life, messageCapacity(v.font, view.Y))
}

// PostMessageUnlessNewest is PostMessage that drops a one-piece post equal to
// the newest line standing (MISSION-MSGLINE-057). It reports whether it posted.
func (v *Viewer) PostMessageUnlessNewest(s string, ink MessageInk, life time.Duration) bool {
	view := viewportSize(v.frameW, v.frameH)
	pieces := wrapMessage(v.font, strings.ReplaceAll(s, "", " "), view.X)
	if n := len(v.messages.lines); len(pieces) == 1 && n > 0 && v.messages.lines[n-1].Text == pieces[0] {
		return false
	}
	v.messages.post(pieces, ink, life, messageCapacity(v.font, view.Y))
	return len(pieces) > 0
}

// MessageLines is the lines the message line holds now, oldest first.
func (v *Viewer) MessageLines() []MessageLine {
	lines := slices.Clone(v.messages.lines)
	if v.playerPaused {
		label := v.playerPauseLabel
		if label == "" {
			label = "PAUSED"
		}
		lines = append([]MessageLine{{Text: label, Ink: MessageWhite}}, lines...)
	}
	return lines
}

func (v *Viewer) setPlayerPaused(paused bool) {
	if v.playerPaused != paused {
		v.playerPaused = paused
		v.messages.serial++
	}
}

// stepMessages delivers one clock reading to the message line.
func (v *Viewer) stepMessages(now time.Time) {
	v.messages.tick(now)
}

// composeMessages draws lines top to bottom one pitch apart, each as a flat
// shadow one pixel down and right of its text in the line's ink. It is nil for
// no font, no lines or lines that measure to no width. The text starts at the
// picture's left edge.
func composeMessages(lines []MessageLine, f *text.Font) *image.RGBA {
	if f == nil || f.Height() <= 0 || len(lines) == 0 {
		return nil
	}
	width := 0
	for _, ln := range lines {
		w, _ := f.Measure(ln.Text)
		width = max(width, w, f.Advance(ln.Text))
	}
	if width <= 0 {
		return nil
	}
	pitch := f.Height() + messageLinePad
	img := image.NewRGBA(image.Rect(0, 0, width+messageShadow, (len(lines)-1)*pitch+f.Height()+messageShadow))
	for i, ln := range lines {
		f.DrawFlat(img, ln.Text, messageShadow, i*pitch+messageShadow, messageShadowColor)
		f.Draw(img, ln.Text, 0, i*pitch, messageInkColor(ln.Ink))
	}
	return img
}

// messagePresent is the picture to draw this frame and where its top-left
// corner goes in frame pixels, or false for a frame that draws no message
// line: no font or no lines. The picture is composed again only when the
// lines have changed.
func (v *Viewer) messagePresent() (*image.RGBA, image.Point, bool) {
	if v.font == nil || len(v.messages.lines) == 0 && !v.playerPaused {
		return nil, image.Point{}, false
	}
	if v.messagePic == nil || v.messageBuilt != v.messages.serial || v.messageSmooth != v.textSmoothingEnabled {
		v.messagePic, v.messageBlit, v.messageText = glyphPicture(v.textSmoothingEnabled, func() *image.RGBA {
			return composeMessages(v.MessageLines(), v.font)
		})
		v.messageFresh = true
		v.messageBuilt, v.messageSmooth = v.messages.serial, v.textSmoothingEnabled
	}
	if v.messagePic == nil {
		return nil, image.Point{}, false
	}
	text.Append(v.messageText, 0, 0)
	return v.messagePic, image.Pt(messageOriginX, messageOriginY), true
}

// messageBlitOf is the picture the frame blits for the message line: the
// picture without its glyphs while the overlay draws them, the picture itself
// otherwise.
func (v *Viewer) messageBlitOf(pic *image.RGBA) *image.RGBA {
	if v.messageBlit != nil && v.textSmoothingEnabled {
		return v.messageBlit
	}
	return pic
}

// MessageDraw is one drawn line of the message line: its text, its ink and the
// frame pixel its first glyph's cell starts at.
type MessageDraw struct {
	Text string
	Ink  MessageInk
	Pen  image.Point
}

// MessageLog is the message line as this frame draws it: the composed picture,
// its top-left corner in frame pixels and each line with its pen, or false for
// a frame that draws none.
func (v *Viewer) MessageLog() (*image.RGBA, image.Point, []MessageDraw, bool) {
	pic, at, ok := v.messagePresent()
	if !ok {
		return nil, image.Point{}, nil, false
	}
	pitch := v.font.Height() + messageLinePad
	out := make([]MessageDraw, len(v.messages.lines))
	for i, ln := range v.messages.lines {
		out[i] = MessageDraw{Text: ln.Text, Ink: ln.Ink, Pen: at.Add(image.Pt(0, i*pitch))}
	}
	return pic, at, out, true
}
