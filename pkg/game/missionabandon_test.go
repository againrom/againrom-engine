package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func abandonBaseTown(t *testing.T, second bool) (*FrontEnd, *ui.App, *townScreen, int) {
	t.Helper()
	if second {
		f, a, screen, _ := secondBagFront(t, secondBagSample())
		f.Carried[0], _ = memberWithPack(f.Carried[0], nil, f.Table, true)
		return f, a, screen, 10
	}
	f := winRouteFirstFront(t, mod.MissionData{})
	f.Archives.Containers = missionArchive(t, 20)
	f.Tiles, f.Words = &terrain.Tileset{}, ui.AuthoredWords()
	f.arriveInTown()
	f.Town.available[20] = true
	a := f.App("abandon")
	a.Layout(1024, 768)
	return f, a, f.TownScreen().(*townScreen), 20
}

func TestBaseAbandonReturnsTheEntryPartyInBothGames(t *testing.T) {
	for _, second := range []bool{false, true} {
		name := "rom1"
		if second {
			name = "rom2"
		}
		t.Run(name, func(t *testing.T) {
			f, a, screen, n := abandonBaseTown(t, second)
			if !f.ModSet().Empty() {
				t.Fatal("the base witness enabled a mod")
			}
			party, gold := mapload.CloneParty(f.Carried), f.Town.Gold()
			var bag []string
			if second {
				bag = bagLabels(*f.Town.second.heroBag())
				secondBagEnter(t, f, a, screen)
			} else if err := a.OpenMission(f.MissionOpener(n)); err != nil {
				t.Fatal(err)
			}
			mw := f.live
			if !mw.world.SetPurse(sim.SelfSlot, uint32(gold+777)) {
				t.Fatal("could not change the mission purse")
			}
			if err := mw.world.HeadlessDamage(mw.mission.ids[0], 1); err != nil {
				t.Fatal(err)
			}
			if second {
				id := mw.mission.ids[0]
				hero, _ := mw.world.Entity(id)
				stacks, _ := mw.world.CarriedStacks(id)
				found := false
				for slot, stack := range stacks {
					if stack.Code == secondBagWeapon.Code {
						mw.pending = append(mw.pending, sim.DropCarriedStack(id, sim.ItemSlot(slot), sim.CellPoint{X: hero.X + 1, Y: hero.Y}))
						found = true
						break
					}
				}
				for range 32 {
					mw.tick()
				}
				if !found || len(mw.world.Sacks()) == 0 {
					t.Fatal("the loss control did not drop the packed mission weapon")
				}
			}
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessActivate("Abandon mission"); err != nil {
				t.Fatal(err)
			}
			rows := a.HeadlessRows()
			if len(rows) != 2 || rows[0].Text != "Abandon, return to town" {
				t.Fatalf("confirm rows: %+v", rows)
			}
			if err := a.HeadlessActivate(rows[0].Text); err != nil || a.Screen() != ui.ScreenTown {
				t.Fatalf("confirm: %v on %s (%s)", err, a.Screen(), a.HeadlessMessage())
			}
			if second {
				c := f.Town.second
				if c.current != secondTown(1) {
					t.Fatalf("abandon returned to %v, want the entry town", c.current)
				}
				if !reflect.DeepEqual(bagLabels(*c.heroBag()), bag) {
					t.Fatalf("entry bag %v, returned %v", bag, bagLabels(*c.heroBag()))
				}
			}
			for i := range party {
				if len(party[i].Carried) == 0 && len(f.Carried[i].Carried) == 0 {
					party[i].Carried = f.Carried[i].Carried
				}
				if len(party[i].CarriedItems) == 0 && len(f.Carried[i].CarriedItems) == 0 {
					party[i].CarriedItems = f.Carried[i].CarriedItems
				}
			}
			if !reflect.DeepEqual(f.Carried, party) || f.Town.Gold() != gold || f.Town.Done(n) {
				campaignFlowValueDiff(t, "party", party, f.Carried)
				t.Fatalf("abandon changed entry party, gold or completion: party=%t gold=%d/%d won=%t", reflect.DeepEqual(f.Carried, party), f.Town.Gold(), gold, f.Town.Done(n))
			}
			offered := containsMission(f.Town.Available(), n)
			if second {
				offered = f.Town.second.has(partyPlace{kind: placeMission, id: n})
			}
			if !offered {
				t.Fatalf("mission %d is no longer offered: %v", n, f.Town.Available())
			}
			if second {
				raw := currentTownSave(t, f)
				cold := secondMissionFront(t)
				if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
					t.Fatalf("returned town SAVE/LOAD: town=%t %v", town, err)
				}
				if cold.Town.Gold() != gold || cold.Town.second.current != secondTown(1) ||
					!reflect.DeepEqual(bagLabels(*cold.Town.second.heroBag()), bag) {
					t.Fatal("returned town SAVE/LOAD changed the purse, town or bag")
				}
			}
		})
	}
}

func TestBaseAbandonControlsBothGames(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, state := range []string{"off", "won", "lost", "no town"} {
			t.Run(fmt.Sprintf("second=%t/%s", second, state), func(t *testing.T) {
				f, a, screen, n := abandonBaseTown(t, second)
				if state == "off" {
					p := rules.Defaults()
					p.Abandon = 0
					f.Table.Rules, _ = rules.New(p)
					a = f.App("off")
				}
				if second {
					secondBagEnter(t, f, a, screen)
				} else if err := a.OpenMission(f.MissionOpener(n)); err != nil {
					t.Fatal(err)
				}
				if state == "won" || state == "lost" {
					op := int32(sim.ScriptInstantWin)
					if state == "lost" {
						op = sim.ScriptInstantLose
					}
					f.live.world = continuityMission(t, n, op).World
				}
				if state == "no town" {
					f.Town.open = false
					if second {
						f.Town.second.entryTown = partyPlace{}
					}
				}
				if err := a.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				for _, row := range a.HeadlessRows() {
					if row.Text == "Abandon mission" {
						t.Fatalf("%s still shows abandon", state)
					}
				}
			})
		}
	}
}

func TestExampleModReplacesTheBaseAbandonAction(t *testing.T) {
	entries, err := mod.Resolve(abandonModDir, []string{"mission-abandon"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, "rom1-en", nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	f, a, _, n := abandonBaseTown(t, false)
	if err := a.SetModScreens(ModScreens(res.Screens, res.Rules, f.Words.Engine)); err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(f.MissionOpener(n)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, row := range a.HeadlessRows() {
		if row.Text == "Abandon mission" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("the example mod shows %d abandon entries", count)
	}
}
