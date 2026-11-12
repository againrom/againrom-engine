package ui

import (
	"againrom/pkg/namefold"
	"strconv"
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
	missionKeyNone
	missionKeyShortcut
	missionKeySupport
)

type missionKeyBinding struct {
	name      string
	key       ebiten.Key
	action    missionKeyAction
	noCtrl    bool
	noShift   bool
	shiftOnly bool
	debug     bool
}

type missionKeyPolicy struct {
	noCtrl bool
	// supportKey binds bare L to the second game's L order (R2-ENGINE-1179).
	supportKey    bool
	shortcutFirst int
	functionKeys  [9]missionKeyAction
	shiftKeys     [9]missionKeyAction
}

var missionKeyPolicies = map[missionKeyAction]missionKeyPolicy{
	missionKeyPatrol: {shortcutFirst: 5, functionKeys: [9]missionKeyAction{missionKeyQuickSave, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyQuickLoad, missionKeyGrid, missionKeyReadout, missionKeyFPS}, shiftKeys: [9]missionKeyAction{missionKeyNone, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyQuickLoad, missionKeyGrid, missionKeyReadout, missionKeyFPS}},
	missionKeyPickup: {noCtrl: true, supportKey: true, shortcutFirst: 4, functionKeys: [9]missionKeyAction{missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut}, shiftKeys: [9]missionKeyAction{missionKeyQuickSave, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyShortcut, missionKeyQuickLoad, missionKeyGrid, missionKeyReadout, missionKeyFPS}},
}

var missionKeyProfiles = map[string]missionKeyAction{"pickup-p": missionKeyPickup}

// MissionShortcutCount is the number of spell keys in a mission profile.
func MissionShortcutCount(name string) int {
	n := 0
	for _, action := range missionKeyPolicies[missionKeyProfiles[name]].functionKeys {
		if action == missionKeyShortcut {
			n++
		}
	}
	return n
}

// SetMissionKeys selects the edition's mission bindings.
func (a *App) SetMissionKeys(name string) { a.missionKeys = missionKeyProfiles[name] }

func (a *App) activeMissionKeys() missionKeyAction {
	if a.flow != nil && a.flow.mapShowing() {
		return a.missionKeys
	}
	return missionKeyPatrol
}

func missionBindings(p missionKeyAction) []missionKeyBinding {
	policy := missionKeyPolicies[p]
	rows := []missionKeyBinding{
		{name: "f", key: ebiten.KeyF, action: missionKeyGrab, noCtrl: true},
		{name: "p", key: ebiten.KeyP, action: p, noCtrl: policy.noCtrl},
		{name: "g", key: ebiten.KeyG, action: missionKeyGuard, noCtrl: policy.noCtrl},
		{name: "d", key: ebiten.KeyD, action: missionKeyDefend, noCtrl: policy.noCtrl},
		{name: "c", key: ebiten.KeyC, action: missionKeyCast, noCtrl: policy.noCtrl},
		{name: "s", key: ebiten.KeyS, action: missionKeyMarch, noCtrl: policy.noCtrl},
		{name: "t", key: ebiten.KeyT, action: missionKeyStandGround, noCtrl: policy.noCtrl},
		{name: "r", key: ebiten.KeyR, action: missionKeyRetreat, noCtrl: policy.noCtrl},
		{name: "m", key: ebiten.KeyM, action: missionKeyMove, noCtrl: policy.noCtrl},
	}
	if policy.supportKey {
		rows = append(rows, missionKeyBinding{name: "l", key: ebiten.KeyL, action: missionKeySupport, noCtrl: true})
	}
	for i, action := range policy.functionKeys {
		rows = append(rows, missionKeyBinding{name: "f" + strconv.Itoa(i+4), key: ebiten.Key(int(ebiten.KeyF4) + i), action: action, noShift: true})
	}
	for i, action := range policy.shiftKeys {
		rows = append(rows, missionKeyBinding{name: "f" + strconv.Itoa(i+4), key: ebiten.Key(int(ebiten.KeyF4) + i), action: action, shiftOnly: true, noCtrl: action == missionKeyQuickSave})
	}
	for _, row := range []missionKeyBinding{
		{name: "f4", key: ebiten.KeyF4, action: missionKeyReveal, debug: true},
		{name: "f10", key: ebiten.KeyF10, action: missionKeyGrid, debug: true},
		{name: "f11", key: ebiten.KeyF11, action: missionKeyReadout, debug: true},
	} {
		rows = append(rows, row)
	}
	return rows
}

func (in *appInput) readMissionKeys(p missionKeyAction, pressed func(ebiten.Key) bool, ctrl bool) {
	policy := missionKeyPolicies[p]
	for _, b := range missionBindings(p) {
		if !pressed(b.key) || b.noCtrl && ctrl || b.noShift && in.Viewer.Shift || b.shiftOnly && !in.Viewer.Shift || b.debug && !(ctrl && in.Viewer.Shift) {
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
			first := policy.shortcutFirst
			if b.shiftOnly {
				first = 5
			}
			in.QuickSpell[int(b.key-ebiten.KeyF4)+4-first] = true
		}
	}
}

func (in *appInput) missionKeyNamed(p missionKeyAction, name string) bool {
	name = namefold.ArchiveText(strings.TrimSpace(name))
	ctrl, shift := in.Viewer.Ctrl, in.Viewer.Shift
	if strings.HasPrefix(name, "ctrl-") {
		ctrl, name = true, strings.TrimPrefix(name, "ctrl-")
	}
	if strings.HasPrefix(name, "shift-") {
		shift, name = true, strings.TrimPrefix(name, "shift-")
	}
	for _, b := range missionBindings(p) {
		if b.name == name {
			if ctrl != in.Viewer.Ctrl || shift != in.Viewer.Shift {
				in.AttackHeld, in.ShiftHeld = ctrl, shift
			}
			in.Viewer.Ctrl, in.Viewer.Shift = ctrl, shift
			in.readMissionKeys(p, onlyKey(b.key), in.Viewer.Ctrl)
			return true
		}
	}
	return false
}
