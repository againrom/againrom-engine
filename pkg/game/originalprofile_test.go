package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// Independent wire fields: offsets are transcribed from the archive programme,
// never read back through the projection under test or calculated from tables.
type profileFixture1107 struct {
	attack    [24]byte
	defence   [22]byte
	modifier  [64]byte
	stats     [3]uint16
	periods   [2]uint16
	fractions [2]byte
}

func literalProfile1107() *profileFixture1107 {
	p := &profileFixture1107{stats: [3]uint16{0xfffd, 0x8001, 32766}, periods: [2]uint16{100, 100}, fractions: [2]byte{77, 51}}
	binary.LittleEndian.PutUint16(p.attack[:], 2000)
	copy(p.attack[14:], []byte{40, 0, 1, 20, 0, 8, 0, 1, 211, 212})
	binary.LittleEndian.PutUint16(p.defence[:], 0xfffe)
	binary.LittleEndian.PutUint16(p.defence[2:], 7)
	for i, v := range []uint16{999, 10, 20, 30, 40, 0xfffb} {
		binary.LittleEndian.PutUint16(p.defence[4+2*i:], v)
	}
	copy(p.defence[16:], []byte{233, 1, 2, 3, 128, 255})
	binary.LittleEndian.PutUint16(p.modifier[10:], 50)
	binary.LittleEndian.PutUint16(p.modifier[14:], 50)
	return p
}

func assertCurrent1107(t *testing.T, e sim.Entity) {
	t.Helper()
	if e.CurrentProfileBasis != sim.ProfileOriginalCurrent || e.ToHit != 2000 || e.DamageBase != 40 || e.DamageSpread != 0 || e.XPSlot != 1 ||
		e.SecondBase != 20 || e.SecondSpread != 0 || e.SecondaryDamage != (sim.SecondaryDamage{Base: 8}) ||
		e.Defence != -2 || e.Absorption != 7 || e.Protection != ([5]int32{10, 20, 30, 40, -5}) || e.Resistance != ([5]uint8{1, 2, 3, 128, 255}) ||
		e.Reaction != -3 || e.Mind != -32767 || e.Spirit != 32766 || e.HealthRegenPeriod != 100 || e.ManaRegenPeriod != 100 ||
		e.HealthRegeneration != 50 || e.ManaRegeneration != 50 || e.HealthHundredths != 77 || e.ManaHundredths != 51 {
		t.Fatalf("wrong current profile for MapUnitID%d: %+v", e.MapUnitID, e)
	}
}

func TestOriginalProfile1107ExactAllPlayerClassesAndAliases(t *testing.T) {
	var actors []*poolFixtureActor
	for i, class := range []string{"Unit", "Humanoid", "Human"} {
		actors = append(actors, &poolFixtureActor{class: class, mapID: uint16(91 + i), cell: uint16(0x0605 + i), hp: 10, maxHP: 100, mana: 10, maxMana: 100, profile: literalProfile1107()})
	}
	first := &poolFixturePlayer{groups: [][]*poolFixtureActor{{nil, actors[0], actors[0]}}}
	last := &poolFixturePlayer{groups: [][]*poolFixtureActor{{actors[1]}, {}, {actors[2], actors[0]}}}
	payload := savedContainer(poolFixtureBody([]*poolFixturePlayer{first, nil, last, first}, nil))
	sf, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	got, err := sf.ActorCurrentProfiles()
	if err != nil || len(got) != 3 {
		t.Fatal(len(got), err)
	}
	for i, p := range got {
		if p.Class != actors[i].class || p.MapUnitID != uint16(91+i) || p.ToHit != 2000 || p.Absorption != 7 || p.Mind != -32767 || p.HealthRegeneration != 50 || p.Protection != ([5]int16{10, 20, 30, 40, -5}) || p.Resistance != ([5]uint8{1, 2, 3, 128, 255}) {
			t.Fatalf("projection%d=%+v", i, p)
		}
	}
	got[0].Protection[0] = 300
	again, _ := sf.ActorCurrentProfiles()
	if again[0].Protection[0] != 10 {
		t.Fatal("projection aliases source")
	}
	for _, cut := range []int{100, len(sf.Body) - 4} {
		bad := *sf
		bad.Body = append([]byte(nil), sf.Body[:cut]...)
		if partial, err := bad.ActorCurrentProfiles(); err == nil || len(partial) != 0 {
			t.Fatal("partial malformed archive admitted", cut, err)
		}
	}
	f := poolFixtureFront(t, 91, 92, 93)
	ms, r, err := loadOriginalMission(f, payload)
	if err != nil || r.ProfilesRestored != 3 {
		t.Fatal(r, err)
	}
	for _, id := range []uint16{91, 92, 93} {
		assertCurrent1107(t, poolEntity(t, ms.World, id))
	}
}

func TestOriginalProfile1107BothAppDoorsNewRegenAndNativeContinuation(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		t.Run(map[bool]string{false: "main menu", true: "mission menu"}[fromMap], func(t *testing.T) {
			f := currentPoolFixtureFront(t, 91, 92)
			app := f.App("profile")
			if fromMap {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			payload := poolFixtureSave(&poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, mana: 10, maxMana: 100, profile: literalProfile1107(), equipmentRuntime: &[3]byte{1, 8, 4}},
				&poolFixtureActor{mapID: 92, cell: 0x0606, hp: 10, maxHP: 100, mana: 10, maxMana: 100, human: true, profile: literalProfile1107()})
			// Synthetic continuation cut: phase12 at92, independent Full4.
			payload = withClockFixture1112(t, payload, 80, 4)
			originals := t.TempDir()
			if err := os.WriteFile(filepath.Join(originals, "game1107.sav"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game1107.sav")
			for _, id := range []uint16{91, 92} {
				assertCurrent1107(t, poolEntity(t, f.live.world, id))
			}
			if f.live.world.Tick() != rawSavedSubTick1112(t, payload) {
				t.Fatal("LOAD ticked")
			}
			for i := 0; i < 13; i++ {
				f.live.tick()
			}
			e := poolEntity(t, f.live.world, 91)
			if e.HP != 13 || e.Mana != 12 || e.HealthHundredths != 77 || e.ManaHundredths != 1 {
				t.Fatalf("literal next regen: %+v", e)
			}
			before := f.live
			// A real newly ordered attack remains active across ordinary SAVE.
			target := poolEntity(t, before.world, 92)
			before.pending = append(before.pending, sim.Command{Kind: sim.KindAttack, Entity: e.ID, X: int32(target.ID)})
			before.tick()
			moving := poolEntity(t, before.world, 91)
			for i := 0; i < 64 && moving.HasAttackTarget && moving.AttackPhase == sim.AttackReady; i++ {
				before.tick()
				moving = poolEntity(t, before.world, 91)
			}
			if !moving.HasAttackTarget || moving.AttackPhase == sim.AttackReady {
				t.Fatalf("new attack action did not start: tick%d id%d phase%d hp%d targetid%d targethp%d attack=%t", before.world.Tick(), moving.ID, moving.AttackPhase, moving.HP, target.ID, poolEntity(t, before.world, 92).HP, moving.HasAttackTarget)
			}
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
			freshApp := fresh.App("fresh native")
			fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(fs, fl, ff)
			groundAppLoad(t, freshApp, fl, entries[0].Name)
			if fresh.live.world.Hash() != before.world.Hash() {
				currentMenuWorldDiagnostics(t, before.world, fresh.live.world)
				t.Fatal("fresh native load replaced imported current state")
			}
			for i := 0; i < 96; i++ {
				before.tick()
				fresh.live.tick()
				if before.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("native continuation", i)
				}
			}
		})
	}
}

func TestOriginalProfile1107LateRefusalLeavesActiveSession(t *testing.T) {
	for _, kind := range []string{"ambiguous source", "bad active", "bad elemental", "zero mana period", "full mana zero period"} {
		t.Run(kind, func(t *testing.T) {
			f := poolFixtureFront(t, 91, 92)
			app := f.App("refuse")
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			old := f.live
			hash := old.world.Hash()
			party := mapload.CloneParty(f.Carried)
			a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, mana: 10, maxMana: 100, profile: literalProfile1107()}
			b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 10, maxHP: 100, mana: 10, maxMana: 100, profile: literalProfile1107()}
			a.holdings, b.holdings = &holdingFixture{}, &holdingFixture{}
			a.book = []*poolFixtureSpell{{id: 1, rangeByte: 7, cost: 3}}
			b.book = []*poolFixtureSpell{{id: 1, rangeByte: 9, cost: 4}}
			switch kind {
			case "ambiguous source":
				b.mapID = 91
			case "bad active":
				b.profile.attack[16] = 6
			case "bad elemental":
				b.profile.attack[21] = 6
			case "zero mana period":
				b.profile.periods[1] = 0
			case "full mana zero period":
				b.profile.periods[1] = 0
				b.mana = b.maxMana
			}
			payload := poolFixtureSave(a, b)
			// A refused LOAD returns no mission and no report.
			if ms, _, err := loadOriginalMission(f, payload); err == nil || ms != nil {
				t.Fatal("LOAD admitted invalid current profile")
			}
			if _, _, err := f.RestoreOriginal(payload); err == nil {
				t.Fatal("bad late entry admitted")
			}
			if f.live != old || old.world.Hash() != hash || !reflect.DeepEqual(f.Carried, party) {
				t.Fatal("failed LOAD changed active session")
			}
		})
	}
}

func TestOriginalProfile1107JoinExcludesPartyDeadOffmapAndRefusesAmbiguousTarget(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, []sim.Entity{{ID: 1, X: 5, Y: 6, HP: 10, MaxHP: 100, MapUnitID: 91}, {ID: 2, X: 6, Y: 6, HP: 10, MaxHP: 100, MapUnitID: 92}})
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{Map: &alm.Map{Width: 40, Height: 40}, World: w, Start: mapload.Start{IDs: []sim.EntityID{2}}}
	p := sav.ActorCurrent{ActorPools: sav.ActorPools{MapUnitID: 91, Cell: 0x0605, HP: 10, MaxHP: 100}, DamageBase: 90}
	party, dead, offmap, zero, missing := p, p, p, p, p
	party.MapUnitID = 92
	dead.Stage = 1
	offmap.Cell = 0xffff
	zero.MapUnitID = 0
	missing.MapUnitID = 93
	var r OriginalSaveResume
	if err := applyOriginalActorProfiles(ms, []sav.ActorCurrent{p, party, dead, offmap, zero, missing}, &r); err != nil {
		t.Fatal(err)
	}
	if r.ProfilesRestored != 1 || r.ProfilesParty != 1 || r.ProfilesExcluded != 3 || r.ProfilesUnmatched != 1 {
		t.Fatal(r)
	}
	if poolEntity(t, w, 92).CurrentProfileBasis != sim.ProfileNative {
		t.Fatal("persistent party imported")
	}
	w2, _ := sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, []sim.Entity{{ID: 1, X: 5, Y: 6, HP: 10, MaxHP: 100, MapUnitID: 91}, {ID: 3, X: 7, Y: 6, HP: 10, MaxHP: 100, MapUnitID: 91}})
	ms.World = w2
	old := w2.Hash()
	if err := applyOriginalActorProfiles(ms, []sav.ActorCurrent{p}, &OriginalSaveResume{}); err == nil || !strings.Contains(err.Error(), "ambiguous") || w2.Hash() != old {
		t.Fatal("ambiguous target", err)
	}
}

func TestOriginalProfile1107RawHighPoolWordsAreNotLoadArithmetic(t *testing.T) {
	f := poolFixtureFront(t, 91)
	p := literalProfile1107()
	p.periods = [2]uint16{1, 1}
	p.fractions = [2]byte{255, 0}
	p.modifier = [64]byte{}
	payload := withClockFixture1112(t, poolFixtureSave(&poolFixtureActor{
		mapID: 91, cell: 0x0605, hp: 10, maxHP: 65535, mana: 65530, maxMana: 65535, profile: p,
	}), 80, 4)
	ms, _, err := loadOriginalMission(f, payload)
	if err != nil {
		t.Fatal(err)
	}
	e := poolEntity(t, ms.World, 91)
	if e.HP != 10 || e.MaxHP != 65535 || e.Mana != 65530 || e.MaxMana != 65535 || e.HealthHundredths != 255 || ms.World.Tick() != rawSavedSubTick1112(t, payload) {
		t.Fatal("LOAD normalized raw pool words", e)
	}
	for i := 0; i < 13; i++ {
		sim.Step(ms.World, nil)
	}
	e = poolEntity(t, ms.World, 91)
	// Consumer reads 65530/65535 as -6/-1; health's signed maximum gate
	// leaves its raw fraction untouched. Neither result is a LOAD rewrite.
	if e.HP != 10 || e.HealthHundredths != 255 || e.Mana != -7 || e.ManaHundredths != 0 {
		t.Fatal("signed read of raw pool words", e)
	}
}

func TestOriginalProfile1110ElementalSelectorMatchesResolverPermutation(t *testing.T) {
	for raw := uint8(1); raw <= 5; raw++ {
		w, err := sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil,
			[]sim.Entity{{ID: 1, X: 5, Y: 6, HP: 10, MaxHP: 100, MapUnitID: 91}})
		if err != nil {
			t.Fatal(err)
		}
		ms := &Mission{Map: &alm.Map{Width: 40, Height: 40}, World: w}
		p := sav.ActorCurrent{ActorPools: sav.ActorPools{MapUnitID: 91, Cell: 0x0605, HP: 10, MaxHP: 100},
			ElementalBase: 8, ElementalKind: raw}
		var r OriginalSaveResume
		if err := applyOriginalActorProfiles(ms, []sav.ActorCurrent{p}, &r); err != nil {
			t.Fatalf("raw kind %d refused: %v", raw, err)
		}
		want := data.ElementalSelectorOrder[raw-1]
		if e := poolEntity(t, w, 91); e.SecondaryDamage != (sim.SecondaryDamage{Base: 8, Selector: want}) {
			t.Fatalf("raw kind %d: SecondaryDamage = %+v, want Selector %d", raw, e.SecondaryDamage, want)
		}
	}
}

// A real Humans-row roster is rebuilt by native LOAD. Its cache is deliberately
// unlike the saved current sheet, so the post-potion overwrite is observable.
func profileHumanFront1107(t *testing.T) *FrontEnd {
	f := poolFixtureFront(t, 91)
	b := poolFixtureMap(91)
	for off := 20; off+20 <= len(b); {
		size := int(binary.LittleEndian.Uint32(b[off+8:]))
		if binary.LittleEndian.Uint32(b[off+12:]) == 6 {
			binary.LittleEndian.PutUint16(b[off+20+8:], 7)
			break
		}
		off += 20 + size
	}
	path := filepath.Join(t.TempDir(), ScenarioArchive)
	if err := os.WriteFile(path, synth.Archive([]synth.File{{Path: "10.alm", Data: b}, {Path: "npc.reg", Data: synth.NPCReg(nil)}}), 0600); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fs
	f.Table = eqDefsTable(t)
	f.Table.Humans = dbCollection{{}, personDBRow("Nonparty", 30, 20, 18, 17, [data.SkillSlots]int32{}, 7)}
	return f
}

func TestOriginalProfile1110RealPotionContinuesSourceBasisAcrossNativeLoad(t *testing.T) {
	f := withCurrentMenuDefinitions(t, profileHumanFront1107(t))
	app := f.App("native mutation")
	potion := sim.ItemInstance{Code: 0x0e01, Kind: 3, WeightPresent: true, Effects: []sim.ItemEffect{{Kind: 2, Mode: 0, Operand: 1}}}
	payload := poolFixtureSave(&poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, mana: 10, maxMana: 100, human: true, profile: literalProfile1107(),
		holdings: &holdingFixture{items: []*holdingFixtureItem{{class: "Item", code: potion.Code, count: 1, kind: potion.Kind, effects: potion.Effects}}}})
	originals := t.TempDir()
	if err := os.WriteFile(filepath.Join(originals, "game1107.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game1107.sav")
	e := poolEntity(t, f.live.world, 91)
	assertCurrent1107(t, e)
	if len(f.live.derives) == 0 {
		t.Fatal("no actual cached roster")
	}
	pack, _ := f.live.world.CarriedStacks(e.ID)
	if len(pack) != 1 || !sim.StackStateEqual(pack[0], sim.StackItem(potion, 1)) {
		t.Fatal("original LOAD did not restore the actual potion", pack)
	}
	if !f.live.world.SetPotionHeadroom(e.ID, [4]int32{1, 0, 0, 0}) {
		t.Fatal("potion admission")
	}
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindUsePotion, Entity: e.ID, X: 0})
	f.live.tick()
	got := poolEntity(t, f.live.world, 91)
	// Body30 ->31; source signed Reaction=-3, Mind=-32767 and Spirit's
	// effective cap50 are not replaced from the cached row. The retained
	// second-pair modifier is zero, so the reached derive clears live20.
	if got.PotionStats[0] != 0 || got.CurrentProfileBasis != sim.ProfileNativeRetired || got.ActorLoad.Source.Class != 2 || got.ActorLoad.Source.Stats[0] != 31 || got.Capacity != 311 || got.Speed != -3 || got.MaxHP != 73 || got.ToHit != 3 || got.SecondBase != 0 {
		t.Fatalf("source potion used the native cached row: %+v", got)
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
	fresh := withCurrentMenuDefinitions(t, profileHumanFront1107(t))
	freshApp := fresh.App("retired native")
	fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(fs, fl, ff)
	groundAppLoad(t, freshApp, fl, entries[0].Name)
	if fresh.live.world.Hash() != old.world.Hash() || poolEntity(t, fresh.live.world, 91).CurrentProfileBasis != sim.ProfileNativeRetired {
		t.Fatal("native load reclaimed source basis")
	}
	for i := 0; i < 32; i++ {
		old.tick()
		fresh.live.tick()
		if old.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("retired continuation", i)
		}
	}
}
