package ui

import (
	"fmt"
	"strconv"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// The slider adds player pause before the nine shipped positive speeds.
const gameSpeedLevels = 9

// gameOptionsDraft is the Game Options dialog's local copy of every value it
// edits (MENU-073, MENU-074). A click changes the draft only. OK writes the
// draft; Cancel, Escape and any other way off the page drop it, and nothing
// outside the dialog is restored because nothing outside was changed.
type gameOptionsDraft struct {
	autosave      TimedAutosaveSettings
	autosaveStart TimedAutosaveSettings
	values        GameOptionValues
	tips          bool
	tipsStart     bool
	speed         int
	speedStart    int
	speedChanged  bool
	delay         int
	delayStart    int
	slider        sliderInput
	// radio is the radio group a held press began on, or 0.
	radio gameMenuAction
}

func (f *flow) newGameOptionsDraft() *gameOptionsDraft {
	d := &gameOptionsDraft{values: f.readGameOptions()}
	if f.gameOptions.Autosave.Read != nil {
		d.autosave = f.gameOptions.Autosave.Read()
	}
	d.autosaveStart = d.autosave
	if f.menuTips != nil {
		d.tips = f.menuTips()
	}
	d.tipsStart = d.tips
	d.speed = min(max(f.menuSpeedRung()-terrain.CadenceShippedLo+1, 1), gameSpeedLevels)
	if f.menuBack == ScreenMap && f.stopped {
		d.speed = 0
	}
	d.speedStart = d.speed
	d.delay = DefaultTooltipDelay
	if f.tooltip != nil {
		d.delay = f.tooltip.delay
	}
	d.delayStart = d.delay
	return d
}

// optionValues is what the page shows: the draft while the page is open, the
// stored state otherwise.
func (f *flow) optionValues() GameOptionValues {
	if f.gameOptions.draft != nil {
		return f.gameOptions.draft.values
	}
	return f.readGameOptions()
}

// cycleDraftOption advances one checkbox or radio group of the local copy.
func (f *flow) cycleDraftOption(o GameOption) {
	d := f.gameOptions.draft
	if d == nil || o >= gameOptionCount {
		return
	}
	d.values[o] = (max(d.values[o], -1) + 1) % o.Choices()
}

func (f *flow) setDraftOption(o GameOption, value int) {
	if d := f.gameOptions.draft; d != nil && value >= 0 && value < o.Choices() {
		d.values[o] = value
	}
}

func (f *flow) setDraftSpeed(level int) {
	if d := f.gameOptions.draft; d != nil {
		d.speed = min(max(level, 0), gameSpeedLevels)
		if f.menuBack != ScreenMap && d.speed == 0 {
			d.speed = 1
		}
		d.speedChanged = true
	}
}

func (f *flow) cycleDraftTooltipDelay() {
	if d := f.gameOptions.draft; d != nil && f.tooltip != nil {
		d.delay = (d.delay + 100) % 600
	}
}

func (f *flow) toggleDraftTips() {
	if d := f.gameOptions.draft; d != nil && f.menuTips != nil && f.setMenuTips != nil {
		d.tips = !d.tips
	}
}

// setMenuSpeedRung applies one absolute speed rung: it is remembered as the
// preference and, over a map, becomes the running cadence.
func (f *flow) setMenuSpeedRung(rung int) {
	rung = terrain.ClampCadenceRung(rung)
	f.rememberCadenceRung(rung)
	if f.menuBack == ScreenMap {
		reset := f.unpaced
		f.rung, f.unpaced, f.stopped = rung, false, false
		f.syncCadence(reset)
	}
}

// commitGameOptions is the OK arm (MENU-074). It writes the speed, Tips and the
// Day/Night flag, then the four graphics flags with Animation 0 forcing
// Lighting 0, then the party flags, then sends the formation, retreat and
// autohealing commands on every OK, changed or not. It reports the first
// failed write and carries on with the rest.
func (f *flow) commitGameOptions() error {
	d := f.gameOptions.draft
	if d == nil {
		return nil
	}
	var first error
	if d.autosave != d.autosaveStart && f.gameOptions.Autosave.Write != nil {
		if err := f.gameOptions.Autosave.Write(d.autosave); err != nil {
			return err
		}
		d.autosaveStart = d.autosave
	}
	note := func(err error) {
		if err != nil && first == nil {
			first = err
		}
	}
	if f.menuTips != nil && f.setMenuTips != nil && d.tips != d.tipsStart {
		f.setMenuTips(d.tips)
	}
	v := d.values
	if v[GameOptionAnimation] == 0 {
		v[GameOptionLighting] = 0
	}
	write := func(o GameOption) {
		if v[o] < 0 || f.gameOptions.Write == nil {
			return
		}
		note(f.gameOptions.Write(f.gameOptionsOnMap(), o, v[o]))
	}
	for _, o := range []GameOption{GameOptionDayNight, GameOptionSmoothing, GameOptionShadows, GameOptionLighting,
		GameOptionAnimation, GameOptionHealth, GameOptionDamage, GameOptionPathfinding,
		GameOptionFormation, GameOptionRetreat, GameOptionAutoHealing} {
		write(o)
	}
	if f.tooltip != nil && d.delay != d.delayStart {
		f.tooltip.setDelay(d.delay)
		if f.persistTooltipDelay != nil {
			note(f.persistTooltipDelay(d.delay))
		}
	}
	if first == nil && d.speedChanged {
		if d.speed == 0 && f.menuBack == ScreenMap {
			f.stopped = true
			f.syncCadence(false)
		} else {
			f.setMenuSpeedRung(terrain.CadenceShippedLo + d.speed - 1)
		}
	}
	return first
}

// gameOptionsPageRows lists the page's controls in focus order. The speed
// slider is two rows that share one rectangle.
func (f *flow) gameOptionsPageRows() []gameMenuRow {
	d := f.gameOptions.draft
	if d == nil {
		d = f.newGameOptionsDraft()
	}
	w := f.gameOptions.Words
	speed := w.Speed
	if speed == "" {
		speed = "Game Speed"
	}
	rows := []gameMenuRow{
		{Label: speed + ": " + strconv.Itoa(d.speed), Literal: true, Status: true},
		{Label: "~SLOWER", Fallback: 'S', Action: gameMenuSpeedDown, Enabled: d.speed > 0},
		{Label: "~FASTER", Fallback: 'F', Action: gameMenuSpeedUp, Enabled: d.speed < gameSpeedLevels},
	}
	if d.speed == 0 {
		rows[0].Label = speed + ": " + f.pauseLabel()
	}
	if f.menuSelector() == text.SelectorConverting {
		rows[1].Label, rows[2].Label = f.menuDisplayText(gameSpeedRU.Slower), f.menuDisplayText(gameSpeedRU.Faster)
	}
	enabled := f.gameOptions.Write != nil
	option := func(o GameOption) gameMenuRow {
		state := "OFF"
		if d.values[o] != 0 {
			state = "ON"
		}
		if d.values[o] >= 0 && d.values[o] < 3 {
			switch o {
			case GameOptionFormation:
				state = w.Formation[d.values[o]]
			case GameOptionRetreat:
				state = w.Retreat[d.values[o]]
			case GameOptionAutoHealing:
				state = w.AutoHealing[d.values[o]]
			}
		}
		if o == GameOptionAutoHealing && d.values[o] < 0 {
			state = "--"
		}
		return gameMenuRow{Label: w.Labels[o] + ": " + state, Literal: true, Enabled: enabled, Action: optionAction(o)}
	}
	for _, o := range []GameOption{GameOptionDayNight, GameOptionSmoothing, GameOptionShadows,
		GameOptionLighting, GameOptionAnimation} {
		rows = append(rows, option(o))
	}
	delay := f.tooltipDelayRow()
	label, unit := "Tooltip delay: ", " ms"
	if f.menuSelector() == text.SelectorConverting {
		label, unit = f.menuDisplayText(tooltipRU.Delay), f.menuDisplayText(tooltipRU.Unit)
	}
	delay.Label = label + strconv.Itoa(d.delay) + unit
	rows = append(rows, delay, option(GameOptionHealth), option(GameOptionDamage))
	tips := gameMenuRow{Label: w.Tips + ": " + onOff(d.tips), Literal: true, Action: gameMenuToggleTips,
		Enabled: f.menuTips != nil && f.setMenuTips != nil}
	rows = append(rows, tips, option(GameOptionAutoHealing), option(GameOptionPathfinding),
		option(GameOptionFormation), option(GameOptionRetreat))
	if f.gameOptions.Autosave.Read != nil {
		toggle, interval := f.timedAutosaveLabels(d)
		rows = append(rows,
			gameMenuRow{Label: toggle + ": " + onOff(d.autosave.Enabled), Literal: true, Action: gameMenuTimedAutosave, Enabled: f.gameOptions.Autosave.Write != nil},
			gameMenuRow{Label: interval, Literal: true, Action: gameMenuAutosaveMinutes, Enabled: f.gameOptions.Autosave.Write != nil})
	}
	ok, cancel := w.OK, w.Cancel
	if ok == "" {
		ok = "OK"
	}
	if cancel == "" {
		cancel = "Cancel"
	}
	return append(rows,
		gameMenuRow{Label: ok, Fallback: 'O', Action: gameMenuPageReturn, Enabled: true},
		gameMenuRow{Label: cancel, Fallback: 'C', Action: gameMenuOptionsCancel, Enabled: true})
}

func onOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}

// gameOptionsFailure is the line a failed OK reports.
func gameOptionsFailure(err error) string { return fmt.Sprintf("Setting not saved: %v", err) }
