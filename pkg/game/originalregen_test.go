package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Real App original LOAD -> new attack -> ordinary SAVE -> fresh native LOAD.
// The expected signed results are literals; the two copies then dispatch the
// actual mission loop. This proves native continuation, not ROM1's first tick.
func TestOriginalProfile1107SignedAppActionAndNativeContinuation(t *testing.T) {
	for _, tc := range []struct {
		name                                    string
		hp, maxHP, healthModifier, healthPeriod uint16
		wantHP                                  int32
	}{
		{"narrowed health", 32766, 32767, 0, 1, 32764},
		{"negative health", 1, 101, 65435, 1, -1},
		{"negative period", 10, 100, 0, 65533, -56},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := currentPoolFixtureFront(t, 91, 92)
			app := f.App("signed profile")
			p := literalProfile1107()
			p.periods = [2]uint16{tc.healthPeriod, 1}
			p.fractions = [2]byte{}
			binary.LittleEndian.PutUint16(p.modifier[10:], tc.healthModifier)
			binary.LittleEndian.PutUint16(p.modifier[14:], 65435) // signed -101
			payload := poolFixtureSave(&poolFixtureActor{mapID: 91, cell: 0x0605, hp: tc.hp, maxHP: tc.maxHP, mana: 0, maxMana: 101,
				human: true, profile: p, holdings: profileHoldings1107()},
				&poolFixtureActor{mapID: 92, cell: 0x0606, hp: 32767, maxHP: 32767, profile: literalProfile1107()})
			// Explicit synthetic phase cut; Full4 admits the next health filter.
			payload = withClockFixture1112(t, payload, 80, 4)
			originals := t.TempDir()
			if err := os.WriteFile(filepath.Join(originals, "signed.sav"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "signed.sav")
			e := poolEntity(t, f.live.world, 91)
			assertProfileHoldings1107(t, f.live.world, e.ID)
			if e.HP != int32(tc.hp) || e.HealthRegenPeriod != int32(int16(tc.healthPeriod)) || e.ManaRegeneration != -101 || f.live.world.Tick() != rawSavedSubTick1112(t, payload) {
				t.Fatal("LOAD changed signed group or tick", e)
			}
			target := poolEntity(t, f.live.world, 92)
			f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindAttack, Entity: e.ID, X: int32(target.ID)})
			f.live.tick()
			if !poolEntity(t, f.live.world, 91).HasAttackTarget {
				t.Fatal("new attack not admitted")
			}
			old := f.live
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 {
				t.Fatal(entries, err, app.HeadlessMessage())
			}
			fresh := currentPoolFixtureFront(t, 91, 92)
			freshApp := fresh.App("signed native")
			fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(fs, fl, ff)
			groundAppLoad(t, freshApp, fl, entries[0].Name)
			if old.world.Hash() != fresh.live.world.Hash() {
				currentMenuWorldDiagnostics(t, old.world, fresh.live.world)
				t.Fatal("active attack native load")
			}
			assertProfileHoldings1107(t, fresh.live.world, e.ID)
			for old.world.Tick() <= 92 {
				old.tick()
				fresh.live.tick()
				if old.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("first native dispatch differs")
				}
			}
			got := poolEntity(t, old.world, 91)
			wantHP := tc.wantHP
			if wantHP < 0 {
				wantHP-- // Full4 also admits HERO-DECAY-069's live-negative arm
			}
			if wantHP < 0 && e.DyingTime <= 1 {
				wantHP--
			} // native same-tick teardown reaches the separate even-Full dead arm
			if got.HP != wantHP || got.Mana != -1 || got.ManaHundredths != 255 {
				t.Fatalf("signed dispatch HP/mana/rest=%d/%d/%d want%d/-1/255", got.HP, got.Mana, got.ManaHundredths, wantHP)
			}
			assertProfileHoldings1107(t, old.world, e.ID)
			// SAVE the negative pool and byte255, then fresh LOAD again. Merely
			// saving the pre-dispatch source would not exercise decoder admission.
			if err := freshApp.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := freshApp.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			saved, err := store.List()
			if err != nil || len(saved) != 2 {
				t.Fatal(saved, err)
			}
			var latest string
			for _, s := range saved {
				if s.Name != entries[0].Name {
					latest = s.Name
				}
			}
			if tc.name == "narrowed health" {
				raw, err := os.ReadFile(filepath.Join(store.Dir, latest))
				if err != nil {
					t.Fatal(err)
				}
				assertCurrentSignedPoolBoundary(t, fresh, raw, e.ID)
			}
			last := currentPoolFixtureFront(t, 91, 92)
			lastApp := last.App("signed post-pool native")
			ls, ll, lf := last.SaveSeams(store, OriginalStore{}, nil)
			lastApp.SetSaveSeams(ls, ll, lf)
			groundAppLoad(t, lastApp, ll, latest)
			if old.world.Hash() != last.live.world.Hash() {
				currentMenuWorldDiagnostics(t, old.world, last.live.world)
				t.Fatal("negative-pool save was changed")
			}
			assertProfileHoldings1107(t, last.live.world, e.ID)
			for i := 0; i < 96; i++ {
				old.tick()
				last.live.tick()
				if old.world.Hash() != last.live.world.Hash() {
					t.Fatal("signed continuation", i)
				}
				if tc.name == "narrowed health" && old.world.Tick() == 29 {
					e := poolEntity(t, old.world, 91)
					if e.Mana != 0 || e.ManaHundredths != 54 {
						t.Fatal("unsigned byte reload", e)
					}
				}
			}
			assertProfileHoldings1107(t, last.live.world, e.ID)
		})
	}
}

func TestCurrentSourceRearmKeepsAbsentItemDefinition(t *testing.T) {
	item := sim.ItemInstance{Code: 0x0101, Kind: 2, Price: -201, WeightPresent: true, SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon}}
	for _, class := range []uint8{0, 1, 2} {
		w, err := sim.NewStockedWorld(7, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{{ID: 1, X: 2, Y: 2, HP: 30, MaxHP: 40}}, nil, sim.Relations{}, nil,
			[]sim.Stock{{ID: 1, EquippedItems: [sim.EquipSlots]sim.ItemInstance{item}}})
		if err != nil {
			t.Fatal(err)
		}
		if class != 0 {
			load := sim.ActorLoadSnapshot{Capacity: 300, Speed: 1, Inventory: sim.ActorLoad{Present: true, Source: sim.SourceActor{
				Class: class, Stats: [14]uint16{30, 20, 10, 5, 1, 0, 0, 300, 30, 40, 100, 0, 0, 100},
			}}}
			if err := w.RestoreActorLoad(1, load); err != nil {
				t.Fatal(err)
			}
			hash := w.Hash()
			load.Inventory.Source.Attack[16] = 6
			if w.RestoreActorLoad(1, load) == nil || w.Hash() != hash {
				t.Fatal("malformed current load changed world")
			}
		}
		hash := w.Hash()
		weapon, ok := Rearm(w, 1, data.Hero{Body: 99}, data.Profile{}, nil, true, nil, 0)
		if ok != (class != 0) || weapon != nil || w.Hash() != hash {
			t.Fatal("current source required an absent definition or derived its live state", class, ok, weapon)
		}
		items, _ := w.EquippedItems(1)
		if !reflect.DeepEqual(items[0], item) || items[0].SourceEquipment.Definition.Present {
			t.Fatal("rearm changed the literal held Item")
		}
		if _, ok := Rearm(w, 99, data.Hero{}, data.Profile{}, nil, true, nil, 0); ok || w.Hash() != hash {
			t.Fatal("missing actor rearm changed world")
		}
	}
}

func assertCurrentSignedPoolBoundary(t *testing.T, live *FrontEnd, raw []byte, id sim.EntityID) {
	t.Helper()
	for _, kind := range []string{"ordinary edit", "missing node", "missing policy", "bad lift", "legacy word"} {
		t.Run(kind, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil {
				t.Fatal("missing current policy fixture", err)
			}
			value, present := a.Values[id]
			lift := -1
			for i, x := range value.Widths {
				if x.Field == 2 && x.Wire == 65535 && x.Lift == -65536 {
					lift = i
				}
			}
			if !present || lift < 0 {
				t.Fatal("signed Mana lacks its exact ordinary anchor")
			}
			var object uint16
			for _, binding := range a.Bindings {
				if binding.ID == id && !binding.Structure && !binding.Missing {
					object = binding.Object
				}
			}
			if object == 0 {
				t.Fatal("signed actor lacks its current binding")
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			switch kind {
			case "ordinary edit":
				mustSetValue(&doc.Objects[object-1], "Mana", 65534)
			case "missing node":
				delete(a.Values, id)
			case "missing policy":
				for i, record := range doc.State.ValueRecords {
					if record.Path == sav.NativeActionsPath {
						doc.State.ValueRecords = append(doc.State.ValueRecords[:i], doc.State.ValueRecords[i+1:]...)
						break
					}
				}
			case "bad lift":
				value.Widths[lift].Lift++
				a.Values[id] = value
			}
			if kind == "missing node" || kind == "bad lift" {
				b, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&doc.State, b); err != nil {
					t.Fatal(err)
				}
			}
			changed, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "legacy word" {
				changed = poolFixtureSave(&poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, mana: 65535, maxMana: 101})
			}
			if kind == "ordinary edit" {
				back, err := sav.DecodeDocumentData(changed)
				if err != nil {
					t.Fatal(err)
				}
				unchanged, _, _ := sav.NativeActions(back.State)
				if !bytes.Equal(leaf, unchanged) {
					t.Fatal("ordinary Mana edit changed the width policy")
				}
				cold := currentPoolFixtureFront(t, 91, 92)
				open, town, err := cold.RestoreOriginal(changed)
				if err == nil && !town {
					err = cold.App("edited ordinary mana").OpenMission(open)
				}
				if err != nil || town {
					t.Fatal("edited ordinary Mana refused", err)
				}
				got := poolEntity(t, cold.live.world, 91)
				if got.Mana != 65534 || got.MaxMana != 101 || got.ActorLoad.Source.Stats[11] != 65534 {
					t.Fatal("ordinary Mana restored stale signed value", got.Mana, got.MaxMana, got.ActorLoad.Source.Stats[11])
				}
				return
			}
			prior := live.live
			hash := prior.world.Hash()
			before, _, err := live.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			open, _, loadErr := live.RestoreOriginal(changed)
			if loadErr == nil {
				loadErr = live.App("invalid signed pools").OpenMission(open)
			}
			after, _, err := live.Snapshot(true)
			if loadErr == nil || err != nil || prior != live.live || hash != live.live.world.Hash() || !reflect.DeepEqual(before, after) {
				t.Fatal("invalid signed-pool input changed the live session", loadErr, err)
			}
			if kind == "legacy word" && !strings.Contains(loadErr.Error(), "unsupported pools") {
				t.Fatal("legacy word failed before its pool boundary", loadErr)
			}
			t.Log("atomic refusal:", loadErr)
		})
	}
}
