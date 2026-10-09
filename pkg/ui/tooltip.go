package ui

import (
	"image"
	"strings"
	"time"

	"againrom/pkg/render/text"
)

const (
	DefaultTooltipDelay = 500
	tooltipMaxWidth     = 320
)

// ValidTooltipDelay names the owner's six millisecond choices. The clock is
// presentation state and never participates in simulation or save hashes.
func ValidTooltipDelay(ms int) bool { return ms >= 0 && ms <= 500 && ms%100 == 0 }

type tooltipController struct {
	delay    int
	point    image.Point
	key      string
	since    time.Time
	now      time.Time
	baseFont *text.Font
	// ball is interface/Ball.bmp, the hover box's corner (MENU-128).
	ball image.Image
}

// Hover help uses the character card's font without resampling its glyphs.
func (h *tooltipController) picture(t tooltipTarget, p image.Point, bounds image.Rectangle) (*image.RGBA, image.Point, bool) {
	if h.baseFont != nil && t.font != nil {
		t.font = h.baseFont
	}
	return tooltipPicture(t, p, bounds, h.ball)
}

func (h *tooltipController) reset() {
	h.key = ""
	h.since = time.Time{}
}

func (h *tooltipController) setDelay(ms int) {
	if !ValidTooltipDelay(ms) {
		ms = DefaultTooltipDelay
	}
	if h.delay != ms {
		h.delay = ms
		h.reset()
	}
}

func (h *tooltipController) observe(now time.Time, p image.Point, key string, blocked bool) {
	if blocked || key == "" {
		h.reset()
		h.now = now
		return
	}
	if key != h.key || p != h.point || now.Before(h.now) {
		h.since = now
	}
	h.key, h.point, h.now = key, p, now
}

func (h *tooltipController) visible(key string) bool {
	if h == nil || key == "" || key != h.key {
		return false
	}
	elapsed := h.now.Sub(h.since)
	delay := time.Duration(h.delay) * time.Millisecond
	return elapsed >= delay && elapsed < delay+25*time.Second
}

type tooltipTarget struct {
	kind  uint8
	id    string
	lines []string
	font  *text.Font
}

const (
	tooltipText uint8 = iota
	tooltipItem
	tooltipSpell
	tooltipCommand
)

func (t tooltipTarget) key() string {
	if t.id == "" || len(t.lines) == 0 || t.font == nil {
		return ""
	}
	return t.id + "\x00" + strings.Join(t.lines, "\x00")
}

// Installed strings retain their code-page bytes; '#' is the authored line
// separator (TEXT-HOVERPAINT-053), not text to display to the player.
func tooltipLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "#")
}

func tooltipPicture(t tooltipTarget, p image.Point, bounds image.Rectangle, ball image.Image) (*image.RGBA, image.Point, bool) {
	if t.key() == "" || bounds.Empty() {
		return nil, image.Point{}, false
	}
	// Bound all hover help to a readable column, including item and spell
	// descriptions. Source-authored breaks remain intact and height may grow.
	var lines []string
	width := min(tooltipMaxWidth, bounds.Dx()) - hoverExtraW
	for _, line := range t.lines {
		for _, part := range strings.Split(line, "#") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			lines = append(lines, wrapTooltipLine(part, t.font, width)...)
		}
	}
	pic := composeHoverBox(lines, t.font, ball)
	if pic == nil {
		return nil, image.Point{}, false
	}
	// TEXT-HOVERPAINT-053: the cursor anchors the lower-left corner.
	// Growing text moves the top upward, leaving the hovered item visible.
	at := image.Pt(p.X, p.Y-pic.Bounds().Dy())
	if at.X+pic.Bounds().Dx() > bounds.Max.X {
		at.X = bounds.Max.X - pic.Bounds().Dx()
	}
	if at.Y+pic.Bounds().Dy() > bounds.Max.Y {
		at.Y = bounds.Max.Y - pic.Bounds().Dy()
	}
	at.X, at.Y = max(bounds.Min.X, at.X), max(bounds.Min.Y, at.Y)
	return pic, at, true
}

// ComposeTooltipHint paints lines with font exactly as every hover hint is
// painted (tooltipPicture, above), positioned within bounds from the anchor
// point p. It exists for an install-gated witness that has no live App or
// Viewer to hover through and needs production's own paint for one screen's
// hint, not the screen beneath it — see cmd/tooltipshot.
func ComposeTooltipHint(lines []string, font *text.Font, p image.Point, bounds image.Rectangle) (*image.RGBA, image.Point, bool) {
	return tooltipPicture(tooltipTarget{tooltipText, "witness", lines, font}, p, bounds, nil)
}

func wrapTooltipLine(s string, font *text.Font, width int) []string {
	if width <= 0 {
		return nil
	}
	var out []string
	for s != "" {
		end, space := 0, 0
		for i := 1; i <= len(s); i++ {
			w, _ := font.Measure(s[:i])
			if max(w, font.Advance(s[:i])) > width && i > 1 {
				break
			}
			end = i
			if s[i-1] == ' ' {
				space = i
			}
		}
		if end == 0 {
			end = 1
		}
		if end < len(s) && space > 0 {
			end = space
		}
		out = append(out, strings.TrimRight(s[:end], " "))
		s = strings.TrimLeft(s[end:], " ")
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}
