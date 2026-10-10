package ui

import (
	"strconv"

	"againrom/pkg/render/text"
)

func (a *App) SetTooltipDelayPreference(ms int, persist func(int) error) {
	a.tooltip.setDelay(ms)
	a.flow.tooltip = &a.tooltip
	a.flow.persistTooltipDelay = persist
	if a.flow.viewer != nil {
		a.flow.viewer.tooltip = &a.tooltip
	}
}

func (a *App) TooltipDelay() int { return a.tooltip.delay }

// SetTooltipFont gives every hover surface the character card's installed font.
func (a *App) SetTooltipFont(font *text.Font) { a.tooltip.baseFont = font }

func (f *flow) tooltipDelayRow() gameMenuRow {
	ms := DefaultTooltipDelay
	if f.tooltip != nil {
		ms = f.tooltip.delay
	}
	label, unit := f.menuWord("tooltip.delay"), f.menuWord("tooltip.unit")
	return gameMenuRow{Label: label + strconv.Itoa(ms) + unit, Fallback: 'D',
		Action: gameMenuTooltipDelay, Enabled: f.tooltip != nil}
}

func (f *flow) cycleTooltipDelay() error {
	if f.tooltip == nil {
		return nil
	}
	ms := (f.tooltip.delay + 100) % 600
	f.tooltip.setDelay(ms)
	if f.persistTooltipDelay != nil {
		return f.persistTooltipDelay(ms)
	}
	return nil
}
