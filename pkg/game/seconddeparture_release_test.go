package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type secondLaterProof struct {
	Town        *secondTownSample
	TownGraph   *cityObjectTopology
	TownSpells  map[sim.SavedObjectID]sim.SourceItemSpell
	Mission     []secondSaveSample
	Actor       sim.EntityID
	X, Y, Ticks int
}

func secondScriptAdmissionReport(t *testing.T, f *FrontEnd) mapload.ScriptReport {
	t.Helper()
	m := f.live.mission.state
	_, rep, err := mapload.CompileROM2Script(m.Map, campaignScriptRefs(m.Map, f.Table, m.Party))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Unresolved) != 0 || len(rep.OmittedTriggers) != 0 || len(rep.OmittedChecks) != 0 || len(rep.OmittedActions) != 0 || len(f.live.world.Script().Unsupported()) != 0 {
		t.Fatalf("admitted roster has unresolved script support: %+v", rep)
	}
	return rep
}

func secondLaterChoose(t *testing.T, app *ui.App, targets ...string) {
	t.Helper()
	for _, target := range targets {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
}

func secondLaterAssertGraph(t *testing.T, want, got *cityObjectTopology, generated map[sim.SavedObjectID]sim.SourceItemSpell) {
	t.Helper()
	if want == nil || got == nil {
		t.Fatal("town continuation lacks its current item/child graph")
	}
	want = want.Clone()
	if len(want.HumanTails) == 0 {
		for _, root := range want.Roots {
			want.HumanTails = append(want.HumanTails, cityHumanTails{PartyID: bytes.Clone(root.PartyID)})
		}
	}
	keys := make(map[uint32]bool)
	for id, row := range got.SpellRecords {
		if row.This == 0 || keys[row.This] {
			t.Fatal("city Spell constructor lacks a distinct retained handle", id)
		}
		keys[row.This] = true
		if _, present := want.SpellRecords[id]; present {
			continue
		}
		value, expected := generated[id]
		record := sim.SavedSpellObject{ID: id, This: row.This, Value: value}
		if !expected || !reflect.DeepEqual(row, record) {
			t.Fatal("generated city Spell supplement changed independent book value", id, row, value)
		}
		if want.SpellRecords == nil {
			want.SpellRecords = make(map[sim.SavedObjectID]sim.SavedSpellObject)
		}
		want.SpellRecords[id] = record
	}
	if reflect.DeepEqual(want, got) {
		return
	}
	a, b := reflect.ValueOf(*want), reflect.ValueOf(*got)
	for field := 0; field < a.NumField(); field++ {
		if !reflect.DeepEqual(a.Field(field).Interface(), b.Field(field).Interface()) {
			t.Logf("city graph %s before=%+v after=%+v", a.Type().Field(field).Name, a.Field(field).Interface(), b.Field(field).Interface())
		}
	}
	t.Fatal("town LOAD changed item/child graph")
}

func secondLaterTownAction(t *testing.T, app *ui.App) {
	t.Helper()
	secondLaterChoose(t, app, "GATES", "town 2", "CANCEL", "town 2", "ENTER")
}

func secondLaterColdProof(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var proof secondLaterProof
	if err := json.Unmarshal(raw, &proof); err != nil {
		t.Fatal(err)
	}
	if proof.Town != nil {
		f, app := secondTownCold(t, filepath.Dir(path), filepath.Base(path))
		secondTownAssertSample(t, *proof.Town, secondTownSampleNow(t, f, app))
		secondLaterAssertGraph(t, proof.TownGraph, f.Town.cityObjects, proof.TownSpells)
		secondLaterTownAction(t, app)
		secondTownAssertSample(t, *proof.Town, secondTownSampleNow(t, f, app))
		secondTownNamedSave(t, f, app, filepath.Dir(path), "Fresh process town")
		cold, _ := secondTownCold(t, filepath.Dir(path), "Fresh process town.sav")
		secondLaterAssertGraph(t, f.Town.cityObjects, cold.Town.cityObjects, nil)
	} else {
		f, app := secondMissionColdAt(t, filepath.Dir(path), filepath.Base(path), 21)
		secondAssertSample(t, proof.Mission[0], secondSaveSampleNow(t, f, app))
		secondMissionMove(t, app, proof.Actor, proof.X, proof.Y)
		for range proof.Ticks {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		secondAssertSample(t, proof.Mission[1], secondSaveSampleNow(t, f, app))
		secondMissionNamedSave(t, f, app, filepath.Dir(path), "Fresh process mission")
	}
	t.Log("fresh-process LOAD -> same composed presentation/current state -> next action -> resave")
}

func secondLaterGraph(t *testing.T, live *mapWorld, f *FrontEnd) *cityObjectTopology {
	t.Helper()
	registry, graph := live.world.SavedObjects(), f.Town.cityObjects
	if registry == nil || graph == nil || graph.NextID != registry.NextID || len(graph.Roots) != len(f.Carried) {
		t.Fatal("return lost current registry/party graph")
	}
	for i, p := range f.Carried {
		id := live.mission.ids[i]
		want := cityPartyObjectRoots{PartyID: []byte(p.ID)}
		for _, container := range registry.Containers {
			if container.Owner == (sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: id}) {
				want.Pack = append(want.Pack, container.Items...)
			}
		}
		for _, root := range registry.ItemRoots {
			if root.Owner.Kind == sim.SavedOwnerActorWorn && root.Owner.Entity == id {
				want.Worn[root.Owner.Slot-1] = root.ID
			}
		}
		if !reflect.DeepEqual(want, graph.Roots[i]) {
			t.Fatal("return changed independent live pack/equipment roots", p.ID, want, graph.Roots[i])
		}
	}
	for _, item := range graph.Items {
		row, exists := registry.Item(item.ID)
		if !exists || !reflect.DeepEqual(graph.ItemRecords[item.ID], row) || !reflect.DeepEqual(item.Effects, row.Effects) || item.Spell != row.Spell {
			t.Fatal("return changed independent live Item identity/children", item.ID)
		}
	}
	for id, row := range graph.EffectRecords {
		found := false
		for _, source := range registry.Effects {
			found = found || source.ID == id && reflect.DeepEqual(row, source)
		}
		if !found {
			t.Fatal("return changed independent live Effect record", id)
		}
	}
	for id, row := range graph.SpellRecords {
		found := false
		for _, source := range registry.Spells {
			found = found || source.ID == id && reflect.DeepEqual(row, source)
		}
		if !found {
			t.Fatal("return changed independent live Spell record", id)
		}
	}
	return graph.Clone()
}

func secondLaterSpellValues(t *testing.T, graph *cityObjectTopology, party []mapload.PartyMember) map[sim.SavedObjectID]sim.SourceItemSpell {
	t.Helper()
	values := make(map[sim.SavedObjectID]sim.SourceItemSpell)
	for _, book := range graph.Books {
		for _, p := range party {
			if p.ID != string(book.PartyID) {
				continue
			}
			for slot, id := range book.Slots {
				if id == 0 {
					continue
				}
				if !p.Book.HasInstances() {
					t.Fatal("graph Spell lacks its live member book instance", id)
				}
				value := p.Book.Slots[slot]
				values[id] = sim.SourceItemSpell{Present: true, ID: uint8(slot + 1), Range: value.Range, Defensive: value.Defensive, ManaCost: value.ManaCost}
			}
		}
	}
	return values
}

func secondLaterEmitProof(t *testing.T, path string, proof secondLaterProof) {
	t.Helper()
	raw, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".json", raw, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_SECOND_LATER_INPUT")
}

func secondLaterLossControls(t *testing.T, f *FrontEnd, raw []byte) {
	t.Helper()
	for _, loss := range []string{"restoration", "stage", "completed", "unknown input", "foreign current", "duplicate", "unavailable21", "completed21"} {
		changed := secondTownChangeJSON(t, raw, func(a map[string]any) {
			c := a["Session"].(map[string]any)["Second"].(map[string]any)
			bank := c["Bank"].([]any)
			switch loss {
			case "restoration":
				bank[775] = 1
			case "stage":
				bank[768] = 40
			case "completed":
				bank[916] = 0
			case "unknown input":
				c["Auxiliary"] = []int{10000, 10000, 10000, 10000}
			case "foreign current":
				c["Current"] = map[string]any{"Kind": 2, "ID": 3}
			case "duplicate":
				c["Available"] = append(c["Available"].([]any), c["Available"].([]any)[0])
			case "unavailable21":
				bank[772] = 0
				c["Available"] = []map[string]any{{"Kind": 2, "ID": 2}, {"Kind": 1, "ID": 21}}
			case "completed21":
				bank[917] = 1
				c["Available"] = []map[string]any{{"Kind": 2, "ID": 2}, {"Kind": 1, "ID": 21}}
			}
		})
		before, _, err := f.Snapshot(f.live != nil)
		if err != nil {
			t.Fatal(err)
		}
		town, live := f.Town, f.live
		if _, _, err := f.RestoreOriginal(changed); err == nil {
			t.Fatal("invalid later continuation admitted", loss)
		}
		after, _, err := f.Snapshot(f.live != nil)
		if err != nil || town != f.Town || live != f.live || !reflect.DeepEqual(before, after) {
			t.Fatal("invalid later continuation changed source", loss, err)
		}
	}
}

func secondTwentyWin(t *testing.T, f *FrontEnd, app *ui.App, unlock bool) {
	t.Helper()
	live := f.live
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, live)
	id := live.mission.ids[0]
	if unlock {
		if err := live.world.HeadlessPlace(id, 26, 32); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		live.view.Camera().CenterOn(26*32, 32*32)
		secondMissionMove(t, app, id, 24, 32)
		for tick := 0; tick < 160; tick++ {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			dismissSecondMissionDialogue(t, app, live)
			bank, _ := live.world.ROM2ScenarioState()
			if bank[772] != 0 {
				break
			}
		}
	}
	bank, _ := live.world.ROM2ScenarioState()
	if (bank[772] != 0) != unlock {
		t.Fatalf("installed proximity branch772=%d want unlocked%v", bank[772], unlock)
	}
	if err := live.world.HeadlessPlace(id, 42, 15); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	live.view.Camera().CenterOn(42*32, 15*32)
	secondMissionMove(t, app, id, 42, 10)
	for tick := 0; tick < 240; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, live)
		if live.mission.open && live.mission.kind == ui.NoticeSuccess {
			break
		}
	}
	bank, _ = live.world.ROM2ScenarioState()
	won, _ := live.world.ScriptCounters()
	if live.world.Outcome() != sim.OutcomeWon || !live.mission.open || live.mission.kind != ui.NoticeSuccess || won == 0 || (bank[772] != 0) != unlock {
		t.Fatalf("installed M20 route: outcome=%v open=%v kind=%v bank772=%d wins=%d", live.world.Outcome(), live.mission.open, live.mission.kind, bank[772], won)
	}
	t.Logf("M20 installed pointer/tick victory tick%d bank772=%d; disclosed HeadlessPlace near optional proximity and final marker", live.world.Tick(), bank[772])
}

func secondLaterMissionProof(t *testing.T, f *FrontEnd, app *ui.App, cold *FrontEnd, a *ui.App, out string) {
	t.Helper()
	for _, app := range []*ui.App{app, a} {
		secondLaterChoose(t, app, "GATES", "mission 21", "CANCEL", "mission 21", "ENTER")
	}
	if f.liveMission != 21 || cold.liveMission != 21 {
		t.Fatal("available21 did not enter")
	}
	rep := secondScriptAdmissionReport(t, f)
	t.Logf("M21 retained-roster unresolved=%+v omittedTriggers=%v omittedChecks=%v omittedActions=%v unsupported=%v", rep.Unresolved, rep.OmittedTriggers, rep.OmittedChecks, rep.OmittedActions, f.live.world.Script().Unsupported())
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
	dismissSecondMissionDialogue(t, a, cold.live)
	first := secondSaveSampleNow(t, f, app)
	secondAssertSample(t, first, secondSaveSampleNow(t, cold, a))
	raw := secondMissionNamedSave(t, f, app, out, "Mission21")
	secondLaterLossControls(t, f, raw)
	loaded, b := secondMissionColdAt(t, out, "Mission21.sav", 21)
	secondAssertSample(t, first, secondSaveSampleNow(t, loaded, b))
	id := f.live.mission.ids[0]
	e, _ := f.live.world.Entity(id)
	x, y := int(e.X)-1, int(e.Y)
	secondMissionMove(t, app, id, x, y)
	secondMissionMove(t, b, id, x, y)
	for range 8 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := b.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		assertCurrentWorldEqual(t, f.live.world, loaded.live.world, "M21 next ordinary tick")
	}
	last := secondSaveSampleNow(t, f, app)
	secondAssertSample(t, last, secondSaveSampleNow(t, loaded, b))
	if bytes.Equal(first.World, last.World) {
		t.Fatal("M21 next action changed no World")
	}
	secondMissionNamedSave(t, loaded, b, out, "Continued21")
	again, c := secondMissionColdAt(t, out, "Continued21.sav", 21)
	secondAssertSample(t, last, secondSaveSampleNow(t, again, c))
	secondLaterEmitProof(t, filepath.Join(out, "Mission21.sav"), secondLaterProof{Mission: []secondSaveSample{first, last}, Actor: id, X: x, Y: y, Ticks: 8})
}

func TestReleaseSecondTownTwoContinuation(t *testing.T) {
	secondGameRoot(t)
	if path := os.Getenv("AGAINROM_SECOND_LATER_INPUT"); path != "" {
		secondLaterColdProof(t, path)
		return
	}
	for _, enriched := range []bool{false, true} {
		for _, unlock := range []bool{false, true} {
			for _, continued := range []bool{false, true} {
				if !enriched && continued {
					continue
				}
				roster := "ordinary"
				if enriched {
					roster = "changed"
				}
				t.Run(fmt.Sprintf("roster=%s/unlock=%v/continued=%v", roster, unlock, continued), func(t *testing.T) {
					out := secondMissionSaveDirectory(t)
					f := secondGameFront(t)
					app := f.App("later continuation")
					app.Layout(1024, 768)
					secondLaterChoose(t, app, "new game")
					if enriched {
						secondTownMutate(f)
						companion := mapload.CloneParty(f.Carried)[0]
						companion.ID, companion.Name = "retained", "Retained companion"
						companion.StartingHero = false
						companion.Hero.Body++
						f.Carried = append(f.Carried, companion)
					}
					secondLaterChoose(t, app, "TAVERN", "TALK 517")
					for page := 0; page < 64; page++ {
						if app.HeadlessActivate("notice") != nil {
							break
						}
					}
					secondLaterChoose(t, app, "GATES", "mission 10", "ENTER")
					enterSecondCampaignNextMission(t, f, app, false)
					if enriched {
						f.live.world.SetPurse(sim.SelfSlot, 913)
					}
					rep := secondScriptAdmissionReport(t, f)
					t.Logf("M20 retained-roster unresolved=%+v omittedTriggers=%v omittedChecks=%v omittedActions=%v unsupported=%v", rep.Unresolved, rep.OmittedTriggers, rep.OmittedChecks, rep.OmittedActions, f.live.world.Script().Unsupported())
					secondTwentyWin(t, f, app, unlock)
					live := f.live
					carried := mapload.CarryRoster(live.mission.party, live.world, live.mission.ids, live.mission.state.Start.Roster)
					if len(carried) < 1 || enriched && (len(carried) < 2 || len(carried[0].CarriedItems) == 0) {
						t.Fatal("changed roster/inventory absent from live witness")
					}
					purse := int(live.world.Purse(sim.SelfSlot))
					if continued {
						secondLaterChoose(t, app, "continue")
						if f.Town.second.bank[916] != 0 || !live.mission.delayedVictory {
							t.Fatal("Continue acknowledged M20")
						}
						if err := app.HeadlessKey("escape"); err != nil {
							t.Fatal(err)
						}
						for _, target := range []string{"end", "victory"} {
							if err := app.HeadlessGameMenuAction(target); err != nil {
								t.Fatal(err)
							}
						}
					} else {
						secondLaterChoose(t, app, "notice")
					}
					c := f.Town.second
					want := []secondLocation{{2, 2}}
					if unlock {
						want = append(want, secondLocation{1, 21})
					}
					if app.Screen() != ui.ScreenTown || f.live != nil || c.current != (secondLocation{}) || !reflect.DeepEqual(c.available, want) || c.bank[916] != 1 || c.bank[768] != 30 || c.bank[532] != 1 || c.bank[552] != 2 || f.Town.gold != purse {
						t.Fatalf("M20 destination/carry: screen=%v current=%v available=%v stage=%d completed=%d purse=%d", app.Screen(), c.current, c.available, c.bank[768], c.bank[916], f.Town.gold)
					}
					if len(f.Carried) != len(carried) {
						t.Fatal("completion replaced live roster")
					}
					for i, p := range carried {
						got := f.Carried[i]
						if got.ID != p.ID || got.Name != p.Name || got.Hero != p.Hero || got.Worn != p.Worn || !reflect.DeepEqual(got.WornItems, p.WornItems) || !reflect.DeepEqual(got.CarriedItems, p.CarriedItems) {
							t.Fatalf("retained member%d changed identity/base/equipment/inventory", i)
						}
						if i < len(live.mission.ids) {
							id := live.mission.ids[i]
							e, exists := live.world.Entity(id)
							items, _ := live.world.CarriedItems(id)
							equipped, _ := live.world.EquippedItems(id)
							for j := range items {
								items[j].ObjectID = 0
							}
							for j := range equipped {
								equipped[j].ObjectID = 0
							}
							if !exists || !e.Alive() || got.ID != live.mission.party[i].ID || got.Hero.Body != live.mission.party[i].Hero.Body || !reflect.DeepEqual(got.CarriedItems, items) || !reflect.DeepEqual(got.WornItems, equipped) {
								t.Fatal("return differs from independent live actor/holdings sample", id)
							}
						}
					}
					before := captureSecondCampaign(c)
					graph := secondLaterGraph(t, live, f)
					spells := secondLaterSpellValues(t, graph, f.Carried)
					if app.HeadlessActivate("notice") == nil || !reflect.DeepEqual(before, captureSecondCampaign(c)) {
						t.Fatal("duplicate acknowledgement advanced again")
					}
					secondLaterChoose(t, app, "town 2", "CANCEL", "town 2", "ENTER")
					if app.HeadlessActivate("TALK 517") == nil {
						t.Fatal("town2 reused the first-town talk")
					}
					sample := secondTownSampleNow(t, f, app)
					frame, _, err := app.HeadlessFrame()
					if err != nil {
						t.Fatal(err)
					}
					var encoded bytes.Buffer
					if err := png.Encode(&encoded, frame); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(out, "TownBefore.png"), encoded.Bytes(), 0600); err != nil {
						t.Fatal(err)
					}
					raw := secondTownNamedSave(t, f, app, out, "Second town")
					secondTownShape(t, raw)
					secondLaterLossControls(t, f, raw)
					cold, a := secondTownCold(t, out, "Second town.sav")
					secondTownAssertSample(t, sample, secondTownSampleNow(t, cold, a))
					secondLaterAssertGraph(t, graph, cold.Town.cityObjects, spells)
					secondLaterTownAction(t, a)
					secondTownAssertSample(t, sample, secondTownSampleNow(t, cold, a))
					secondTownNamedSave(t, cold, a, out, "Continued town")
					again, b := secondTownCold(t, out, "Continued town.sav")
					secondTownAssertSample(t, sample, secondTownSampleNow(t, again, b))
					secondLaterAssertGraph(t, cold.Town.cityObjects, again.Town.cityObjects, nil)
					secondLaterEmitProof(t, filepath.Join(out, "Second town.sav"), secondLaterProof{Town: &sample, TownGraph: graph, TownSpells: spells})
					if unlock {
						secondLaterMissionProof(t, f, app, cold, a, out)
					}
					t.Logf("retained %d actual live members/items/equipment/purse -> town2 SAV/fresh LOAD/action/resave; available21=%v", len(carried), unlock)
				})
			}
		}
	}
}

func TestReleaseSecondMissionTwentyFailureDoesNotLeave(t *testing.T) {
	f, app := secondMissionSettledAt(t, 20)
	live := f.live
	before := captureSecondCampaign(f.Town.second)
	party := mapload.CloneParty(f.Carried)
	gold, offered := f.Town.gold, f.Offered
	if err := live.world.HeadlessKill(live.mission.ids[0]); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 300 && !live.mission.open; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !live.mission.open || live.mission.kind != ui.NoticeFailure {
		t.Fatal("ordinary M20 hero death did not show failure")
	}
	secondLaterChoose(t, app, "exit to main menu")
	if app.Screen() != ui.ScreenMenu || !reflect.DeepEqual(before, captureSecondCampaign(f.Town.second)) || !reflect.DeepEqual(party, f.Carried) || f.Town.gold != gold || f.Offered != offered {
		t.Fatal("M20 loss acknowledged departure or carried failed state")
	}
}
