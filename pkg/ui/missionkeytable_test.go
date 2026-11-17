package ui

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

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
	if b.action == missionKeyShortcut || b.action == missionKeyBindSpell || b.action == missionKeyBindItem || b.action == missionKeyGroup {
		line = append(line, strconv.Itoa(b.arg))
	}
	return strings.Join(line, " ")
}

// The mission key table of each game, one line per row. A rebind is one row
// of missionBindings and one line here.
func TestMissionKeyTableListsEveryKeyOnce(t *testing.T) {
	both := []string{
		"f no-ctrl = grab",
		"a no-ctrl no-alt = attack", "a ctrl no-alt = autocast", "w no-ctrl = weapon-swap", "w ctrl = retreat-setting",
		"f ctrl = formation", "n ctrl = time-flow", "o ctrl = smoothing", "u ctrl = auto-healing",
		"h ctrl = show-health", "l ctrl = numerals",
		"k ctrl shift = kill", "semicolon ctrl shift no-alt = chip", "f4 ctrl shift no-alt = reveal",
		"f10 ctrl shift no-alt = grid", "f11 ctrl shift no-alt = readout", "f12 ctrl shift no-alt = fps",
		"e = select-all", "i = inventory", "backquote no-alt = inventory", "x = worn", "b = book", "q = book",
		"j = doll", "space no-shift no-alt = panels", "z no-ctrl no-alt hold = highlight",
		"f1 no-alt = help", "pause no-alt = pause-text", "f2 no-ctrl no-alt = quick-save", "f2 ctrl no-alt = save-game",
		"f3 no-ctrl no-alt = quick-load", "f3 ctrl no-alt = load-game",
		"0 no-ctrl no-shift no-alt = pause", "numpad-0 no-ctrl no-shift no-alt = pause",
		"equal no-alt = faster", "numpad-plus no-ctrl no-alt = faster", "minus no-alt = slower", "numpad-minus no-ctrl no-alt = slower",
		"numpad-plus ctrl no-alt = unpaced", "numpad-minus ctrl no-alt = paced",
		"home no-ctrl no-shift no-alt = zoom-reset", "end no-ctrl no-shift no-alt = speed-reset", "backspace no-alt = clear-messages",
		"0 no-alt = group 0", "1 = group 1", "2 = group 2", "3 = group 3", "4 = group 4",
		"5 = group 5", "6 = group 6", "7 = group 7", "8 = group 8", "9 = group 9",
		"numpad-0 = group 0", "numpad-1 = group 1", "numpad-2 = group 2", "numpad-3 = group 3",
		"numpad-4 = group 4", "numpad-5 = group 5", "numpad-6 = group 6", "numpad-7 = group 7",
		"numpad-8 = group 8", "numpad-9 = group 9",
	}
	slotRows := func(profile string) (rows []string) {
		for slot, key := range []string{"f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12"} {
			n := strconv.Itoa(shortcutOrdinal(missionKeyProfiles[profile], slot))
			rows = append(rows, key+" no-ctrl no-shift no-alt = spell "+n, key+" ctrl no-shift no-alt = bind-spell "+n, key+" no-ctrl shift no-alt = bind-item "+n)
		}
		return rows
	}
	first := append(append(slices.Clone(both), slotRows("")...),
		"p = patrol", "g = guard", "d = defend", "c = cast", "s = march", "t = stand-ground", "r = retreat", "m = move",
	)
	second := append(append(slices.Clone(both), slotRows("pickup-p")...),
		"p no-ctrl = pickup", "g no-ctrl = guard", "d no-ctrl = defend", "c no-ctrl = cast", "s no-ctrl = march",
		"t no-ctrl = stand-ground", "r no-ctrl = retreat", "m no-ctrl = move", "l no-ctrl = support",
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
	in.Faster = (pressed(ebiten.KeyEqual) || pressed(ebiten.KeyNumpadAdd) && !ctrl) && !alt
	in.Slower = (pressed(ebiten.KeyMinus) || pressed(ebiten.KeyNumpadSubtract) && !ctrl) && !alt
	in.Unpaced = pressed(ebiten.KeyNumpadAdd) && ctrl && !alt
	in.Paced = pressed(ebiten.KeyNumpadSubtract) && ctrl && !alt
	in.Kill = pressed(ebiten.KeyK) && ctrl && shift
	in.Chip = pressed(ebiten.KeySemicolon) && ctrl && shift && !alt
	in.Numerals = pressed(ebiten.KeyL) && ctrl
	in.Attack = pressed(ebiten.KeyA) && !ctrl && !alt
	in.HighlightHeld = held(ebiten.KeyZ) && !ctrl && !alt
	in.SaveGame = pressed(ebiten.KeyF2) && ctrl && !alt
	in.LoadGame = pressed(ebiten.KeyF3) && ctrl && !alt
	in.QuickSave = pressed(ebiten.KeyF2) && !ctrl && !alt
	in.QuickLoad = pressed(ebiten.KeyF3) && !ctrl && !alt
	in.TimeFlow = pressed(ebiten.KeyN) && ctrl
	in.Smoothing = pressed(ebiten.KeyO) && ctrl
	in.AutoHealing = pressed(ebiten.KeyU) && ctrl
	in.ShowHealth = pressed(ebiten.KeyH) && ctrl
	in.Formation = pressed(ebiten.KeyF) && ctrl
	in.Retreat = pressed(ebiten.KeyW) && ctrl
	in.PauseText = pressed(ebiten.KeyPause) && !alt
	in.Help = pressed(ebiten.KeyF1) && !alt
	in.Inventory = pressed(ebiten.KeyI) || pressed(ebiten.KeyBackquote) && !alt
	in.Worn = pressed(ebiten.KeyX)
	in.SelectAll = pressed(ebiten.KeyE)
	in.Book = pressed(ebiten.KeyB) || pressed(ebiten.KeyQ)
	in.Doll = pressed(ebiten.KeyJ)
	in.Autocast = pressed(ebiten.KeyA) && ctrl && !alt
	in.WeaponSwap = pressed(ebiten.KeyW) && !ctrl
	in.ZoomReset = pressed(ebiten.KeyHome) && !ctrl && !shift && !alt
	in.SpeedReset = pressed(ebiten.KeyEnd) && !ctrl && !shift && !alt
	in.ClearMessages = pressed(ebiten.KeyBackspace) && !alt
	digit := -1
	for n := 0; n <= 9; n++ {
		if pressed(ebiten.Key(int(ebiten.KeyDigit0)+n)) && !(n == 0 && alt) || pressed(ebiten.Key(int(ebiten.KeyNumpad0)+n)) {
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
	for i := 0; i < 9; i++ {
		if pressed(ebiten.Key(int(ebiten.KeyF4)+i)) && !alt {
			switch {
			case ctrl && !shift:
				in.BindSpell[i] = true
			case !ctrl && shift:
				in.BindItem[i] = true
			case !ctrl && !shift:
				in.QuickSpell[i] = true
			}
		}
	}
	if ctrl && shift && !alt {
		in.Reveal = in.Reveal || pressed(ebiten.KeyF4)
		in.Grid = in.Grid || pressed(ebiten.KeyF10)
		in.Readout = in.Readout || pressed(ebiten.KeyF11)
		in.FPS = pressed(ebiten.KeyF12)
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
			"f2": {QuickSave: true}, "f3": {QuickLoad: true}, "numpad-plus": {Faster: true},
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

// shortcutOrdinal is the number a quick-spell slot's profile name carries,
// less one: the first game's F5 to F8 slots keep spell-1 to spell-4.
func shortcutOrdinal(p missionKeyAction, slot int) int {
	if missionKeyPolicies[p].games&firstGameKeys != 0 {
		return (slot + 8) % 9
	}
	return slot
}

func shortcutSlotName(profile string, slot int) string {
	return "spell-" + strconv.Itoa(shortcutOrdinal(missionKeyProfiles[profile], slot)+1)
}
