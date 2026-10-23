package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// invisibilityWitnessSAV loads the tester's mission-140 SAV named by
// AGAINROM_INVIZ_SAV on the install AGAINROM_ASSETS names.
func invisibilityWitnessSAV(t *testing.T) *FrontEnd {
	t.Helper()
	path := os.Getenv("AGAINROM_INVIZ_SAV")
	if path == "" {
		t.Skip("set AGAINROM_INVIZ_SAV to the tester's Invisibility mission SAV")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "dd60db8509ad8a693cfddcd138c6499d613839bdebd6f1a4ea1acaa279288b7e" {
		t.Fatal("the tester's Invisibility SAV SHA differs")
	}
	f := releaseFront(t)
	openOriginalSAVApp(t, f, raw, "inviz.sav")
	if f.live == nil || f.live.world == nil {
		t.Fatal("the SAV loaded no mission")
	}
	return f
}

const (
	invizMage, invizWarrior            = sim.EntityID(171), sim.EntityID(172)
	invizSpell, invizBless, invizHaste = uint16(15), uint16(23), uint16(24)
)

type invizCast struct {
	tick           uint64
	caster, target sim.EntityID
	spell          uint16
}

// invizTrace logs a line for each tick whose observed state differs from the
// previous tick's, apart from the falling counters: the warrior's
// Invisibility, Bless and Haste, the mage's cell, distance, mana, wind-up and
// order progress, and the mage's admission verdict for each spell at the
// warrior. It records every applied cast and whether the mage left its cell.
type invizTrace struct {
	t       *testing.T
	name    string
	last    string
	casts   []invizCast
	started bool
	x, y    int32
	moved   bool
}

func invizRemaining(w *sim.World, target sim.EntityID, spell uint16) (uint16, bool) {
	for _, a := range w.ActiveEffects() {
		if a.Target == target && a.Spell == spell {
			return a.Remaining, true
		}
	}
	return 0, false
}

func invizDistance(w *sim.World) int32 {
	m, _ := w.Entity(invizMage)
	v, _ := w.Entity(invizWarrior)
	dx, dy := m.X-v.X, m.Y-v.Y
	return max(dx, -dx, dy, -dy)
}

func (tr *invizTrace) observe(f *FrontEnd, casts []sim.CastEvent) {
	w := f.live.world
	for _, c := range casts {
		tr.casts = append(tr.casts, invizCast{w.Tick(), c.Caster, c.Target, c.Spell})
		tr.t.Logf("%s tick %d CAST caster %d target %d spell %d", tr.name, w.Tick(), c.Caster, c.Target, c.Spell)
	}
	m, _ := w.Entity(invizMage)
	if !tr.started {
		tr.started, tr.x, tr.y = true, m.X, m.Y
	}
	tr.moved = tr.moved || m.X != tr.x || m.Y != tr.y
	windup := "-"
	if spell, left, ok := w.CastingSpell(invizMage); ok {
		windup = fmt.Sprintf("%d/%d", spell, left)
	}
	var counters, key strings.Builder
	for _, s := range []uint16{invizSpell, invizBless, invizHaste} {
		r, ok := invizRemaining(w, invizWarrior, s)
		fmt.Fprintf(&counters, "spell %d remaining %d (%v) ", s, r, ok)
		fmt.Fprintf(&key, "%v ", ok)
	}
	fmt.Fprintf(&key, "| mage %d,%d dist %d mana %d windup %s progress %d | refuse inv %q bless %q haste %q",
		m.X, m.Y, invizDistance(w), m.Mana, windup, w.ActorOrderProgress(invizMage),
		w.BookSpellRefusal(invizMage, invizWarrior, uint32(invizSpell)), w.BookSpellRefusal(invizMage, invizWarrior, uint32(invizBless)), w.BookSpellRefusal(invizMage, invizWarrior, uint32(invizHaste)))
	if key.String() != tr.last {
		tr.t.Logf("%s tick %d %s%s", tr.name, w.Tick(), counters.String(), key.String())
		tr.last = key.String()
	}
}

func (tr *invizTrace) run(f *FrontEnd, ticks int) {
	for i := 0; i < ticks; i++ {
		tr.observe(f, f.LiveAdvanceCasts(1))
	}
}

func (tr *invizTrace) castsAt(target sim.EntityID, spell uint16, from uint64) int {
	n := 0
	for _, c := range tr.casts {
		if c.target == target && c.spell == spell && c.tick >= from {
			n++
		}
	}
	return n
}

// TestReleaseAPlayersCastsSeeItsOwnInvisibleUnit proves the owner direction
// DIV-3996 on the tester's SAV; it does not state the original's rule. The
// mage (171) holds Average with Own; the warrior (172) stands two cells away
// and nobody has a SeeInvisible radius. Control: the support cast renews the
// warrior's Bless and Haste. Case a: it still renews them once the warrior is
// invisible. Case b: a second book Invisibility at the invisible warrior is
// cast from where the mage stands, with no walk, and resets the remaining
// time. Case c: a second Invisibility on the mage itself sets the remaining
// time back to one fresh duration (MAGIC-400).
func TestReleaseAPlayersCastsSeeItsOwnInvisibleUnit(t *testing.T) {
	const loadTick, landTick, fresh = 6646786, 6646804, 6011
	const notSeen = "target not currently visible"

	f := invisibilityWitnessSAV(t)
	tr := &invizTrace{t: t, name: "control"}
	tr.observe(f, nil)
	tr.run(f, 400)
	if tr.castsAt(invizWarrior, invizBless, 0) != 1 || tr.castsAt(invizWarrior, invizHaste, 0) != 1 {
		t.Fatalf("control: Bless and Haste on the visible warrior were not renewed once each: %v", tr.casts)
	}

	f = invisibilityWitnessSAV(t)
	tr = &invizTrace{t: t, name: "a"}
	tr.observe(f, nil)
	f.live.pending = append(f.live.pending, sim.Cast(invizMage, invizWarrior, sim.SpellID(invizSpell)))
	tr.run(f, 18)
	if r, ok := invizRemaining(f.live.world, invizWarrior, invizSpell); !ok || r != fresh || f.live.world.Tick() != landTick {
		t.Fatalf("a: Invisibility on the warrior at tick %d: %d %v", f.live.world.Tick(), r, ok)
	}
	tr.run(f, 382)
	w := f.live.world
	_, bless := invizRemaining(w, invizWarrior, invizBless)
	_, haste := invizRemaining(w, invizWarrior, invizHaste)
	if !bless || !haste || tr.castsAt(invizWarrior, invizBless, landTick) != 1 || tr.castsAt(invizWarrior, invizHaste, landTick) != 1 ||
		w.BookSpellRefusal(invizMage, invizWarrior, uint32(invizBless)) == notSeen || tr.castsAt(174, invizHaste, landTick) != 1 {
		t.Fatalf("a: the support cast skipped the invisible warrior: Bless %v Haste %v, casts %v", bless, haste, tr.casts)
	}

	f = invisibilityWitnessSAV(t)
	tr = &invizTrace{t: t, name: "b"}
	tr.observe(f, nil)
	f.live.pending = append(f.live.pending, sim.Cast(invizMage, invizWarrior, sim.SpellID(invizSpell)))
	tr.run(f, 60)
	second := f.live.world.Tick()
	f.live.pending = append(f.live.pending, sim.Cast(invizMage, invizWarrior, sim.SpellID(invizSpell)))
	tr.run(f, 400)
	w = f.live.world
	r, ok := invizRemaining(w, invizWarrior, invizSpell)
	var lands []uint64
	for _, c := range tr.casts {
		if c.target == invizWarrior && c.spell == invizSpell {
			lands = append(lands, c.tick)
		}
	}
	if len(lands) != 2 || lands[1] < second || !ok || uint64(r) != fresh-(w.Tick()-lands[1]) || tr.moved || invizDistance(w) != 2 {
		t.Fatalf("b: second Invisibility from tick %d: landings %v, remaining %d %v at tick %d, moved %v, distance %d, casts %v",
			second, lands, r, ok, w.Tick(), tr.moved, invizDistance(w), tr.casts)
	}
	t.Logf("b: second Invisibility ordered at tick %d landed at tick %d from distance 2", second, lands[1])

	f = invisibilityWitnessSAV(t)
	w = f.live.world
	m, _ := w.Entity(invizMage)
	rule, _ := w.Spell(uint32(invizSpell))
	power := min(max(m.Skill[rule.School]+m.Mind-30, 0), 100)
	t.Logf("c mage power %d, Invisibility MaxRange %d, cast range %d", power, rule.MaxRange, int32(rule.MaxRange)+power/30)
	f.live.pending = append(f.live.pending, sim.Cast(invizMage, invizMage, sim.SpellID(invizSpell)))
	var landed []uint16
	for i := 0; i < 120; i++ {
		before, _ := invizRemaining(w, invizMage, invizSpell)
		for _, c := range f.LiveAdvanceCasts(1) {
			if c.Spell == invizSpell {
				after, _ := invizRemaining(w, invizMage, invizSpell)
				landed = append(landed, before, after)
				t.Logf("c tick %d self Invisibility: remaining before %d after %d", w.Tick(), before, after)
			}
		}
		if i == 59 {
			f.live.pending = append(f.live.pending, sim.Cast(invizMage, invizMage, sim.SpellID(invizSpell)))
		}
	}
	if fmt.Sprint(landed) != "[0 6011 5960 6011]" || loadTick+120 != w.Tick() {
		t.Fatalf("c: self Invisibility landings %v at tick %d", landed, w.Tick())
	}
}
