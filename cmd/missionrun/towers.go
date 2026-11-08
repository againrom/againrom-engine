package main

import (
	"fmt"
	"io"
	"sort"

	"againrom/pkg/game"
	"againrom/pkg/sim"
)

// towerWatch reports the world's building casters and each cast one of them
// queues, by tick. It reads the queue after each step: a building caster's cast
// is queued on the turn phase and resolved at the start of the next step. A
// queued cast whose origin is no caster's cell is a script's and is not counted.
type towerWatch struct {
	out     io.Writer
	casters map[[2]int32]bool
	bySpell map[uint16]int
	byUnit  map[sim.EntityID]int
	total   int
}

func newTowerWatch(on bool, out io.Writer, ms *game.Mission) *towerWatch {
	if !on {
		return nil
	}
	m, w := ms.Map, ms.World
	t := &towerWatch{out: out, casters: map[[2]int32]bool{}, bySpell: map[uint16]int{}, byUnit: map[sim.EntityID]int{}}
	list := w.BuildingCasters()
	fmt.Fprintf(out, "map caster records %d  map type-9 records %d  map objects %d  world structures %d  game builds casters %v\n",
		len(m.BuildingCasters()), len(m.Enchantments), len(m.Objects), len(w.Structures()), w.BuildsCasters())
	fmt.Fprintf(out, "building casters %d\n", len(list))
	for i, c := range list {
		t.casters[[2]int32{int32(c.X), int32(c.Y)}] = true
		fmt.Fprintf(out, "caster %d  cell (%d,%d) spell %d power %d reach %d owner %d building cell word %#x\n",
			i, c.X, c.Y, c.Spell, c.Power, c.Reach, c.Owner, c.Cell)
	}
	return t
}

// sample prints every building caster's cast queued by the last step.
func (t *towerWatch) sample(w *sim.World) {
	if t == nil {
		return
	}
	for _, c := range w.ScriptCasts() {
		if !t.casters[[2]int32{c.FromX, c.FromY}] {
			continue
		}
		e, _ := w.Entity(c.Target)
		fmt.Fprintf(t.out, "tick %d  tower (%d,%d) casts spell %d power %d at unit %d owner %d at (%d,%d) hp %d/%d\n",
			w.Tick(), c.FromX, c.FromY, c.Spell, c.Power, c.Target, e.Owner, e.X, e.Y, e.HP, e.MaxHP)
		t.total++
		t.bySpell[c.Spell]++
		t.byUnit[c.Target]++
	}
}

// report prints the totals by spell and by target unit.
func (t *towerWatch) report() {
	if t == nil {
		return
	}
	fmt.Fprintf(t.out, "tower casts %d\n", t.total)
	spells := make([]int, 0, len(t.bySpell))
	for s := range t.bySpell {
		spells = append(spells, int(s))
	}
	sort.Ints(spells)
	for _, s := range spells {
		fmt.Fprintf(t.out, "  spell %d: %d\n", s, t.bySpell[uint16(s)])
	}
	units := make([]int, 0, len(t.byUnit))
	for u := range t.byUnit {
		units = append(units, int(u))
	}
	sort.Ints(units)
	for _, u := range units {
		fmt.Fprintf(t.out, "  unit %d: %d\n", u, t.byUnit[sim.EntityID(u)])
	}
}
