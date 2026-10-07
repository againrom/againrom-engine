package game

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// skillRuleS is the threshold table written out from the experience rule
// itself, S(n) = ftol((1.1^n - 1) * 1000), so the assertions below hold no
// engine arithmetic.
func skillRuleS(n int32) int32 { return int32((math.Pow(1.1, float64(n)) - 1) * 1000) }

func TestSkillRuleThresholds(t *testing.T) {
	for n, want := range map[int32]int32{19: 5115, 20: 5727, 21: 6400, 22: 7140, 25: 9834, 54: 170871, 55: 188059} {
		if got := skillRuleS(n); got != want {
			t.Fatalf("S(%d) = %d, want %d", n, got, want)
		}
		if got := data.SkillXPFor(n); got != want {
			t.Fatalf("data.SkillXPFor(%d) = %d, want %d", n, got, want)
		}
	}
}

// A slot is settled when its level is the one its experience implies: the raise
// fires only for experience strictly above S(level), one level per award, so a
// settled slot holds S(level-1) < xp <= S(level) once it has been raised and
// xp <= S(level) at creation.
func TestRepairSkillLevelBoundaries(t *testing.T) {
	cases := []struct{ level, xp, want int32 }{
		{22, 171859, 55},
		{22, skillRuleS(22), 22},
		{22, skillRuleS(23), 22},
		{22, skillRuleS(23) + 1, 24},
		{22, skillRuleS(54), 54},
		{22, skillRuleS(54) + 1, 55},
		{55, 186990, 55},
		{0, 1 << 30, 0},
		{99, 1 << 30, 100},
		{100, 1 << 30, 100},
		{1, 0, 1},
	}
	for _, c := range cases {
		if got := data.RepairSkillLevel(c.level, c.xp); got != c.want {
			t.Errorf("RepairSkillLevel(%d, %d) = %d, want %d", c.level, c.xp, got, c.want)
		}
	}
}

// savSheet is one character's skill fields as the SAV bytes hold them.
type savSheet struct {
	hero   bool
	levels [6]int32
	base   [6]int32
	xp     [6]uint32
	sum    uint32
	stats  [14]uint16
	sight  uint16
	known  uint32
	name   string
	worn   int
}

func savSheets(t *testing.T, raw []byte) []savSheet {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := file.Walk()
	if err != nil || rec == nil {
		t.Fatalf("walk: %v", err)
	}
	chars, err := file.Party()
	if err != nil {
		t.Fatal(err)
	}
	var out []savSheet
	for _, c := range chars {
		for _, a := range rec.Refs["Actors"] {
			if a.Off != c.Off || a.Class != "Human" {
				continue
			}
			attack, base := a.Raw["UA6"], a.Raw["U114"]
			if len(attack) != 24 || len(base) != 24 {
				t.Fatalf("attack/base block widths %d/%d", len(attack), len(base))
			}
			s := savSheet{hero: c.Hero, xp: c.SkillXP, sum: uint32(a.Value["U130"]), stats: c.Stats, sight: uint16(a.Value["UA4"]), known: c.KnownSpells(), name: c.Name, worn: len(c.Worn)}
			for i := 0; i < 6; i++ {
				s.levels[i] = int32(int16(uint16(attack[2+2*i]) | uint16(attack[3+2*i])<<8))
				s.base[i] = int32(int16(uint16(base[2+2*i]) | uint16(base[3+2*i])<<8))
			}
			out = append(out, s)
		}
	}
	return out
}

func savHero(t *testing.T, raw []byte) savSheet {
	t.Helper()
	for _, s := range savSheets(t, raw) {
		if s.hero {
			return s
		}
	}
	t.Fatal("save holds no starting hero")
	return savSheet{}
}

func liveHeroSheet(e sim.Entity) savSheet {
	s := savSheet{hero: true}
	for i := range s.levels {
		s.levels[i] = e.Skill[i]
		s.base[i] = e.Skill[i]
		s.xp[i] = uint32(e.SkillXP[i])
		s.sum += s.xp[i]
	}
	if e.NativeTraining.Present {
		s.base = e.NativeTraining.Levels
	}
	return s
}

// settledSkill checks the band the experience rule leaves a slot in.
func settledSkill(level int32, xp uint32) bool {
	if level <= 0 {
		return true
	}
	return int64(xp) <= int64(skillRuleS(level)) && (level == 1 || int64(xp) > int64(skillRuleS(level-1)))
}

func assertSheet(t *testing.T, label string, got, live savSheet, checkBase bool) {
	t.Helper()
	for i := 1; i < 6; i++ {
		if got.levels[i] != live.levels[i] {
			t.Errorf("%s: level word slot %d = %d, live level %d", label, i, got.levels[i], live.levels[i])
		}
		if checkBase && got.base[i] != live.base[i] {
			t.Errorf("%s: base word slot %d = %d, live base %d", label, i, got.base[i], live.base[i])
		}
		if got.xp[i] != live.xp[i] {
			t.Errorf("%s: experience slot %d = %d, live %d", label, i, got.xp[i], live.xp[i])
		}
		if !settledSkill(got.levels[i], got.xp[i]) {
			t.Errorf("%s: slot %d level %d is out of band for experience %d", label, i, got.levels[i], got.xp[i])
		}
	}
	if got.sum != live.sum {
		t.Errorf("%s: +0x130 = %d, sum of slot experience %d", label, got.sum, live.sum)
	}
}

type skillCase struct {
	name    string
	mage    bool
	levels  [6]int32
	spell   uint32
	slot    int
	raises  int
	maxCast int
}

func skillCases() []skillCase {
	return []skillCase{
		{"mage fire 1", true, [6]int32{0, 1, 0, 0, 0, 0}, 1, 1, 2, 60},
		{"mage fire 20", true, [6]int32{0, 20, 0, 0, 0, 0}, 1, 1, 2, 60},
		{"mage fire 50 mixed", true, [6]int32{0, 50, 20, 50, 99, 0}, 1, 1, 2, 80},
		{"mage fire 99", true, [6]int32{0, 99, 0, 0, 0, 0}, 1, 1, 1, 10},
		{"mage water 20 mixed", true, [6]int32{0, 20, 20, 50, 99, 0}, 6, 2, 2, 60},
		{"mage air 50 mixed", true, [6]int32{0, 1, 20, 50, 99, 0}, 12, 3, 2, 60},
		{"fighter weapon 20", false, [6]int32{0, 20, 0, 0, 0, 0}, 0, 1, 2, 60},
		{"fighter weapon 50", false, [6]int32{0, 50, 0, 0, 0, 0}, 0, 1, 2, 80},
	}
}

func skillCaseParty(f *FrontEnd, c skillCase) []mapload.PartyMember {
	party := MissionPartyAs(c.mage, f.StartWeapon.Value(), f.Bodies, f.Table)
	hero := &party[0]
	hero.Hero.Skill = c.levels
	carry := &mapload.Carry{}
	for i, level := range c.levels {
		if level > 0 {
			carry.SkillXP[i] = skillRuleS(level)
		}
	}
	hero.Carry = carry
	return party
}

func openSkillMission(t *testing.T, f *FrontEnd, mission int, party []mapload.PartyMember) sim.EntityID {
	t.Helper()
	f.SetDeterministicFrames(true)
	app := f.App("skill")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
		t.Fatal(err)
	}
	return f.live.mission.ids[0]
}

// actSkill makes one cast, or one melee swing, by the hero at the nearest
// hostile actor and lets it resolve.
func actSkill(t *testing.T, f *FrontEnd, id sim.EntityID, spell uint32) {
	t.Helper()
	hero := releaseEntity(t, f.live, id)
	var hostile []sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.Owner != hero.Owner && e.Owner != 0 && e.HP > 0 && e.MaxHP > 100 {
			hostile = append(hostile, e)
		}
	}
	sort.Slice(hostile, func(i, j int) bool {
		di := (hostile[i].X-hero.X)*(hostile[i].X-hero.X) + (hostile[i].Y-hero.Y)*(hostile[i].Y-hero.Y)
		dj := (hostile[j].X-hero.X)*(hostile[j].X-hero.X) + (hostile[j].Y-hero.Y)*(hostile[j].Y-hero.Y)
		return di < dj
	})
	gap := int32(3)
	if spell == 0 {
		gap = 1
	}
	var victim sim.Entity
	for _, e := range hostile {
		if f.live.world.HeadlessPlace(id, e.X-gap, e.Y) == nil {
			victim = e
			break
		}
	}
	if victim.ID == 0 {
		t.Fatal("mission holds no reachable hostile actor to train on")
	}
	for tick := 0; releaseEntity(t, f.live, id).CastWait != 0 && tick < 512; tick++ {
		f.live.tick()
	}
	_ = f.live.world.HeadlessHeal(id)
	_ = f.live.world.HeadlessHeal(victim.ID)
	f.live.attackOrCast(uint32(id), uint32(victim.ID), spell, int(victim.X), int(victim.Y), false)
	for tick := 0; tick < 48; tick++ {
		f.live.tick()
	}
}

func trainSkill(t *testing.T, f *FrontEnd, id sim.EntityID, c skillCase, mustRise bool) {
	t.Helper()
	raised, start := 0, releaseEntity(t, f.live, id).Skill[c.slot]
	for n := 0; n < c.maxCast && raised < c.raises; n++ {
		before := releaseEntity(t, f.live, id).Skill[c.slot]
		actSkill(t, f, id, c.spell)
		after := releaseEntity(t, f.live, id)
		if after.Skill[c.slot] > before+2 {
			t.Fatalf("one cast (a cast award and a hit award) raised slot %d from %d to %d", c.slot, before, after.Skill[c.slot])
		}
		if after.Skill[c.slot] > before {
			raised++
		}
		if after.Skill[c.slot] >= 100 {
			break
		}
	}
	if mustRise && releaseEntity(t, f.live, id).Skill[c.slot] == start {
		t.Fatalf("slot %d did not rise from %d", c.slot, start)
	}
}

func skillExport(t *testing.T, f *FrontEnd, onMap bool, how string) []byte {
	t.Helper()
	snap, label, err := f.Snapshot(onMap)
	if err != nil {
		t.Fatal(err)
	}
	if how == "autosave" {
		view, captured := f.detachedExporter(snap)
		raw, _, err := view.playerMissionSave(captured, "skill autosave")
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	raw, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func coldMission(t *testing.T, raw []byte) (*FrontEnd, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open, _, err := f.RestoreOriginal(raw)
	if err != nil || open == nil {
		t.Fatalf("cold LOAD of a mission save: open=%v err=%v", open != nil, err)
	}
	app := f.App("skill-cold")
	app.Layout(1024, 768)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f, f.live.mission.ids[0]
}

func coldHeroLevels(t *testing.T, label string, f *FrontEnd, id sim.EntityID, live sim.Entity) {
	t.Helper()
	e := releaseEntity(t, f.live, id)
	if e.Skill != live.Skill || e.SkillXP != live.SkillXP {
		t.Errorf("%s: cold LOAD hero skill %v xp %v, live skill %v xp %v", label, e.Skill, e.SkillXP, live.Skill, live.SkillXP)
	}
}

// A level raised in play stays raised through every save path, a cold LOAD, the
// carry home, the town SAVE, its cold LOAD and the next mission entry, with the
// experience per slot unchanged and in band at every step.
func TestReleaseSkillLevelsSurviveEverySavePath(t *testing.T) {
	const mission = 90
	for _, c := range skillCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := releaseFront(t)
			id := openSkillMission(t, f, mission, skillCaseParty(f, c))
			start := releaseEntity(t, f.live, id)
			for i, level := range c.levels {
				if start.Skill[i] != level {
					t.Fatalf("mission entry slot %d = %d, want %d", i, start.Skill[i], level)
				}
			}
			exerciseSavePaths(t, f, id, c, mission)
		})
	}
}

// exerciseSavePaths trains the hero's slot until it rises, then drives every
// save path and checks the level words, the base words and the experience.
func exerciseSavePaths(t *testing.T, f *FrontEnd, id sim.EntityID, c skillCase, mission int) {
	t.Helper()
	trainSkill(t, f, id, c, true)
	_ = f.live.world.HeadlessDamage(id, 3)
	live := releaseEntity(t, f.live, id)
	want := liveHeroSheet(live)
	for i := 1; i < 6; i++ {
		if !settledSkill(live.Skill[i], uint32(live.SkillXP[i])) {
			t.Fatalf("live slot %d level %d is out of band for experience %d", i, live.Skill[i], live.SkillXP[i])
		}
	}
	for _, how := range []string{"mission SAVE", "autosave"} {
		raw := skillExport(t, f, true, how)
		assertSheet(t, how, savHero(t, raw), want, true)
		for _, m := range censusMismatch(savHero(t, raw), live, false) {
			t.Errorf("%s: census: %s", how, m)
		}
		for _, m := range censusIdentity(savHero(t, raw), f, f.live.mission.party[0], id) {
			t.Errorf("%s: census: %s", how, m)
		}
		cold, cid := coldMission(t, raw)
		coldHeroLevels(t, how, cold, cid, live)
		trainSkill(t, cold, cid, skillCase{c.name, c.mage, c.levels, c.spell, c.slot, 99, 6}, false)
		again := releaseEntity(t, cold.live, cid)
		for i := 1; i < 6; i++ {
			if again.Skill[i] < live.Skill[i] {
				t.Errorf("%s: continuing after cold LOAD lowered slot %d from %d to %d", how, i, live.Skill[i], again.Skill[i])
			}
			if !settledSkill(again.Skill[i], uint32(again.SkillXP[i])) {
				t.Errorf("%s: continuing after cold LOAD left slot %d level %d out of band for experience %d", how, i, again.Skill[i], again.SkillXP[i])
			}
		}
	}
	raw := skillExport(t, f, true, "mission SAVE")
	home, _ := coldMission(t, raw)
	ms := home.live.mission
	if refused := home.CampaignSession.carryMissionHome(home.townInstall(), mission, ms.party, home.live.world, ms.ids, nil); refused != "" {
		t.Fatal(refused)
	}
	town := skillExport(t, home, false, "town SAVE")
	assertSheet(t, "town SAVE", savHero(t, town), want, false)
	for _, m := range censusMismatch(savHero(t, town), live, true) {
		t.Errorf("town SAVE: census: %s", m)
	}
	cold := releaseFront(t)
	open, isCity, err := cold.RestoreOriginal(town)
	if err != nil || open != nil || !isCity {
		t.Fatalf("cold LOAD of the town save: open=%v city=%v err=%v", open != nil, isCity, err)
	}
	carried := cold.NextParty()[0]
	if carried.Hero.Skill != live.Skill {
		t.Errorf("town cold LOAD hero skill %v, live %v", carried.Hero.Skill, live.Skill)
	}
	for i := 1; i < 6; i++ {
		if carried.Carry == nil || carried.Carry.SkillXP[i] != live.SkillXP[i] {
			t.Errorf("town cold LOAD slot %d experience %v, live %d", i, carried.Carry, live.SkillXP[i])
		}
	}
	cold.SetDeterministicFrames(true)
	app := cold.App("skill-town")
	app.Layout(1024, 768)
	if err := app.OpenMission(cold.MissionOpener(mission)); err != nil {
		t.Fatal(err)
	}
	entry := releaseEntity(t, cold.live, cold.live.mission.ids[0])
	if entry.Skill != live.Skill || entry.SkillXP != live.SkillXP {
		t.Errorf("town to mission entry skill %v xp %v, live skill %v xp %v", entry.Skill, entry.SkillXP, live.Skill, live.SkillXP)
	}
}

const beforeKargallasSHA = "86605e1b3d7e851d0a9c8c63e4312478f8bfd2bf4bebcc4e3f12ec97973510a8"

// The owner's town save holds Fire level 22 beside 171859 Fire experience. The
// rule raises a slot one level per award for experience above S(level), so the
// save's own experience implies level 55, which is what the player had.
func TestReleaseBeforeKargallasLoadsFireAtItsExperienceLevel(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-10-07/beforekargallas.sav", beforeKargallasSHA)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open, isCity, err := f.RestoreOriginal(raw)
	if err != nil || open != nil || !isCity {
		t.Fatalf("restore: open=%v city=%v err=%v", open != nil, isCity, err)
	}
	var reniesta, hero *mapload.PartyMember
	party := f.NextParty()
	for i := range party {
		switch {
		case party[i].Hero.Mind == 30 && party[i].Hero.Spirit == 42:
			reniesta = &party[i]
		case party[i].StartingHero:
			hero = &party[i]
		}
	}
	if reniesta == nil || hero == nil {
		t.Fatal("party lacks the hero or Reniesta")
	}
	if reniesta.Hero.Skill != [6]int32{0, 55, 55, 0, 21, 25} {
		t.Errorf("Reniesta skill %v, want Fire 55 Water 55 Air 0 Earth 21 Astral 25", reniesta.Hero.Skill)
	}
	if reniesta.Carry.SkillXP != [6]int32{0, 171859, 186990, 0, 6354, 9606} {
		t.Errorf("Reniesta experience %v changed", reniesta.Carry.SkillXP)
	}
	if hero.Hero.Skill[4] != 62 || hero.Carry.SkillXP[4] != 357412 {
		t.Errorf("hero Earth %d experience %d, want 62 and 357412", hero.Hero.Skill[4], hero.Carry.SkillXP[4])
	}
	for _, m := range party {
		for i := 1; i < 6; i++ {
			if m.Hero.Skill[i] > 0 && m.Carry.SkillXP[i] > skillRuleS(m.Hero.Skill[i]+1) {
				t.Errorf("%q slot %d level %d is out of band for experience %d", m.Name, i, m.Hero.Skill[i], m.Carry.SkillXP[i])
			}
		}
	}
	town := skillExport(t, f, false, "town SAVE")
	for _, s := range savSheets(t, town) {
		for i := 1; i < 6; i++ {
			if s.levels[i] > 0 && int64(s.xp[i]) > int64(skillRuleS(s.levels[i]+1)) {
				t.Errorf("re-saved level %d out of band for experience %d", s.levels[i], s.xp[i])
			}
		}
		var sum uint32
		for _, x := range s.xp {
			sum += x
		}
		if s.sum != sum {
			t.Errorf("re-saved +0x130 = %d, sum of slot experience %d", s.sum, sum)
		}
	}
	// A town save written for the loaded party keeps Fire at 55 and the next
	// Fire cast raises it by at most one level.
	app := f.App("kargallas")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(90)); err != nil {
		t.Fatal(err)
	}
	var rid sim.EntityID
	for _, id := range f.live.mission.ids {
		if e := releaseEntity(t, f.live, id); e.Mind == 30 && e.MaxMana > 200 {
			rid = id
		}
	}
	before := releaseEntity(t, f.live, rid)
	if before.Skill[1] != 55 {
		t.Fatalf("mission entry Fire %d, want 55", before.Skill[1])
	}
	actSkill(t, f, rid, 1)
	after := releaseEntity(t, f.live, rid)
	if after.Skill[1] > 56 {
		t.Errorf("one Fire cast moved Fire from %d to %d", before.Skill[1], after.Skill[1])
	}
}

// No save the corpus holds loads into a slot whose level lies below its band.
func TestReleaseCorpusLoadsNoOutOfBandSlot(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS")
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	loaded := 0
	err := filepath.WalkDir(corpus, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".sav" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, _, err := f.RestoreOriginal(raw); err != nil {
			return nil
		}
		loaded++
		for _, m := range f.NextParty() {
			if m.Carry == nil {
				continue
			}
			for i := 1; i < 6; i++ {
				if m.Hero.Skill[i] > 0 && m.Carry.SkillXP[i] > skillRuleS(m.Hero.Skill[i]+1) {
					t.Errorf("%s: %q slot %d level %d is out of band for experience %d", path, m.Name, i, m.Hero.Skill[i], m.Carry.SkillXP[i])
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if loaded == 0 {
		t.Fatal("no corpus save loaded")
	}
}

// An original-written save loads its hero as a source-backed actor. A level
// raised there survives the same save paths.
func TestReleaseOriginalLoadedHeroLevelsSurviveEverySavePath(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-10-07/beforekargallas.sav", beforeKargallasSHA)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if _, isCity, err := f.RestoreOriginal(raw); err != nil || !isCity {
		t.Fatalf("restore: city=%v err=%v", isCity, err)
	}
	app := f.App("orig")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(90)); err != nil {
		t.Fatal(err)
	}
	id := f.live.mission.ids[0]
	exerciseSavePaths(t, f, id, skillCase{"original mage air", true, [6]int32{}, 12, 3, 1, 150}, 90)
}

// censusMismatch lists every modelled Human field whose saved value differs from
// the live actor's, so a field taken from a stale document snapshot is named.
func censusMismatch(got savSheet, e sim.Entity, town bool) []string {
	var bad []string
	check := func(name string, saved, live int64) {
		if saved != live {
			bad = append(bad, fmt.Sprintf("%s saved %d live %d", name, saved, live))
		}
	}
	live := liveHeroSheet(e)
	for i := 1; i < 6; i++ {
		check(fmt.Sprintf("level[%d]", i), int64(got.levels[i]), int64(live.levels[i]))
		check(fmt.Sprintf("base[%d]", i), int64(got.base[i]), int64(live.base[i]))
		check(fmt.Sprintf("xp[%d]", i), int64(got.xp[i]), int64(live.xp[i]))
	}
	check("total +0x130", int64(got.sum), int64(live.sum))
	check("Reaction", int64(int16(got.stats[1])), int64(e.Reaction))
	check("Mind", int64(int16(got.stats[2])), int64(e.Mind))
	check("Spirit", int64(int16(got.stats[3])), int64(e.Spirit))
	if !town {
		check("Health", int64(int16(got.stats[8])), int64(e.HP))
		check("Mana", int64(int16(got.stats[11])), int64(e.Mana))
	}
	check("HealthMax", int64(int16(got.stats[9])), int64(e.MaxHP))
	check("ManaMax", int64(int16(got.stats[12])), int64(e.MaxMana))
	if !town {
		// The town writer takes the carried-weight word from the reloaded
		// carry, which differs from the mission's by the carried stack weight;
		// that word is outside this census.
		check("Load", int64(got.stats[6]), int64(e.Load))
	}
	check("Capacity", int64(got.stats[7]), int64(e.Capacity))
	check("Sight", int64(got.sight), int64((e.Mind+e.Reaction)*256/25+4*256))
	return bad
}

func TestReleaseWriterCensusNative(t *testing.T) {
	const mission = 90
	for _, c := range skillCases()[:1] {
		f := releaseFront(t)
		id := openSkillMission(t, f, mission, skillCaseParty(f, c))
		trainSkill(t, f, id, c, true)
		_ = f.live.world.HeadlessDamage(id, 3)
		live := releaseEntity(t, f.live, id)
		for _, how := range []string{"mission SAVE", "autosave"} {
			raw := skillExport(t, f, true, how)
			for _, m := range censusMismatch(savHero(t, raw), live, false) {
				t.Errorf("%s: %s", how, m)
			}
		}
	}
}

// censusIdentity checks the fields a mission does not change: the name, the
// known spells and the worn equipment still equal the live party member's.
func censusIdentity(got savSheet, f *FrontEnd, m mapload.PartyMember, id sim.EntityID) []string {
	var bad []string
	if got.name != m.Name {
		bad = append(bad, fmt.Sprintf("name saved %q live %q", got.name, m.Name))
	}
	if got.known != m.KnownSpells {
		bad = append(bad, fmt.Sprintf("known spells saved %#x live %#x", got.known, m.KnownSpells))
	}
	worn := 0
	if items, ok := f.live.world.EquippedItems(id); ok {
		for _, item := range items {
			if item.Code != 0 {
				worn++
			}
		}
	}
	if got.worn != worn {
		bad = append(bad, fmt.Sprintf("worn pieces saved %d live %d", got.worn, worn))
	}
	return bad
}

// modelledHumanFields are the Human fields the writer census compares with live
// state; none may be taken from the loaded document.
var modelledHumanFields = []string{"Attack.Skill", "Base.Skill", "SkillXP", "Experience", "Body", "Reaction", "Mind", "Spirit",
	"Sight", "Speed", "Health", "HealthMax", "Mana", "ManaMax", "Load", "Capacity", "Spellbook", "Equipment", "Name"}

func TestSnapshotHumanFieldsHoldNoModelledField(t *testing.T) {
	for _, s := range snapshotHumanFields {
		if s.Reason == "" {
			t.Errorf("snapshot field %s has no reason", s.Field)
		}
		for _, m := range modelledHumanFields {
			if s.Field == m {
				t.Errorf("modelled field %s is taken from the loaded document as %s", m, s.Field)
			}
		}
	}
}
