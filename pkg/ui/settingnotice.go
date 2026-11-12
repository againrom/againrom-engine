package ui

import (
	"fmt"
	"time"

	"againrom/pkg/render/terrain"
)

// The first main.txt line of each setting's state lines: the line a settings
// key posts is the base plus the setting's new state (MENU-057). The speed
// step's base is noticeSlotSpeed plus the speed index (MENU-058).
const (
	noticeSlotRetreat      = 94
	noticeSlotFormation    = 97
	noticeSlotShowHealth   = 100
	noticeSlotFlyingDamage = 102
	noticeSlotDayNight     = 104
	noticeSlotSmoothing    = 106
	noticeSlotSpeed        = 108
	noticeSlotAutoHealing  = 218

	// noticeSlotZoom is the first slot past main.txt's: the map zoom, whose
	// state is the zoom in percent (DIV-4877).
	noticeSlotZoom = len(Words{}.SettingNotice)

	// settingNoticeLife is how long a notice line stays once it is the oldest
	// on the message line (MENU-059).
	settingNoticeLife = 2000 * time.Millisecond
)

func noticeState(on bool) int {
	if on {
		return 1
	}
	return 0
}

// engineNotice is a setting notice slot past main.txt's: the engine word whose
// format takes the state, and whether the slot's line replaces the line it
// posted before instead of appending below it.
type engineNotice struct {
	format   func(Words) string
	replaces bool
}

var engineNotices = map[int]engineNotice{
	noticeSlotZoom: {format: func(w Words) string { return w.ZoomNotice }, replaces: true},
}

// postSettingNotice posts main.txt line slot+state in the grey ramp, appended
// below the lines standing: a toggle never drops a duplicate (MENU-057,
// MENU-059). A line the install did not state posts nothing. An engine slot
// posts its word formatted with the state, in the same ramp and life.
func (v *Viewer) postSettingNotice(base, state int) {
	if n, ok := engineNotices[base]; ok {
		format := n.format(v.words)
		if state < 0 || format == "" {
			return
		}
		slot := 0
		if n.replaces {
			slot = base
		}
		v.postMessageSlot(fmt.Sprintf(format, state), MessageGrey, settingNoticeLife, slot)
		return
	}
	slot := base + state
	if state < 0 || slot >= len(v.words.SettingNotice) || v.words.SettingNotice[slot] == "" {
		return
	}
	v.PostMessage(v.words.SettingNotice[slot], MessageGrey, settingNoticeLife)
}

// postSpeedNotice posts the line of the speed index the cadence rung stands
// on, clamped to the game's nine speeds, unless it equals the newest line
// (MENU-058).
func (v *Viewer) postSpeedNotice(rung int) {
	index := min(max(rung-terrain.CadenceShippedLo, terrain.SpeedIndexMin), terrain.SpeedIndexMax)
	slot := noticeSlotSpeed + index
	if v.words.SettingNotice[slot] == "" {
		return
	}
	v.PostMessageUnlessNewest(v.words.SettingNotice[slot], MessageGrey, settingNoticeLife)
}

// postOptionNotice posts the line for the state the option holds after a
// shortcut changed it.
func (a *App) postOptionNotice(v *Viewer, o GameOption, base int) {
	v.postSettingNotice(base, a.flow.readGameOptions()[o])
}
