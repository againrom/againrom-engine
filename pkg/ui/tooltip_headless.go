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
