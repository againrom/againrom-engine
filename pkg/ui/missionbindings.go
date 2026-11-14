package ui

import (
	"againrom/pkg/namefold"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type missionKeyAction uint8

const (
	missionKeyPatrol missionKeyAction = iota
	missionKeyGrab
	missionKeyPickup
	missionKeyGuard
	missionKeyDefend
	missionKeyCast
	missionKeyMarch
	missionKeyStandGround
	missionKeyRetreat
	missionKeyMove
	missionKeyQuickSave
	missionKeyQuickLoad
	missionKeyGrid
	missionKeyReadout
	missionKeyFPS
	missionKeyReveal
	missionKeyShortcut
	missionKeySupport
	missionKeyAttack
	missionKeyAutocast
	missionKeyWeaponSwap
	missionKeyRetreatSetting
	missionKeyFormation
	missionKeyTimeFlow
	missionKeySmoothing
	missionKeyAutoHealing
	missionKeyShowHealth
	missionKeyNumerals
	missionKeyKill
	missionKeyChip
	missionKeySelectAll
	missionKeyInventory
	missionKeyWorn
	missionKeyBook
	missionKeyDoll
	missionKeyPanels
	missionKeyHighlight
	missionKeyHelp
	missionKeySaveGame
	missionKeyLoadGame
	missionKeyPauseText
	missionKeyPause
	missionKeyFaster
	missionKeySlower
	missionKeyUnpaced
	missionKeyPaced
	missionKeyZoomReset
	missionKeySpeedReset
	missionKeyClearMessages
	missionKeyGroup
)

// keyMod is what a row asks of one modifier: either state, held, or up.
type keyMod uint8

const (
	modAny keyMod = iota
	modHeld
	modFree
)

func (m keyMod) admits(down bool) bool { return m == modAny || (m == modHeld) == down }

// missionKeyGames are the games a row binds in; zero is both.
type missionKeyGames uint8

const (
	firstGameKeys missionKeyGames = 1 << iota
	secondGameKeys
)

// missionKeyBinding is one row: a key with what it asks of Ctrl, Shift and
// Alt, a press edge or (hold) a level, and the action it sets. arg is the
// quick-spell slot or the group digit of the actions that take one.
type missionKeyBinding struct {
	only             missionKeyGames
	name             string
	key              ebiten.Key
	ctrl, shift, alt keyMod
	hold             bool
	action           missionKeyAction
	arg              int
}

// missionKeyPolicy is a profile's data beside its rows: the game whose rows
// it reads and the Ctrl its order letters ask (the second game refuses it).
type missionKeyPolicy struct {
	games  missionKeyGames
	orders keyMod
}

var missionKeyPolicies = map[missionKeyAction]missionKeyPolicy{
	missionKeyPatrol: {games: firstGameKeys},
	missionKeyPickup: {games: secondGameKeys, orders: modFree},
}

var missionKeyProfiles = map[string]missionKeyAction{"pickup-p": missionKeyPickup}

// MissionShortcutCount is the number of spell keys in a mission profile.
func MissionShortcutCount(name string) int {
	slots := map[int]bool{}
	for _, b := range missionBindings(missionKeyProfiles[name]) {
		if b.action == missionKeyShortcut {
			slots[b.arg] = true
		}
	}
	return len(slots)
}

// missionShortcutOffset is the function-key number of a profile's first
// spell slot less five, the offset the book's slot marks are drawn with.
func missionShortcutOffset(p missionKeyAction) int {
	for _, b := range missionBindings(p) {
		if b.action == missionKeyShortcut && b.arg == 0 && b.shift == modFree {
			return int(b.key-ebiten.KeyF4) + 4 - 5
		}
	}
	return 0
}

// SetMissionKeys selects the edition's mission bindings.
func (a *App) SetMissionKeys(name string) { a.missionKeys = missionKeyProfiles[name] }

func (a *App) activeMissionKeys() missionKeyAction {
	if a.flow != nil && a.flow.mapShowing() {
		return a.missionKeys
	}
	return missionKeyPatrol
}

// missionBindings is the mission key table: every key the mission reads, as
// rows of profile p. A profile is named by the action of its P key.
func missionBindings(p missionKeyAction) []missionKeyBinding {
	policy := missionKeyPolicies[p]
	orders := policy.orders
	table := []missionKeyBinding{
		// Orders (AI-KEY-125; DIV-236, DIV-237, DIV-4888).
		{name: "f", key: ebiten.KeyF, ctrl: modFree, action: missionKeyGrab},
		{name: "p", key: ebiten.KeyP, ctrl: orders, action: p},
		{name: "g", key: ebiten.KeyG, ctrl: orders, action: missionKeyGuard},
		{name: "d", key: ebiten.KeyD, ctrl: orders, action: missionKeyDefend},
		{name: "c", key: ebiten.KeyC, ctrl: orders, action: missionKeyCast},
		{name: "s", key: ebiten.KeyS, ctrl: orders, action: missionKeyMarch},
		{name: "t", key: ebiten.KeyT, ctrl: orders, action: missionKeyStandGround},
		{name: "r", key: ebiten.KeyR, ctrl: orders, action: missionKeyRetreat},
		{name: "m", key: ebiten.KeyM, ctrl: orders, action: missionKeyMove},
		{only: secondGameKeys, name: "l", key: ebiten.KeyL, ctrl: modFree, action: missionKeySupport},
		{name: "a", key: ebiten.KeyA, ctrl: modFree, action: missionKeyAttack},
		{name: "a", key: ebiten.KeyA, ctrl: modHeld, action: missionKeyAutocast},
		{name: "w", key: ebiten.KeyW, ctrl: modFree, action: missionKeyWeaponSwap},
		// Settings under Ctrl (MENU-057, ANIM-NUM-020, keyboard.tsv rows 48 and 50).
		{name: "w", key: ebiten.KeyW, ctrl: modHeld, action: missionKeyRetreatSetting},
		{name: "f", key: ebiten.KeyF, ctrl: modHeld, action: missionKeyFormation},
		{name: "n", key: ebiten.KeyN, ctrl: modHeld, action: missionKeyTimeFlow},
		{name: "o", key: ebiten.KeyO, ctrl: modHeld, action: missionKeySmoothing},
		{name: "u", key: ebiten.KeyU, ctrl: modHeld, action: missionKeyAutoHealing},
		{name: "h", key: ebiten.KeyH, ctrl: modHeld, action: missionKeyShowHealth},
		{name: "l", key: ebiten.KeyL, ctrl: modHeld, action: missionKeyNumerals},
		// Debug chords; the map arm asks the cheat unlock (DIV-4862, DIV-2178).
		{name: "k", key: ebiten.KeyK, ctrl: modHeld, shift: modHeld, action: missionKeyKill},
		{name: "semicolon", key: ebiten.KeySemicolon, ctrl: modHeld, shift: modHeld, action: missionKeyChip},
		{name: "f4", key: ebiten.KeyF4, ctrl: modHeld, shift: modHeld, action: missionKeyReveal},
		{name: "f10", key: ebiten.KeyF10, ctrl: modHeld, shift: modHeld, action: missionKeyGrid},
		{name: "f11", key: ebiten.KeyF11, ctrl: modHeld, shift: modHeld, action: missionKeyReadout},
		// Selection and panels (AI-SELECT-122, keyboard.tsv; DIV-296, DIV-1817).
		{name: "e", key: ebiten.KeyE, action: missionKeySelectAll},
		{name: "i", key: ebiten.KeyI, action: missionKeyInventory},
		{name: "backquote", key: ebiten.KeyBackquote, action: missionKeyInventory},
		{name: "x", key: ebiten.KeyX, action: missionKeyWorn},
		{name: "b", key: ebiten.KeyB, action: missionKeyBook},
		{name: "q", key: ebiten.KeyQ, action: missionKeyBook},
		{name: "j", key: ebiten.KeyJ, action: missionKeyDoll},
		{name: "space", key: ebiten.KeySpace, shift: modFree, alt: modFree, action: missionKeyPanels},
		{name: "z", key: ebiten.KeyZ, ctrl: modFree, alt: modFree, hold: true, action: missionKeyHighlight},
		{name: "f1", key: ebiten.KeyF1, action: missionKeyHelp},
		{name: "pause", key: ebiten.KeyPause, action: missionKeyPauseText},
		// Function keys (AI-KEY-125; DIV-526, DIV-4885, DIV-4887).
		{name: "f2", key: ebiten.KeyF2, ctrl: modFree, action: missionKeyQuickSave},
		{name: "f2", key: ebiten.KeyF2, ctrl: modHeld, action: missionKeySaveGame},
		{name: "f3", key: ebiten.KeyF3, ctrl: modFree, action: missionKeyQuickLoad},
		{name: "f3", key: ebiten.KeyF3, ctrl: modHeld, action: missionKeyLoadGame},
		{only: firstGameKeys, name: "f4", key: ebiten.KeyF4, shift: modFree, action: missionKeyQuickSave},
		{only: firstGameKeys, name: "f5", key: ebiten.KeyF5, shift: modFree, action: missionKeyShortcut, arg: 0},
		{only: firstGameKeys, name: "f6", key: ebiten.KeyF6, shift: modFree, action: missionKeyShortcut, arg: 1},
		{only: firstGameKeys, name: "f7", key: ebiten.KeyF7, shift: modFree, action: missionKeyShortcut, arg: 2},
		{only: firstGameKeys, name: "f8", key: ebiten.KeyF8, shift: modFree, action: missionKeyShortcut, arg: 3},
		{only: firstGameKeys, name: "f9", key: ebiten.KeyF9, shift: modFree, action: missionKeyQuickLoad},
		{only: firstGameKeys, name: "f10", key: ebiten.KeyF10, shift: modFree, action: missionKeyGrid},
		{only: firstGameKeys, name: "f11", key: ebiten.KeyF11, shift: modFree, action: missionKeyReadout},
		{only: firstGameKeys, name: "f12", key: ebiten.KeyF12, shift: modFree, action: missionKeyFPS},
		{only: secondGameKeys, name: "f4", key: ebiten.KeyF4, shift: modFree, action: missionKeyShortcut, arg: 0},
		{only: secondGameKeys, name: "f5", key: ebiten.KeyF5, shift: modFree, action: missionKeyShortcut, arg: 1},
		{only: secondGameKeys, name: "f6", key: ebiten.KeyF6, shift: modFree, action: missionKeyShortcut, arg: 2},
		{only: secondGameKeys, name: "f7", key: ebiten.KeyF7, shift: modFree, action: missionKeyShortcut, arg: 3},
		{only: secondGameKeys, name: "f8", key: ebiten.KeyF8, shift: modFree, action: missionKeyShortcut, arg: 4},
		{only: secondGameKeys, name: "f9", key: ebiten.KeyF9, shift: modFree, action: missionKeyShortcut, arg: 5},
		{only: secondGameKeys, name: "f10", key: ebiten.KeyF10, shift: modFree, action: missionKeyShortcut, arg: 6},
		{only: secondGameKeys, name: "f11", key: ebiten.KeyF11, shift: modFree, action: missionKeyShortcut, arg: 7},
		{only: secondGameKeys, name: "f12", key: ebiten.KeyF12, shift: modFree, action: missionKeyShortcut, arg: 8},
		{name: "f5", key: ebiten.KeyF5, shift: modHeld, action: missionKeyShortcut, arg: 0},
		{name: "f6", key: ebiten.KeyF6, shift: modHeld, action: missionKeyShortcut, arg: 1},
		{name: "f7", key: ebiten.KeyF7, shift: modHeld, action: missionKeyShortcut, arg: 2},
		{name: "f8", key: ebiten.KeyF8, shift: modHeld, action: missionKeyShortcut, arg: 3},
		{name: "f10", key: ebiten.KeyF10, shift: modHeld, action: missionKeyGrid},
		{name: "f11", key: ebiten.KeyF11, shift: modHeld, action: missionKeyReadout},
		{name: "f12", key: ebiten.KeyF12, shift: modHeld, action: missionKeyFPS},
		// The clock and the camera (SESS-CLOCK-005, keyboard.tsv rows 21..24;
		// DIV-2263, DIV-2950).
		{name: "0", key: ebiten.KeyDigit0, ctrl: modFree, shift: modFree, alt: modFree, action: missionKeyPause},
		{name: "numpad-0", key: ebiten.KeyNumpad0, ctrl: modFree, shift: modFree, alt: modFree, action: missionKeyPause},
		{name: "equal", key: ebiten.KeyEqual, action: missionKeyFaster},
		{name: "numpad-plus", key: ebiten.KeyNumpadAdd, ctrl: modFree, action: missionKeyFaster},
		{name: "minus", key: ebiten.KeyMinus, action: missionKeySlower},
		{name: "numpad-minus", key: ebiten.KeyNumpadSubtract, ctrl: modFree, action: missionKeySlower},
		{name: "numpad-plus", key: ebiten.KeyNumpadAdd, ctrl: modHeld, action: missionKeyUnpaced},
		{name: "numpad-minus", key: ebiten.KeyNumpadSubtract, ctrl: modHeld, action: missionKeyPaced},
		{name: "home", key: ebiten.KeyHome, ctrl: modFree, shift: modFree, action: missionKeyZoomReset},
		{name: "end", key: ebiten.KeyEnd, ctrl: modFree, shift: modFree, action: missionKeySpeedReset},
		{name: "backspace", key: ebiten.KeyBackspace, alt: modFree, action: missionKeyClearMessages},
		// Groups: the digit row and the numpad's digits are one input.
		{name: "0", key: ebiten.KeyDigit0, action: missionKeyGroup, arg: 0},
		{name: "1", key: ebiten.KeyDigit1, action: missionKeyGroup, arg: 1},
		{name: "2", key: ebiten.KeyDigit2, action: missionKeyGroup, arg: 2},
		{name: "3", key: ebiten.KeyDigit3, action: missionKeyGroup, arg: 3},
		{name: "4", key: ebiten.KeyDigit4, action: missionKeyGroup, arg: 4},
		{name: "5", key: ebiten.KeyDigit5, action: missionKeyGroup, arg: 5},
		{name: "6", key: ebiten.KeyDigit6, action: missionKeyGroup, arg: 6},
		{name: "7", key: ebiten.KeyDigit7, action: missionKeyGroup, arg: 7},
		{name: "8", key: ebiten.KeyDigit8, action: missionKeyGroup, arg: 8},
		{name: "9", key: ebiten.KeyDigit9, action: missionKeyGroup, arg: 9},
		{name: "numpad-0", key: ebiten.KeyNumpad0, action: missionKeyGroup, arg: 0},
		{name: "numpad-1", key: ebiten.KeyNumpad1, action: missionKeyGroup, arg: 1},
		{name: "numpad-2", key: ebiten.KeyNumpad2, action: missionKeyGroup, arg: 2},
		{name: "numpad-3", key: ebiten.KeyNumpad3, action: missionKeyGroup, arg: 3},
		{name: "numpad-4", key: ebiten.KeyNumpad4, action: missionKeyGroup, arg: 4},
		{name: "numpad-5", key: ebiten.KeyNumpad5, action: missionKeyGroup, arg: 5},
		{name: "numpad-6", key: ebiten.KeyNumpad6, action: missionKeyGroup, arg: 6},
		{name: "numpad-7", key: ebiten.KeyNumpad7, action: missionKeyGroup, arg: 7},
		{name: "numpad-8", key: ebiten.KeyNumpad8, action: missionKeyGroup, arg: 8},
		{name: "numpad-9", key: ebiten.KeyNumpad9, action: missionKeyGroup, arg: 9},
	}
	rows := table[:0]
	for _, b := range table {
		if b.only == 0 || b.only&policy.games != 0 {
			rows = append(rows, b)
		}
	}
	return rows
}

// readMissionKeys reads the rows of p with one test for press edges and
// levels alike, as a named key or a single test key is.
func (in *appInput) readMissionKeys(p missionKeyAction, pressed func(ebiten.Key) bool, ctrl bool) {
	in.readMissionKeyRows(p, pressed, pressed, ctrl)
}

// readMissionKeyRows is the one reader of mission keys: every row of p whose
// key is down and whose modifiers admit Ctrl, Shift and Alt sets its action.
// Where two rows set one field, the lowest group digit wins, and a bare zero
// is the clock pause and not group 0 (DIV-2263).
func (in *appInput) readMissionKeyRows(p missionKeyAction, pressed, held func(ebiten.Key) bool, ctrl bool) {
	for _, b := range missionBindings(p) {
		down := pressed
		if b.hold {
			down = held
		}
		if !down(b.key) || !b.ctrl.admits(ctrl) || !b.shift.admits(in.Viewer.Shift) || !b.alt.admits(in.Viewer.Alt) {
			continue
		}
		switch b.action {
		case missionKeyPatrol:
			in.Patrol = true
		case missionKeyGrab:
			in.Grab = true
		case missionKeyPickup:
			in.Grab, in.GrabReply = true, true
		case missionKeyGuard:
			in.Guard = true
		case missionKeyDefend:
			in.Defend = true
		case missionKeyCast:
			in.Cast = true
		case missionKeyMarch:
			in.March = true
		case missionKeyStandGround:
			in.StandGround = true
		case missionKeyRetreat:
			in.PlayerRetreat = true
		case missionKeyMove:
			in.Move = true
		case missionKeySupport:
			in.Support = true
		case missionKeyQuickSave:
			in.QuickSave = true
		case missionKeyQuickLoad:
			in.QuickLoad = true
		case missionKeyGrid:
			in.Grid = true
		case missionKeyReadout:
			in.Readout = true
		case missionKeyFPS:
			in.FPS = true
		case missionKeyReveal:
			in.Reveal = true
		case missionKeyShortcut:
			in.QuickSpell[b.arg] = true
		case missionKeyAttack:
			in.Attack = true
		case missionKeyAutocast:
			in.Autocast = true
		case missionKeyWeaponSwap:
			in.WeaponSwap = true
		case missionKeyRetreatSetting:
			in.Retreat = true
		case missionKeyFormation:
			in.Formation = true
		case missionKeyTimeFlow:
			in.TimeFlow = true
		case missionKeySmoothing:
			in.Smoothing = true
		case missionKeyAutoHealing:
			in.AutoHealing = true
		case missionKeyShowHealth:
			in.ShowHealth = true
		case missionKeyNumerals:
			in.Numerals = true
		case missionKeyKill:
			in.Kill = true
		case missionKeyChip:
			in.Chip = true
		case missionKeySelectAll:
			in.SelectAll = true
		case missionKeyInventory:
			in.Inventory = true
		case missionKeyWorn:
			in.Worn = true
		case missionKeyBook:
			in.Book = true
		case missionKeyDoll:
			in.Doll = true
		case missionKeyPanels:
			in.Panels = true
		case missionKeyHighlight:
			in.HighlightHeld = true
		case missionKeyHelp:
			in.Help = true
		case missionKeySaveGame:
			in.SaveGame = true
		case missionKeyLoadGame:
			in.LoadGame = true
		case missionKeyPauseText:
			in.PauseText = true
		case missionKeyPause:
			in.Pause = true
		case missionKeyFaster:
			in.Faster = true
		case missionKeySlower:
			in.Slower = true
		case missionKeyUnpaced:
			in.Unpaced = true
		case missionKeyPaced:
			in.Paced = true
		case missionKeyZoomReset:
			in.ZoomReset = true
		case missionKeySpeedReset:
			in.SpeedReset = true
		case missionKeyClearMessages:
			in.ClearMessages = true
		case missionKeyGroup:
			if !in.GroupKey || b.arg < in.GroupDigit {
				in.GroupKey, in.GroupDigit = true, b.arg
			}
		}
	}
	if in.Pause {
		in.GroupKey = false
	}
}

// namedMissionKey is the press test of the key a row of p is named by.
func namedMissionKey(p missionKeyAction, name string) (func(ebiten.Key) bool, bool) {
	for _, b := range missionBindings(p) {
		if b.name == name {
			return onlyKey(b.key), true
		}
	}
	return nil, false
}

// missionKeyNamed presses the key a row is named by, as a scenario names it:
// "ctrl-" and "shift-" hold those modifiers for the press.
func (in *appInput) missionKeyNamed(p missionKeyAction, name string) bool {
	name = namefold.ArchiveText(strings.TrimSpace(name))
	ctrl, shift := in.Viewer.Ctrl, in.Viewer.Shift
	if strings.HasPrefix(name, "ctrl-") {
		ctrl, name = true, strings.TrimPrefix(name, "ctrl-")
	}
	if strings.HasPrefix(name, "shift-") {
		shift, name = true, strings.TrimPrefix(name, "shift-")
	}
	key, ok := namedMissionKey(p, name)
	if !ok {
		return false
	}
	if ctrl != in.Viewer.Ctrl || shift != in.Viewer.Shift {
		in.AttackHeld, in.ShiftHeld = ctrl, shift
	}
	in.Viewer.Ctrl, in.Viewer.Shift = ctrl, shift
	in.readMissionKeys(p, key, in.Viewer.Ctrl)
	return true
}
