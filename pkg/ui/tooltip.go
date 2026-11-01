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
	// style is the game's own hover style on every screen; nil draws the
	// shared hover help.
	style *hoverStyle
}

// Hover help uses the character card's font without resampling its glyphs,
// or the game's hover style when it has one.
func (h *tooltipController) picture(t tooltipTarget, p image.Point, bounds image.Rectangle) (*image.RGBA, image.Point, bool) {
	return h.styled(t, p, bounds, h.style)
}

// styled is picture in style s; a styled target keeps its own font unless
// the style names one.
func (h *tooltipController) styled(t tooltipTarget, p image.Point, bounds image.Rectangle, s *hoverStyle) (*image.RGBA, image.Point, bool) {
	switch {
	case s != nil && s.font != nil && t.font != nil:
		t.font = s.font
	case s == nil && h.baseFont != nil && t.font != nil:
		t.font = h.baseFont
	}
	return tooltipPicture(t, p, bounds, h.ball, s)
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
	delay := time.Duration(h.delay) * time.Millisecond
	shown := cadence{period: delay, gate: gateAtLeast}
	expired := cadence{period: delay + 25*time.Second, gate: gateAtLeast}
	return shown.due(h.since, h.now) && !expired.due(h.since, h.now)
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

func tooltipPicture(t tooltipTarget, p image.Point, bounds image.Rectangle, ball image.Image, s *hoverStyle) (*image.RGBA, image.Point, bool) {
	if t.key() == "" || bounds.Empty() {
		return nil, image.Point{}, false
	}
	// Bound all hover help to a readable column, including item and spell
	// descriptions. Source-authored breaks remain intact and height may grow.
	var lines []string
	width := min(tooltipMaxWidth, bounds.Dx()) - hoverExtraW
	for _, line := range t.lines {
		if s != nil && s.authored {
			// A line that begins with '#' is an installed fragment appended
			// to the text before it (an item's "#Defense +"): its '#' is the
			// break the line boundary already makes, not a blank line.
			line = strings.TrimPrefix(line, "#")
		}
		for _, part := range strings.Split(line, "#") {
			if s != nil && s.authored {
				lines = append(lines, part)
				continue
			}
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			lines = append(lines, tooltipFlowLines(part, t.font, width)...)
		}
	}
	return placeHover(hoverDescription{lines, t.font, s, hoverAnchor{hoverAbove, p, bounds}}, ball)
}

// ComposeTooltipHint paints lines with font exactly as every hover hint is
// painted (tooltipPicture, above), positioned within bounds from the anchor
// point p. It exists for an install-gated witness that has no live App or
// Viewer to hover through and needs production's own paint for one screen's
// hint, not the screen beneath it — see cmd/tooltipshot. ball is the
// installed corner picture the App was given (FrontEnd.HoverBall), look the
// game's hover look (FrontEnd.HoverHelp); a nil look is the shared hover help.
func ComposeTooltipHint(lines []string, font *text.Font, p image.Point, bounds image.Rectangle, ball image.Image, look *HoverLook) (*image.RGBA, image.Point, bool) {
	h := &tooltipController{ball: ball}
	return h.styled(tooltipTarget{tooltipText, "witness", lines, font}, p, bounds, look.hoverStyle())
}

// tooltipFlowLines breaks one hover help line to width, counting the wider of the
// font's box and pen.
func tooltipFlowLines(s string, font *text.Font, width int) []string {
	return tooltipRule.plain(lineBox{Measure: reachOf(font), Width: width}, s)
}
