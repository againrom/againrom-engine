package main

// The regeneration census (0109, AC-10).
//
// AC-10 asks two questions a shipped install is the only witness for: how many
// of the definition table's parameterised Units rows carry a mana maximum above
// zero, and how many of a given map's placements resolve to one. Neither is
// answerable from -databin, which reports collection SIZES and which rows carry
// parameters, but no per-column value.
//
// It answers a third question the first two only make sharp: how many rows and
// placements can regenerate AT ALL. A pool's gain per qualifying tick is
// max * factor * 100 / period hundredths, so a period above a hundred times the
// factored maximum leaves the remainder empty every tick and that unit never
// moves. Whether that case is populated is the difference between a rule the
// corpus exercises and a rule it does not, and no assertion in the tree can
// establish it because no fixture is a shipped row.
//
// IT PRINTS COUNTS AND NOTHING ELSE — no name, no parameter, no per-row value.
// The evidence ships; the assets never do (golden rule 2).

import (
	"fmt"
	"io"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// NOTHING HERE NAMES pkg/sim. cmd/classdump -> pkg/sim is a DAG violation and
// internal/archtest fails on it; the entities below are ranged over with their
// type inferred from mapload's own return, which is all this census needs.

// regenTally is one population's answer to the same five questions, whether the
// population is the table's rows or a map's placements.
type regenTally struct {
	total int

	// mana is how many carry a mana maximum above zero — AC-10's own question.
	mana int

	// noHealthPeriod and noManaPeriod are how many carry a period at or below
	// zero on that arm, which the pass answers by regenerating nothing.
	noHealthPeriod int
	noManaPeriod   int

	// deadHealth and deadMana are how many have a usable period and a positive
	// maximum and STILL gain nothing, because the gain truncates to zero.
	deadHealth int
	deadMana   int

	// wounded is how many are born below their own health maximum. It is zero on
	// every shipped map, and that is the fact that makes regeneration invisible
	// on an undriven one: the pass returns at once on cur >= max.
	wounded int

	poolWithoutPeriod int
}

// gainless reports whether a pool with this maximum and this period gains
// nothing on a qualifying tick. It is the pass's own arithmetic, taken from
// pkg/sim's constants rather than from a literal here, so the two cannot come
// to disagree about what "never regenerates" means.
func gainless(max, period, factor int32) bool {
	if max <= 0 || period <= 0 {
		return false // no maximum or no period is a different answer, counted elsewhere
	}
	return int64(max)*int64(factor)*100/int64(period) == 0
}

func (t *regenTally) add(hp, maxHP, healthPeriod, mana, maxMana, manaPeriod int32) {
	t.total++
	if maxMana > 0 {
		t.mana++
	}
	if healthPeriod <= 0 {
		t.noHealthPeriod++
	}
	if manaPeriod <= 0 {
		t.noManaPeriod++
		if maxMana > 0 {
			t.poolWithoutPeriod++
		}
	}
	if gainless(maxHP, healthPeriod, 2) {
		t.deadHealth++
	}
	if gainless(maxMana, manaPeriod, 1) {
		t.deadMana++
	}
	if hp < maxHP {
		t.wounded++
	}
	_ = mana
}

func (t regenTally) write(w io.Writer, what string) {
	fmt.Fprintf(w, "%-22s %6d\n", what, t.total)
	fmt.Fprintf(w, "  mana maximum above zero      %6d\n", t.mana)
	fmt.Fprintf(w, "  health period at or below 0  %6d\n", t.noHealthPeriod)
	fmt.Fprintf(w, "  mana period at or below 0    %6d\n", t.noManaPeriod)
	fmt.Fprintf(w, "  health gain truncates to 0   %6d\n", t.deadHealth)
	fmt.Fprintf(w, "  mana gain truncates to 0     %6d\n", t.deadMana)
	fmt.Fprintf(w, "  born below its health max    %6d\n", t.wounded)
	fmt.Fprintf(w, "  a mana pool and NO period    %6d\n", t.poolWithoutPeriod)
}

// runRegen is the verb: the table's own rows, then one map's placements —
// resolved the way the loader resolves them, through an assembled world, so the
// figure is about what a player's mission holds and not about what a search
// would have reached.
func runRegen(root, only string, w io.Writer) error {
	c, err := openCampaign(root)
	if err != nil {
		return err
	}

	// The rows. A row with no parameter array is a name and nothing else; the
	// parameterised ones are the population AC-10 names.
	var rows regenTally
	u := c.table.Units
	for i := 0; i < u.Len(); i++ {
		p := u.EntryParams(i)
		if len(p) == 0 {
			continue
		}
		d, err := data.NewUnitDef(u.EntryName(i), p)
		if err != nil {
			return fmt.Errorf("units row %d: %w", i, err)
		}
		rows.add(d.Health, d.HealthMax, d.HealthRegenPeriod, d.Mana, d.ManaMax, d.ManaRegenPeriod)
	}
	rows.write(w, "parameterised rows")

	if only == "" {
		return nil
	}

	srcs, err := c.sources(root)
	if err != nil {
		return err
	}
	for _, s := range srcs {
		if s.name != only {
			continue
		}
		b, err := s.read()
		if err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
		m, err := alm.Open(b)
		if err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
		world, err := mapload.FromALMWith(m, c.table, mapload.DifficultyNormal)
		if err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
		// widest is the largest health gain one qualifying tick adds to any
		// entity of this world, in hundredths — the figure that says whether
		// the rule can move this mission at all, as against how often it fires.
		var placed regenTally
		var widest int64
		for _, e := range world.Entities() {
			placed.add(e.HP, e.MaxHP, e.HealthRegenPeriod, e.Mana, e.MaxMana, e.ManaRegenPeriod)
			if e.MaxHP <= 0 || e.HealthRegenPeriod <= 0 {
				continue
			}
			if g := int64(e.MaxHP) * 2 * 100 / int64(e.HealthRegenPeriod); g > widest {
				widest = g
			}
		}
		fmt.Fprintln(w)
		placed.write(w, s.name)
		fmt.Fprintf(w, "  one qualifying tick adds     %6d hundredths to the widest health pool here\n", widest)
		return nil
	}
	return fmt.Errorf("%s: no such map under %s", only, root)
}
