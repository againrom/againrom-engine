package ui

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// Authored translations enter the same UTF-8-to-installed-font conversion as
// map prose. Production Go literals remain ASCII under the drawn-text guard.
//
//go:embed gamemenuspeed_ru.json
var gameSpeedRUText string

var gameSpeedRU = func() (labels struct{ Speed, Slower, Faster, UnpacedSpeed, Unpaced, Paused string }) {
	if err := json.Unmarshal([]byte(gameSpeedRUText), &labels); err != nil {
		panic(err)
	}
	return labels
}()

func (f *flow) pauseLabel() string {
	if f.menuSelector() == text.SelectorConverting {
		return f.menuDisplayText(gameSpeedRU.Paused)
	}
	return "PAUSED"
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
	label, down, up := "GAME SPEED: ", "~SLOWER", "~FASTER"
	if unpaced {
		value = "UNLIMITED"
	}
	if f.menuSelector() == text.SelectorConverting {
		label, down, up = gameSpeedRU.Speed, gameSpeedRU.Slower, gameSpeedRU.Faster
		if unpaced {
			label, value = gameSpeedRU.UnpacedSpeed, gameSpeedRU.Unpaced
		}
		label, down, up = f.menuDisplayText(label), f.menuDisplayText(down), f.menuDisplayText(up)
		value = f.menuDisplayText(value)
	}
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
