package game

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// meleeBlow is one load of a melee attack cycle.
type meleeBlow struct {
	tick, body, bearing int
	turning             bool
}

func (b meleeBlow) faces() bool { return b.bearing >= 0 && circ16(b.body, b.bearing) <= 1 }

type meleeCase struct {
	walkCase
	victimAt *sim.CellPoint
}

// meleeCases are the walking cases and pursuit cases whose victim walks
// beside the attacker's path while it closes.
func meleeCases() []meleeCase {
	var cs []meleeCase
	for _, c := range walkCases() {
		cs = append(cs, meleeCase{walkCase: c})
	}
	starts := []sim.CellPoint{{X: 9, Y: 8}, {X: 9, Y: 6}, {X: 9, Y: 10}, {X: 7, Y: 5}}
	dests := []sim.CellPoint{{X: 5, Y: 7}, {X: 5, Y: 9}, {X: 4, Y: 6}, {X: 6, Y: 10}, {X: 3, Y: 10}, {X: 7, Y: 8}, {X: 2, Y: 7}, {X: 8, Y: 4}}
	for _, slow := range []bool{false, true} {
		for _, hero := range []bool{false, true} {
			for _, st := range starts {
				for _, d := range dests {
					for _, k := range []int{0, 2, 4, 6, 8, 10, 13, 16, 20, 25} {
						st, d, k := st, d, k
						cs = append(cs, meleeCase{victimAt: &st, walkCase: walkCase{
							name: fmt.Sprintf("slow=%v hero=%v attack victim at %d,%d, victim walks to %d,%d at tick %d", slow, hero, st.X, st.Y, d.X, d.Y, k),
							hero: hero, slow: slow,
							order: func(mw *mapWorld, n int) {
								if n == 0 {
									mw.strike(1, 2)
								}
								if n == k {
									mw.pending = append(mw.pending, sim.MoveTo(2, d))
								}
							}}})
					}
				}
			}
		}
	}
	return cs
}

func meleeWalkWorld(t *testing.T, c meleeCase) *mapWorld {
	t.Helper()
	a := archerEnt(1, 1, 3, 8)
	a.Reach = 1
	a.Facing, a.DesiredFacing = 64, 64
	if c.slow {
		a.Speed = 6
	}
	victim := archerVictim(2, 1, 8)
	if c.victimAt != nil {
		victim.X, victim.Y = c.victimAt.X, c.victimAt.Y
	}
	ents := []sim.Entity{a, victim, archerVictim(3, 3, 11)}
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

func meleeWalkRun(mw *mapWorld, ticks int, order func(n int)) []meleeBlow {
	var out []meleeBlow
	prev, _ := mw.entity(1)
	for n := 0; n < ticks; n++ {
		order(n)
		mw.tick()
		e, _ := mw.entity(1)
		if e.AttackPhase == sim.AttackCharging && prev.AttackPhase != sim.AttackCharging {
			bearing := -1
			if v, ok := mw.entity(e.AttackTarget); ok {
				bearing = EffectFacing(int(v.X-e.X)*256, int(v.Y-e.Y)*256)
			}
			out = append(out, meleeBlow{tick: n, body: (sim.FacingDir(e.Facing)*2 + 8) & 15, bearing: bearing, turning: e.Turning()})
		}
		prev = e
	}
	return out
}

// Every load of a melee cycle finds the body facing its victim (AI-405).
func TestAWalkingMeleeAttackerFacesItsVictimAtEveryLoad(t *testing.T) {
	var b strings.Builder
	b.WriteString("scenario | loads tick: body/bearing (sixteenths, 0 south 8 north), turning | off-facing first load | off-facing any load\n")
	firstOff, anyOff, scenarios, loads := 0, 0, 0, 0
	for _, c := range meleeCases() {
		mw := meleeWalkWorld(t, c)
		blows := meleeWalkRun(mw, 160, func(n int) { c.order(mw, n) })
		var parts []string
		off := false
		for _, r := range blows {
			parts = append(parts, fmt.Sprintf("%d: %d/%d tn=%v", r.tick, r.body, r.bearing, r.turning))
			off = off || !r.faces()
		}
		first := len(blows) > 0 && !blows[0].faces()
		if len(blows) > 0 {
			scenarios++
		}
		loads += len(blows)
		if first {
			firstOff++
		}
		if off {
			anyOff++
		}
		fmt.Fprintf(&b, "%s | %s | %v | %v\n", c.name, strings.Join(parts, "; "), first, off)
	}
	fmt.Fprintf(&b, "scenarios with a load %d of %d; loads %d; off-facing first load %d; off-facing any load %d\n", scenarios, len(meleeCases()), loads, firstOff, anyOff)
	if p := os.Getenv("MELEE_WALK_TABLE"); p != "" {
		if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if scenarios < len(meleeCases())/2 {
		t.Fatalf("only %d of %d scenarios load a blow; the witness no longer exercises the melee load", scenarios, len(meleeCases()))
	}
	if anyOff != 0 {
		t.Errorf("%d walking scenarios load a melee blow the body does not face (%d on the first blow)", anyOff, firstOff)
	}
}
