package game

import (
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const monsterReturnTicks = 60 * 16

func monsterCases() []sessionCase {
	cases := sessionEndCases()
	cases[0].front = func(t *testing.T) *FrontEnd {
		f := firstDeathRollFront(t)
		profile, _ := base.Find(base.ROM1EN)
		f.Archives.Base = base.Match{Profile: profile}
		f.Font = resolved(missionFont(), nil)
		return f
	}
	cases[0].mapB = func(t *testing.T) []byte {
		units := []synth.ALMUnit{
			{X: 12<<8 | 0x80, Y: 16<<8 | 0x80, ClassID: 0x50, ClassSubID: 1, UnitID: 31},
			{X: 15<<8 | 0x80, Y: 16<<8 | 0x80, ClassID: 0x42, ClassSubID: 1, UnitID: 32},
		}
		b := synth.ALM(synth.ALMOptions{Width: 128, Height: 128, Name: "Arena", Units: units})
		binary.LittleEndian.PutUint32(b[alm.FileHeaderSize+alm.RecordHeaderSize+alm.MetaWord70:], 2)
		return b
	}
	cases[1].front = func(t *testing.T) *FrontEnd {
		f := secondMissionFrontOf(t, secondGuardMission())
		f.Font = resolved(missionFont(), nil)
		return f
	}
	cases[1].mapB = func(t *testing.T) []byte { return secondMissionMapOf(128, 128, secondGuardMission()) }
	return cases
}

type monster struct {
	id   sim.EntityID
	x, y int32
}

func monsterSession(t *testing.T, c sessionCase) (*FrontEnd, []monster) {
	t.Helper()
	f, app, _ := sessionApp(t, c)
	sessionKey(t, app, "enter", "enter")
	if app.Screen() != ui.ScreenMap || f.live == nil || f.live.mission.fall != fallReturns {
		t.Fatalf("no session map entered: %s", app.HeadlessMessage())
	}
	var out []monster
	for _, e := range f.live.world.Entities() {
		if e.Owner != sim.SelfSlot && e.Alive() {
			out = append(out, monster{e.ID, e.X, e.Y})
		}
	}
	if len(out) < 2 {
		t.Fatalf("the map holds %d monsters, want at least 2", len(out))
	}
	return f, out
}

func monstersDown(t *testing.T, f *FrontEnd, ms []monster) {
	t.Helper()
	for _, m := range ms {
		e, _ := f.live.world.Entity(m.id)
		if err := f.live.world.HeadlessDamage(m.id, e.HP+20); err != nil {
			t.Fatal(err)
		}
	}
}

func monstersStanding(f *FrontEnd, ms []monster) int {
	n := 0
	for _, m := range ms {
		if e, ok := f.live.world.Entity(m.id); ok && e.Alive() {
			n++
		}
	}
	return n
}

func TestSessionMonsterGroupReturnsAfterTheDelay(t *testing.T) {
	for _, c := range monsterCases() {
		t.Run(c.name, func(t *testing.T) {
			f, ms := monsterSession(t, c)
			full := map[sim.EntityID][2]int32{}
			for _, m := range ms {
				e, _ := f.live.world.Entity(m.id)
				full[m.id] = [2]int32{e.MaxHP, e.MaxMana}
			}
			monstersDown(t, f, ms)
			ticks := 0
			for ; monstersStanding(f, ms) == len(ms) && ticks < 200; ticks++ {
				f.live.tick()
			}
			for ; monstersStanding(f, ms) < len(ms) && ticks < 4*monsterReturnTicks; ticks++ {
				f.live.tick()
			}
			if monstersStanding(f, ms) != len(ms) || ticks < monsterReturnTicks {
				t.Fatalf("%d of %d monsters stand after %d ticks, want all after at least %d", monstersStanding(f, ms), len(ms), ticks, monsterReturnTicks)
			}
			for _, m := range ms {
				e, _ := f.live.world.Entity(m.id)
				if e.X != m.x || e.Y != m.y || e.HP != full[m.id][0] || e.Mana != full[m.id][1] {
					t.Fatalf("monster %d at %d,%d health %d mana %d, want cell %d,%d health %d mana %d",
						m.id, e.X, e.Y, e.HP, e.Mana, m.x, m.y, full[m.id][0], full[m.id][1])
				}
			}
		})
	}
}

func TestSessionMonsterGroupStaysDownWhileAMemberStands(t *testing.T) {
	for _, c := range monsterCases() {
		t.Run(c.name, func(t *testing.T) {
			f, ms := monsterSession(t, c)
			monstersDown(t, f, ms[1:])
			for range 3 * monsterReturnTicks {
				f.live.tick()
			}
			if n := monstersStanding(f, ms); n != 1 {
				t.Fatalf("%d monsters stand, want the one that was never hit", n)
			}
		})
	}
}

func TestCampaignMonstersStayDown(t *testing.T) {
	f := firstDeathRollFront(t)
	if err := f.App("campaign monsters").OpenMission(f.MissionOpener(10)); err != nil || f.live == nil {
		t.Fatal("mission 10", err)
	}
	var ms []monster
	for _, e := range f.live.world.Entities() {
		if e.Owner != sim.SelfSlot && e.Alive() {
			ms = append(ms, monster{id: e.ID})
		}
	}
	if len(ms) < 2 {
		t.Fatalf("%d monsters, want at least 2", len(ms))
	}
	monstersDown(t, f, ms)
	for range 3 * monsterReturnTicks {
		f.live.tick()
	}
	if n := monstersStanding(f, ms); n != 0 {
		t.Fatalf("%d campaign monsters stand again", n)
	}
}
