package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/sim"
)

// tpHero is the save's mage, who knows Teleport (Spell 26).
const (
	tpHero     = sim.EntityID(35)
	tpTeleport = sim.SpellID(26)
)

func tpEnt(w *sim.World, id sim.EntityID) sim.Entity { e, _ := w.Entity(id); return e }

func tpCheb(ax, ay, bx, by int32) int32 {
	dx, dy := ax-bx, ay-by
	return max(dx, -dx, dy, -dy)
}

// tpRun drives one case through the game's own order queue and tick. Each
// tick within the relocation window and each tick whose step ran the group
// pass appends the world's byte form to the case file under
// AGAINROM_TELEPORT_OUT, for the package-internal reader in pkg/sim.
type tpRun struct {
	t     *testing.T
	f     *FrontEnd
	name  string
	out   *os.File
	reloc []uint64
	ring  [][]byte
	dumps int
	heals int
	// window is how many ticks after the last relocation each group pass
	// is still dumped.
	window uint64
}

func (r *tpRun) world() *sim.World { return r.f.live.world }
func (r *tpRun) hero() sim.Entity  { return tpEnt(r.world(), tpHero) }

func (r *tpRun) order(c sim.Command) { r.f.live.pending = append(r.f.live.pending, c) }

// tick runs one game tick and dumps the world when it is inside a recorded
// window: 8 ticks before to 40 after a relocation every tick, then each group
// pass up to 700 ticks after the last relocation.
func (r *tpRun) tick(record bool) {
	before := r.hero()
	var moved []sim.Relocation
	r.f.live.tickStep(nil, false, func(rep sim.Report) { moved = append(moved, rep.Relocations...) })
	w := r.world()
	h := r.hero()
	for _, m := range moved {
		if m.Caster == tpHero {
			r.reloc = append(r.reloc, w.Tick())
			r.t.Logf("%s relocation at %d: %d,%d to %d,%d (hop %d) mana %d->%d hp %d", r.name, w.Tick(), m.FromX, m.FromY, m.ToX, m.ToY,
				tpCheb(m.FromX, m.FromY, m.ToX, m.ToY), before.Mana, h.Mana, h.HP)
		}
	}
	if before.HP >= 1 && h.HP < 1 {
		r.t.Logf("%s hero fell at %d", r.name, w.Tick())
	}
	// The instrument keeps the 19-point hero standing through the mission
	// script's health write, which writes no order, sight or notice field.
	if h.HP >= 1 && h.HP < 12 {
		if err := w.HeadlessHeal(tpHero); err != nil {
			r.t.Fatal(err)
		}
		r.heals++
	}
	if !record || r.out == nil {
		return
	}
	blob, err := w.MarshalBinary()
	if err != nil {
		r.t.Fatal(err)
	}
	var head [12]byte
	binary.LittleEndian.PutUint64(head[:], w.Tick())
	binary.LittleEndian.PutUint32(head[8:], uint32(len(blob)))
	entry := append(head[:], blob...)
	n := len(r.reloc)
	relocated := n > 0 && r.reloc[n-1] == w.Tick()
	if relocated {
		for _, e := range r.ring {
			r.write(e)
		}
		r.ring = r.ring[:0]
	}
	dump := w.ScriptPassJustRan()
	if n > 0 {
		last := r.reloc[n-1]
		dump = dump && w.Tick() <= last+r.window || w.Tick() <= last+40
	}
	if !dump {
		if r.ring = append(r.ring, entry); len(r.ring) > 8 {
			r.ring = r.ring[1:]
		}
		return
	}
	r.ring = r.ring[:0]
	r.write(entry)
}

func (r *tpRun) write(entry []byte) {
	if _, err := r.out.Write(entry); err != nil {
		r.t.Fatal(err)
	}
	r.dumps++
}

func (r *tpRun) ticks(n int, record bool) {
	for i := 0; i < n; i++ {
		r.tick(record)
	}
}

// walk orders the hero to a cell and ticks until it stands there with no
// destination, or limit ticks pass.
func (r *tpRun) walk(x, y int32, limit int, record bool) bool {
	r.order(sim.MoveTo(tpHero, sim.CellPoint{X: x, Y: y}))
	for i := 0; i < limit; i++ {
		r.tick(record)
		if h := r.hero(); h.X == x && h.Y == y && !h.HasTarget {
			return true
		}
	}
	h := r.hero()
	r.t.Logf("%s walk to %d,%d stopped at %d,%d", r.name, x, y, h.X, h.Y)
	return false
}

// cast orders Teleport at a cell and ticks until the hero relocates or
// limit ticks pass, recording.
func (r *tpRun) cast(x, y int32, limit int) bool {
	h := r.hero()
	r.t.Logf("%s tick %d cast Teleport from %d,%d at %d,%d hp %d mana %d", r.name, r.world().Tick(), h.X, h.Y, x, y, h.HP, h.Mana)
	n := len(r.reloc)
	r.order(sim.CastAt(tpHero, tpTeleport, sim.CellPoint{X: x, Y: y}))
	for i := 0; i < limit; i++ {
		before := r.hero()
		r.tick(true)
		a := r.hero()
		if a.X != before.X || a.Y != before.Y {
			r.t.Logf("%s tick %d hero %d,%d", r.name, r.world().Tick(), a.X, a.Y)
		}
		if len(r.reloc) > n {
			return true
		}
		if a.Mana < before.Mana-30 {
			r.t.Logf("%s tick %d paid %d", r.name, r.world().Tick(), before.Mana-a.Mana)
			for k := 0; k < 60 && len(r.reloc) == n; k++ {
				r.tick(true)
			}
			if len(r.reloc) == n {
				r.t.Logf("%s tick %d no relocation 60 ticks after the payment", r.name, r.world().Tick())
			}
			return len(r.reloc) > n
		}
		if a.HP < 1 {
			return false
		}
	}
	r.t.Logf("%s cast at %d,%d did not relocate in %d ticks", r.name, x, y, limit)
	return false
}

func (r *tpRun) chasers() []string {
	var out []string
	for _, e := range r.world().Entities() {
		if e.Alive() && e.HasAttackTarget && e.AttackTarget == tpHero {
			out = append(out, fmt.Sprintf("%d@%d,%d", e.ID, e.X, e.Y))
		}
	}
	return out
}

func (r *tpRun) status(label string) {
	h := r.hero()
	r.t.Logf("%s %s tick %d hero %d,%d hp %d mana %d chasers %v", r.name, label, r.world().Tick(), h.X, h.Y, h.HP, h.Mana, r.chasers())
}

func (r *tpRun) finish() {
	r.status("end")
	r.t.Logf("%s dumped %d worlds, %d health restorations", r.name, r.dumps, r.heals)
}

// reach is the largest distance south of the hero that Teleport admission
// does not refuse as out of range.
func (r *tpRun) reach() int32 {
	h := r.hero()
	reach := int32(0)
	for _, dir := range [][2]int32{{0, 1}, {0, -1}, {1, 0}, {-1, 0}} {
		for d := int32(1); d <= 40; d++ {
			refusal := r.world().BookSpellCellRefusal(tpHero, h.X+d*dir[0], h.Y+d*dir[1], uint32(tpTeleport))
			if refusal == "cell outside the map" || refusal == "caster not alive" {
				break
			}
			if refusal != "cell out of range" {
				reach = max(reach, d)
			}
		}
	}
	r.t.Logf("%s tick %d Teleport reach %d (book slot range %d)", r.name, r.world().Tick(), reach, h.Book.Slots[tpTeleport-1].Range)
	return reach
}

// warm casts Teleport onto a free neighbouring cell, unrecorded. The
// application refreshes the book's cached range from the current power, as
// the SAV's own Spell record holds the constructor base.
func (r *tpRun) warm() {
	r.reach()
	h := r.hero()
	occupied := map[[2]int32]bool{}
	for _, e := range r.world().Entities() {
		occupied[[2]int32{e.X, e.Y}] = true
	}
	for _, d := range [][2]int32{{1, 0}, {0, 1}, {-1, 0}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
		x, y := h.X+d[0], h.Y+d[1]
		if occupied[[2]int32{x, y}] || r.world().BookSpellCellRefusal(tpHero, x, y, uint32(tpTeleport)) != "" {
			continue
		}
		r.t.Logf("%s warm-up", r.name)
		out := r.out
		r.out = nil
		ok := r.cast(x, y, 200)
		r.out, r.reloc = out, nil
		if ok {
			r.reach()
			return
		}
	}
	r.t.Fatalf("%s: no warm-up Teleport relocated", r.name)
}

// holders lists the given units that hold the hero as their victim.
func (r *tpRun) holders(ids ...sim.EntityID) int {
	n := 0
	for _, id := range ids {
		if e := tpEnt(r.world(), id); e.Alive() && e.HasAttackTarget && e.AttackTarget == tpHero {
			n++
		}
	}
	return n
}

const tpLone = sim.EntityID(13)
