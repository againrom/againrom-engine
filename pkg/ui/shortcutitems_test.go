package ui

import (
	"bytes"
	"image/color"
	"strings"
	"testing"
	"time"

	"againrom/pkg/words"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestShortcutInventoryBindingMarkAndWaiting(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		a := inventoryTestApp(t)
		a.SetMissionKeys(profile)
		v := a.flow.viewer
		v.SetFont(panelFont())
		layoutViewport(v, MenuWindowW, MenuWindowH)
		v.SetEntities([]MapEntity{{ID: 5, Owner: 1, Life: LifeAlive}})
		v.sel = selection{5}
		subject := packOf(5, solidPic(20, 20, color.RGBA{R: 200, A: 255}))
		subject.PackKeys = []string{"armour"}
		v.SetInventorySubject(subject)
		slots := new(ShortcutSlots)
		v.SetQuickSpells(slots)
		base, _, ok := v.packBarPresent()
		if !ok {
			t.Fatal("no inventory")
		}
		x, y := packCellCenter(t, v, 0)
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessKey("shift-f12"); err != nil {
			t.Fatal(err)
		}
		if slots[8].Item != "armour" {
			t.Fatal("Shift did not bind item", *slots)
		}
		marked, _, _ := v.packBarPresent()
		if bytes.Equal(base.Pix, marked.Pix) {
			t.Fatal("item key mark is missing")
		}
		if err := a.HeadlessKey("shift-f4"); err != nil {
			t.Fatal(err)
		}
		if slots[8] != (Shortcut{}) || slots[0].Item != "armour" {
			t.Fatal("item did not move", *slots)
		}
		if err := a.HeadlessKey("f4"); err != nil {
			t.Fatal(err)
		}
		if i, ok := v.TakeInventoryEquip(); !ok || i != 0 {
			t.Fatal("shortcut missed inventory use", i, ok)
		}
		subject.PackKeys[0] = "replacement"
		v.SetInventorySubject(subject)
		if err := a.HeadlessKey("f4"); err != nil {
			t.Fatal(err)
		}
		if _, ok := v.TakeInventoryEquip(); ok || slots[0].Item != "armour" {
			t.Fatal("absent item acted or binding was lost")
		}
		subject.PackKeys[0] = "armour"
		v.SetInventorySubject(subject)
		if err := a.HeadlessKey("f4"); err != nil {
			t.Fatal(err)
		}
		if _, ok := v.TakeInventoryEquip(); !ok {
			t.Fatal("returned item remained inert")
		}
	}
}

func TestShortcutMovedDiagnosticsAndKeysRows(t *testing.T) {
	for _, p := range missionKeyGameProfiles {
		for _, tc := range []struct {
			key    ebiten.Key
			action missionKeyAction
			read   func(appInput) bool
		}{
			{ebiten.KeyF10, missionKeyGrid, func(in appInput) bool { return in.Grid }},
			{ebiten.KeyF11, missionKeyReadout, func(in appInput) bool { return in.Readout }},
			{ebiten.KeyF12, missionKeyFPS, func(in appInput) bool { return in.FPS }},
		} {
			for _, ctrl := range []bool{false, true} {
				in := appInput{Viewer: Input{Shift: true, Ctrl: ctrl}}
				in.readMissionKeys(p, onlyKey(tc.key), ctrl)
				if tc.read(in) != ctrl {
					t.Fatal("diagnostic chord", p, tc.key, ctrl)
				}
			}
		}
	}
	for _, profile := range []string{"", "pickup-p"} {
		a, _ := onKeysPage(t, profile)
		for _, lang := range []string{"en", "ru"} {
			a.flow.words.Engine = words.For(lang)
			for _, name := range []string{"spell-1", "spell-9", "bind-spell-1", "bind-spell-9", "bind-item-1", "bind-item-9", "grid", "readout", "fps"} {
				row := keysPageRow(t, a, name)
				if text := a.keyRowText(row, 1000); text == "" || strings.Contains(text, "keys.action.") {
					t.Fatal(name, text)
				}
			}
			if row := keysPageRow(t, a, "fps"); !strings.Contains(a.keyRowText(row, 1000), "Ctrl+Shift+F12") {
				t.Fatal("diagnostic key missing")
			}
		}
	}
}

func TestShortcutFunctionKeyLayout(t *testing.T) {
	for _, p := range missionKeyGameProfiles {
		for slot := 0; slot < 9; slot++ {
			for _, mods := range []Input{{}, {Ctrl: true}, {Shift: true}} {
				in := appInput{Viewer: mods}
				in.readMissionKeys(p, onlyKey(ebiten.Key(int(ebiten.KeyF4)+slot)), mods.Ctrl)
				hit := in.QuickSpell[slot]
				if mods.Ctrl {
					hit = in.BindSpell[slot]
				}
				if mods.Shift {
					hit = in.BindItem[slot]
				}
				if !hit || in.QuickSave || in.QuickLoad || in.Grid || in.Readout || in.FPS {
					t.Fatalf("profile %v slot %d modifiers %+v: %+v", p, slot, mods, in)
				}
			}
		}
	}
}

func TestShortcutScrollOnlyBookBinding(t *testing.T) {
	a, v, slots := quickSpellApp(t)
	v.SetSpellbook(sbUnitID, []SpellEntry{{ID: 17, Unavailable: true, ItemCast: &BookItemCast{Index: 2, Key: "scroll"}}})
	v.SetItemCastSink(func(uint32, int, string, uint32, int, int, bool) {})
	x, y := sbEntryPoint(t, v, v.spellbook, 0)
	a.step(appInput{BindSpell: [9]bool{true}, CursorX: x, CursorY: y, Viewer: Input{Ctrl: true}}, time.Unix(100, 0))
	if slots[0].Spell != 17 {
		t.Fatal("scroll spell was not bound", *slots)
	}
	v.toggleHudPanel(hudPanelBook)
	v.quickSpell(0, false, -1, -1)
	if v.itemCast == nil || v.itemCast.index != 2 || v.itemCast.key != "scroll" {
		t.Fatal("shortcut did not arm the carried scroll")
	}
}

func TestShortcutInvocationActsOnAnyChord(t *testing.T) {
	a, v, slots := quickSpellApp(t)
	a.SetMissionKeys("")
	a.SetPlayerMissionKeys(PlayerMissionKeys{}, func(PlayerMissionKeys) error { return nil })
	*slots = spellSlots(89)
	v.selectedSpell = 17
	if err := a.bindMissionKey(shortcutSlotName("", 0), MissionKeyChord{Key: ebiten.KeyQ, Ctrl: true}); err != nil {
		t.Fatal(err)
	}
	in := pressOnMap(a, MissionKeyChord{Key: ebiten.KeyQ, Ctrl: true})
	if !in.QuickSpell[0] || in.BindSpell[0] {
		t.Fatalf("Ctrl+Q does not read as slot 1 invocation: %+v %+v", in.QuickSpell, in.BindSpell)
	}
	a.step(in, time.Unix(200, 0))
	if *slots != spellSlots(89) || v.selectedSpell != 89 {
		t.Fatalf("invocation on Ctrl+Q assigned or missed: slots %v current %d", *slots, v.selectedSpell)
	}
}

func TestShortcutCtrlOverScrollOnlySpellBindsAndArmsNothing(t *testing.T) {
	for _, bookOpen := range []bool{true, false} {
		a, v, slots := quickSpellApp(t)
		v.SetSpellbook(sbUnitID, []SpellEntry{{ID: 17, Unavailable: true, ItemCast: &BookItemCast{Index: 2, Key: "scroll"}}})
		v.SetItemCastSink(func(uint32, int, string, uint32, int, int, bool) {})
		if !bookOpen {
			v.toggleHudPanel(hudPanelBook)
		}
		v.selectedSpell = 17
		a.step(appInput{BindSpell: [9]bool{true}, Viewer: Input{Ctrl: true}}, time.Unix(100, 0))
		if slots[0].Spell != 17 || v.itemCast != nil || v.spellArmed {
			t.Fatalf("book open %v: bound %v itemCast %v armed %v", bookOpen, *slots, v.itemCast, v.spellArmed)
		}
	}
}

// A profile entry stored before the common layout keeps its key slot: the first
// game's spell-1 to spell-4 are the F5 to F8 slots, the second game's spell-1
// to spell-9 the F4 to F12 slots.
func TestShortcutStoredProfileEntriesKeepTheirSlots(t *testing.T) {
	a, _ := onKeysPage(t, "")
	a.SetPlayerMissionKeys(PlayerMissionKeysFrom("", map[string]string{"FirstGameKeys.spell-1": "G", "FirstGameKeys.spell-4": "H"}), nil)
	for key, slot := range map[ebiten.Key]int{ebiten.KeyG: 1, ebiten.KeyH: 4} {
		in := pressOnMap(a, MissionKeyChord{Key: key})
		want := [9]bool{}
		want[slot] = true
		if in.QuickSpell != want {
			t.Fatalf("stored first-game entry on %v drives %v, want slot %d", key, in.QuickSpell, slot)
		}
	}
	row := keysPageRow(t, a, "spell-1")
	if text := a.keyRowText(row, 1000); !strings.Contains(text, "G") {
		t.Fatal("Keys row of the F5 slot does not show the stored key", text)
	}
	if marks := missionShortcutMarks(a.activeMissionKeys(), a.player.keys); marks[1] != "G" || marks[0] != "F4" {
		t.Fatal("marks", marks)
	}
	b, _ := onKeysPage(t, "pickup-p")
	b.SetPlayerMissionKeys(PlayerMissionKeysFrom("pickup-p", map[string]string{"SecondGameKeys.spell-1": "G", "SecondGameKeys.spell-9": "H"}), nil)
	for key, slot := range map[ebiten.Key]int{ebiten.KeyG: 0, ebiten.KeyH: 8} {
		in := pressOnMap(b, MissionKeyChord{Key: key})
		want := [9]bool{}
		want[slot] = true
		if in.QuickSpell != want {
			t.Fatalf("stored second-game entry on %v drives %v, want slot %d", key, in.QuickSpell, slot)
		}
	}
}

// Two identical carried items share one key; the key acts on the first.
func TestShortcutIdenticalItemsActOnTheFirstInPackOrder(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	v.SetFont(panelFont())
	layoutViewport(v, MenuWindowW, MenuWindowH)
	v.SetEntities([]MapEntity{{ID: 5, Owner: 1, Life: LifeAlive}})
	v.sel = selection{5}
	subject := packOf(5, solidPic(20, 20, color.RGBA{R: 200, A: 255}), solidPic(20, 20, color.RGBA{R: 200, A: 255}))
	subject.PackKeys = []string{"potion", "potion"}
	v.SetInventorySubject(subject)
	slots := new(ShortcutSlots)
	slots[0] = Shortcut{Item: "potion"}
	v.SetQuickSpells(slots)
	if err := a.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	if i, ok := v.TakeInventoryEquip(); !ok || i != 0 {
		t.Fatal("the key did not act on the first identical item", i, ok)
	}
	cells, _, _ := v.packBarPresent()
	solo := new(ShortcutSlots)
	v.SetQuickSpells(solo)
	plain, _, _ := v.packBarPresent()
	if bytes.Equal(cells.Pix, plain.Pix) {
		t.Fatal("the bound item shows no mark")
	}
}
