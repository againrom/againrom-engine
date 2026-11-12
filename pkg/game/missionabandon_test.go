package game

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/internal/synth"
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

func TestAbandonKeepsEntryBackpackAndBag(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, withMod := range []bool{false, true} {
			t.Run(fmt.Sprintf("second=%t/mod=%t", second, withMod), func(t *testing.T) {
				f, a, screen, n := abandonBaseTown(t, second)
				if !second {
					f.Campaign = resolved(saveCampaign(), nil)
					if _, town, err := f.RestoreOriginal(currentTownSave(t, currentTrainingCity(t))); err != nil || !town {
						t.Fatalf("entry town: town=%t error=%v", town, err)
					}
					f.Archives.Containers = poolFixtureFrontMap(t, synth.ALM(synth.ALMOptions{Width: 128, Height: 128})).Archives.Containers
					n = 10
					a = f.App("abandon inventory")
					a.Layout(1024, 768)
					screen = f.TownScreen().(*townScreen)
				}
				secondBagPotionTable(f)
				potion := mapload.ItemInstanceFromCode(secondBagPotion.Code, f.Table)
				items := []sim.ItemInstance{secondBagWeapon, potion, potion, secondBagArmour}
				stacks := slices.Insert(sim.FoldItems(items), 1, sim.ItemStack{})
				if err := f.partyItems(f.Table).writeShopStacks(0, stacks, true); err != nil {
					t.Fatal(err)
				}
				menu, confirm := "Abandon mission", "Abandon, return to town"
				if withMod {
					menu, confirm = "Leave this mission", "Return unchanged"
					data := mod.ScreenData{Screens: []mod.Screen{{Mod: "leave", Key: "leave", Kind: mod.ActionAbandon,
						Game: true, MenuLabel: menu, Title: confirm}}}
					if err := a.SetModScreens(ModScreens(data, f.Table.Rules, f.Words.Engine)); err != nil {
						t.Fatal(err)
					}
				}
				party := mapload.CloneParty(f.Carried)
				var bag []sim.ItemInstance
				if second {
					bag = mapload.CloneParty(f.Town.second.party)[0].CarriedItems
					secondBagEnter(t, f, a, screen)
				} else if err := a.OpenMission(f.MissionOpener(n)); err != nil {
					t.Fatal(err)
				}
				mw := f.live
				if withMod {
					entryPacks, entryWorn := secondBagHeld(mw.world, mw.mission.ids)
					dest, line, open := f.continuity(n, mw.mission.state, mw.advanceNotice)(ui.NoticeRestart)
					if dest != ui.NoticeToMission || open == nil {
						t.Fatalf("restart: destination=%v message=%q", dest, line)
					}
					if err := a.OpenMission(open); err != nil {
						t.Fatal(err)
					}
					mw = f.live
					packs, worn := secondBagHeld(mw.world, mw.mission.ids)
					if !reflect.DeepEqual(packs, entryPacks) || !reflect.DeepEqual(worn, entryWorn) {
						t.Fatal("restart changed the mission entry items")
					}
				}
				id := mw.mission.ids[0]
				actor, _ := mw.world.Entity(id)
				at := sim.CellPoint{X: actor.X + 1, Y: actor.Y}
				found := sim.ItemInstance{Code: 0x1303, Kind: 1, Price: 77}
				if !mw.world.CheatAddItem(id, sim.StackItem(found, 1)) {
					t.Fatal("could not construct the mission loot")
				}
				loot, _ := mw.world.CarriedStacks(id)
				lootSlot := slices.IndexFunc(loot, func(s sim.ItemStack) bool { return s.Code == found.Code })
				mw.pending = append(mw.pending, sim.DropCarriedStack(id, sim.ItemSlot(lootSlot), at))
				for range 32 {
					mw.tick()
				}
				onGround, _ := mw.world.CarriedItems(id)
				if slices.ContainsFunc(onGround, func(it sim.ItemInstance) bool { return it.Code == found.Code }) || len(mw.world.Sacks()) == 0 {
					t.Fatal("pickup control did not put the loot on the ground")
				}
				mw.pending = append(mw.pending, sim.PickUp(id, at))
				for range 400 {
					mw.tick()
				}
				picked, _ := mw.world.CarriedItems(id)
				if !slices.ContainsFunc(picked, func(it sim.ItemInstance) bool { return it.Code == found.Code }) {
					t.Fatal("pickup control did not gain the ground item")
				}
				stacks, _ = mw.world.CarriedStacks(id)
				slot := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == secondBagWeapon.Code })
				if slot < 0 {
					t.Fatal("drop control has no weapon")
				}
				actor, _ = mw.world.Entity(id)
				mw.pending = append(mw.pending, sim.DropCarriedStack(id, sim.ItemSlot(slot), sim.CellPoint{X: actor.X - 1, Y: actor.Y}))
				for range 32 {
					mw.tick()
				}
				dropped, _ := mw.world.CarriedItems(id)
				if slices.ContainsFunc(dropped, func(it sim.ItemInstance) bool { return it.Code == secondBagWeapon.Code }) || len(mw.world.Sacks()) == 0 {
					t.Fatalf("drop control did not lose the weapon: items=%v sacks=%v", bagLabels(dropped), mw.world.Sacks())
				}
				secondBagDrink(t, mw, potion.Code, 30, func() { mw.tick() })
				if err := a.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				for _, label := range []string{menu, confirm} {
					if err := a.HeadlessActivate(label); err != nil {
						t.Fatal(err)
					}
				}
				if a.Screen() != ui.ScreenTown {
					t.Fatalf("abandon left %s: %s", a.Screen(), a.HeadlessMessage())
				}
				for range 1000 {
					if !screen.AtWorldMap() {
						break
					}
					screen.WorldMapTick()
				}
				requireAbandonItems(t, f, party, bag)
				raw := currentTownSave(t, f)
				cold, _, _, _ := abandonBaseTown(t, second)
				cold.Campaign = f.Campaign
				secondBagPotionTable(cold)
				if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
					t.Fatalf("town SAV cold LOAD: town=%t error=%v", town, err)
				}
				requireAbandonItems(t, cold, party, bag)
			})
		}
	}
}

func TestDefeatKeepsTheExistingInventoryBoundary(t *testing.T) {
	for _, second := range []bool{false, true} {
		t.Run(fmt.Sprintf("second=%t", second), func(t *testing.T) {
			f, a, screen, n := abandonBaseTown(t, second)
			if second {
				secondBagEnter(t, f, a, screen)
			} else if err := a.OpenMission(f.MissionOpener(n)); err != nil {
				t.Fatal(err)
			}
			party := mapload.CloneParty(f.Carried)
			var bag []mapload.PartyMember
			if second {
				bag = mapload.CloneParty(f.Town.second.party)
			}
			lost := continuityMission(t, n, sim.ScriptInstantLose)
			if dest, _, _ := f.continuity(n, lost, menuAdvance)(); dest != ui.NoticeToMenu {
				t.Fatalf("defeat destination %v, want menu", dest)
			}
			if !reflect.DeepEqual(f.Carried, party) || second && !reflect.DeepEqual(f.Town.second.party, bag) {
				t.Fatal("defeat changed the retained party or bag")
			}
		})
	}
}

func TestAbandonKeepsTheDepartureCompanion(t *testing.T) {
	f, _, screen, _ := secondBagFront(t, nil)
	f.Table = identityTable()
	f.Carried = []mapload.PartyMember{identityHero()}
	c := f.Town.second
	c.current, c.bank[773] = secondTown(2), 23
	destinationTownCampaign{screen}.leave()
	if len(f.Carried) != 2 || f.Carried[1].CompanionNPC != 23 {
		t.Fatal("departure did not add its companion")
	}
	joined := mapload.CloneParty(f.Carried[1:])
	if refusal := f.CampaignSession.leaveLive(f.townInstall(), 10, nil, false); refusal != "" {
		t.Fatal(refusal)
	}
	if len(f.Carried) != 2 || !reflect.DeepEqual(f.Carried[1:], joined) {
		t.Fatal("abandon changed the departure companion")
	}
}

func requireAbandonItems(t *testing.T, f *FrontEnd, party []mapload.PartyMember, bag []sim.ItemInstance) {
	t.Helper()
	if len(f.Carried) != len(party) {
		t.Fatalf("party size %d, want %d", len(f.Carried), len(party))
	}
	for i, want := range party {
		gotPack, wantPack := mapload.MemberCarriedItems(f.Carried[i], f.Table), mapload.MemberCarriedItems(want, f.Table)
		if !reflect.DeepEqual(gotPack, wantPack) {
			t.Fatalf("member %d backpack changed: got %+v, want %+v", i, gotPack, wantPack)
		}
		if got, want := cityMemberStacks(f.Carried[i], f.Table), cityMemberStacks(want, f.Table); !reflect.DeepEqual(got, want) {
			t.Fatalf("member %d backpack slots changed: got %+v, want %+v", i, got, want)
		}
		if got, want := mapload.MemberItemEquipment(f.Carried[i], f.Table), mapload.MemberItemEquipment(want, f.Table); !reflect.DeepEqual(got, want) {
			t.Fatalf("member %d worn slots changed: got %+v, want %+v", i, got, want)
		}
	}
	if c := secondOf(&f.CampaignSession); c != nil && !reflect.DeepEqual(*c.heroBag(), bag) {
		t.Fatalf("town bag changed: got %+v, want %+v", *c.heroBag(), bag)
	}
}

func TestLoadedSecondMissionAbandonUsesOrdinaryArrival(t *testing.T) {
	seed, app, screen, _ := secondBagFront(t, secondBagSample())
	items := []sim.ItemInstance{secondBagWeapon, {Code: 0x0e4d, Price: -1}}
	if err := seed.partyItems(seed.Table).writeShopStacks(0, sim.FoldItems(items), true); err != nil {
		t.Fatal(err)
	}
	secondBagEnter(t, seed, app, screen)
	raw := skillExport(t, seed, true, "loaded abandon")
	load := func(f *FrontEnd, a *ui.App) {
		t.Helper()
		oldTown := f.Town
		oldSecond := secondOf(&f.CampaignSession)
		var entryParty, entryBag []mapload.PartyMember
		if oldSecond != nil {
			entryParty, entryBag = mapload.CloneParty(oldSecond.entryParty), mapload.CloneParty(oldSecond.entryBag)
		}
		open, town, err := f.RestoreOriginal(raw)
		if err != nil || town || open == nil {
			t.Fatalf("mission LOAD: town=%t error=%v", town, err)
		}
		if f.Town != oldTown {
			t.Fatal("LOAD preparation replaced the running town")
		}
		if oldSecond != nil && (!reflect.DeepEqual(oldSecond.entryParty, entryParty) || !reflect.DeepEqual(oldSecond.entryBag, entryBag)) {
			t.Fatal("LOAD preparation changed the running entry copies")
		}
		if err := a.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		if f.Town == oldTown || f.Town.second.entryParty != nil || f.Town.second.entryBag != nil {
			t.Fatal("committed LOAD retained the previous town or its entry copies")
		}
	}
	for _, previous := range []bool{false, true} {
		t.Run(fmt.Sprintf("previous departure=%t", previous), func(t *testing.T) {
			f := secondMissionFrontOf(t, secondMissionContent{side: 128})
			a := f.App("loaded abandon")
			if previous {
				var town *townScreen
				f, a, town, _ = secondBagFront(t, []sim.ItemInstance{secondBagIglezPack})
				f.Carried[0].Name = "Previous party"
				secondBagEnter(t, f, a, town)
				if f.Town.second.entryParty == nil {
					t.Fatal("previous departure retained no entry copies")
				}
			}
			load(f, a)
			if f.canLeaveMission(10, f.live.world) {
				t.Fatal("a cold mission LOAD unexpectedly supplied a departure town")
			}
			inventory := func(g *FrontEnd) []string {
				all := slices.Clone(*g.Town.second.heroBag())
				for _, member := range g.Carried {
					all = append(all, mapload.MemberCarriedItems(member, g.Table)...)
				}
				return secondBagForm(all)
			}
			missionItems := slices.Clone(*f.Town.second.heroBag())
			for _, actor := range f.live.world.Entities() {
				if actor.Owner == sim.SelfSlot {
					items, _ := f.live.world.CarriedItems(actor.ID)
					missionItems = append(missionItems, items...)
				}
			}
			before := secondBagForm(missionItems)
			control := secondMissionFrontOf(t, secondMissionContent{side: 128})
			load(control, control.App("base arrival"))
			ms := control.live.mission.state
			if refusal := control.carryMissionHome(control.townInstall(), 10, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster); refusal != "" {
				t.Fatal(refusal)
			}
			if err := control.partyItems(control.Table).arriveTown(); err != nil {
				t.Fatal(err)
			}
			f.Town.second.entryTown = secondTown(1)
			ms = f.live.mission.state
			if dest, line, _ := f.continuity(10, ms, f.live.advanceNotice)(ui.NoticeAbandon); dest != ui.NoticeToTown {
				t.Fatalf("abandon: destination=%v message=%q", dest, line)
			}
			requireAbandonItems(t, f, control.Carried, *control.Town.second.heroBag())
			if got := inventory(f); !slices.Equal(got, before) {
				t.Fatalf("loaded items lost or doubled: got %v, want %v", got, before)
			}
			if len(*f.Town.second.heroBag()) == 0 || len(mapload.MemberCarriedItems(f.Carried[0], f.Table)) != 1 {
				t.Fatal("ordinary arrival did not split bag-form units from the unpriced pack unit")
			}
		})
	}
}

func TestLoadedPartyAdoptionClearsEntryCopies(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, secondBagSample())
	secondBagEnter(t, f, app, screen)
	if f.Town.second.entryParty == nil || f.Town.second.entryBag == nil {
		t.Fatal("departure retained no entry copies")
	}
	party := mapload.CloneParty(f.Carried)
	party[0].Name = "Loaded party"
	f.CampaignSession.adopt(&restoreCandidate{town: f.Town, carried: party})
	if f.Town.second.entryParty != nil || f.Town.second.entryBag != nil || f.Carried[0].Name != "Loaded party" {
		t.Fatal("party adoption retained entry copies from another state")
	}
}
