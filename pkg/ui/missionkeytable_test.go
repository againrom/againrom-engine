package ui

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

var missionKeyActionNames = map[missionKeyAction]string{
	missionKeyPatrol: "patrol", missionKeyGrab: "grab", missionKeyPickup: "pickup", missionKeyGuard: "guard",
	missionKeyDefend: "defend", missionKeyCast: "cast", missionKeyMarch: "march", missionKeyStandGround: "stand-ground",
	missionKeyRetreat: "retreat", missionKeyMove: "move", missionKeyQuickSave: "quick-save",
	missionKeyQuickLoad: "quick-load", missionKeyGrid: "grid", missionKeyReadout: "readout", missionKeyFPS: "fps",
	missionKeyReveal: "reveal", missionKeyShortcut: "spell", missionKeySupport: "support", missionKeyAttack: "attack",
	missionKeyAutocast: "autocast", missionKeyWeaponSwap: "weapon-swap", missionKeyRetreatSetting: "retreat-setting",
	missionKeyFormation: "formation", missionKeyTimeFlow: "time-flow", missionKeySmoothing: "smoothing",
	missionKeyAutoHealing: "auto-healing", missionKeyShowHealth: "show-health", missionKeyNumerals: "numerals",
	missionKeyKill: "kill", missionKeyChip: "chip", missionKeySelectAll: "select-all", missionKeyInventory: "inventory",
	missionKeyWorn: "worn", missionKeyBook: "book", missionKeyDoll: "doll", missionKeyPanels: "panels",
	missionKeyHighlight: "highlight", missionKeyHelp: "help", missionKeySaveGame: "save-game",
	missionKeyLoadGame: "load-game", missionKeyPauseText: "pause-text", missionKeyPause: "pause",
	missionKeyFaster: "faster", missionKeySlower: "slower", missionKeyUnpaced: "unpaced", missionKeyPaced: "paced",
	missionKeyZoomReset: "zoom-reset", missionKeySpeedReset: "speed-reset", missionKeyClearMessages: "clear-messages", missionKeyGroup: "group",
}

// missionKeyLine is one row as the table test lists it: the key's name, what
// it asks of each modifier, hold for a level, and the action.
func missionKeyLine(b missionKeyBinding) string {
	line := []string{b.name}
	for _, m := range []struct {
		name string
		mod  keyMod
	}{{"ctrl", b.ctrl}, {"shift", b.shift}, {"alt", b.alt}} {
		switch m.mod {
		case modHeld:
			line = append(line, m.name)
		case modFree:
			line = append(line, "no-"+m.name)
		}
	}
	if b.hold {
		line = append(line, "hold")
	}
	action, ok := missionKeyActionNames[b.action]
	if !ok {
		action = "action-" + strconv.Itoa(int(b.action))
	}
	line = append(line, "=", action)
	if b.action == missionKeyShortcut || b.action == missionKeyGroup {
		line = append(line, strconv.Itoa(b.arg))
	}
	return strings.Join(line, " ")
}

// The mission key table of each game, one line per row. A rebind is one row
// of missionBindings and one line here.
func TestMissionKeyTableListsEveryKeyOnce(t *testing.T) {
	both := []string{
		"f no-ctrl = grab",
		"a no-ctrl = attack", "a ctrl = autocast", "w no-ctrl = weapon-swap", "w ctrl = retreat-setting",
		"f ctrl = formation", "n ctrl = time-flow", "o ctrl = smoothing", "u ctrl = auto-healing",
		"h ctrl = show-health", "l ctrl = numerals",
		"k ctrl shift = kill", "semicolon ctrl shift = chip", "f4 ctrl shift = reveal",
		"f10 ctrl shift = grid", "f11 ctrl shift = readout",
		"e = select-all", "i = inventory", "backquote = inventory", "x = worn", "b = book", "q = book",
		"j = doll", "space no-shift no-alt = panels", "z no-ctrl no-alt hold = highlight",
		"f1 = help", "pause = pause-text", "f2 = save-game", "f3 = load-game",
		"f5 shift = spell 0", "f6 shift = spell 1", "f7 shift = spell 2", "f8 shift = spell 3",
		"f9 shift = quick-load", "f10 shift = grid", "f11 shift = readout", "f12 shift = fps",
		"0 no-ctrl no-shift no-alt = pause", "numpad-0 no-ctrl no-shift no-alt = pause",
		"equal = faster", "numpad-plus no-ctrl = faster", "minus = slower", "numpad-minus no-ctrl = slower",
		"numpad-plus ctrl = unpaced", "numpad-minus ctrl = paced",
		"home no-ctrl no-shift = zoom-reset", "end no-ctrl no-shift = speed-reset", "backspace no-alt = clear-messages",
		"0 = group 0", "1 = group 1", "2 = group 2", "3 = group 3", "4 = group 4",
		"5 = group 5", "6 = group 6", "7 = group 7", "8 = group 8", "9 = group 9",
		"numpad-0 = group 0", "numpad-1 = group 1", "numpad-2 = group 2", "numpad-3 = group 3",
		"numpad-4 = group 4", "numpad-5 = group 5", "numpad-6 = group 6", "numpad-7 = group 7",
		"numpad-8 = group 8", "numpad-9 = group 9",
	}
	first := append(slices.Clone(both),
		"p = patrol", "g = guard", "d = defend", "c = cast", "s = march", "t = stand-ground", "r = retreat", "m = move",
		"f4 no-shift = quick-save", "f5 no-shift = spell 0", "f6 no-shift = spell 1", "f7 no-shift = spell 2",
		"f8 no-shift = spell 3", "f9 no-shift = quick-load", "f10 no-shift = grid", "f11 no-shift = readout",
		"f12 no-shift = fps",
	)
	second := append(slices.Clone(both),
		"p no-ctrl = pickup", "g no-ctrl = guard", "d no-ctrl = defend", "c no-ctrl = cast", "s no-ctrl = march",
		"t no-ctrl = stand-ground", "r no-ctrl = retreat", "m no-ctrl = move", "l no-ctrl = support",
		"f4 no-shift = spell 0", "f5 no-shift = spell 1", "f6 no-shift = spell 2", "f7 no-shift = spell 3",
		"f8 no-shift = spell 4", "f9 no-shift = spell 5", "f10 no-shift = spell 6", "f11 no-shift = spell 7",
		"f12 no-shift = spell 8", "f4 no-ctrl shift = quick-save",
	)
	for _, tc := range []struct {
		profile string
		want    []string
	}{{"", first}, {"pickup-p", second}} {
		var got []string
		keyOf := map[string]ebiten.Key{}
		for _, b := range missionBindings(missionKeyProfiles[tc.profile]) {
			got = append(got, missionKeyLine(b))
			if k, ok := keyOf[b.name]; ok && k != b.key {
				t.Errorf("profile %q: name %q names two keys", tc.profile, b.name)
			}
			keyOf[b.name] = b.key
		}
		slices.Sort(got)
		want := slices.Clone(tc.want)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("profile %q rows:\n got %q\nwant %q", tc.profile, got, want)
		}
		if c := slices.Compact(slices.Clone(got)); len(c) != len(got) {
			t.Errorf("profile %q lists a row twice", tc.profile)
		}
		names := map[ebiten.Key]string{}
		for name, k := range keyOf {
			if other, ok := names[k]; ok {
				t.Errorf("profile %q: key %v has two names, %q and %q", tc.profile, k, other, name)
			}
			names[k] = name
		}
	}
}

// missionKeyGameProfiles is each game's profile: the first game's and the
// second game's.
var missionKeyGameProfiles = []missionKeyAction{missionKeyPatrol, missionKeyPickup}

// missionRowsFor is every row of profile p that sets action.
func missionRowsFor(p missionKeyAction, action missionKeyAction) []missionKeyBinding {
	var out []missionKeyBinding
	for _, b := range missionBindings(p) {
		if b.action == action {
			out = append(out, b)
		}
	}
	return out
}

// requireMissionRow fails unless each game binds key to action in one row,
// read on a press edge, whose Ctrl test is ctrl.
func requireMissionRow(t *testing.T, action missionKeyAction, key ebiten.Key, ctrl keyMod) {
	t.Helper()
	for _, p := range missionKeyGameProfiles {
		b := missionRow(t, p, action, key)
		if b.ctrl != ctrl || b.hold {
			t.Errorf("profile %d: %s on %v reads Ctrl %d hold %v, want Ctrl %d on a press edge",
				p, missionKeyActionNames[action], key, b.ctrl, b.hold, ctrl)
		}
	}
}

// missionRow is the one row of profile p that binds key to action.
func missionRow(t *testing.T, p missionKeyAction, action missionKeyAction, key ebiten.Key) missionKeyBinding {
	t.Helper()
	var found []missionKeyBinding
	for _, b := range missionBindings(p) {
		if b.action == action && b.key == key {
			found = append(found, b)
		}
	}
	if len(found) != 1 {
		t.Fatalf("profile %d: %d rows bind %v to %s, want 1", p, len(found), key, missionKeyActionNames[action])
	}
	return found[0]
}

// baseMissionInput is what the base (e49d4aeb4) read for the mission fields:
// readAppInput's own key lines, barePauseKey, groupDigit and the table of
// that time, transcribed. pressed is the press edge and held the level.
func baseMissionInput(second bool, pressed, held func(ebiten.Key) bool, ctrl, shift, alt bool) appInput {
	in := appInput{Viewer: Input{Ctrl: ctrl, Shift: shift, Alt: alt}}
	bareZero := (pressed(ebiten.KeyDigit0) || pressed(ebiten.KeyNumpad0)) && !ctrl && !alt && !shift
	in.Panels = pressed(ebiten.KeySpace) && !alt && !shift
	in.Pause = bareZero
	in.Faster = pressed(ebiten.KeyEqual) || pressed(ebiten.KeyNumpadAdd) && !ctrl
	in.Slower = pressed(ebiten.KeyMinus) || pressed(ebiten.KeyNumpadSubtract) && !ctrl
	in.Unpaced = pressed(ebiten.KeyNumpadAdd) && ctrl
	in.Paced = pressed(ebiten.KeyNumpadSubtract) && ctrl
	in.Kill = pressed(ebiten.KeyK) && ctrl && shift
	in.Chip = pressed(ebiten.KeySemicolon) && ctrl && shift
	in.Numerals = pressed(ebiten.KeyL) && ctrl
	in.Attack = pressed(ebiten.KeyA) && !ctrl
	in.HighlightHeld = held(ebiten.KeyZ) && !ctrl && !alt
	in.SaveGame = pressed(ebiten.KeyF2)
	in.LoadGame = pressed(ebiten.KeyF3)
	in.TimeFlow = pressed(ebiten.KeyN) && ctrl
	in.Smoothing = pressed(ebiten.KeyO) && ctrl
	in.AutoHealing = pressed(ebiten.KeyU) && ctrl
	in.ShowHealth = pressed(ebiten.KeyH) && ctrl
	in.Formation = pressed(ebiten.KeyF) && ctrl
	in.Retreat = pressed(ebiten.KeyW) && ctrl
	in.PauseText = pressed(ebiten.KeyPause)
	in.Help = pressed(ebiten.KeyF1)
	in.Inventory = pressed(ebiten.KeyI) || pressed(ebiten.KeyBackquote)
	in.Worn = pressed(ebiten.KeyX)
	in.SelectAll = pressed(ebiten.KeyE)
	in.Book = pressed(ebiten.KeyB) || pressed(ebiten.KeyQ)
	in.Doll = pressed(ebiten.KeyJ)
	in.Autocast = pressed(ebiten.KeyA) && ctrl
	in.WeaponSwap = pressed(ebiten.KeyW) && !ctrl
	in.ZoomReset = pressed(ebiten.KeyHome) && !ctrl && !shift
	in.SpeedReset = pressed(ebiten.KeyEnd) && !ctrl && !shift
	in.ClearMessages = pressed(ebiten.KeyBackspace) && !alt
	digit := -1
	for n := 0; n <= 9; n++ {
		if pressed(ebiten.Key(int(ebiten.KeyDigit0)+n)) || pressed(ebiten.Key(int(ebiten.KeyNumpad0)+n)) {
			digit = n
			break
		}
	}
	in.GroupKey, in.GroupDigit = digit >= 0 && !bareZero, max(digit, 0)

	orders := func(k ebiten.Key) bool { return pressed(k) && !(second && ctrl) }
	in.Grab = pressed(ebiten.KeyF) && !ctrl
	if orders(ebiten.KeyP) {
		if second {
			in.Grab, in.GrabReply = true, true
		} else {
			in.Patrol = true
		}
	}
	in.Guard, in.Defend, in.Cast = orders(ebiten.KeyG), orders(ebiten.KeyD), orders(ebiten.KeyC)
	in.March, in.StandGround = orders(ebiten.KeyS), orders(ebiten.KeyT)
	in.PlayerRetreat, in.Move = orders(ebiten.KeyR), orders(ebiten.KeyM)
	in.Support = second && pressed(ebiten.KeyL) && !ctrl
	bare := []string{"save", "spell", "spell", "spell", "spell", "load", "grid", "readout", "fps"}
	shifted := []string{"", "spell", "spell", "spell", "spell", "load", "grid", "readout", "fps"}
	firstSlot := 5
	if second {
		bare = []string{"spell", "spell", "spell", "spell", "spell", "spell", "spell", "spell", "spell"}
		shifted[0], firstSlot = "save", 4
	}
	apply := func(what string, slot int) {
		switch what {
		case "save":
			in.QuickSave = true
		case "load":
			in.QuickLoad = true
		case "grid":
			in.Grid = true
		case "readout":
			in.Readout = true
		case "fps":
			in.FPS = true
		case "spell":
			in.QuickSpell[slot] = true
		}
	}
	for i := 0; i < 9; i++ {
		if !pressed(ebiten.Key(int(ebiten.KeyF4) + i)) {
			continue
		}
		if !shift {
			apply(bare[i], i+4-firstSlot)
		}
		if shift && !(shifted[i] == "save" && ctrl) {
			apply(shifted[i], i+4-5)
		}
	}
	if ctrl && shift {
		in.Reveal = in.Reveal || pressed(ebiten.KeyF4)
		in.Grid = in.Grid || pressed(ebiten.KeyF10)
		in.Readout = in.Readout || pressed(ebiten.KeyF11)
	}
	return in
}

// Every key, alone or beside a contested key, under every combination of
// Ctrl, Shift and Alt and as an edge, a level or both, sets in each game what
// the base set: the same actions, the same refusals, the same precedence.
func TestMissionKeyRowsReadAsTheBaseRead(t *testing.T) {
	never := func(ebiten.Key) bool { return false }
	contested := []ebiten.Key{ebiten.KeyA, ebiten.KeyF, ebiten.KeyL, ebiten.KeyW, ebiten.KeySpace, ebiten.KeyEqual,
		ebiten.KeyMinus, ebiten.KeyNumpadAdd, ebiten.KeyNumpadSubtract, ebiten.KeyHome, ebiten.KeyEnd, ebiten.KeyBackspace, ebiten.KeyZ}
	for n := 0; n <= 9; n++ {
		contested = append(contested, ebiten.Key(int(ebiten.KeyDigit0)+n), ebiten.Key(int(ebiten.KeyNumpad0)+n))
	}
	for i := 0; i < 9; i++ {
		contested = append(contested, ebiten.Key(int(ebiten.KeyF4)+i))
	}
	type frame struct {
		name           string
		pressed, held  func(ebiten.Key) bool
		ctrl, shf, alt bool
	}
	var frames []frame
	for m := 0; m < 8; m++ {
		ctrl, shf, alt := m&1 != 0, m&2 != 0, m&4 != 0
		for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
			frames = append(frames,
				frame{k.String() + " edge+level", onlyKey(k), onlyKey(k), ctrl, shf, alt},
				frame{k.String() + " edge", onlyKey(k), never, ctrl, shf, alt},
				frame{k.String() + " level", never, onlyKey(k), ctrl, shf, alt})
		}
		for _, x := range contested {
			for _, y := range contested {
				both := func(k ebiten.Key) bool { return k == x || k == y }
				frames = append(frames, frame{x.String() + "+" + y.String(), both, both, ctrl, shf, alt})
			}
		}
	}
	for _, profile := range []string{"", "pickup-p"} {
		p := missionKeyProfiles[profile]
		for _, f := range frames {
			got := appInput{Viewer: Input{Ctrl: f.ctrl, Shift: f.shf, Alt: f.alt}}
			got.readMissionKeyRows(p, f.pressed, f.held, f.ctrl)
			want := baseMissionInput(profile != "", f.pressed, f.held, f.ctrl, f.shf, f.alt)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("profile %q %s Ctrl %v Shift %v Alt %v:\n got %+v\nwant %+v", profile, f.name, f.ctrl, f.shf, f.alt, got, want)
			}
		}
	}
}

// A scenario name of a moved key presses through the table and gives the
// frame the base's own case gave.
func TestMovedMissionKeyNamesPressAsTheBaseCasesDid(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		for name, want := range map[string]appInput{
			"a": {Attack: true}, "w": {WeaponSwap: true}, "e": {SelectAll: true}, "b": {Book: true},
			"q": {Book: true}, "i": {Inventory: true}, "backquote": {Inventory: true}, "j": {Doll: true},
			"space": {Panels: true}, "0": {Pause: true}, "numpad-0": {Pause: true}, "f1": {Help: true},
			"f2": {SaveGame: true}, "f3": {LoadGame: true}, "numpad-plus": {Faster: true},
			"numpad-minus": {Slower: true},
		} {
			var got appInput
			if !got.missionKeyNamed(missionKeyProfiles[profile], name) {
				t.Fatalf("profile %q: no row is named %q", profile, name)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("profile %q %q:\n got %+v\nwant %+v", profile, name, got, want)
			}
		}
	}
}
