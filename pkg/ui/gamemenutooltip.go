package ui

import (
	_ "embed"
	"encoding/json"
	"strconv"

	"againrom/pkg/render/text"
)

//go:embed gamemenutooltip_ru.json
var tooltipRUText string

var tooltipRU = func() (labels struct{ Delay, Unit, NotSaved string }) {
	if err := json.Unmarshal([]byte(tooltipRUText), &labels); err != nil {
		panic(err)
	}
	return labels
}()

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
	label, unit := "TOOLTIP ~DELAY: ", " ms"
	if f.menuSelector() == text.SelectorConverting {
		label, unit = f.menuDisplayText(tooltipRU.Delay), f.menuDisplayText(tooltipRU.Unit)
	}
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
