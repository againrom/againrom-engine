package game

import (
	"encoding/binary"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func holdingsNativeFresh(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore, terminalSave ui.SaveGame) (*FrontEnd, *ui.App) {
	t.Helper()
	hash := f.live.world.Hash()
	native := nativeCheckpoint1170(f, store)
	_, nativeList, nativeLoad := agsSaveSeams(f, store, OriginalStore{}, nil)
	app.SetSaveSeams(native, nativeList, nativeLoad)
	if terminalSave != nil {
		// A saved Victory modal captures menu keys. This instrument selects
		// the native codec explicitly, including for the terminal callback.
		if _, err := native(true); err != nil {
			t.Fatal(err)
		}
		t.Log("terminal Victory: explicit native checkpoint callback")
	} else {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		t.Log("App menu with explicit native checkpoint codec")
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatalf("ordinary App SAVE: %+v %v", entries, err)
	}
	fresh := releaseFront(t)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("1108-fresh-native-load")
	s, l, load := agsSaveSeams(fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(s, l, load)
	groundAppLoad(t, app2, l, entries[0].Name)
	if fresh.live.world.Hash() != hash {
		t.Fatalf("fresh FrontEnd LOAD: hash %016x want %016x", fresh.live.world.Hash(), hash)
	}
	t.Logf("native SAVE=%s fresh FrontEnd LOAD hash=%016x", entries[0].Name, hash)
	return fresh, app2
}

func itemOnlyAt(w *sim.World, id sim.SavedObjectID, owner sim.SavedObjectOwner) bool {
	at := w.SavedObjects().Locations(id)
	return len(at) == 1 && at[0].Owner == owner
}

// Natural owner saves; literal expected canonical values come from the
// independent seat baseline, not ActorHoldings or the importer under test.
// EN/RU are two installed consumers of each same source, not two recordings.
func TestReleaseOriginalHoldings1108NaturalEmptyAndValuedStaff(t *testing.T) {
	for _, tc := range []struct {
		name, path, sha string
		mission         int
		mapID           uint16
	}{
		{"empty-witch", "2026-08-02/game0006.sav", "c6b9506e986f5dcc3e4260b68b0fae41b510d6307b33909f26c801d68329ccde", 10, 21},
		{"valued-staff", "2026-08-15/game0017.sav", "eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0", 40, 32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			path, payload := groundCorpusFile(t, tc.path, tc.sha)
			source, err := sav.Open(payload)
			if err != nil {
				t.Fatal(err)
			}
			actors, err := source.ActorHoldings()
			if err != nil {
				t.Fatal(err)
			}
			var src *sav.ActorHoldings
			for i := range actors {
				if actors[i].MapUnitID == tc.mapID {
					if src != nil {
						t.Fatal("source ambiguous")
					}
					src = &actors[i]
				}
			}
			if src == nil || len(src.Items) != 0 {
				t.Fatal("natural source is no longer empty-pack")
			}
			var want [sim.EquipSlots]sim.ItemInstance
			if tc.mapID == 21 {
				if src.Off != 15484 || src.HeldWeapon != nil || src.HeldShield != nil {
					t.Fatal("Witch natural anchor changed")
				}
			} else {
				// Literal words from the pinned source body, independently
				// located before this assertion. Never copy ActorHoldings's
				// weight into the expected instance.
				for _, literal := range []struct{ at, code, weight int }{{27922, 0x810e, 5}, {28200, 0xf70d, 8}, {28278, 0xf80b, 6}} {
					if int(binary.LittleEndian.Uint16(source.Body[literal.at:])) != literal.code || int(int16(binary.LittleEndian.Uint16(source.Body[literal.at+9:]))) != literal.weight {
						t.Fatal("natural raw item weight anchor changed", literal.at)
					}
				}
				want[0] = sim.ItemInstance{Code: 0x810e, Kind: 2, Price: 981, Weight: 5, WeightPresent: true, Effects: []sim.ItemEffect{{Kind: 41, Mode: 0, Operand: 327681}}}
				want[6] = sim.ItemInstance{Code: 0xf70d, Kind: 1, Price: 125, Weight: 8, WeightPresent: true}
				want[7] = sim.ItemInstance{Code: 0xf80b, Kind: 1, Price: 150, Weight: 6, WeightPresent: true}
				// Fixed source-body positions, independent of Piece and the
				// importer: Token row, then the concrete class's saved blocks.
				for _, row := range []struct{ at, value int }{{27851, 14}, {28175, 13}, {28253, 11}} {
					if int(source.Body[row.at]) != row.value {
						t.Fatalf("natural definition row at %d = %d want %d", row.at, source.Body[row.at], row.value)
					}
				}
				p := f.Table.Weapons.EntryParams(14)
				if p[5] != 3 || p[12] != 8 || p[13] != 4 || p[14] != 2 || p[15] != 2 {
					t.Fatal("natural staff definition columns changed")
				}
				// Existing Spell class9 tag, followed by ID/range/defensive,
				// cost and runtime key. These offsets do not use Piece.
				if binary.LittleEndian.Uint16(source.Body[27981:]) != 0x8009 ||
					[5]byte(source.Body[27983:27988]) != [5]byte{1, 7, 0, 3, 0} {
					t.Fatal("natural owned Spell anchor changed")
				}
				want[0].SourceEquipment = sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 14,
					OwnKind: source.Body[27980], Attack: [24]byte(source.Body[27934:27958]), Defence: [22]byte(source.Body[27958:27980]),
					Definition: sim.SourceWeaponDefinition{Present: true, AttackType: 3, Hands: 2, Charge: 8, Relax: 4, Suitable: 2},
					Spell:      sim.SourceItemSpell{Present: true, ID: 1, Range: 7, ManaCost: 3}}
				want[6].SourceEquipment = sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 13,
					OwnKind: source.Body[28234], Defence: [22]byte(source.Body[28212:28234])}
				want[7].SourceEquipment = sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 11,
					OwnKind: source.Body[28312], Defence: [22]byte(source.Body[28290:28312])}
				if src.Off != 27305 || src.HeldWeapon == nil || src.HeldWeapon.Price != 981 {
					t.Fatal("staff natural anchor changed")
				}
			}
			freshMap, err := StartMission(f.Archives.Containers, tc.mission, f.Table, mapload.DifficultyNormal, nil)
			if err != nil {
				t.Fatal(err)
			}
			baseline := poolEntity(t, freshMap.World, tc.mapID)
			baseWorn, _ := freshMap.World.EquippedItems(baseline.ID)
			basePack, _ := freshMap.World.CarriedStacks(baseline.ID)
			if tc.mapID == 21 {
				if len(basePack) != 1 || basePack[0].Count != 3 || basePack[0].Code != 0x0e06 {
					t.Fatal("fresh Witch no longer has three starter potions")
				}
			} else if baseWorn[0].Price != 981 {
				t.Fatal("fresh staff price changed")
			}
			f.SetDeterministicFrames(true)
			app := f.App("1108-natural-holdings")
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			actor := poolEntity(t, f.live.world, tc.mapID)
			if !actor.Alive() || actor.OffMap || originalPartyCarriesMapUnit(f.liveParty, tc.mapID) {
				t.Fatal("not living non-party target")
			}
			var sourceStaff sim.SavedItemObject
			if tc.mapID == 32 {
				doc, origins, err := sav.DecodeDocumentDataWithOrigins(payload)
				if err != nil {
					t.Fatal(err)
				}
				graph, err := source.ActorGraph()
				if err != nil {
					t.Fatal(err)
				}
				var actorIndex uint16
				for _, a := range graph.Actors {
					if a.Off != src.Off {
						continue
					}
					if a.MapUnitID != tc.mapID || actor.SourceBinding.ArchiveIndex != a.ArchiveIndex || actor.SourceBinding.Identity != a.Identity {
						t.Fatal("native holder is not the exact natural source actor")
					}
					for _, origin := range origins {
						if origin.ArchiveIndex == a.ArchiveIndex {
							actorIndex = origin.ObjectIndex
						}
					}
				}
				if actorIndex == 0 {
					t.Fatal("natural source actor has no exact DTO origin")
				}
				record := &doc.Objects[actorIndex-1]
				held, _ := savedObjectRefs(record, "HeldWeapon")
				worn, _ := savedObjectRefs(record, "Worn")
				if len(held) != 1 || len(worn) != sim.EquipSlots {
					t.Fatal("natural exact held/worn edge shape changed")
				}
				seen := make(map[sim.SavedObjectID]bool)
				for _, slot := range []int{1, 7, 8} {
					index := held[0]
					if slot != 1 {
						index = worn[slot-1]
					}
					row := releaseSourceItemObject1076(t, f.live.mission.state.savedDocument, f.live.world, doc, index, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: actor.ID, Slot: uint32(slot)})
					if seen[row.ID] {
						t.Fatal("distinct source worn objects were aliased")
					}
					seen[row.ID] = true
					want[slot-1].ObjectID = row.ID
					if slot == 1 {
						sourceStaff = row
					}
				}
			}
			assert := func(w *sim.World) {
				t.Helper()
				e := poolEntity(t, w, tc.mapID)
				worn, _ := w.EquippedItems(e.ID)
				pack, _ := w.CarriedStacks(e.ID)
				if !reflect.DeepEqual(worn, want) || len(pack) != 0 {
					t.Fatalf("natural holdings: worn=%+v pack=%+v", worn, pack)
				}
				for slot, item := range want {
					if item.ObjectID == 0 {
						continue
					}
					row, ok := w.SavedObjects().Item(item.ObjectID)
					if !ok || !w.SavedObjects().HasRoot(item.ObjectID, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: e.ID, Slot: uint32(slot + 1)}) || !reflect.DeepEqual(row.Value.Instance(), item) {
						t.Fatal("natural initial Item identity/owner/value changed")
					}
				}
			}
			assert(f.live.world)
			diagnostic, report, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, f.Difficulty, nil, f.Bodies)
			if err != nil {
				t.Fatal(err)
			}
			assert(diagnostic.World)
			// The Witch's Victory modal captures menu keys.
			var fresh *FrontEnd
			var freshApp *ui.App
			if tc.mapID == 21 {
				fresh, freshApp = holdingsNativeFresh(t, f, app, store, save)
			} else {
				savedStore, name, _ := menuSAVE(t, f, app, OriginalStore{})
				fresh, freshApp = loadSAVWindow(t, savedStore, name)
			}
			assert(fresh.live.world)
			w := fresh.live.world
			actor = poolEntity(t, w, tc.mapID)
			if actor.SuppressCorpseLoot {
				t.Fatal("natural subject no longer eligible for corpse loot")
			}
			// Controlled terminal kill uses the production death dispatch. It is
			// not an observation of the original actor dying in the owner save.
			if tc.mapID == 21 {
				// OutcomeWon freezes Step. The explicit scenario kill reaches
				// the same terminal loot dispatch without rewriting the save.
				if err := w.HeadlessKill(actor.ID); err != nil {
					t.Fatal(err)
				}
			} else {
				releaseTerminalDeath(t, w, actor.ID)
			}
			if poolEntity(t, w, tc.mapID).Alive() {
				t.Fatal("death control did not kill subject")
			}
			if tc.mapID == 21 {
				if sack := groundAt(w.Sacks(), actor.X, actor.Y); sack != nil {
					for _, item := range sack.ItemInstances {
						if item.Code == 0x0e06 {
							t.Fatal("cleared starter potions reappeared in death sack")
						}
					}
				}
			} else {
				removed := want[0].Clone()
				removed.SourceEquipment.Spell = sim.SourceItemSpell{}
				if !releaseSackAtContains(w.Sacks(), actor.X, actor.Y, removed) {
					t.Fatal("saved-price staff did not reach death sack")
				}
				deathSack := groundAt(w.Sacks(), actor.X, actor.Y)
				staff, ok := w.SavedObjects().Item(sourceStaff.ID)
				if !ok || deathSack.ObjectID == 0 || !itemOnlyAt(w, sourceStaff.ID, sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: deathSack.ObjectID}) || staff.Spell != 0 || !reflect.DeepEqual(staff.Effects, sourceStaff.Effects) || !itemMutationSpell1115(t, w.SavedObjects(), sourceStaff.Spell).Retired {
					t.Fatal("death lost source staff/Effect identity or old Spell retirement")
				}
				taker := fresh.live.mission.ids[0]
				liveTakeAt(t, fresh.live, taker, actor.X, actor.Y)
				pack, _ := w.CarriedStacks(taker)
				index := -1
				for i, s := range pack {
					if reflect.DeepEqual(s.Instance(), removed) {
						index = i
						break
					}
				}
				if index < 0 {
					t.Fatal("loot lost staff price/effects")
				}
				staff, ok = w.SavedObjects().Item(sourceStaff.ID)
				if !ok || !itemOnlyAt(w, sourceStaff.ID, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: taker}) || staff.Spell != 0 || !reflect.DeepEqual(staff.Effects, sourceStaff.Effects) {
					t.Fatal("pickup rerolled source staff/Effect identity")
				}
				sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: taker, X: int32(index), Y: 1}})
				eq, _ := w.EquippedItems(taker)
				if !reflect.DeepEqual(eq[0], want[0]) {
					t.Fatal("looted staff changed on equip")
				}
				staff, ok = w.SavedObjects().Item(sourceStaff.ID)
				if !ok || !itemOnlyAt(w, sourceStaff.ID, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: taker, Slot: 1}) || staff.Spell == 0 || staff.Spell == sourceStaff.Spell || !reflect.DeepEqual(staff.Effects, sourceStaff.Effects) || itemMutationSpell1115(t, w.SavedObjects(), staff.Spell).Value != want[0].SourceEquipment.Spell {
					t.Fatal("equip changed Item identity or failed to create a new owned Spell")
				}
				store, name, written := menuSAVE(t, fresh, freshApp, OriginalStore{})
				back, _ := loadSAVWindow(t, store, name)
				eq, _ = back.live.world.EquippedItems(taker)
				if !reflect.DeepEqual(eq[0], want[0]) || !itemOnlyAt(back.live.world, sourceStaff.ID, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: taker, Slot: 1}) {
					t.Fatal("death/loot/equip staff is not worn by its taker after the SAV LOAD")
				}
				requireSameItemObjects(t, w, back.live.world)
				requireAlteredItemWeight(t, w, written, sourceStaff.ID)
				for range 20 {
					sim.Step(w, nil)
					sim.Step(back.live.world, nil)
					if w.Hash() != back.live.world.Hash() {
						t.Fatal("staff death/pickup/equip continuation differs after the SAV LOAD")
					}
				}
				requireSameItemObjects(t, w, back.live.world)
			}
			t.Logf("source=%s sha=%s actor=%d mapID=%d stockRestored=%d: fresh pack=%+v weaponPrice=%d -> saved pack empty weaponPrice=%d; both original LOAD doors, fresh LOAD, production death/loot passed", path, tc.sha, actor.ID, tc.mapID, report.Stocked, basePack, baseWorn[0].Price, want[0].Price)
		})
	}
}
