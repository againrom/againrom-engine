package ui

import (
	"fmt"
	"strings"

	"againrom/pkg/render/terrain"
)

func (f *flow) pauseLabel() string {
	return f.menuWord("speed.paused")
}

func (f *flow) menuSpeedRung() int {
	if f.menuBack == ScreenMap {
		return terrain.ClampCadenceRung(f.rung)
	}
	return f.cadencePreference()
}

// Game Options uses the same normal speed and persistence sink as +/-.
// Authored labels express owner direction; original submenu contents are DIV-099.
func (f *flow) gameOptionsRows() []gameMenuRow {
	if f.gameOptions.Read != nil {
		return f.gameOptionsPageRows()
	}
	tips := literalMenuRow("TIPS MODE UNAVAILABLE")
	if f.menuTips != nil {
		state := "OFF"
		if f.menuTips() {
			state = "ON"
		}
		tips = gameMenuRow{Label: "~TIPS: " + state, Fallback: 'T', Action: gameMenuToggleTips, Enabled: f.setMenuTips != nil}
	}
	rung := f.menuSpeedRung()
	value := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", float64(terrain.CadencePeriod(terrain.DefaultCadenceRung))/float64(terrain.CadencePeriod(rung))), "0"), ".") + "x"
	unpaced := f.menuBack == ScreenMap && f.unpaced
	label, down, up := f.word("speed.label"), f.word("speed.slower"), f.word("speed.faster")
	if unpaced {
		label, value = f.word("speed.unpaced_label"), f.word("speed.unpaced")
	}
	label, down, up, value = f.menuText(label), f.menuText(down), f.menuText(up), f.menuText(value)
	rows := []gameMenuRow{
		tips,
		{Label: label + value, Literal: true, Status: true},
		{Label: down, Fallback: 'S', Action: gameMenuSpeedDown, Enabled: rung > terrain.CadenceRungMin || unpaced},
		{Label: up, Fallback: 'F', Action: gameMenuSpeedUp, Enabled: rung < terrain.CadenceRungMax || unpaced},
		f.tooltipDelayRow(),
	}
	return append(rows, pageReturnRow())
}

func (f *flow) rememberCadenceRung(rung int) {
	rung = terrain.ClampCadenceRung(rung)
	changed := f.cadencePreference() != rung
	f.preferredRung, f.preferredRungSet = rung, true
	if changed && f.persistRung != nil {
		f.persistRung(rung)
	}
}

func (f *flow) stepMenuSpeed(delta int) {
	f.setMenuSpeedRung(f.menuSpeedRung() + delta)
}
