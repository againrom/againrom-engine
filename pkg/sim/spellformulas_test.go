package sim

import (
	"slices"
	"testing"
)

func formulaRules(t *testing.T, set SpellFormulaSet) Rules {
	t.Helper()
	r, err := Rules{}.WithSpellFormulas(set)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func constantTable(n int, v int32) []int32 {
	out := make([]int32, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func TestFormulasWithoutTablesAreTheOriginalArithmetic(t *testing.T) {
	t.Parallel()

	r := Rules{}
	rule := SpellRule{ID: 24, DamageMin: 4, DamageMax: 9, MaxRange: 7, SpellDuration: 5}
	for _, power := range []int32{0, 29, 30, 100, 101, 255} {
		wb, ws := spellDamage(4, 9, power)
		if gb, gs := spellDamageUnder(r, rule, power); gb != wb || gs != ws {
			t.Errorf("damage at %d: %d %d, want %d %d", power, gb, gs, wb, ws)
		}
		if got, want := spellRangeUnder(r, rule, power), spellRange(rule, power); got != want {
			t.Errorf("range at %d: %d, want %d", power, got, want)
		}
		if got, want := spellRecordRangeUnder(r, rule, power), spellRecordRange(rule, power); got != want {
			t.Errorf("record range at %d: %d, want %d", power, got, want)
		}
		if got, want := spellLastingTicksUnder(r, rule, power), spellLastingTicks(rule, power); got != want {
			t.Errorf("duration at %d: %d, want %d", power, got, want)
		}
	}
	for _, sum := range []int32{-5, 0, 29, 30, 130, 300, 600} {
		if got, want := spellPowerUnder(r, rule, sum, 0), spellPower(sum, 0); got != want {
			t.Errorf("power at %d: %d, want %d", sum, got, want)
		}
		if got, want := spellRecordPowerUnder(r, rule, sum, 0), spellRecordPower(sum, 0); got != want {
			t.Errorf("record power at %d: %d, want %d", sum, got, want)
		}
	}
}

func TestPowerTableIsReadAtSkillPlusMindPerSpell(t *testing.T) {
	t.Parallel()

	var set SpellFormulaSet
	set.Global[FormulaPower] = []int32{0, 10, 250}
	set.Spell[FormulaPower][3] = []int32{7}
	r := formulaRules(t, set)
	fire := SpellRule{ID: 1}
	for _, c := range []struct {
		level, mind int32
		want        int32
	}{{-9, 0, 0}, {0, 0, 0}, {0, 1, 10}, {1, 1, 250}, {200, 300, 250}} {
		if got := spellPowerUnder(r, fire, c.level, c.mind); got != c.want {
			t.Errorf("sum %d: power %d, want %d", c.level+c.mind, got, c.want)
		}
		if got := spellRecordPowerUnder(r, fire, c.level, c.mind); got != c.want {
			t.Errorf("sum %d: record power %d, want %d", c.level+c.mind, got, c.want)
		}
	}
	if got := spellPowerUnder(r, SpellRule{ID: 3}, 200, 50); got != 7 {
		t.Errorf("the row's own table gave %d, want 7", got)
	}
}

func TestDamageFactorScalesThePairInThirtiethsAboveHundred(t *testing.T) {
	t.Parallel()

	table := constantTable(256, 30)
	table[200] = 150
	var set SpellFormulaSet
	set.Spell[FormulaDamage][1] = table
	r := formulaRules(t, set)
	rule := SpellRule{ID: 1, DamageMin: 4, DamageMax: 9}
	if base, spread := spellDamageUnder(r, rule, 0); base != 4 || spread != 5 {
		t.Errorf("factor 30: %d %d, want 4 5", base, spread)
	}
	if base, spread := spellDamageUnder(r, rule, 200); base != 20 || spread != 25 {
		t.Errorf("power 200 at factor 150: %d %d, want 20 25", base, spread)
	}
	if base, spread := spellDamageUnder(r, rule, 1000); base != 4 || spread != 5 {
		t.Errorf("past the table: %d %d, want the last entry's 4 5", base, spread)
	}
	if base, spread := spellDamageUnder(r, SpellRule{ID: 2, DamageMin: 4, DamageMax: 9}, 200); base != 4*230/30 || spread != 9*230/30-4*230/30 {
		t.Errorf("another row changed: %d %d", base, spread)
	}
}

func TestRangeBonusReplacesPowerOverThirtyAndSaturates(t *testing.T) {
	t.Parallel()

	var set SpellFormulaSet
	set.Global[FormulaRange] = []int32{1, 2, 255}
	set.Spell[FormulaRange][26] = []int32{0}
	r := formulaRules(t, set)
	rule := SpellRule{ID: 1, MaxRange: 10}
	for _, c := range []struct {
		power int32
		want  int64
	}{{-3, 11}, {0, 11}, {1, 12}, {2, 255}, {200, 255}} {
		if got := spellRangeUnder(r, rule, c.power); got != c.want {
			t.Errorf("range at %d: %d, want %d", c.power, got, c.want)
		}
		if got := spellRecordRangeUnder(r, rule, c.power); got != c.want {
			t.Errorf("record range at %d: %d, want %d", c.power, got, c.want)
		}
	}
	if got := spellRangeUnder(r, SpellRule{ID: 1}, 1); got != 0 {
		t.Errorf("a zero range column took a bonus: %d", got)
	}
	if got := spellRangeUnder(r, SpellRule{ID: 1, MaxRange: 9, bookInstance: true}, 1); got != 9 {
		t.Errorf("a book instance range changed: %d", got)
	}
	if got := spellRangeUnder(r, SpellRule{ID: 26, MaxRange: 12}, 90); got != 12 {
		t.Errorf("Teleport under its own table: %d, want 12", got)
	}
	if got := spellRangeUnder(Rules{}, SpellRule{ID: 26, MaxRange: 12}, 90); got != 42 {
		t.Errorf("Teleport original: %d, want 42", got)
	}
}

func TestDurationFactorIsThousandthsOfSixteenthsAboveHundred(t *testing.T) {
	t.Parallel()

	table := constantTable(256, 1000)
	table[200] = 2500
	var set SpellFormulaSet
	set.Global[FormulaDuration] = table
	r := formulaRules(t, set)
	haste := SpellRule{ID: 24, SpellDuration: 5}
	if got := spellLastingTicksUnder(r, haste, 0); got != 80 {
		t.Errorf("factor 1000: %d, want 80", got)
	}
	if got := spellLastingTicksUnder(r, haste, 200); got != 200 {
		t.Errorf("power 200 at factor 2500: %d, want 200", got)
	}
	if got := spellLastingTicksUnder(r, SpellRule{ID: 15}, 200); got != 120 {
		t.Errorf("Invisibility keeps its literal base of 3: %d, want 120", got)
	}
	stone := SpellRule{ID: 20, SpellDuration: 2}
	if got := spellLastingTicksUnder(r, stone, 250); got != 32 {
		t.Errorf("Stone Curse reads the table whole, not in segments: %d, want 32", got)
	}
	if got := spellLastingTicksUnder(r, SpellRule{ID: 24}, 200); got != 0 {
		t.Errorf("a zero base gave %d", got)
	}
	set.Global[FormulaDuration] = constantTable(1, 1000000)
	if got := spellLastingTicksUnder(formulaRules(t, set), SpellRule{ID: 24, SpellDuration: 4095}, 0); got != durationTickCeiling {
		t.Errorf("saturation: %d", got)
	}
	if got := spellPointDurationUnder(r, SpellRule{ID: 7, EffectDuration: 33}, 0); got != 33 {
		t.Errorf("a row off the power duration arm changed: %d", got)
	}
	if got := spellRecordDurationUnder(r, haste, 200); got != 200 {
		t.Errorf("record duration: %d", got)
	}
}

func TestMagnitudeTableReplacesTheArmOfOneSpellOnly(t *testing.T) {
	t.Parallel()

	var set SpellFormulaSet
	set.Spell[FormulaMagnitude][24] = []int32{2, -7, 90}
	set.Spell[FormulaMagnitude][23] = []int32{44}
	set.Spell[FormulaMagnitude][20] = []int32{99}
	w := hlWorld(t, 1, Relations{}, nil, spEnt(1, 1, 1), spEnt(2, 2, 1))
	w.SetRules(formulaRules(t, set))
	haste := SpellRule{ID: 24, SpellDuration: 1, EffectKind: EffectSpeed, EffectMode: EffectDuration}
	for _, c := range []struct {
		power int32
		want  int32
	}{{0, 2}, {1, -7}, {2, 90}, {250, 90}} {
		if _, mag, _, _ := w.pointEffect(1, haste, c.power); mag != c.want {
			t.Errorf("haste at power %d: magnitude %d, want %d", c.power, mag, c.want)
		}
	}
	if _, mag, _, _ := w.pointEffect(1, SpellRule{ID: 23, SpellDuration: 1}, 100); mag != 44 {
		t.Errorf("bless: %d, want 44", mag)
	}
	if _, mag, _, _ := w.pointEffect(1, SpellRule{ID: 27, SpellDuration: 1}, 100); mag != 100*4/5+20 {
		t.Errorf("curse took bless's table: %d", mag)
	}
	if _, mag, _, _ := w.pointEffect(1, SpellRule{ID: 20, SpellDuration: 1, EffectMagnitude: 5}, 100); mag != 5 {
		t.Errorf("a row off the magnitude arms changed: %d", mag)
	}
	if _, mag, _, _ := w.pointEffect(1, SpellRule{ID: 5, SpellDuration: 1}, 100); mag != 50 {
		t.Errorf("an untabled protection changed: %d", mag)
	}
	for _, id := range []uint16{5, 7, 8, 10, 12, 16, 17, 18, 22, 23, 24, 27, 28} {
		if !MagnitudeArm(id) {
			t.Errorf("spell %d is not a magnitude arm", id)
		}
	}
	if MagnitudeArm(20) || MagnitudeArm(1) || DurationArm(7) || !DurationArm(15) {
		t.Error("arm lists")
	}
}

func TestACastReadsEveryTableThroughTheWorld(t *testing.T) {
	t.Parallel()

	spell := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 5, DamageMin: 6, DamageMax: 6, TargetsUnit: true, Damaging: true}
	cast := func(r Rules, dist int32) *World {
		caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
		w := spWorld(t, 42, []SpellRule{spell}, caster, spEnt(2, dist, 0))
		w.SetRules(r)
		spRunCast(w, spCast(1, 2, 1))
		return w
	}
	if hp := spAt(t, cast(Rules{}, 3), 2).HP; hp != 100-6*(30+30)/30 {
		t.Fatalf("unmodded damage left %d", hp)
	}
	var set SpellFormulaSet
	set.Spell[FormulaPower][1] = constantTable(1, 200)
	damage := constantTable(256, 30)
	damage[200] = 150
	set.Spell[FormulaDamage][1] = damage
	reach := constantTable(256, 0)
	reach[200] = 4
	set.Spell[FormulaRange][1] = reach
	r := formulaRules(t, set)
	if hp := spAt(t, cast(r, 3), 2).HP; hp != 100-30 {
		t.Errorf("a power of 200 at factor 150 left %d health, want 70", hp)
	}
	if got := spAt(t, cast(r, 9), 2).HP; got != 100-30 {
		t.Errorf("a range bonus of 4 over 5 did not reach 9 cells: %d health", got)
	}
	if got := spAt(t, cast(r, 10), 2).HP; got != 100 {
		t.Errorf("a cast 10 cells away over a reach of 9 landed: %d health", got)
	}
	if got := spAt(t, cast(Rules{}, 9), 2).HP; got != 100 {
		t.Errorf("without the table 9 cells is out of range: %d health", got)
	}
}

func TestBookRangeCacheFollowsTheRangeTable(t *testing.T) {
	t.Parallel()

	rule := SpellRule{ID: 1, MaxRange: 10, School: 1}
	e := spMage(1, 0, 0, 60, 10, 10, 1<<1)
	e.Book = Spellbook{State: BookPresent}
	e.Book.Slots[0] = BookSpell{Range: 1}
	RefreshBook(Rules{}, &e, []SpellRule{rule})
	if got := e.Book.Slots[0].Range; got != 11 {
		t.Fatalf("original reach %d, want 10 + 30/30", got)
	}
	var set SpellFormulaSet
	set.Global[FormulaRange] = constantTable(256, 33)
	r := formulaRules(t, set)
	RefreshBook(r, &e, []SpellRule{rule})
	if got := e.Book.Slots[0].Range; got != 43 {
		t.Errorf("tabled reach %d, want 43", got)
	}
	w := hlWorld(t, 1, Relations{}, []SpellRule{rule}, e)
	w.SetRules(r)
	saved := SourceItemSpell{Present: true, ID: 1, Range: 43}
	if !w.bookRootValueMatches(saved, saved) {
		t.Error("equal slots differ")
	}
	derived := saved
	saved.Range = 44
	if w.bookRootValueMatches(saved, derived) {
		t.Error("a saved reach above every table reach matched")
	}
	set.Global[FormulaRange] = []int32{40, 10, 25}
	w.SetRules(formulaRules(t, set))
	saved.Range, derived.Range = 35, 45
	if !w.bookRootValueMatches(saved, derived) {
		t.Error("a saved reach inside the table's span did not match")
	}
	saved.Range = 51
	if w.bookRootValueMatches(saved, derived) {
		t.Error("a saved reach above the table's span matched")
	}
}

func TestSpellCharacteristicsCarryTheTablesForThePopup(t *testing.T) {
	t.Parallel()

	var set SpellFormulaSet
	set.Spell[FormulaDamage][24] = constantTable(256, 60)
	set.Spell[FormulaMagnitude][24] = constantTable(256, 11)
	set.Spell[FormulaRange][24] = constantTable(256, 3)
	set.Spell[FormulaDuration][24] = constantTable(256, 2000)
	set.Spell[FormulaPower][24] = []int32{0, 0, 120}
	r := formulaRules(t, set)
	rule := SpellRule{ID: 24, School: 1, DamageMin: 3, DamageMax: 6, MaxRange: 4, SpellDuration: 5}
	e := spEnt(1, 0, 0)
	e.Skill[1], e.Mind = 40, 30
	c := SpellCharacteristicsFor(r, e, rule)
	if c.Power != 120 || c.RecordPower != 120 || !c.HasDamageFactor || c.DamageFactor != 60 || !c.HasMagnitude || c.Magnitude != 11 {
		t.Errorf("%+v", c)
	}
	if c.Range != 7 || c.Duration != 160 {
		t.Errorf("range %d duration %d, want 7 and 160", c.Range, c.Duration)
	}
	plain := SpellCharacteristicsFor(Rules{}, e, rule)
	if plain.HasDamageFactor || plain.HasMagnitude || plain.Power != 40 {
		t.Errorf("unmodded %+v", plain)
	}
	rule.Damaging, rule.TargetsUnit = true, true
	got, ok := weaponSpellCharacteristics(r, rule, 10)
	if !ok || got.DamageMin != 6 || got.DamageMax != 12 {
		t.Errorf("tooltip %+v %v, want damage 6-12", got, ok)
	}
}

func TestFormulaTablesAreNotInTheWorldByteForm(t *testing.T) {
	t.Parallel()

	rule := SpellRule{ID: 1, MaxRange: 10, School: 1, DamageMin: 1, DamageMax: 2, Damaging: true, TargetsUnit: true}
	plain := hlWorld(t, 3, Relations{}, []SpellRule{rule}, spEnt(1, 1, 1))
	var set SpellFormulaSet
	set.Global[FormulaDamage] = []int32{99}
	tabled := hlWorld(t, 3, Relations{}, []SpellRule{rule}, spEnt(1, 1, 1))
	tabled.SetRules(formulaRules(t, set))
	if !slices.Equal(hlBytes(t, plain), hlBytes(t, tabled)) {
		t.Error("a formula table changed the world bytes")
	}
}
