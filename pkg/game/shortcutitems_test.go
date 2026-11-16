package game

import (
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestShortcutItemIdentitySurvivesSAV(t *testing.T) {
	for _, second := range []bool{false, true} {
		var f *FrontEnd
		if second {
			front, app, screen, _ := secondBagFront(t, secondBagSample())
			secondBagEnter(t, front, app, screen)
			f = front
		} else {
			f = currentArchiveFixture(t)
		}
		id := f.live.mission.ids[0]
		stacks, _ := f.live.world.CarriedStacks(id)
		if len(stacks) == 0 {
			t.Fatal("fixture has no item")
		}
		key := inventoryItemKeys(stacks)[0]
		f.quickSpells[8] = ui.Shortcut{Item: key}
		s, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		cold := &FrontEnd{InstallResources: f.InstallResources}
		open, town, err := cold.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal(town, err)
		}
		if err := cold.App("shortcut identity").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		loaded, _ := cold.live.world.CarriedStacks(cold.live.mission.ids[0])
		if cold.quickSpells[8].Item != key || !slices.Contains(inventoryItemKeys(loaded), key) {
			t.Fatal("saved binding no longer identifies its item", second, key, inventoryItemKeys(loaded))
		}
	}
}

func TestShortcutSAVKeepsNineSlotsInBothGames(t *testing.T) {
	for _, game := range []base.Game{base.GameROM1, base.GameROM2} {
		t.Run(string(game), func(t *testing.T) {
			var f *FrontEnd
			if game == base.GameROM1 {
				f, _ = originalCityRouteFixture(t)
			} else {
				f, _, _, _ = secondBagFront(t, nil)
			}
			want := spellSlots(17, 1, 2, 3, 4, 89, 23, 65535)
			want[8] = ui.Shortcut{Item: itemCastKey(sim.ItemInstance{Code: 0xe10, Kind: 4})}
			f.quickSpells = want
			for cycle := 0; cycle < 2; cycle++ {
				s, label, err := f.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := f.ExportCurrentSave(s, label)
				if err != nil {
					t.Fatal(err)
				}
				doc := engineSaveDocument(t, raw)
				a, err := readCurrentActions(&doc)
				if err != nil || a == nil || a.Session == nil || a.Session.ShortcutLayout != 1 {
					t.Fatal("shortcut record", err)
				}
				if game == base.GameROM1 {
					file, err := sav.Open(raw)
					if err != nil {
						t.Fatal(err)
					}
					leaf, ok, err := file.StateLeaves().ReadLeaf("/SpellBook/Shortcuts")
					first := int32(0)
					if cycle == 1 {
						first = -1
					}
					if err != nil || !ok || len(leaf.Ints) != 4 || leaf.Ints[0] != first || leaf.Ints[1] != 1 || leaf.Ints[2] != 2 || leaf.Ints[3] != 3 {
						t.Fatal("ordinary F5 to F8", leaf, err)
					}
				}
				cold := &FrontEnd{InstallResources: f.InstallResources}
				if _, town, err := cold.RestoreOriginal(raw); err != nil || !town || cold.quickSpells != want {
					t.Fatal("cold SAV", cycle, town, err, cold.quickSpells)
				}
				f = cold
				if cycle == 0 {
					want[1], want[8] = want[8], want[1]
					f.quickSpells = want
				}
			}
		})
	}
}

func TestShortcutOriginalAndOlderSessionDefaults(t *testing.T) {
	raw, newFront := spellbookCitySource(t)
	f := newFront()
	if _, _, err := f.RestoreOriginal(raw); err != nil || f.quickSpells != (ui.ShortcutSlots{}) {
		t.Fatal("original defaults", f.quickSpells, err)
	}
	file, err := sav.Open(originalContainer(sav.Magic, savedTrailer(append(savedBody(0, tenActors()), 0)), shortcutsStore(0, 1, 2, 3)))
	if err != nil {
		t.Fatal(err)
	}
	old := &currentSessionData{QuickSpells: []currentQuickSpell{{Slot: 0, ID: 89}}}
	got, err := currentQuickSpells(file, ui.ShortcutSlots{}, old)
	if err != nil || got != spellSlots(0, 89) {
		t.Fatal("old first-game residue", got, err)
	}
	old.Game = base.GameROM2
	got, err = currentQuickSpells(file, spellSlots(0, 1, 2, 3, 4), old)
	if err != nil || got != spellSlots(1, 2, 3, 4) {
		t.Fatal("old second-game ordinary cells", got, err)
	}
}

func TestShortcutItemActionsUseTheInventoryRoute(t *testing.T) {
	for _, tc := range []struct {
		name string
		item sim.ItemInstance
		kind uint8
	}{
		{"weapon", sim.ItemInstance{Code: eqSwordCode}, sim.KindEquip},
		{"armour", sim.ItemInstance{Code: eqShieldCode}, sim.KindEquip},
		{"potion", sim.ItemInstance{Code: 0xe01, Kind: 3}, sim.KindUsePotion},
		{"scroll", sim.ItemInstance{Code: 0xe10, Kind: 4, Effects: []sim.ItemEffect{{Kind: 41, Operand: 1 | 40<<16}}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{{ID: 7, Owner: sim.SelfSlot, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{tc.item}}})
			if err != nil {
				t.Fatal(err)
			}
			mw := equipMission(t, w, 7, eqHero(), nil, eqDefsTable(t))
			slots := ui.ShortcutSlots{}
			mw.view.SetQuickSpells(&slots)
			a := ui.NewApp("shortcuts", nil, nil, nil)
			if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
				return mw.view, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
			}); err != nil {
				t.Fatal(err)
			}
			a.Layout(1024, 768)
			if err := a.HeadlessSelectEntity(7); err != nil {
				t.Fatal(err)
			}
			x, y, err := a.HeadlessPackCellPoint(0)
			if err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessKey("shift-f4"); err != nil {
				t.Fatal(err)
			}
			if slots[0].Item == "" {
				t.Fatal("no item bound")
			}
			if err := a.HeadlessKey("f4"); err != nil {
				t.Fatal(err)
			}
			mw.equipFromPack()
			if tc.kind != 0 && (len(mw.pending) != 1 || mw.pending[0].Kind != tc.kind) {
				t.Fatal("inventory action", mw.pending)
			}
			if tc.kind == 0 {
				if len(mw.pending) != 0 {
					t.Fatal("scroll spent at arm")
				}
				mw.view.SetFogReveal(true)
				x, y, err := a.HeadlessEntityPoint(7)
				if err != nil {
					t.Fatal(err)
				}
				for _, edge := range []string{"press", "release"} {
					if err := a.HeadlessPointer(edge, x, y); err != nil {
						t.Fatal(err)
					}
				}
				if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindUseScroll {
					t.Fatal("shortcut did not arm the scroll route", mw.pending)
				}
			}
		})
	}
}
