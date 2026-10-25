package ui

import "image"

// HeadlessTooltipState observes the production timer and popup compositor.
// It carries no installed text bytes and never advances the UI clock.
type HeadlessTooltipState struct {
	DelayMS int             `json:"delay_ms"`
	Target  string          `json:"target,omitempty"`
	Visible bool            `json:"visible"`
	Bounds  image.Rectangle `json:"bounds"`
}

func (a *App) HeadlessTooltip() (HeadlessTooltipState, *image.RGBA) {
	state := HeadlessTooltipState{DelayMS: a.TooltipDelay(), Target: a.tooltipTarget().id}
	var pic *image.RGBA
	var at image.Point
	if a.flow.screen == ScreenMap && a.flow.viewer != nil {
		pic, at, state.Visible = a.flow.viewer.tooltipPresent()
	} else {
		pic, at, state.Visible = a.tooltipPresent(image.Rectangle{Max: a.place.FrameSize()})
	}
	if state.Visible {
		state.Bounds = pic.Bounds().Add(at)
	}
	return state, pic
}

// HeadlessTooltipLines is the text of the hover box HeadlessTooltip reports
// visible, one entry per drawn line, in the font's code page; nil while none
// shows. The route witness records it beside the frame.
func (a *App) HeadlessTooltipLines() []string {
	var target tooltipTarget
	var visible bool
	if a.flow.screen == ScreenMap && a.flow.viewer != nil {
		v := a.flow.viewer
		target = v.tooltipTarget()
		visible = v.tooltip.visible(target.key())
	} else {
		target = a.tooltipTarget()
		visible = a.tooltip.visible(target.key())
	}
	if !visible {
		return nil
	}
	return append([]string(nil), target.lines...)
}
