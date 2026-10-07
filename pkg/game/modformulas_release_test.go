package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

const (
	blessSpellID = 23

	formulaHealDamage  = 10
	formulaBlessMagnit = 77
	formulaBlessFactor = 7000
	formulaArrowRange  = 4
	formulaPower       = 150
	formulaBlessPower  = 200
)

type formulaRows struct {
	heal, arrow, bless data.Spell
	healLow, healHigh  int32
}

func installedFormulaRows(t *testing.T) formulaRows {
	t.Helper()
	f := releaseFront(t)
	rows := releaseSpellRows(t, f)
	r := formulaRows{heal: rows[healSpellID-1], arrow: rows[fireArrowID-1], bless: rows[blessSpellID-1]}
	casterID, _ := formulasMission(t, f)
	caster, _ := f.live.entity(casterID)
	rule, _ := f.live.world.Spell(healSpellID)
	if power := sim.SpellCharacteristicsFor(f.live.world.Rules(), caster, rule).Power; power != 0 {
		t.Fatalf("fixture: the unmodded caster has power %d, want 0", power)
	}
	r.healLow, r.healHigh = r.heal.DamageMin*30/30, r.heal.DamageMax*30/30
	return r
}

func formulaTableText(vals []int32) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprint(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func powerTable(base, at, value int32) []int32 {
	out := make([]int32, 256)
	for i := range out {
		out[i] = base
	}
	out[at] = value
	return out
}

func formulaMod(healAmount int32) func(name func(int) string) string {
	return func(name func(int) string) string {
		return fmt.Sprintf("[[spell]]\ntarget = %q\npower = [%d]\ndamage = { min = %d, max = %d }\ndamage_factor = %s\n\n",
			name(healSpellID), formulaPower, formulaHealDamage, formulaHealDamage,
			formulaTableText(powerTable(30, formulaPower, 3*healAmount))) +
			fmt.Sprintf("[[spell]]\ntarget = %q\npower = [%d]\nduration_factor = %s\nmagnitude = [%d]\n\n",
				name(blessSpellID), formulaBlessPower,
				formulaTableText(powerTable(1000, formulaBlessPower, formulaBlessFactor)), formulaBlessMagnit) +
			fmt.Sprintf("[[spell]]\ntarget = %q\nrange_bonus = [%d]\n", name(fireArrowID), formulaArrowRange)
	}
}

func formulasMission(t *testing.T, f *FrontEnd) (casterID, allyID sim.EntityID) {
	t.Helper()
	casterID, allyID = radiusMission(t, f)
	mw := f.live
	caster, _ := mw.entity(casterID)
	book, known := caster.Book, caster.KnownSpells
	for _, id := range []uint32{fireArrowID, healSpellID, blessSpellID} {
		rule, ok := mw.world.Spell(id)
		if !ok {
			t.Fatalf("no spell row %d", id)
		}
		slot := sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
		if rule.Defensive {
			slot.Defensive = 1
		}
		book.Slots[id-1] = slot
		known |= 1 << id
	}
	caster.Book, caster.KnownSpells = book, known
	sim.RefreshBook(mw.world.Rules(), &caster, mw.world.Spells())
	if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: casterID, KnownSpells: known, Book: caster.Book}}); err != nil {
		t.Fatal(err)
	}
	if !mw.world.SetSkillLevels(casterID, caster.Skill) {
		t.Fatal("the caster's book roots cannot be refreshed")
	}
	return casterID, allyID
}

func popupOf(mw *mapWorld, casterID sim.EntityID) map[uint32][]string {
	caster, _ := mw.entity(casterID)
	out := map[uint32][]string{}
	for _, s := range spellbookOf(mw.world.Rules(), caster, mw.world.Spells(), mw.spellNames, mw.view.Words(), nil) {
		out[s.ID] = s.Info
	}
	return out
}

func linesOf(lines []string) string { return strings.Join(lines, " | ") }

func savedSpellRanges(t *testing.T, raw []byte, id uint32) []uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var out []uint32
	for _, o := range doc.Objects {
		if o.Class == "Spell" && savedRecordValueForTest(t, o, "S08") == id {
			out = append(out, savedRecordValueForTest(t, o, "S09"))
		}
	}
	return out
}

func healAlly(t *testing.T, mw *mapWorld, casterID, allyID sim.EntityID, wound int32) int32 {
	t.Helper()
	if err := mw.world.HeadlessDamage(allyID, wound); err != nil {
		t.Fatal(err)
	}
	before := hpOf(mw, allyID)
	mw.castAt(uint32(casterID), uint32(allyID), healSpellID)
	for tick := 0; hpOf(mw, allyID) == before; tick++ {
		if tick > 300 {
			t.Fatalf("the heal never reached the ally (health %d)", before)
		}
		mw.tick()
	}
	return hpOf(mw, allyID) - before
}

func blessOf(mw *mapWorld, allyID sim.EntityID) (sim.ActiveEffect, bool) {
	for _, e := range mw.world.ActiveEffects() {
		if e.Spell == blessSpellID && e.Target == allyID {
			return e, true
		}
	}
	return sim.ActiveEffect{}, false
}

func castBless(t *testing.T, mw *mapWorld, casterID, allyID sim.EntityID) sim.ActiveEffect {
	t.Helper()
	mw.castAt(uint32(casterID), uint32(allyID), blessSpellID)
	for tick := 0; tick < 300; tick++ {
		mw.tick()
		if e, ok := blessOf(mw, allyID); ok {
			return e
		}
	}
	t.Fatal("Bless never attached to the ally")
	return sim.ActiveEffect{}
}

func arrowReach(t *testing.T, mw *mapWorld, casterID sim.EntityID) (stated, booked int32) {
	t.Helper()
	caster, _ := mw.entity(casterID)
	rule, _ := mw.world.Spell(fireArrowID)
	return int32(sim.SpellCharacteristicsFor(mw.world.Rules(), caster, rule).Range), int32(caster.Book.Slots[fireArrowID-1].Range)
}

func TestReleaseModFormulaTablesReachTheCastTheBookAndTheSave(t *testing.T) {
	r := installedFormulaRows(t)
	healAmount := r.healHigh + 7
	blessTicks := r.bless.SpellDuration * 16 * formulaBlessFactor / 1000
	reach := r.arrow.MaxRange + formulaArrowRange
	f := targetsModFront(t, formulaMod(healAmount))
	casterID, allyID := formulasMission(t, f)
	mw := f.live
	if !mw.world.Rules().HasSpellFormulas() {
		t.Fatal("the mission world runs without the tables")
	}
	checkBook := func(where string, m *mapWorld) {
		t.Helper()
		book := popupOf(m, casterID)
		w := m.view.Words()
		for _, c := range []struct {
			spell uint32
			line  string
		}{
			{healSpellID, w.Hover[spellLabelDamage] + ": " + fmt.Sprint(healAmount)},
			{blessSpellID, w.Hover[spellLabelDuration] + ": " + fmt.Sprintf("%5.1f", float64(blessTicks)*0.0625)},
			{blessSpellID, w.Hover[185] + ": " + fmt.Sprintf("+%d%%", formulaBlessMagnit)},
			{fireArrowID, w.Hover[spellLabelRange] + ": " + fmt.Sprint(reach)},
		} {
			if !slices.Contains(book[c.spell], c.line) {
				t.Errorf("%s: the popup of spell %d lacks %q: %s", where, c.spell, c.line, linesOf(book[c.spell]))
			}
		}
	}
	checkBook("mission", mw)

	if gain := healAlly(t, mw, casterID, allyID, healAmount+20); gain != healAmount {
		t.Errorf("the Heal restored %d health, want the table's exact %d (installed pair %d-%d)", gain, healAmount, r.healLow, r.healHigh)
	}
	live := castBless(t, mw, casterID, allyID)
	if live.Magnitude != formulaBlessMagnit || int32(live.Remaining) != blessTicks {
		t.Errorf("Bless attached %+v, want magnitude %d for %d ticks", live, formulaBlessMagnit, blessTicks)
	}
	if stated, booked := arrowReach(t, mw, casterID); stated != reach || booked != reach {
		t.Errorf("the Fire Arrow reach is %d (book %d), want the row's %d plus the table's %d", stated, booked, r.arrow.MaxRange, formulaArrowRange)
	}

	raw := exportSave(t, f)
	if got := savedSpellRanges(t, raw, fireArrowID); !slices.Contains(got, uint32(reach)) {
		t.Errorf("the SAV's Fire Arrow records %v lack the tabled reach %d", got, reach)
	}
	saved, _ := blessOf(mw, allyID)

	g := targetsModFront(t, formulaMod(healAmount))
	open, town, err := g.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("cold load: town %v err %v", town, err)
	}
	if err := g.App("cold load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	late := g.live
	if !late.world.Rules().HasSpellFormulas() {
		t.Fatal("the restored world runs without the tables")
	}
	checkBook("cold load", late)
	if got, ok := blessOf(late, allyID); !ok || got != saved {
		t.Errorf("the restored Bless is %+v %v, saved %+v", got, ok, saved)
	}
	if gain := healAlly(t, late, casterID, allyID, healAmount+20); gain != healAmount {
		t.Errorf("after the cold LOAD the Heal restored %d health, want %d", gain, healAmount)
	}
	if stated, booked := arrowReach(t, late, casterID); stated != reach || booked != reach {
		t.Errorf("after the cold LOAD the Fire Arrow reach is %d (book %d), want %d", stated, booked, reach)
	}

	plain := releaseFront(t)
	if _, _, err := plain.RestoreOriginal(raw); err == nil {
		t.Error("a game without the mod loaded the SAV written under it")
	}
}

func TestReleaseWithoutFormulaTablesTheCastKeepsTheOriginalNumbers(t *testing.T) {
	r := installedFormulaRows(t)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	casterID, allyID := formulasMission(t, f)
	mw := f.live
	if mw.world.Rules().HasSpellFormulas() {
		t.Fatal("the installed game carries formula tables")
	}
	book := popupOf(mw, casterID)
	w := mw.view.Words()
	if line := w.Hover[spellLabelDamage] + ": " + spellPair(int64(r.heal.DamageMin*130/30), int64(r.heal.DamageMax*130/30), "%d", "-"); !slices.Contains(book[healSpellID], line) {
		t.Errorf("the Heal popup lacks %q: %s", line, linesOf(book[healSpellID]))
	}
	if line := w.Hover[spellLabelRange] + ": " + fmt.Sprint(r.arrow.MaxRange); !slices.Contains(book[fireArrowID], line) {
		t.Errorf("the Fire Arrow popup lacks %q: %s", line, linesOf(book[fireArrowID]))
	}
	if gain := healAlly(t, mw, casterID, allyID, r.healHigh+20); gain < r.healLow || gain > r.healHigh {
		t.Errorf("the installed Heal restored %d health, want %d to %d", gain, r.healLow, r.healHigh)
	}
	bless := castBless(t, mw, casterID, allyID)
	if want := r.bless.SpellDuration * 16; bless.Magnitude != 20 || int32(bless.Remaining) != want {
		t.Errorf("the installed Bless attached %+v, want magnitude 20 for %d ticks at power 0", bless, want)
	}
	if stated, booked := arrowReach(t, mw, casterID); stated != r.arrow.MaxRange || booked != r.arrow.MaxRange {
		t.Errorf("the installed Fire Arrow reach is %d (book %d), want %d", stated, booked, r.arrow.MaxRange)
	}
	if got := savedSpellRanges(t, exportSave(t, f), fireArrowID); slices.Contains(got, uint32(r.arrow.MaxRange+formulaArrowRange)) {
		t.Errorf("the installed SAV holds the tabled reach: %v", got)
	}
}
