package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

const teleportPursuitSAVSHA = "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c"

// tpHero is the save's mage, who knows Teleport (Spell 26).
const (
	tpHero     = sim.EntityID(35)
	tpTeleport = sim.SpellID(26)
)

// teleportPursuitFront loads the EN install's mission-10 SAV named by
// AGAINROM_TELEPORT_SAV through the App LOAD path.
func teleportPursuitFront(t *testing.T) *FrontEnd {
	t.Helper()
	path := os.Getenv("AGAINROM_TELEPORT_SAV")
	if path == "" {
		t.Skip("set AGAINROM_TELEPORT_SAV to the EN install's game0021.sav")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != teleportPursuitSAVSHA {
		t.Fatal("the Teleport SAV SHA differs")
	}
	f := releaseFront(t)
	openOriginalSAVApp(t, f, raw, "teleport.sav")
	if f.live == nil || f.live.world == nil {
		t.Fatal("the SAV loaded no mission")
	}
	return f
}

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

func newTPRun(t *testing.T, name string) *tpRun {
	r := &tpRun{t: t, f: teleportPursuitFront(t), name: name, window: 700}
	if dir := os.Getenv("AGAINROM_TELEPORT_OUT"); dir != "" {
		out, err := os.Create(filepath.Join(dir, name+".worlds"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { out.Close() })
		r.out = out
	}
	return r
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

// tpGuard is the Guard group of three (Group 8) the measured cases name, and
// tpLone the single Guard member (Group 11) on the road south of it.
var tpGuard = []sim.EntityID{6, 7, 8}

const tpLone = sim.EntityID(13)

func tpCellEnv(name string, x, y int32) (int32, int32) {
	if v := os.Getenv(name); v != "" {
		var a, b int32
		if _, err := fmt.Sscanf(v, "%d,%d", &a, &b); err == nil {
			return a, b
		}
	}
	return x, y
}

// TestReleaseTeleportPursuitWitness drives the EN mission-10 SAV into each
// measured case and casts Teleport. It states what this build does; the
// package-internal reader in pkg/sim prints the pursuers' fields.
func TestReleaseTeleportPursuitWitness(t *testing.T) {
	t.Run("reach", func(t *testing.T) {
		// The hero at (43,39) orders the cast on (43,43), first with the
		// book as loaded, then after one applied Teleport.
		r := newTPRun(t, "reach")
		r.walk(43, 39, 3000, false)
		r.status("before")
		r.reach()
		r.cast(43, 43, 400)
		r.status("after")
		r.reach()
		r.finish()
	})
	t.Run("reachwarm", func(t *testing.T) {
		r := newTPRun(t, "reachwarm")
		r.warm()
		r.walk(43, 39, 3000, false)
		r.status("before")
		r.reach()
		r.cast(43, 43, 400)
		r.status("after")
		r.finish()
	})
	t.Run("b1", func(t *testing.T) {
		// A Guard group of three pursues the hero at (41,35); hop to (43,37).
		r := newTPRun(t, "b1")
		r.warm()
		r.walk(41, 35, 3000, false)
		r.status("before")
		r.ticks(8, true)
		x, y := tpCellEnv("AGAINROM_TELEPORT_B1", 43, 37)
		r.cast(x, y, 200)
		r.ticks(700, true)
		r.finish()
	})
	t.Run("b2", func(t *testing.T) {
		// The hero at (33,27) strikes the lone pursuer near (31,23); hop to
		// (36,29).
		r := newTPRun(t, "b2")
		r.warm()
		r.walk(33, 27, 3000, false)
		r.status("at 33,27")
		var near sim.EntityID
		best := int32(99)
		for _, id := range tpGuard {
			e := tpEnt(r.world(), id)
			if d := tpCheb(e.X, e.Y, 31, 23); e.Alive() && d < best {
				near, best = id, d
			}
		}
		r.t.Logf("b2 strikes %d (%d from 31,23), refusal %q", near, best, r.world().BookSpellRefusal(tpHero, near, 1))
		r.order(sim.Cast(tpHero, near, 1))
		hp := tpEnt(r.world(), near).HP
		for i := 0; i < 200 && tpEnt(r.world(), near).HP == hp; i++ {
			r.tick(true)
		}
		r.status("struck")
		r.ticks(4, true)
		x, y := tpCellEnv("AGAINROM_TELEPORT_B2", 36, 29)
		r.cast(x, y, 200)
		r.ticks(700, true)
		r.finish()
	})
	t.Run("q4", func(t *testing.T) {
		// The hero at (43,39), the group idle at its posts; hop to (43,37).
		r := newTPRun(t, "q4")
		r.warm()
		r.walk(43, 39, 3000, false)
		r.status("before")
		r.ticks(8, true)
		r.cast(43, 37, 200)
		r.ticks(700, true)
		r.finish()
	})
	t.Run("b3", func(t *testing.T) {
		// The group has just dropped the hero at (43,37); hop to (43,40).
		r := newTPRun(t, "b3")
		r.warm()
		r.walk(43, 37, 3000, false)
		r.status("at 43,37")
		held := r.holders(tpGuard...)
		i := 0
		for ; i < 2000 && !(held > 0 && r.holders(tpGuard...) == 0); i++ {
			held = max(held, r.holders(tpGuard...))
			r.tick(true)
			if r.hero().HP < 1 {
				break
			}
		}
		r.status(fmt.Sprintf("dropped after %d", i))
		r.cast(43, 40, 200)
		r.ticks(700, true)
		r.finish()
	})
	t.Run("far", func(t *testing.T) {
		// A relocation longer than this hero's Teleport reaches: the
		// instrument places the hero at its start cell during the pursuit,
		// out of the pursuing Group's activity cover, as a long Teleport would
		// (the relocation stores into no other actor, MAGIC-417).
		r := newTPRun(t, "far")
		r.window = 1500
		r.walk(41, 35, 3000, false)
		r.status("before")
		r.ticks(8, true)
		x, y := tpCellEnv("AGAINROM_TELEPORT_FAR", 17, 66)
		if err := r.world().HeadlessPlace(tpHero, x, y); err != nil {
			t.Fatal(err)
		}
		r.reloc = append(r.reloc, r.world().Tick())
		r.status("placed")
		r.ticks(1500, true)
		r.finish()
	})
	t.Run("hops", func(t *testing.T) {
		// Several hops away in a row during the pursuit: a one-cell hop
		// with the book as loaded, then a three-cell hop.
		r := newTPRun(t, "hops")
		r.walk(41, 35, 3000, false)
		r.status("before")
		r.ticks(8, true)
		x, y := tpCellEnv("AGAINROM_TELEPORT_HOP1", 42, 36)
		r.cast(x, y, 200)
		h := r.hero()
		x, y = tpCellEnv("AGAINROM_TELEPORT_HOP2", h.X, h.Y+3)
		r.cast(x, y, 200)
		r.ticks(700, true)
		r.finish()
	})
}
