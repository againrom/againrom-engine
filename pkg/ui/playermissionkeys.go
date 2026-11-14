package ui

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// MissionKeyChord is a key and the modifiers held with it (DIV-4939); None is no key.
type MissionKeyChord struct {
	Key              ebiten.Key
	Ctrl, Shift, Alt bool
	None             bool
}

// PlayerMissionKeys is the player's layout of one game, by listed action name.
type PlayerMissionKeys map[string]MissionKeyChord

var (
	errMissionKeyFixed   = errors.New("the key is fixed")
	errMissionKeyUnknown = errors.New("no such mission key action")
)

type missionKeySlot struct {
	action missionKeyAction
	arg    int
}

// missionKeyActionNames name the actions; a spell slot counts from one.
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

func (s missionKeySlot) name() string {
	n := missionKeyActionNames[s.action]
	switch s.action {
	case missionKeyShortcut:
		return n + "-" + strconv.Itoa(s.arg+1)
	case missionKeyGroup:
		return n + "-" + strconv.Itoa(s.arg)
	}
	return n
}

func (b missionKeyBinding) slot() missionKeySlot { return missionKeySlot{b.action, b.arg} }

// exactRow is the row that answers chord c and no other.
func exactRow(c MissionKeyChord) missionKeyBinding {
	return missionKeyBinding{name: c.String(), key: c.Key, ctrl: exactMod(c.Ctrl), shift: exactMod(c.Shift), alt: exactMod(c.Alt)}
}

func exactMod(held bool) keyMod {
	if held {
		return modHeld
	}
	return modFree
}

func (m keyMod) meets(o keyMod) bool { return m == modAny || o == modAny || m == o }

// overlaps is whether one chord is answered by both rows.
func (b missionKeyBinding) overlaps(o missionKeyBinding) bool {
	return b.key == o.key && b.ctrl.meets(o.ctrl) && b.shift.meets(o.shift) && b.alt.meets(o.alt)
}

func overlapsAny(b missionKeyBinding, rows []missionKeyBinding) bool {
	return slices.ContainsFunc(rows, b.overlaps)
}

// rowChord is the chord a row answers when it is kept as the player's.
func rowChord(b missionKeyBinding) MissionKeyChord {
	return MissionKeyChord{Key: b.key, Ctrl: b.ctrl == modHeld, Shift: b.shift == modHeld, Alt: b.alt == modHeld}
}

// chordsOf are the chords layout k gives action name, the first under the
// name and the others under name.2, name.3 and on; ok is false when k leaves
// the action its table keys.
func (k PlayerMissionKeys) chordsOf(name string) (chords []MissionKeyChord, ok bool) {
	for i := 1; ; i++ {
		c, found := k[chordEntry(name, i)]
		if !found {
			return chords, len(chords) > 0
		}
		chords = append(chords, c)
	}
}

func chordEntry(name string, i int) string {
	if i == 1 {
		return name
	}
	return name + "." + strconv.Itoa(i)
}

// setChords gives the action exactly these chords, or none.
func (k PlayerMissionKeys) setChords(name string, chords []MissionKeyChord) {
	for i := 1; ; i++ {
		if _, found := k[chordEntry(name, i)]; !found {
			break
		}
		delete(k, chordEntry(name, i))
	}
	if len(chords) == 0 {
		chords = []MissionKeyChord{{None: true}}
	}
	for i, c := range chords {
		k[chordEntry(name, i+1)] = c
	}
}

// MissionKeyActions are the action names profile p lists, in table order.
func MissionKeyActions(profile string) []string {
	var names []string
	for _, s := range listedMissionKeys(missionKeyProfiles[profile]) {
		names = append(names, s.name())
	}
	return names
}

// listedMissionKeys are the actions with a row that is not fixed.
func listedMissionKeys(p missionKeyAction) []missionKeySlot {
	seen := map[missionKeySlot]bool{}
	var slots []missionKeySlot
	for _, b := range missionBindings(p) {
		if !b.fixed && !seen[b.slot()] {
			seen[b.slot()] = true
			slots = append(slots, b.slot())
		}
	}
	return slots
}

// missionKeyRows is p's table with each action k names answering its chord.
func missionKeyRows(p missionKeyAction, k PlayerMissionKeys) []missionKeyBinding {
	table := missionBindings(p)
	if len(k) == 0 {
		return table
	}
	rows := make([]missionKeyBinding, 0, len(table))
	first := map[missionKeySlot]missionKeyBinding{}
	for _, b := range table {
		if _, ok := k.chordsOf(b.slot().name()); ok && !b.fixed {
			if _, seen := first[b.slot()]; !seen {
				first[b.slot()] = b
			}
			continue
		}
		rows = append(rows, b)
	}
	for _, s := range listedMissionKeys(p) {
		chords, _ := k.chordsOf(s.name())
		for _, c := range chords {
			if c.None {
				continue
			}
			m := exactRow(c)
			m.hold, m.action, m.arg = first[s].hold, s.action, s.arg
			if !slices.Contains(rows, m) {
				rows = append(rows, m)
			}
		}
	}
	return rows
}

// missionKeyFixed: Esc, F1, Enter and Alt+letter (cheat entry), a lone
// modifier, the keys other map readers take (the camera's arrows, Tab,
// PageUp and PageDown), and a chord a fixed row answers.
func missionKeyFixed(p missionKeyAction, chords ...MissionKeyChord) bool {
	for _, c := range chords {
		if missionKeyChordFixed(p, c) {
			return true
		}
	}
	return false
}

func missionKeyChordFixed(p missionKeyAction, c MissionKeyChord) bool {
	switch c.Key {
	case ebiten.KeyEscape, ebiten.KeyF1, ebiten.KeyEnter, ebiten.KeyNumpadEnter,
		ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyTab, ebiten.KeyPageUp, ebiten.KeyPageDown:
		return true
	}
	if loneModifier(c.Key) {
		return true
	}
	if c.Alt && c.Key >= ebiten.KeyA && c.Key <= ebiten.KeyZ {
		return true
	}
	for _, b := range missionBindings(p) {
		if b.fixed && b.overlaps(exactRow(c)) {
			return true
		}
	}
	return false
}

func loneModifier(k ebiten.Key) bool {
	switch k {
	case ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight, ebiten.KeyShift, ebiten.KeyShiftLeft,
		ebiten.KeyShiftRight, ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight, ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight:
		return true
	}
	return false
}

// bind gives name the chord c; an action that answered c is not assigned.
func (k PlayerMissionKeys) bind(p missionKeyAction, name string, c MissionKeyChord) (PlayerMissionKeys, error) {
	listed := false
	for _, s := range listedMissionKeys(p) {
		listed = listed || s.name() == name
	}
	if !listed {
		return k, fmt.Errorf("%w: %q", errMissionKeyUnknown, name)
	}
	chords := slotChords(p, name, c)
	if c.None || missionKeyFixed(p, chords...) {
		return k, errMissionKeyFixed
	}
	next := PlayerMissionKeys{}
	for n, held := range k {
		next[n] = held
	}
	next.takeOver(p, name, chords)
	next.setChords(name, chords)
	return next, nil
}

// slotChords are the chords binding action name to c answers: a spell slot
// that has a Shift form in the table answers it on the new key too.
func slotChords(p missionKeyAction, name string, c MissionKeyChord) []MissionKeyChord {
	chords := []MissionKeyChord{c}
	if c.Shift || c.None {
		return chords
	}
	for _, b := range missionBindings(p) {
		if b.action == missionKeyShortcut && b.slot().name() == name && b.shift == modHeld {
			shifted := c
			shifted.Shift = true
			return append(chords, shifted)
		}
	}
	return chords
}

// takeOver takes the chords from every other action that answers one of
// them: the action keeps its other keys and reads "not assigned" when no key
// is left.
func (k PlayerMissionKeys) takeOver(p missionKeyAction, name string, chords []MissionKeyChord) {
	var taken []missionKeyBinding
	for _, c := range chords {
		taken = append(taken, exactRow(c))
	}
	rows := missionKeyRows(p, k)
	for _, s := range listedMissionKeys(p) {
		if s.name() == name {
			continue
		}
		var kept []MissionKeyChord
		hit := false
		for _, b := range rows {
			if b.fixed || b.slot() != s {
				continue
			}
			if overlapsAny(b, taken) {
				hit = true
			} else if c := rowChord(b); !slices.Contains(kept, c) {
				kept = append(kept, c)
			}
		}
		if hit {
			k.setChords(s.name(), kept)
		}
	}
}

// String is the profile text, "none" or "Ctrl+Shift+Alt+Key".
func (c MissionKeyChord) String() string {
	if c.None {
		return "none"
	}
	var parts []string
	for _, m := range []struct {
		held bool
		name string
	}{{c.Ctrl, "Ctrl"}, {c.Shift, "Shift"}, {c.Alt, "Alt"}} {
		if m.held {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, c.Key.String()), "+")
}

// caption is the chord as the page and the book show it: a digit key is
// its digit.
func (c MissionKeyChord) caption() string {
	return strings.Replace(c.String(), "Digit", "", 1)
}

// ParseMissionKeyChord reads String's text.
func ParseMissionKeyChord(s string) (MissionKeyChord, error) {
	if s == "none" {
		return MissionKeyChord{None: true}, nil
	}
	parts := strings.Split(s, "+")
	var c MissionKeyChord
	for _, m := range parts[:len(parts)-1] {
		switch m {
		case "Ctrl":
			c.Ctrl = true
		case "Shift":
			c.Shift = true
		case "Alt":
			c.Alt = true
		default:
			return c, fmt.Errorf("mission key %q: unknown modifier %q", s, m)
		}
	}
	err := c.Key.UnmarshalText([]byte(parts[len(parts)-1]))
	return c, err
}

// MissionKeyPrefix is the profile settings prefix of a game's keys.
func MissionKeyPrefix(profile string) string {
	return missionKeyPolicies[missionKeyProfiles[profile]].store + "."
}

// Entries are layout k as profile settings of the game profile names.
func (k PlayerMissionKeys) Entries(profile string) map[string]string {
	m := map[string]string{}
	for name, c := range k {
		m[MissionKeyPrefix(profile)+name] = c.String()
	}
	return m
}

// PlayerMissionKeysFrom reads a game's layout, skipping unlisted, unreadable
// and fixed entries.
func PlayerMissionKeysFrom(profile string, settings map[string]string) PlayerMissionKeys {
	p := missionKeyProfiles[profile]
	k := PlayerMissionKeys{}
	var claimed []missionKeyBinding
	for _, s := range listedMissionKeys(p) {
		var chords []MissionKeyChord
		unassigned := false
		for i := 1; ; i++ {
			text, ok := settings[MissionKeyPrefix(profile)+chordEntry(s.name(), i)]
			if !ok {
				break
			}
			c, err := ParseMissionKeyChord(text)
			switch {
			case err != nil:
			case c.None:
				unassigned = true
			case missionKeyFixed(p, c) || overlapsAny(exactRow(c), claimed):
			default:
				chords = append(chords, c)
				claimed = append(claimed, exactRow(c))
			}
		}
		if len(chords) > 0 {
			k.takeOver(p, s.name(), chords)
			k.setChords(s.name(), chords)
		} else if unassigned {
			k.setChords(s.name(), nil)
		}
	}
	return k
}

type playerKeys struct {
	keys  PlayerMissionKeys
	write func(PlayerMissionKeys) error
	// capture is the action waiting for its new key on the Keys page.
	capture string
	list    *Picker
}

// SetPlayerMissionKeys installs the player's layout and its writer.
func (a *App) SetPlayerMissionKeys(k PlayerMissionKeys, write func(PlayerMissionKeys) error) {
	a.player.keys, a.player.write = k, write
}

// activePlayerKeys is the player's layout on a map; town and menu keys stay.
func (a *App) activePlayerKeys() *PlayerMissionKeys {
	if a.flow != nil && a.flow.mapShowing() {
		return &a.player.keys
	}
	return nil
}

func (a *App) bindMissionKey(name string, c MissionKeyChord) error {
	next, err := a.player.keys.bind(a.missionKeys, name, c)
	if err != nil {
		return err
	}
	return a.storePlayerKeys(next)
}

func (a *App) restoreMissionKeys() error { return a.storePlayerKeys(PlayerMissionKeys{}) }

func (a *App) storePlayerKeys(k PlayerMissionKeys) error {
	if a.player.write != nil {
		if err := a.player.write(k); err != nil {
			return err
		}
	}
	a.player.keys = k
	return nil
}
