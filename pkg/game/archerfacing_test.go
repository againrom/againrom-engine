package game

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func archerWorld(t *testing.T, ents ...sim.Entity) *mapWorld {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, ents)
	if err != nil {
		t.Fatal(err)
	}
	return archerMapWorld(t, w)
}

func archerHostileWorld(t *testing.T, ents ...sim.Entity) *mapWorld {
	t.Helper()
	var rel sim.Relations
	rel.Set(2, 1, 1)
	w, err := sim.NewRelatedWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{}, ents, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	return archerMapWorld(t, w)
}

func archerMapWorld(t *testing.T, w *sim.World) *mapWorld {
	t.Helper()
	c := worldFixtureArt(16, 16, 8, 14, 4, 4, 64)
	c.Anim = swingAnimDesc()
	c.Corpse = c
	c.Projectile, c.ShootDelay = 3, 5
	v, err := ui.NewViewer("archer", terrain.Grid{Width: 16, Height: 16, Tiles: make([]uint16, 256)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: c}}, v)
	mw.projectiles = &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{3: {Phases: 4}}}
	return mw
}

func archerEnt(id sim.EntityID, owner uint32, x, y int32) sim.Entity {
	e := sim.Entity{ID: id, X: x, Y: y, Class: 1, HP: 1_000_000, MaxHP: 1_000_000, Owner: owner,
		AttackCharge: 12, AttackRelax: 6, Reach: 8, RotationSpeed: 32, Facing: 128, DesiredFacing: 128}
	return e
}

type archerRelease struct {
	tick            int
	body, shot      int
	bearingToVictim int
	turning, moved  bool
}

func circ16(a, b int) int {
	d := (a - b) & 15
	if d > 8 {
		d = 16 - d
	}
	return d
}

func (r archerRelease) faces() bool { return circ16(r.body, r.shot) <= 1 }

func archerRun(mw *mapWorld, ticks int, order func(n int)) []archerRelease {
	var out []archerRelease
	seen := len(mw.world.SavedProjectiles().Items)
	before, _ := mw.entity(1)
	for n := 0; n < ticks; n++ {
		if order != nil {
			order(n)
		}
		mw.tick()
		items := mw.world.SavedProjectiles().Items
		if len(items) > seen {
			p := items[len(items)-1]
			e, _ := mw.entity(1)
			body := (sim.FacingDir(e.DrawnFacing())*2 + 8) & 15
			shot := int(p.ActionDir-8) & 15
			bearing := -1
			if t, ok := mw.entity(e.AttackTarget); ok {
				bearing = EffectFacing(int(t.X-e.X)*256, int(t.Y-e.Y)*256)
			}
			out = append(out, archerRelease{tick: n, body: body, shot: shot, bearingToVictim: bearing,
				turning: e.Turning(), moved: e.Facing != before.Facing})
		}
		seen = len(items)
		before, _ = mw.entity(1)
	}
	return out
}

type archerCase struct {
	name  string
	build func(t *testing.T) *mapWorld
	order func(mw *mapWorld, n int)
}

var archerOffsets = []int{0, 1, 2, 3, 4, 5, 8, 12, 16, 17, 20, 24, 25, 26, 27, 28, 30}

func archerPair(t *testing.T) *mapWorld {
	return archerWorld(t, archerEnt(1, 1, 8, 8), archerVictim(2, 8, 12), archerVictim(3, 8, 4))
}

func archerHostilePair(t *testing.T) *mapWorld {
	h := archerEnt(1, 2, 8, 8)
	h.ScanRange, h.DamageBase, h.AlwaysHits = 8, 3, true
	v := archerVictim(2, 8, 12)
	v.Owner = 1
	return archerHostileWorld(t, h, v)
}

func archerOrderCases() []archerCase {
	var cases []archerCase
	for _, off := range archerOffsets {
		off := off
		cases = append(cases,
			archerCase{fmt.Sprintf("move order north at tick %d", off), archerPair, func(mw *mapWorld, n int) {
				if n == 0 {
					mw.strike(1, 2)
				}
				if n == off {
					mw.pending = append(mw.pending, sim.MoveTo(1, sim.CellPoint{X: 8, Y: 2}))
				}
			}},
			archerCase{fmt.Sprintf("group move order north at tick %d", off), archerPair, func(mw *mapWorld, n int) {
				if n == 0 {
					mw.strike(1, 2)
				}
				if n == off {
					mw.pending = append(mw.pending, sim.GroupMoveTo(1, sim.CellPoint{X: 8, Y: 2}, 1))
				}
			}},
			archerCase{fmt.Sprintf("second attack order on the northern target at tick %d", off), archerPair, func(mw *mapWorld, n int) {
				if n == 0 {
					mw.strike(1, 2)
				}
				if n == off {
					mw.strike(1, 3)
				}
			}},
		)
	}
	return cases
}

func archerVictimCases() []archerCase {
	var cases []archerCase
	for _, off := range archerOffsets {
		off := off
		cases = append(cases,
			archerCase{fmt.Sprintf("player archer, victim walks round to the north at tick %d", off), archerPair, func(mw *mapWorld, n int) {
				if n == 0 {
					mw.strike(1, 2)
				}
				if n == off {
					mw.pending = append(mw.pending, sim.MoveTo(2, sim.CellPoint{X: 8, Y: 3}))
				}
			}},
			archerCase{fmt.Sprintf("hostile archer under its own decision, victim walks round to the north at tick %d", off), archerHostilePair, func(mw *mapWorld, n int) {
				if n == 20+off {
					mw.pending = append(mw.pending, sim.MoveTo(2, sim.CellPoint{X: 8, Y: 3}))
				}
			}},
		)
	}
	return cases
}

func archerTable(t *testing.T, cases []archerCase) (string, []string) {
	var b strings.Builder
	var bad []string
	b.WriteString("scenario | releases (tick: body/shot/bearing in sixteenths; 0 south, 4 west, 8 north, 12 east) | every shot faced\n")
	for _, c := range cases {
		mw := c.build(t)
		rel := archerRun(mw, 90, func(n int) { c.order(mw, n) })
		var parts []string
		ok := true
		for _, r := range rel {
			parts = append(parts, fmt.Sprintf("%d: %d/%d/%d", r.tick, r.body, r.shot, r.bearingToVictim))
			ok = ok && r.faces()
		}
		fmt.Fprintf(&b, "%s | %s | %v\n", c.name, strings.Join(parts, "; "), ok)
		if !ok {
			bad = append(bad, c.name)
		}
	}
	return b.String(), bad
}

func writeArcherTable(t *testing.T, env, table string) {
	t.Helper()
	if p := os.Getenv(env); p != "" {
		if err := os.WriteFile(p, []byte(table), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnOrderDuringAnArchersCycleNeverReleasesAShotOffFacing(t *testing.T) {
	table, bad := archerTable(t, archerOrderCases())
	writeArcherTable(t, "ARCHER_ORDER_TABLE", table)
	if len(bad) != 0 {
		t.Errorf("%d scenario(s) release a shot the drawn body does not face: %s", len(bad), strings.Join(bad, "; "))
	}
	for _, c := range archerOrderCases() {
		mw := c.build(t)
		for _, r := range archerRun(mw, 90, func(n int) { c.order(mw, n) }) {
			if r.turning || r.moved {
				t.Errorf("%s: release at tick %d with the body turning=%v moved=%v", c.name, r.tick, r.turning, r.moved)
			}
		}
	}
}

func archerVictim(id sim.EntityID, x, y int32) sim.Entity {
	e := archerEnt(id, 2, x, y)
	e.Class = 2
	return e
}

func TestAVictimMovingDuringAnArchersCycleLeavesTheLatchedFacing(t *testing.T) {
	table, bad := archerTable(t, archerVictimCases())
	writeArcherTable(t, "ARCHER_VICTIM_TABLE", table)
	if len(bad) == 0 {
		t.Fatal("no scenario moved the victim off the body's facing; the witness no longer exercises the latch")
	}
	for _, c := range archerVictimCases() {
		mw := c.build(t)
		for _, r := range archerRun(mw, 90, func(n int) { c.order(mw, n) }) {
			if r.turning || r.moved {
				t.Errorf("%s: release at tick %d with the body turning=%v moved=%v", c.name, r.tick, r.turning, r.moved)
			}
		}
	}
}
