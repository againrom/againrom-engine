package ui

import (
	"image"
	"time"
)

// sparkleRun is the sparkle's current run: when it began in shown time, how
// long its frames and its gap last, where it is centred, and how many runs
// began before it.
type sparkleRun struct {
	placed bool
	start  time.Duration
	length time.Duration
	at     image.Point
	index  int
}

func (c *Chargen) advancePresentation(now time.Time, active bool) {
	clamp := time.Duration(0)
	if l := c.layout(); l != nil && l.PreCreate.Sparkle != nil {
		clamp = ms(l.PreCreate.Sparkle.ClampMS)
	}
	if active && !c.sparkAt.IsZero() && now.After(c.sparkAt) {
		c.sparkElapsed += min(now.Sub(c.sparkAt), clamp)
	}
	if active {
		c.sparkAt = now
	} else {
		c.sparkAt = time.Time{}
	}
	c.advanceLoops(now, active)
}

// sparkleDraw is one draw from the setup's presentation stream, or 0 without
// one.
func (c *Chargen) sparkleDraw() int {
	if c.setup.Draws == nil {
		return 0
	}
	return c.setup.Draws.Raw()
}

// placeSparkle begins run c.spark.index: its length and its centre.
func (c *Chargen) placeSparkle(s *GeneratorSparkle, frames int) {
	run := &c.spark
	run.length = time.Duration(frames+s.GapSteps)*ms(s.StepMS) + ms(s.GapMS)
	if s.GapDrawDivisor > 0 {
		run.length += ms(c.sparkleDraw() / s.GapDrawDivisor)
	}
	switch {
	case len(s.Tour) > 0:
		run.at = s.Tour[run.index%len(s.Tour)].Pt()
	case len(s.Within) > 0:
		id, _ := generatorControlNamed(s.Within[c.sparkleDraw()%len(s.Within)])
		r := preControlRect(c, id)
		run.at = r.Min
		if !r.Empty() {
			run.at = run.at.Add(image.Pt(c.sparkleDraw()%r.Dx(), c.sparkleDraw()%r.Dy()))
		}
	}
	run.placed = true
}

// paintSparkle draws the sparkle frame the shown time reaches. Frame 0 of
// each run is hidden, and so is every step of its gap.
func (c *Chargen) paintSparkle(dst *image.RGBA) {
	l, p := c.layout(), c.art()
	if l == nil || p == nil || l.PreCreate.Sparkle == nil || len(p.Sparkles) < 2 {
		return
	}
	s, frames := l.PreCreate.Sparkle, p.Sparkles
	if !c.spark.placed {
		c.spark = sparkleRun{}
		c.placeSparkle(s, len(frames))
	}
	for c.spark.length > 0 && c.sparkElapsed >= c.spark.start+c.spark.length {
		c.spark.start += c.spark.length
		c.spark.index++
		c.placeSparkle(s, len(frames))
	}
	frame := int((c.sparkElapsed - c.spark.start) / ms(s.StepMS))
	if frame == 0 || frame >= len(frames) || frames[frame] == nil {
		return
	}
	pic := frames[frame]
	copyNativeOver(dst, pic, c.spark.at.Sub(pic.Bounds().Size().Div(2)), dst.Bounds())
}
