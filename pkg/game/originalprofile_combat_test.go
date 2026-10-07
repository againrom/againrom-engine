package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

// Literal wire selectors and independent outcomes from HERO-DMG2-029 and
// HERO-CLAMP-030. These are new blows, not an assertion that LOAD retained a
// protection word. Both App doors then ordinary SAVE/fresh LOAD exercise the
// production join, canonical selector, combat consumer and native action.
func TestOriginalProfile1107ElementalBlowAppAndNativeContinuation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		wire, slot uint8
		protection [5]int16
		wantHP     int32
	}{
		{"raw1 Fire", 1, 0, [5]int16{10, 20, 30, 40, 50}, 82},
		{"raw2 Earth", 2, 3, [5]int16{10, 20, 30, 40, 50}, 88},
		{"raw3 Air", 3, 2, [5]int16{10, 20, 30, 40, 50}, 86},
		{"raw4 Water", 4, 1, [5]int16{10, 20, 30, 40, 50}, 84},
		{"raw5 Astral", 5, 4, [5]int16{10, 20, 30, 40, 50}, 90},
		{"negative Fire", 1, 0, [5]int16{-50, 20, 30, 40, 50}, 70},
	} {
		for _, fromMap := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/main", true: "/mission"}[fromMap], func(t *testing.T) {
				f := currentPoolFixtureFront(t, 91, 92)
				app := f.App("current elemental blow")
				if fromMap {
					if err := app.OpenMission(f.MissionOpener(10)); err != nil {
						t.Fatal(err)
					}
				}
				attacker := &profileFixture1107{periods: [2]uint16{100, 100}}
				attacker.attack[19], attacker.attack[21] = 20, tc.wire
				target := &profileFixture1107{periods: [2]uint16{100, 100}}
				for i, p := range tc.protection {
					binary.LittleEndian.PutUint16(target.defence[6+2*i:], uint16(p))
				}
				for _, p := range []*profileFixture1107{attacker, target} {
					binary.LittleEndian.PutUint16(p.modifier[10:], 65436) // -100: no regen masks a blow
					binary.LittleEndian.PutUint16(p.modifier[14:], 65436)
				}
				// Both actors sit under the sole real Player, at Slot 1 (SelfSlot):
				// poolFixtureSave's own default leaves an empty Player0 ahead of the
				// populated one, which lands the real Player at Slot 2. The attacker
				// must resolve to SelfSlot for the player-issued KindAttack order to
				// survive engagementPassObserved's per-tick AI decision pass — a group
				// whose owner is not SelfSlot with no scored candidate has its attack
				// released every tick (engage.go's decide) — exactly the per-tick AI
				// arbitration this fixture must NOT engage for a directly commanded
				// actor. actorregistry1111_test.go's own single-Player fixtures already
				// use this same no-dummy-Player0 shape.
				payload := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{
					// A source-backed actor's Reach/AttackCharge/AttackRelax come from the
					// SAV's own cached equipment runtime bytes, not a table recompute
					// (BindSourceItemDefinition's doc). {1, 8, 4} repeats the melee-reach
					// tuple literalProfile1107's fixture already uses in originalprofile_test.go.
					{mapID: 91, cell: 0x0605, hp: 100, maxHP: 100, human: true,
						profile: attacker, holdings: profileHoldings1107(), equipmentRuntime: &[3]byte{1, 8, 4},
						book: []*poolFixtureSpell{{id: 1, rangeByte: 7, cost: 3}}},
					{mapID: 92, cell: 0x0606, hp: 100, maxHP: 100, profile: target, holdings: &holdingFixture{}},
				}}}}, nil))
				originals := t.TempDir()
				if err := os.WriteFile(filepath.Join(originals, "elemental.sav"), payload, 0600); err != nil {
					t.Fatal(err)
				}
				store := SaveStore{Dir: t.TempDir()}
				save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
				app.SetSaveSeams(save, list, load)
				groundAppLoad(t, app, list, "elemental.sav")
				a, v := poolEntity(t, f.live.world, 91), poolEntity(t, f.live.world, 92)
				// Compatibility per originalclock_test.go's rawSavedSubTick1112: the
				// 1112 reconciliation reads this fixture's leading wire word as the
				// source SubTick, so "no repair tick" is no longer zero here.
				if a.CurrentProfileBasis != sim.ProfileOriginalCurrent || v.CurrentProfileBasis != sim.ProfileOriginalCurrent ||
					v.HP != 100 || f.live.world.Tick() != rawSavedSubTick1112(t, payload) || a.Book.State != sim.BookPresent {
					t.Fatal("LOAD changed the profile/book/pool boundary", a, v)
				}
				assertProfileHoldings1107(t, f.live.world, a.ID)
				f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindAttack, Entity: a.ID, X: int32(v.ID)})
				released := false
				for i := 0; i < 128; i++ {
					f.live.tick()
					phase := poolEntity(t, f.live.world, 91).AttackPhase
					if phase == sim.AttackRelaxing || phase == sim.AttackBoundaryOne || phase == sim.AttackBoundaryTwo {
						released = true
						break
					}
				}
				if !released {
					t.Fatal("new attack did not release its first blow")
				}
				if got := poolEntity(t, f.live.world, 92).HP; got != tc.wantHP {
					t.Fatalf("raw selector%d protection%v: first blow HP%d want%d", tc.wire, tc.protection, got, tc.wantHP)
				}
				if got := poolEntity(t, f.live.world, 91).SecondaryDamage.Selector; got != tc.slot {
					t.Fatalf("canonical selector%d want%d", got, tc.slot)
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
				freshApp := fresh.App("native elemental continuation")
				fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
				freshApp.SetSaveSeams(fs, fl, ff)
				groundAppLoad(t, freshApp, fl, entries[0].Name)
				if fresh.live.world.Hash() != old.world.Hash() {
					currentMenuWorldDiagnostics(t, old.world, fresh.live.world)
					t.Fatal("native LOAD changed the active blow/current protection")
				}
				assertProfileHoldings1107(t, fresh.live.world, a.ID)
				for i := 0; i < 96; i++ {
					old.tick()
					fresh.live.tick()
					if old.world.Hash() != fresh.live.world.Hash() {
						t.Fatal("elemental native continuation", i)
					}
				}
			})
		}
	}
}
