package game

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type walkRelease struct {
	tick, body, shot, bearing int
	state                     uint8
	moving, turning           bool
	transit                   int
	phase                     sim.AttackPhase
	hasTarget                 bool
}

func (r walkRelease) faces() bool { return circ16(r.body, r.shot) <= 1 }

type walkCase struct {
	name    string
	hero    bool
	hostile bool
	slow    bool
	order   func(mw *mapWorld, n int)
}

func walkWorld(t *testing.T, c walkCase) *mapWorld {
	t.Helper()
	a := archerEnt(1, 1, 3, 8)
	a.Facing, a.DesiredFacing = 64, 64
	if c.slow {
		a.Speed = 6
	}
	behind := archerVictim(2, 1, 8)
	beside := archerVictim(3, 3, 11)
	ents := []sim.Entity{a, behind, beside}
	var rel sim.Relations
	rel.Set(1, 2, 1)
	if c.hostile {
		rel.Set(2, 1, 1)
	}
	w, err := sim.NewRelatedWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{}, ents, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	mw := archerMapWorld(t, w)
	if c.hero {
		mw.mission = &missionNotices{ids: []sim.EntityID{1}, party: []mapload.PartyMember{{Class: 1, MercenaryType: 1}}}
	}
	return mw
}

func walkRun(mw *mapWorld, ticks int, order func(n int)) []walkRelease {
	var out []walkRelease
	seen := len(mw.world.SavedProjectiles().Items)
	prev, _ := mw.entity(1)
	for n := 0; n < ticks; n++ {
		order(n)
		mw.tick()
		items := mw.world.SavedProjectiles().Items
		e, _ := mw.entity(1)
		if len(items) > seen {
			p := items[len(items)-1]
			bearing := -1
			if t, ok := mw.entity(e.AttackTarget); ok {
				bearing = EffectFacing(int(t.X-e.X)*256, int(t.Y-e.Y)*256)
			}
			out = append(out, walkRelease{tick: n, body: (sim.FacingDir(e.DrawnFacing())*2 + 8) & 15, shot: int(p.ActionDir-8) & 15,
				bearing: bearing, state: e.ActorState, moving: e.X != prev.X || e.Y != prev.Y || e.Transit != 0,
				turning: e.Turning(), transit: int(e.Transit), phase: e.AttackPhase, hasTarget: e.HasTarget})
		}
		seen = len(items)
		prev = e
	}
	return out
}

func walkCases() []walkCase {
	move := func(mw *mapWorld, group bool, x, y int32) {
		if group {
			mw.pending = append(mw.pending, sim.GroupMoveTo(1, sim.CellPoint{X: x, Y: y}, 1))
		} else {
			mw.pending = append(mw.pending, sim.MoveTo(1, sim.CellPoint{X: x, Y: y}))
		}
	}
	var cs []walkCase
	for _, slow := range []bool{false, true} {
		for _, hero := range []bool{false, true} {
			for _, group := range []bool{false, true} {
				tag := fmt.Sprintf("slow=%v hero=%v group=%v", slow, hero, group)
				slow, hero, group := slow, hero, group
				cs = append(cs, walkCase{name: tag + " walk east, enemy behind and beside, automatic acquisition", hero: hero, hostile: true, slow: slow,
					order: func(mw *mapWorld, n int) {
						if n == 0 {
							move(mw, group, 14, 8)
						}
					}})
				for _, k := range []int{1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 20, 24} {
					k := k
					cs = append(cs, walkCase{name: fmt.Sprintf("%s walk east, attack west enemy at tick %d", tag, k), hero: hero, slow: slow,
						order: func(mw *mapWorld, n int) {
							if n == 0 {
								move(mw, group, 14, 8)
							}
							if n == k {
								mw.strike(1, 2)
							}
						}})
					cs = append(cs, walkCase{name: fmt.Sprintf("%s walk east, attack beside enemy at tick %d", tag, k), hero: hero, slow: slow,
						order: func(mw *mapWorld, n int) {
							if n == 0 {
								move(mw, group, 14, 8)
							}
							if n == k {
								mw.strike(1, 3)
							}
						}})
				}
				for _, a := range []int{1, 2, 4, 6, 10} {
					for _, d := range []int{1, 2, 3, 4, 6} {
						a, d := a, d
						cs = append(cs, walkCase{name: fmt.Sprintf("%s move east@0, attack west@%d, move north-east@%d", tag, a, a+d), hero: hero, slow: slow,
							order: func(mw *mapWorld, n int) {
								switch n {
								case 0:
									move(mw, group, 14, 8)
								case a:
									mw.strike(1, 2)
								case a + d:
									move(mw, group, 14, 2)
								}
							}})
						cs = append(cs, walkCase{name: fmt.Sprintf("%s move east@0, attack west@%d, move east again@%d, attack beside@%d", tag, a, a+d, a+2*d), hero: hero, slow: slow,
							order: func(mw *mapWorld, n int) {
								switch n {
								case 0:
									move(mw, group, 14, 8)
								case a:
									mw.strike(1, 2)
								case a + d:
									move(mw, group, 14, 8)
								case a + 2*d:
									mw.strike(1, 3)
								}
							}})
					}
				}
			}
		}
	}
	return cs
}

func TestAWalkingArcherReleaseTable(t *testing.T) {
	var b strings.Builder
	b.WriteString("scenario | releases tick: body/shot/bearing (sixteenths, 0 south 8 north), state, moving, turning, transit, phase, hasDest | off-facing\n")
	bad := 0
	for _, c := range walkCases() {
		mw := walkWorld(t, c)
		rel := walkRun(mw, 110, func(n int) { c.order(mw, n) })
		var parts []string
		off := false
		for _, r := range rel {
			parts = append(parts, fmt.Sprintf("%d: %d/%d/%d st%d mv=%v tn=%v tr%d ph%d dest=%v", r.tick, r.body, r.shot, r.bearing, r.state, r.moving, r.turning, r.transit, r.phase, r.hasTarget))
			off = off || !r.faces()
		}
		if off {
			bad++
		}
		fmt.Fprintf(&b, "%s | %s | %v\n", c.name, strings.Join(parts, "; "), off)
	}
	if p := os.Getenv("ARCHER_WALK_TABLE"); p != "" {
		if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if bad != 0 {
		t.Errorf("%d walking scenarios release a shot the drawn body does not face", bad)
	}
}
