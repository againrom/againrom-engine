package ui

import (
	"image"
	"time"
)

// TOWN-223: one sprite, frame zero hidden, advanced on a 63-ms timer.
// The untraced placement policy uses a repeatable tour of the metal ornaments.
var chargenSparkPoints = [...]image.Point{
	{80, 74}, {104, 144}, {336, 163}, {174, 191},
	{74, 310}, {466, 225}, {622, 178}, {290, 365},
}

func (c *Chargen) advancePresentation(now time.Time, active bool) {
	if active && !c.sparkAt.IsZero() && now.After(c.sparkAt) {
		c.sparkElapsed += min(now.Sub(c.sparkAt), 100*time.Millisecond)
	}
	if active {
		c.sparkAt = now
	} else {
		c.sparkAt = time.Time{}
	}
}

func (c *Chargen) paintSparkle(dst *image.RGBA) {
	frames := c.setup.PreCreate.Art.Sparkles
	if len(frames) < 2 {
		return
	}
	step := int(c.sparkElapsed / (63 * time.Millisecond))
	period := len(frames) + 20
	frame := step % period
	if frame == 0 || frame >= len(frames) || frames[frame] == nil {
		return
	}
	pic := frames[frame]
	at := chargenSparkPoints[(step/period)%len(chargenSparkPoints)].Sub(pic.Bounds().Size().Div(2))
	copyNativeOver(dst, pic, at, dst.Bounds())
}
