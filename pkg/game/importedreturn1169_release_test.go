package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type returnOracle1169 struct {
	PartyID                                           string
	Name                                              string
	Stats                                             [14]uint16
	Skill                                             [6]int32
	XP                                                [6]int32
	Aggregate                                         uint32
	Attack, Base                                      [24]byte
	Defence                                           [22]byte
	Modifier                                          [64]byte
	Reach, Charge, Relax, Mover, HealthRest, ManaRest uint8
	Book                                              sim.Spellbook
	Known                                             uint32
}

func returnOracleFromWorld1169(p mapload.PartyMember, e sim.Entity) returnOracle1169 {
	s := e.ActorLoad.Source
	stats := s.Stats
	stats[4], stats[5], stats[6], stats[7] = uint16(e.HumanMovement.RawSpeed), uint16(e.ActorLoad.OwnWeight), uint16(e.Load), uint16(e.Capacity)
	stats[8], stats[9], stats[11], stats[12] = uint16(e.HP), uint16(e.MaxHP), uint16(e.Mana), uint16(e.MaxMana)
	return returnOracle1169{p.ID, p.Name, stats, e.Skill, e.SkillXP, s.Experience, s.Attack, s.Base, s.Defence, s.Modifier, e.Reach, uint8(e.AttackCharge), uint8(e.AttackRelax), s.MoverSpeed, e.HealthHundredths, e.ManaHundredths, e.Book, e.KnownSpells}
}

// These corpus parties have distinct names. Match the raw characters without
// calling the production NPC resolver: a wrong dialogue identity must not make
// an independent inventory/statistics oracle choose a different actor.
func rawCharacterNamed(t *testing.T, party []sav.Character, name string) sav.Character {
	t.Helper()
	var found sav.Character
	count := 0
	for _, c := range party {
		if c.Name == name {
			found, count = c, count+1
		}
	}
	if name == "" || count != 1 {
		t.Fatalf("raw party has %d characters named %q, need exactly one", count, name)
	}
	return found
}

func returnIDs1169(t *testing.T, f *FrontEnd) (sim.EntityID, sim.EntityID) {
	t.Helper()
	var hero, mage sim.EntityID
	for i, p := range f.live.mission.party {
		if p.StartingHero {
			hero = f.live.mission.ids[i]
		}
		if p.Mage {
			mage = f.live.mission.ids[i]
		}
	}
	if hero == 0 || mage == 0 || hero == mage {
		t.Fatal("source identity roles missing")
	}
	return hero, mage
}

func returnCast1169(t *testing.T, f *FrontEnd, mage sim.EntityID, spell uint16) {
	t.Helper()
	e := releaseEntity(t, f.live, mage)
	cmd := sim.Command{Kind: sim.KindCast, Entity: mage, X: int32(mage), Y: int32(spell)}
	if spell == 12 {
		cmd = sim.Command{Kind: sim.KindCastAt, Entity: mage, X: e.X, Y: e.Y, Spell: spell}
	}
	if spell == 12 {
		if reason := f.live.world.BookSpellCellRefusal(mage, e.X, e.Y, uint32(spell)); reason != "" {
			t.Fatal(reason)
		}
	} else if reason := f.live.world.BookSpellRefusal(mage, mage, uint32(spell)); reason != "" {
		t.Fatal(reason)
	}
	events := sim.StepObserved(f.live.world, []sim.Command{cmd})
	for ticks := 0; ticks < 256; ticks++ {
		for _, ev := range events {
			if ev.Caster == mage && ev.Spell == spell && !ev.Weapon {
				if releaseEntity(t, f.live, mage).Mana >= e.Mana {
					t.Fatal("ordinary cast did not spend mana")
				}
				return
			}
		}
		events = sim.StepObserved(f.live.world, nil)
	}
	t.Fatal("ordinary cast did not release", spell)
}

func returnPlayed1169(t *testing.T, win bool) (*FrontEnd, string, []returnOracle1169) {
	t.Helper()
	f, private := townReturnImported1168(t)
	app := f.App("1169 imported mission return")
	if err := app.OpenMission(f.MissionOpenerWith(30, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	hero, mage := returnIDs1169(t, f)
	initialPacks := make(map[sim.EntityID][]sim.ItemInstance)
	initialEquipment := make(map[sim.EntityID][sim.EquipSlots]sim.ItemInstance)
	for _, id := range f.live.mission.ids {
		initialPacks[id], _ = f.live.world.CarriedItems(id)
		initialEquipment[id], _ = f.live.world.EquippedItems(id)
	}
	initial := releaseEntity(t, f.live, mage)
	returnCast1169(t, f, mage, 12)
	returnCast1169(t, f, mage, 18)
	active := releaseEntity(t, f.live, mage)
	if active.SkillXP == initial.SkillXP || active.Skill == initial.Skill || active.Absorption <= initial.Absorption || len(f.live.world.ActiveEffects()) == 0 {
		t.Fatal("XP/stat/effect controls were not armed")
	}
	granted, _ := f.live.world.CarriedItems(hero)
	var withoutQuest []sim.ItemInstance
	questCount := 0
	for _, item := range granted {
		if item.Code == 0x0e1e {
			questCount++
		} else {
			withoutQuest = append(withoutQuest, item)
		}
	}
	if questCount != 1 || !reflect.DeepEqual(withoutQuest, initialPacks[hero]) {
		t.Fatal("mission30 grant was not exactly the quest object")
	}
	if win {
		// The existing mission30 reach fixture places the hero at the victory
		// trigger. The installed script takes its quest Item 0x0e1e and wins;
		// no test deletes a carried item or synthesizes the outcome.
		if err := f.live.world.HeadlessPlace(hero, 64, 15); err != nil {
			t.Fatal(err)
		}
	}
	// Bounded injury controls use the script's health write; this fixture does
	// not claim the hero walked the whole map or took these wounds in combat.
	headlessDamage(t, f.live.world, hero, 13)
	headlessDamage(t, f.live.world, mage, 2)
	sim.Step(f.live.world, nil)
	if win {
		for tick := 0; tick < 96 && f.live.world.Outcome() == sim.OutcomeUndecided; tick++ {
			sim.Step(f.live.world, nil)
		}
		if f.live.world.Outcome() != sim.OutcomeWon {
			t.Fatal("installed mission30 victory trigger did not win")
		}
	}
	before := releaseEntity(t, f.live, mage)
	if before.HP >= before.MaxHP || before.Mana >= before.MaxMana || len(f.live.world.ActiveEffects()) == 0 {
		t.Fatal("return pool/effect controls were not armed")
	}
	if before.SkillXP != active.SkillXP || before.Skill != active.Skill {
		t.Fatal("victory trigger changed earned progression")
	}
	for _, id := range f.live.mission.ids {
		items, _ := f.live.world.CarriedItems(id)
		equipment, _ := f.live.world.EquippedItems(id)
		if !reflect.DeepEqual(equipment, initialEquipment[id]) {
			t.Fatal("victory trigger changed held equipment")
		}
		want := initialPacks[id]
		if !win && id == hero {
			want = granted
		}
		if !reflect.DeepEqual(items, want) {
			t.Fatal("victory trigger changed non-quest holdings")
		}
	}
	party, ids := mapload.CloneParty(f.live.mission.party), append([]sim.EntityID(nil), f.live.mission.ids...)
	if next, line := f.FinishMissionWithRoster(30, f.live.mission.party, f.live.world, f.live.mission.ids, f.live.mission.state.Start.Roster); next < 0 {
		t.Fatal(line)
	}
	var out []returnOracle1169
	for i, p := range party {
		e := releaseEntity(t, f.live, ids[i])
		if e.HP != e.MaxHP || e.Mana != e.MaxMana || e.HasTarget || e.HasAttackTarget || e.Transit != 0 || e.MapUnitID != 0 {
			t.Fatal("live mission-end normalization missing", p.ID)
		}
		out = append(out, returnOracleFromWorld1169(p, e))
	}
	after := releaseEntity(t, f.live, mage)
	if len(f.live.world.ActiveEffects()) != 0 || after.Absorption != initial.Absorption || after.ActorLoad.Source.Modifier != initial.ActorLoad.Source.Modifier || after.SkillXP != before.SkillXP || after.Skill != before.Skill {
		t.Fatal("temporary effect removal lost permanent progression")
	}
	if f.Town.Chapter() != 40 || f.Town.progress == nil {
		t.Fatal("source campaign40 was not retained")
	}
	t.Logf("mission30 win=%t mage XP%v -> %v skill%v -> %v; HP%d/%d mana%d/%d -> full; absorption%d -> %d", win, initial.SkillXP, after.SkillXP, initial.Skill, after.Skill, before.HP, before.MaxHP, before.Mana, before.MaxMana, before.Absorption, after.Absorption)
	return f, private, out
}

// The output oracle reads named primitive/raw CArchive fields, without using
// CityHuman, cityHumanUpdate, SourceHumanState or the saved-current projector.
func assertReturnRaw1169(t *testing.T, raw []byte, expected []returnOracle1169) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var actors []sav.DocumentRecordData
	for _, r := range doc.Objects {
		if r.Class == "Human" {
			actors = append(actors, r)
		}
	}
	if len(actors) != len(expected) {
		t.Fatal("raw Human roster changed")
	}
	for _, want := range expected {
		var found *sav.DocumentRecordData
		// The source DefRow and hero flag are joined independently by the
		// standard Party reader; its stable ID selects the raw object.
		sf, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		chars, err := sf.Party()
		if err != nil {
			t.Fatal(err)
		}
		_, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
		if err != nil {
			t.Fatal(err)
		}
		graph, err := sf.ActorGraph()
		if err != nil {
			t.Fatal(err)
		}
		c := rawCharacterNamed(t, chars, want.Name)
		for _, a := range graph.Actors {
			if a.Identity != c.Key {
				continue
			}
			for _, o := range origins {
				if o.ArchiveIndex == a.ArchiveIndex {
					r := doc.Objects[o.ObjectIndex-1]
					found = &r
				}
			}
		}
		if found == nil {
			t.Fatal("raw character identity missing", want.PartyID)
		}
		r := *found
		for i, name := range []string{"Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity", "Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen"} {
			if got := actorProjectionValue(t, r, name); got != uint32(want.Stats[i]) {
				t.Fatalf("raw %s %s=%d want%d", want.PartyID, name, got, want.Stats[i])
			}
		}
		for _, v := range []struct {
			name  string
			value uint32
		}{{"U130", want.Aggregate}, {"UA2", uint32(want.HealthRest)}, {"UA3", uint32(want.ManaRest)}, {"U12C", uint32(want.Reach)}, {"U134", uint32(want.Charge)}, {"U135", uint32(want.Relax)}, {"Stage", 0}, {"U5C", 0}, {"U64", 0}, {"U44", 0}, {"U40", 0}} {
			if actorProjectionValue(t, r, v.name) != v.value {
				t.Fatal("raw current scalar lost", want.PartyID, v.name)
			}
		}
		for _, block := range r.Raw {
			switch block.Name {
			case "H1CC":
				for i, xp := range want.XP {
					if binary.LittleEndian.Uint32(block.Bytes[4*i:]) != uint32(xp) {
						t.Fatal("raw XP lost", want.PartyID)
					}
				}
			case "UA6":
				if !reflect.DeepEqual(block.Bytes, want.Attack[:]) {
					t.Fatal("raw Attack lost")
				}
				for i, level := range want.Skill {
					if int32(int16(binary.LittleEndian.Uint16(block.Bytes[2+2*i:]))) != level {
						t.Fatal("raw skill lost")
					}
				}
			case "U114":
				if !reflect.DeepEqual(block.Bytes, want.Base[:]) {
					t.Fatal("raw Base lost")
				}
			case "UBE":
				if !reflect.DeepEqual(block.Bytes, want.Defence[:]) {
					t.Fatal("raw Defence lost")
				}
			case "UD4":
				if !reflect.DeepEqual(block.Bytes, want.Modifier[:]) {
					t.Fatal("raw modifier lost")
				}
			case "U154":
				if block.Bytes[10] != want.Mover {
					t.Fatal("raw mover speed lost")
				}
			}
		}
	}
}

func TestReleaseImportedReturn1169SAVColdNextMission(t *testing.T) {
	f, private, want := returnPlayed1169(t, true)
	for _, b := range f.originalCity.bindings {
		if b.returned == nil {
			t.Fatal("real producer did not certify the return", b.partyID)
		}
	}
	store := SaveStore{Dir: t.TempDir()}
	name, raw := townReturnSave1168(t, f, store.Dir)
	assertReturnRaw1169(t, raw, want)
	if err := os.Remove(private); err != nil {
		t.Fatal(err)
	}
	expectPath := filepath.Join(store.Dir, "expected.json")
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(expectPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestImportedReturn1169ColdProcess$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_1169_SAVE="+filepath.Join(store.Dir, name), "AGAINROM_1169_EXPECT="+expectPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cold process: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
	// AGS remains source-free and must keep the separate return, not rebind
	// either source baseline. Its next ordinary SAVE still takes the city SAV.
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	ags, err := EncodeSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err = DecodeSave(ags)
	if err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	if _, town, err := g.Restore(s); err != nil || !town {
		t.Fatal("native return restore", town, err)
	}
	_, again := townReturnSave1168(t, g, t.TempDir())
	assertReturnRaw1169(t, again, want)
}

func TestImportedReturn1169ColdProcess(t *testing.T) {
	f := releaseFront(t)
	path := os.Getenv("AGAINROM_1169_SAVE")
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(os.Getenv("AGAINROM_1169_EXPECT"))
	if err != nil {
		t.Fatal(err)
	}
	var expected []returnOracle1169
	if err := json.Unmarshal(b, &expected); err != nil {
		t.Fatal(err)
	}
	_, _, load := f.SaveSeams(SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
	if _, town, err := load(localOriginalSaveToken(filepath.Base(path))); err != nil || !town {
		t.Fatal("cold ordinary LOAD", town, err)
	}
	if f.Town.Chapter() != 40 || f.Town.progress == nil {
		t.Fatal("cold campaign40 lost")
	}
	_, second := townReturnSave1168(t, f, t.TempDir())
	assertReturnRaw1169(t, second, expected)
	assertReturnRaw1169(t, raw, expected)
	app := f.App("1169 cold next mission")
	if err := app.OpenMission(f.MissionOpenerWith(40, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	for i, p := range f.live.mission.party {
		e := releaseEntity(t, f.live, f.live.mission.ids[i])
		var want *returnOracle1169
		for j := range expected {
			if expected[j].PartyID == p.ID {
				want = &expected[j]
			}
		}
		if want == nil || returnOracleFromWorld1169(p, e) != *want {
			t.Fatal("next mission live Human values changed", p.ID)
		}
		cell := f.live.mission.state.Start.Cells[i]
		if e.X != cell.X || e.Y != cell.Y || e.MapUnitID != 0 || e.Transit != 0 || e.HasTarget || e.HasAttackTarget {
			t.Fatal("historical placement/action resurrected")
		}
	}
	hero, mage := returnIDs1169(t, f)
	e := releaseEntity(t, f.live, hero)
	sim.Step(f.live.world, []sim.Command{{Kind: sim.KindMoveTo, Entity: hero, X: e.X + 2, Y: e.Y}})
	for i := 0; i < 128 && releaseEntity(t, f.live, hero).X == e.X; i++ {
		sim.Step(f.live.world, nil)
	}
	if releaseEntity(t, f.live, hero).X == e.X {
		t.Fatal("next mission move made no progress")
	}
	returnCast1169(t, f, mage, 12)
	t.Log("cold source-free town LOAD: current Human values, campaign40, second SAV, next-map placement, movement and Light mana spend verified")
}

func TestReleaseImportedLoot1172ControlledRetainedGrant(t *testing.T) {
	f, _, _ := returnPlayed1169(t, false)
	for _, b := range f.originalCity.bindings {
		if b.returned == nil || b.returned.Holdings == nil {
			t.Fatal("current grant/pack graph not captured")
		}
	}
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.ExportOriginalSave(s, label); err != nil {
		t.Fatal("current item graph refused", err)
	}
	save, _, _ := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if name, err := save(false); err != nil || !IsOriginal(name) {
		t.Fatal("retained grant did not write SAV", name, err)
	}
	t.Log("controlled return retains actual mission30 quest grant and writes ordinary SAV; real mission30 victory consumes that quest item")
}

func TestReleaseImportedReturn1169LossControls(t *testing.T) {
	f, _, _ := returnPlayed1169(t, true)
	base, _, baseLoaded := townSAVRoundTrip(t, f)
	hero, companion := f.Carried[0], f.Carried[1]
	for _, test := range []struct {
		name   string
		mutate func()
		check  func(t *testing.T, raw []byte, city *sav.CityProvenance, loaded *FrontEnd)
	}{
		{"identity", func() { f.originalCity.bindings[0].returned.Identity++ }, func(t *testing.T, raw []byte, city *sav.CityProvenance, loaded *FrontEnd) {
			stale := f.originalCity.bindings[0].returned.Identity
			for _, c := range city.Roster() {
				if c.Identity == stale {
					t.Fatalf("SAV names %s by the stale certificate %#x", c.Name, stale)
				}
			}
			if !bytes.Equal(raw, base) {
				t.Fatal("a stale return certificate changed the written file")
			}
			for i, b := range loaded.originalCity.bindings {
				if b.partyID != baseLoaded.originalCity.bindings[i].partyID || b.character.Identity != baseLoaded.originalCity.bindings[i].character.Identity || b.character.Identity == stale {
					t.Fatalf("binding %d loaded %s %#x", i, b.partyID, b.character.Identity)
				}
			}
		}},
		{"historical placement", func() {
			saved := mapload.CloneParty([]mapload.PartyMember{f.originalCity.bindings[1].baseline})[0].Saved
			saved.Cell = mapload.Cell{X: 20, Y: 20}
			f.Carried[0].Saved = saved
		}, func(t *testing.T, raw []byte, city *sav.CityProvenance, loaded *FrontEnd) {
			want := f.Carried[0].Saved
			if want == nil || trainingPartyMember(t, baseLoaded, hero.ID).Saved != nil {
				t.Fatal("placement control not armed")
			}
			file, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			party, err := file.Party()
			if err != nil {
				t.Fatal(err)
			}
			if c := rawCharacterNamed(t, party, hero.Name); int32(c.Col()) != want.Cell.X || int32(c.Row()) != want.Cell.Y {
				t.Fatalf("SAV placement %d,%d, live %+v", c.Col(), c.Row(), want.Cell)
			}
			if got := trainingPartyMember(t, loaded, hero.ID).Saved; got == nil || got.Cell != want.Cell {
				t.Fatalf("loaded placement %+v, live %+v", got, want)
			}
		}},
		{"timed potion", func() { f.Carried[0].PotionEffect = &sim.ActiveEffect{} }, func(t *testing.T, raw []byte, city *sav.CityProvenance, loaded *FrontEnd) {
			if trainingPartyMember(t, baseLoaded, hero.ID).PotionEffect != nil {
				t.Fatal("potion control not armed")
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil {
				t.Fatal("SAV has no current party", err)
			}
			written := 0
			for _, p := range a.Party {
				if string(p.ID) == hero.ID && p.City != nil && p.City.Potion != nil && *p.City.Potion == *f.Carried[0].PotionEffect {
					written++
				}
			}
			if written != 1 {
				t.Fatal("SAV does not carry the current potion")
			}
			if got := trainingPartyMember(t, loaded, hero.ID).PotionEffect; got == nil || *got != *f.Carried[0].PotionEffect {
				t.Fatalf("loaded potion %+v", got)
			}
		}},
		{"current XP", func() { f.Carried[0].Carry.SkillXP[1]++ }, func(t *testing.T, raw []byte, city *sav.CityProvenance, loaded *FrontEnd) {
			want := f.Carried[0].Carry.SkillXP
			if want == trainingPartyMember(t, baseLoaded, hero.ID).Carry.SkillXP {
				t.Fatal("XP control not armed")
			}
			found := 0
			for _, c := range city.Roster() {
				if c.Name == hero.Name {
					found++
					for i := range want {
						if int32(c.SkillXP[i]) != want[i] {
							t.Fatalf("SAV XP %v, live %v", c.SkillXP, want)
						}
					}
				}
			}
			if found != 1 {
				t.Fatal("SAV roster lacks", hero.Name)
			}
			if got := trainingPartyMember(t, loaded, hero.ID).Carry.SkillXP; got != want {
				t.Fatalf("loaded XP %v, live %v", got, want)
			}
			alteredXP := want
			alteredXP[1] += 7
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "altered.sav"), alterSAV(t, raw, func(doc *sav.DocumentData) bool { return setSavedHumanXP(doc, want, alteredXP) }), 0o600); err != nil {
				t.Fatal(err)
			}
			lossy := releaseFront(t)
			openLocalTownSAV(t, lossy, dir, "altered.sav")
			if got := trainingPartyMember(t, lossy, hero.ID).Carry.SkillXP; got != alteredXP {
				t.Fatalf("altered SAV XP loaded as %v, want %v", got, alteredXP)
			}
			for _, g := range []*FrontEnd{f, loaded} {
				if msg := schoolTrain(g, hero.ID); !strings.HasPrefix(msg, "trained ") {
					t.Fatal("next school action", msg)
				}
			}
			live, cold := trainingPartyMember(t, f, hero.ID), trainingPartyMember(t, loaded, hero.ID)
			liveHuman, liveOK := live.OriginalHumanState()
			coldHuman, coldOK := cold.OriginalHumanState()
			if live.Carry.SkillXP != cold.Carry.SkillXP || f.Town.Gold() != loaded.Town.Gold() || liveOK != coldOK || liveHuman != coldHuman {
				t.Fatalf("next school action diverged: live %v gold %d, loaded %v gold %d", live.Carry.SkillXP, f.Town.Gold(), cold.Carry.SkillXP, loaded.Town.Gold())
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			carried := mapload.CloneParty(f.Carried)
			identity := f.originalCity.bindings[0].returned.Identity
			defer func() { f.Carried, f.originalCity.bindings[0].returned.Identity = carried, identity }()
			test.mutate()
			raw, city, loaded := townSAVRoundTrip(t, f)
			if test.name != "identity" && bytes.Equal(raw, base) {
				t.Fatal("current change not written")
			}
			test.check(t, raw, city, loaded)
			if got := trainingPartyMember(t, loaded, companion.ID).Carry.SkillXP; got != companion.Carry.SkillXP {
				t.Fatal("unchanged companion lost XP", got)
			}
		})
	}
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		written bool
		mutate  func(*Snapshot)
	}{
		{"world handle", false, func(s *Snapshot) { s.Party[0].Carry.ItemInstances[0].ObjectID = 999 }},
		{"equipment", true, func(s *Snapshot) {
			worn := s.Party[0].Carry.EquippedItems[:]
			worn[slices.IndexFunc(worn, func(item sim.ItemInstance) bool { return item.Code != 0 })].Price++
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			current, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&current)
			raw, err := f.ExportCurrentSave(current, label)
			if !test.written && err == nil {
				t.Fatal("impossible world handle accepted")
			}
			if test.written && (err != nil || bytes.Equal(raw, unchanged)) {
				t.Fatal("current change not written", err)
			}
		})
	}
	t.Run("current modifier", func(t *testing.T) {
		current, _, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		state, err := originalCityFromSnapshot(current)
		if err != nil {
			t.Fatal(err)
		}
		r := state.bindings[0].returned
		r.Member.Carry.LiveLoad.Inventory.Source.Modifier[44]++
		want := r.Member.Carry.LiveLoad.Inventory.Source.Modifier
		for i := range current.Party {
			if current.Party[i].ID == r.Member.ID {
				current.Party[i] = mapload.CloneParty([]mapload.PartyMember{r.Member})[0]
			}
		}
		raw, err := state.marshal(f, current, label)
		if err != nil {
			t.Fatal("changed current modifier refused", err)
		}
		d, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		for i := range d.Objects {
			for _, field := range d.Objects[i].Raw {
				if d.Objects[i].Class == "Human" && field.Name == "UD4" && bytes.Equal(field.Bytes, want[:]) {
					return
				}
			}
		}
		t.Fatal("no written Human carries the current modifier block")
	})
	for _, test := range []struct {
		name   string
		mutate func(*Snapshot)
	}{
		{"forged baseline", func(s *Snapshot) { s.OriginalCity.Bindings[0].Baseline.Hero.Body++ }},
		{"forged identity", func(s *Snapshot) {
			s.OriginalCity.Bindings[0].PartyID = "forged"
			s.OriginalCity.Bindings[0].Baseline.ID = "forged"
			s.OriginalCity.Bindings[0].Returned.Member.ID = "forged"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			current, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&current)
			state, err := originalCityFromSnapshot(current)
			if err == nil {
				err = f.townInstall().validateOriginalCityBaseline(state)
			}
			if err == nil {
				t.Fatal("forged source baseline or identity admitted")
			}
		})
	}
	ids := append([]sim.EntityID(nil), f.live.mission.ids...)
	ids[0] = 0x7fffffff
	f.captureMissionReturn(30, f.live.mission.party, f.live.world, ids, true, nil, nil)
	for _, b := range f.originalCity.bindings {
		if b.returned != nil {
			t.Fatal("missing actor retained stale certification")
		}
	}
	_, _, uncaptured := townSAVRoundTrip(t, f)
	for _, m := range f.Carried {
		if got := trainingPartyMember(t, uncaptured, m.ID); got.Carry.SkillXP != m.Carry.SkillXP || got.Name != m.Name {
			t.Fatalf("uncaptured return loaded %s XP %v, live %v", m.ID, got.Carry.SkillXP, m.Carry.SkillXP)
		}
	}
	t.Run("failed inverse preserves mission and town", func(t *testing.T) {
		g, _ := townReturnImported1168(t)
		app := g.App("1169 failed cleanup")
		if err := app.OpenMission(g.MissionOpenerWith(30, g.NextParty())); err != nil {
			t.Fatal(err)
		}
		_, mage := returnIDs1169(t, g)
		returnCast1169(t, g, mage, 18)
		g.live.world.BindSourceDerive(func(s sim.SourceActor, _ int32, _ sim.Rules) (sim.SourceActor, error) {
			return s, fmt.Errorf("unsupported inverse")
		})
		worldBefore := g.live.world.Hash()
		before, _, err := g.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		if next, _ := g.FinishMissionWithRoster(30, g.live.mission.party, g.live.world, g.live.mission.ids, g.live.mission.state.Start.Roster); next >= 0 {
			t.Fatal("failed inverse completed campaign")
		}
		after, _, err := g.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		if g.live.world.Hash() != worldBefore || !reflect.DeepEqual(before, after) {
			t.Fatal("failed inverse partly published return")
		}
	})
}

func setSavedHumanXP(doc *sav.DocumentData, from, to [6]int32) bool {
	var want, next [24]byte
	for i := range from {
		binary.LittleEndian.PutUint32(want[4*i:], uint32(from[i]))
		binary.LittleEndian.PutUint32(next[4*i:], uint32(to[i]))
	}
	found := 0
	for i := range doc.Objects {
		for j := range doc.Objects[i].Raw {
			if r := &doc.Objects[i].Raw[j]; doc.Objects[i].Class == "Human" && r.Name == "H1CC" && bytes.Equal(r.Bytes, want[:]) {
				r.Bytes = bytes.Clone(next[:])
				found++
			}
		}
	}
	return found == 1
}
