package game

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const returnWait = 3000

func returnSession(t *testing.T, c sessionCase) (*FrontEnd, *ui.App, string, sim.EntityID, [data.SkillSlots]int32) {
	t.Helper()
	f, app, stored := sessionApp(t, c)
	sessionKey(t, app, "enter", "enter")
	if app.Screen() != ui.ScreenMap || f.live == nil || f.live.mission.fall != fallReturns {
		t.Fatalf("no session map entered: %s", app.HeadlessMessage())
	}
	xp := sessionGain(t, f)
	return f, app, stored, secondMissionActor(t, f.live.world).ID, xp
}

func returnHeroEntity(t *testing.T, f *FrontEnd, id sim.EntityID) sim.Entity {
	t.Helper()
	e, ok := f.live.world.Entity(id)
	if !ok {
		t.Fatalf("the world holds no hero %d", id)
	}
	return e
}

func returnWalk(t *testing.T, f *FrontEnd, id sim.EntityID, x, y int32) {
	t.Helper()
	f.live.enqueue(uint32(id), int(x), int(y))
	for tick := 0; tick < returnWait; tick++ {
		if e := returnHeroEntity(t, f, id); e.X == x && e.Y == y {
			return
		}
		f.live.tick()
	}
	t.Fatalf("the hero did not reach %d,%d", x, y)
}

func returnFall(t *testing.T, f *FrontEnd, id sim.EntityID) {
	t.Helper()
	mw := f.live
	mw.pending = append(mw.pending, sim.Damage(id, returnHeroEntity(t, f, id).HP+3))
	for tick := 0; tick < returnWait; tick++ {
		mw.tick()
		if _, waiting := mw.waitingHero(); waiting {
			return
		}
	}
	t.Fatal("the fallen hero's body was never torn down")
}

func returnPenalty(xp [data.SkillSlots]int32) [data.SkillSlots]int32 {
	for s := data.SkillBlade; s < data.SkillSlots; s++ {
		xp[s] = int32(uint32(xp[s])*9) / 10
	}
	return xp
}

func returnNotLost(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	if _, _, open := f.LiveNotice(); open || f.live.mission.announced || app.Screen() != ui.ScreenMap {
		t.Fatalf("the fall ended the map: notice %v outcome %v screen %v", open, f.live.mission.outcome, app.Screen())
	}
}

func TestSessionHeroReturnsOnSpaceAtTheDropCell(t *testing.T) {
	for _, c := range sessionEndCases() {
		t.Run(c.name, func(t *testing.T) {
			f, app, _, id, xp := returnSession(t, c)
			mw := f.live
			drop := mw.mission.state.Start.Drop
			death := mapload.Cell{X: drop.X + 6, Y: drop.Y}
			returnWalk(t, f, id, death.X, death.Y)
			returnFall(t, f, id)
			returnNotLost(t, f, app)
			line := f.Words.Engine.Text("session.return")
			if !slices.ContainsFunc(mw.view.MessageLines(), func(l ui.MessageLine) bool { return l.Text == line }) {
				t.Fatalf("the message line holds no %q", line)
			}
			for tick := 0; tick < returnWait; tick++ {
				mw.tick()
			}
			if e := returnHeroEntity(t, f, id); e.Alive() {
				t.Fatal("the hero returned without Space")
			}
			returnNotLost(t, f, app)
			sackAt := slices.IndexFunc(mw.world.Sacks(), func(s sim.Sack) bool { return s.X == death.X && s.Y == death.Y })
			if sackAt < 0 || !slices.Contains(mw.world.Sacks()[sackAt].Items, eqSwordCode) {
				t.Fatalf("no sack with the sword on the death cell: %+v", mw.world.Sacks())
			}
			sessionKey(t, app, "space")
			if q := mw.pending; len(q) != 1 || q[0] != sim.ReturnHero(id, sim.CellPoint{X: drop.X, Y: drop.Y}) {
				t.Fatalf("Space queued %+v, want the return at the drop cell %v", q, drop)
			}
			mw.tick()
			e := returnHeroEntity(t, f, id)
			if !e.Alive() || e.X != drop.X || e.Y != drop.Y {
				t.Fatalf("hero alive %v at %d,%d, want back at the drop cell %v", e.Alive(), e.X, e.Y, drop)
			}
			if want := returnPenalty(xp); e.SkillXP != want {
				t.Fatalf("experience %v, want nine tenths %v of %v", e.SkillXP, want, xp)
			}
			if e.HP != e.MaxHP || e.Mana != e.MaxMana || e.MaxHP <= 0 {
				t.Fatalf("health %d/%d mana %d/%d, want both full", e.HP, e.MaxHP, e.Mana, e.MaxMana)
			}
			carried, worn := sessionHolding(t, f)
			if len(carried) != 0 || slices.ContainsFunc(worn[:], func(i sim.ItemInstance) bool { return !i.Empty() }) {
				t.Fatalf("the hero came back holding %+v wearing %+v", carried, worn)
			}
			returnNotLost(t, f, app)
		})
	}
}

func TestSessionHeroReturnsNearTheShopCell(t *testing.T) {
	for _, c := range sessionEndCases() {
		t.Run(c.name, func(t *testing.T) {
			f, app, _, id, _ := returnSession(t, c)
			mw := f.live
			drop := mw.mission.state.Start.Drop
			shop := mapload.Cell{X: drop.X, Y: drop.Y + 6}
			returnWalk(t, f, id, shop.X, shop.Y)
			mw.observeShopUses([]sim.StructureUse{{Entity: id}})
			mw.shop.waiting = false
			returnWalk(t, f, id, drop.X+6, drop.Y)
			returnFall(t, f, id)
			if _, err := mw.world.CheatSummon(sim.Entity{Owner: 3, HP: 50, MaxHP: 50}, nil,
				[sim.EquipSlots]sim.ItemInstance{}, shop.X, shop.Y); err != nil {
				t.Fatal(err)
			}
			sessionKey(t, app, "space")
			if q := mw.pending; len(q) != 1 || q[0] != sim.ReturnHero(id, sim.CellPoint{X: shop.X, Y: shop.Y}) {
				t.Fatalf("Space queued %+v, want the return at the Shop cell %v", q, shop)
			}
			mw.tick()
			if e := returnHeroEntity(t, f, id); !e.Alive() || e.X != shop.X || e.Y != shop.Y-1 {
				t.Fatalf("hero alive %v at %d,%d, want the free cell %d,%d nearest the Shop cell", e.Alive(), e.X, e.Y, shop.X, shop.Y-1)
			}
		})
	}
}

func TestSessionHeroReturnsAtFullHealthAndManaEachTime(t *testing.T) {
	for _, c := range sessionEndCases() {
		t.Run(c.name, func(t *testing.T) {
			f, app, _, id, _ := returnSession(t, c)
			for round := 1; round <= 4; round++ {
				returnFall(t, f, id)
				sessionKey(t, app, "space")
				for tick := 0; tick < 4; tick++ {
					f.live.tick()
					e := returnHeroEntity(t, f, id)
					if !e.Alive() || e.HP != e.MaxHP || e.Mana != e.MaxMana {
						t.Fatalf("return %d tick %d: alive %v health %d/%d mana %d/%d, want both full",
							round, tick, e.Alive(), e.HP, e.MaxHP, e.Mana, e.MaxMana)
					}
				}
				returnNotLost(t, f, app)
			}
		})
	}
}

func TestSessionDeathWritesNoCharacterFile(t *testing.T) {
	for _, c := range sessionEndCases() {
		for _, returned := range []bool{false, true} {
			name := c.name + "/fallen"
			if returned {
				name = c.name + "/returned"
			}
			t.Run(name, func(t *testing.T) {
				f, app, stored, id, xp := returnSession(t, c)
				purse := f.live.world.Purse(sim.SelfSlot)
				returnFall(t, f, id)
				if returned {
					sessionKey(t, app, "space")
					f.live.tick()
				}
				sessionEnd(t, f, app, "leave")
				b, err := os.ReadFile(filepath.Join(stored, c.file))
				if err != nil {
					t.Fatal(err)
				}
				if !returned {
					if !bytes.Equal(b, c.record(t)) {
						t.Fatal("the fallen character's file changed")
					}
					return
				}
				sessionWantFile(t, c, sessionGains{xp: returnPenalty(xp), purse: purse, file: b})
			})
		}
	}
}
