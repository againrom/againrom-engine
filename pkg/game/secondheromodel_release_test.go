package game

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// secondSheetPool is R2-ENGINE-290's maximum, computed here apart from the
// engine's derive: trunc(trunc(stat*base + log1.1(xp/5000+1)*k) *
// (1.1^stat/100+1)).
func secondSheetPool(stat, base, k, xp int32) int32 {
	inner := math.Trunc(float64(stat*base) + math.Log(float64(xp)/5000+1)/math.Log(1.1)*float64(k))
	return int32(inner * (math.Pow(1.1, float64(stat))/100 + 1))
}

// secondSheet is the level 1 health, mana, speed and sight of R2-ENGINE-290's
// four templates, before items (fighters' mana field is zero).
var secondSheet = map[string][4]int32{
	"Start_MF": {142, 0, 19, 6},
	"Start_FF": {123, 0, 19, 6},
	"Start_FM": {29, 157, 16, 6},
	"Start_MM": {42, 99, 16, 6},
}

// Every picture and chosen skill of a new second-game hero: the chosen skill
// at 20 and skill 5 at 10, a chosen skill 5 at 10 (R2-ENGINE-320), the
// experience their curve gives, and health and mana by R2-ENGINE-290; at the
// templates' statistics with 7320 experience, the claim's sheet.
func TestReleaseSecondNewHeroLevelOneSheet(t *testing.T) {
	f := secondGameFront(t)
	for _, h := range f.generator().PreCreate.Heroes {
		row, _, ok := generatorTemplate(missionHumans(f.Table), h.Template)
		if !ok {
			t.Fatalf("no template row %s", h.Template)
		}
		for skill := range data.SkillSlots - 1 {
			res := ui.ChargenResult{Name: "Hero", Choices: []int{h.Sex, h.Class, skill},
				Stats: []int{int(row.Body), int(row.Reaction), int(row.Mind), int(row.Spirit)}}
			party := f.ChargenParty(res)
			if len(party) != 1 {
				t.Fatalf("%s skill %d: party of %d", h.Template, skill, len(party))
			}
			m := party[0]
			slot := data.SkillBlade + int32(skill)
			want := [data.SkillSlots]int32{}
			want[slot] = 20
			want[data.SkillSlots-1] = 10
			if m.Hero.Skill != want || m.MainSkill != slot {
				t.Fatalf("%s skill %d: skills %v main %d, want %v main %d", h.Template, skill, m.Hero.Skill, m.MainSkill, want, slot)
			}
			d, hp, mp := mapload.PartyDisplayWithTable(m, f.Table)
			xp := data.SkillXPFor(want[slot])
			if slot != data.SkillSlots-1 {
				xp += data.SkillXPFor(10)
			}
			if d.Experience != xp {
				t.Fatalf("%s skill %d: experience %d, want %d", h.Template, skill, d.Experience, xp)
			}
			health, mana := secondSheetPool(row.Body, 1, 1, xp), int32(0)
			if h.Class == 0 {
				health = secondSheetPool(row.Body, 2, 2, xp)
			} else {
				mana = secondSheetPool(row.Spirit, 2, 2, xp)
			}
			if hp != health || d.HealthMax != health || mp != mana || d.ManaMax != mana {
				t.Fatalf("%s skill %d: health %d/%d mana %d/%d, want %d and %d", h.Template, skill, hp, d.HealthMax, mp, d.ManaMax, health, mana)
			}
			if sheet := secondSheet[h.Template]; xp == 7320 && [4]int32{hp, mp, d.Speed, d.Sight} != sheet {
				t.Fatalf("%s skill %d: health %d mana %d speed %d sight %d, the claim's sheet %v", h.Template, skill, hp, mp, d.Speed, d.Sight, sheet)
			}
			if m.DefinitionRow == 0 || f.Table.Humans.EntryName(int(m.DefinitionRow)) != h.Template {
				t.Fatalf("%s skill %d: row %d", h.Template, skill, m.DefinitionRow)
			}
		}
	}
}

// secondHeroRecord is the hero's Human record of a second-game file.
func secondHeroRecord(t *testing.T, raw []byte) sav.DocumentRecordData {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	i, err := secondTownHeroRecord(doc)
	if err != nil {
		t.Fatal(err)
	}
	return doc.Objects[i-1]
}

// secondHeroRecordUnsourced are the hero record's fields no claim sources
// (DIV-4305): the record keys, and the position, mover and command bytes.
var secondHeroRecordUnsourced = []string{"Identity", "Reference", "Block12", "U154", "URaw184"}

// secondHeroRecordTail are raw blocks whose last word no claim sources: the
// originals leave memory there (f40f, f2b6 in game0002, 0 in game0000).
var secondHeroRecordTail = []string{"UA6", "U114"}

// secondHeroRecordDiffers names the fields of ours that differ from the
// original's, the unsourced ones and extra aside.
func secondHeroRecordDiffers(ours, orig sav.DocumentRecordData, extra []string) []string {
	skip := append(slices.Clone(secondHeroRecordUnsourced), extra...)
	var out []string
	for _, v := range orig.Values {
		at := slices.IndexFunc(ours.Values, func(w sav.DocumentValueData) bool { return w.Name == v.Name })
		if !slices.Contains(skip, v.Name) && (at < 0 || ours.Values[at].Value != v.Value) {
			out = append(out, v.Name)
		}
	}
	for _, v := range orig.Raw {
		if slices.Contains(skip, v.Name) {
			continue
		}
		at := slices.IndexFunc(ours.Raw, func(w sav.DocumentRawData) bool { return w.Name == v.Name })
		if at < 0 || len(ours.Raw[at].Bytes) != len(v.Bytes) {
			out = append(out, v.Name)
			continue
		}
		a, b := ours.Raw[at].Bytes, v.Bytes
		if slices.Contains(secondHeroRecordTail, v.Name) && len(a) >= 2 {
			a, b = a[:len(a)-2], b[:len(b)-2]
		}
		if string(a) != string(b) {
			out = append(out, v.Name)
		}
	}
	return out
}

// A new hero given the attributes and the skill of an original town file
// writes, in our town SAVE, the original's hero record: row, +0x1c,
// attributes, pools, skills, experience, item modifiers and defence. The
// Start_MM file differs also in its map word T08, U6C and the mana floor
// UA0 (75 there, 95% of 75 here), which no claim sources. A LOAD of
// game0000 and our SAVE write game0000's record again.
func TestReleaseSecondNewHeroRecordIsTheOriginals(t *testing.T) {
	cases := []struct {
		file  string
		hero  ui.ChargenResult
		extra []string
	}{
		{"game0000.sav", ui.ChargenResult{Name: "Hero", Choices: []int{0, 0, 1}, Stats: []int{41, 34, 24, 17}}, nil},
		{"game0002.sav", ui.ChargenResult{Name: "Hero", Choices: []int{0, 1, 3}, Stats: []int{19, 15, 43, 25}}, []string{"T08", "U6C", "UA0"}},
	}
	var orig []byte
	for i, c := range cases {
		raw, err := os.ReadFile(filepath.Join(secondCompanionCorpus, "rom2-gog-original", c.file))
		if err != nil {
			t.Skipf("no corpus file: %v", err)
		}
		if i == 0 {
			orig = raw
		}
		f, app := secondTownNew(t, false, false)
		if err := f.startGeneratedTown(c.hero); err != nil {
			t.Fatal(err)
		}
		ours := secondTownNamedSave(t, f, app, t.TempDir(), "New hero")
		if differ := secondHeroRecordDiffers(secondHeroRecord(t, ours), secondHeroRecord(t, raw), c.extra); len(differ) != 0 {
			t.Fatalf("the new hero's record differs from %s's in %v", c.file, differ)
		}
	}
	in := t.TempDir()
	if err := os.WriteFile(filepath.Join(in, "game0000.sav"), orig, 0o600); err != nil {
		t.Fatal(err)
	}
	lf, lapp := secondTownCold(t, in, "game0000.sav")
	resave := secondTownNamedSave(t, lf, lapp, in, "Loaded hero")
	if differ := secondHeroRecordDiffers(secondHeroRecord(t, resave), secondHeroRecord(t, orig), nil); len(differ) != 0 {
		t.Fatalf("game0000's hero after LOAD and SAVE differs in %v", differ)
	}
}

// secondHeroXP is the hero's per-skill experience and its sum in a file.
func secondHeroXP(t *testing.T, raw []byte) ([sav.CharacterSkillSlots]uint32, uint32) {
	t.Helper()
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	chars, err := sf.Party()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chars {
		if c.Hero {
			return c.SkillXP, c.Experience
		}
	}
	t.Fatal("the file holds no hero")
	return [sav.CharacterSkillSlots]uint32{}, 0
}

// A new hero's experience crosses a town SAVE and a cold LOAD; an original
// mission file whose experience runs ahead of the levels keeps it through
// LOAD and our SAVE.
func TestReleaseSecondHeroExperienceCrossesSaveAndLoad(t *testing.T) {
	f, app := secondTownNew(t, false, false)
	if err := f.startGeneratedTown(ui.ChargenResult{Name: "Hero", Choices: []int{0, 1, 0}, Stats: []int{19, 23, 30, 42}}); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	raw := secondTownNamedSave(t, f, app, out, "Experience")
	if xp, sum := secondHeroXP(t, raw); xp != [6]uint32{0, 5727, 0, 0, 0, 1593} || sum != 7320 {
		t.Fatalf("the town SAVE holds experience %v sum %d", xp, sum)
	}
	cold, _ := secondTownCold(t, out, secondSaveFile(t, out, "Experience"))
	d, _, _ := mapload.PartyDisplayWithTable(nativeCityHeroOf(cold.NextParty()), cold.Table)
	if d.SkillXP != [6]int32{0, 5727, 0, 0, 0, 1593} || d.Experience != 7320 {
		t.Fatalf("the cold LOAD's hero holds experience %v sum %d", d.SkillXP, d.Experience)
	}
	orig, err := os.ReadFile(filepath.Join(secondCompanionCorpus, "rom2-gog-original", "game0006.sav"))
	if err != nil {
		t.Skipf("no corpus file: %v", err)
	}
	wantXP, wantSum := secondHeroXP(t, orig)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0006.sav"), orig, 0o600); err != nil {
		t.Fatal(err)
	}
	lf := secondGameFront(t)
	lapp := lf.App("second hero experience")
	lapp.Layout(1024, 768)
	lf.ConfigureSaveSeams(lapp, SaveStore{Dir: dir}, OriginalStore{}, nil)
	_, list, _ := lf.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	groundAppLoad(t, lapp, list, "game0006.sav")
	s, label, err := lf.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	ours, err := lf.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	if xp, sum := secondHeroXP(t, ours); xp != wantXP || sum != wantSum {
		t.Fatalf("our SAVE of game0006 holds experience %v sum %d, want %v sum %d", xp, sum, wantXP, wantSum)
	}
}

// Every Human record of the original-written corpus files holds at +0x1c
// the rule secondRowExperience gives for its Humans row.
func TestReleaseSecondRowExperienceIsTheOriginals(t *testing.T) {
	f := secondGameFront(t)
	records := 0
	for _, dir := range []string{"rom2-gog-original", "rom2-gog-original-2"} {
		entries, err := os.ReadDir(filepath.Join(secondCompanionCorpus, dir))
		if err != nil {
			t.Skipf("no corpus: %v", err)
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".sav" {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(secondCompanionCorpus, dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatalf("%s/%s: %v", dir, e.Name(), err)
			}
			for i := range doc.Objects {
				r := &doc.Objects[i]
				if r.Class != "Human" {
					continue
				}
				row, _ := savedStructureValue(r, "T0C")
				got, _ := savedStructureValue(r, "T1C")
				want, ok := secondRowExperience(f.Table, row)
				if !ok || got != want {
					t.Fatalf("%s/%s record %d row %d: +0x1c %d, the rule %d (%v)", dir, e.Name(), i+1, row, got, want, ok)
				}
				records++
			}
		}
	}
	t.Logf("%d Human records", records)
}
