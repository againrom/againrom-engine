package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func campaignFlowRows(t *testing.T, source string) mod.MissionData {
	t.Helper()
	d, err := mod.ParseMissions("flow", mod.MissionsFile, []byte(source), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	for i := range d.Missions {
		if d.Missions[i].Map != "" {
			d.Missions[i].MapBytes = synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "detour"})
		}
	}
	return d
}

func campaignFlowRow(number int, fields string) string {
	return fmt.Sprintf("[[mission]]\nkind = \"campaign\"\nkey = \"row%d\"\nnumber = %d\n%s\n", number, number, fields)
}

func campaignFlowFront(t *testing.T, d mod.MissionData) *FrontEnd {
	t.Helper()
	f := winRouteFirstFront(t, mod.MissionData{})
	f.Archives.Containers = missionArchive(t, 10, 20, 30, 31, 40, 50)
	f.Tiles = &terrain.Tileset{}
	c := f.Campaign.Value()
	c.MapObjects = map[int]int{10: 0, 20: 1, 30: 1, 40: 2, 50: 2}
	c.Main, c.Offered = append(c.Main, 50), append(c.Offered, 50)
	c.Last = map[int]bool{50: true}
	c.Chapters[40] = Chapter{Mission: 40, TextDocuments: []int{40}}
	f.Campaign, f.Town = resolved(c, nil), NewTown(c)
	f.Table.Mods.Set = mod.Set{Base: "rom1-en", Mods: []mod.SetEntry{{ID: "flow", Version: "1", Digest: "synthetic"}}}
	if _, err := f.SetModMissions(d); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestModMissionFlowRejectsPastGuestTown(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		line               int
	}{
		{"campaign", campaignFlowRow(30, "guest = 22\nchapter = 30"), "town of chapter 40, not chapter 30", 1},
		{"automatic", campaignFlowRow(20, "AutoGetMission = 30\nguest = 22\nchapter = 30"), "town of chapter 40, not chapter 30", 1},
		{"added", campaignFlowRow(10, "AutoGetMission = 11") + campaignFlowRow(11, "map = \"guest.alm\"\nguest = 22\nchapter = 30"), "town of chapter 20, not chapter 30", 6},
		{"replace", strings.Replace(campaignFlowRow(30, "map = \"guest.alm\"\nguest = 22\nchapter = 30"), "campaign", "replace", 1), "town of chapter 40, not chapter 30", 1},
		{"after", "[[mission]]\nkey = \"guest\"\nnumber = 25\nmap = \"guest.alm\"\nafter = 20\nguest = 22\nchapter = 40\n", "town of chapter 30, not chapter 40", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := campaignFlowFront(t, mod.MissionData{})
			_, err := f.SetModMissions(campaignFlowRows(t, tc.source))
			if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("data/missions.toml:%d:", tc.line)) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("load = %v, want file, line and %q", err, tc.want)
			}
		})
	}
}

func TestModMissionFlowTownMatchesItsSAV(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		missions     []int
		chapter      int
	}{
		{"later main and grant", campaignFlowRow(10, "AutoGetMission = 20"), []int{10, 20}, 30},
		{"later main", campaignFlowRow(20, "AutoGetMission = 30"), []int{10, 20, 30}, 40},
		{"added", campaignFlowRow(20, "AutoGetMission = 21") + campaignFlowRow(21, "map = \"extra.alm\""), []int{10, 20, 21}, 30},
		{"added chain", campaignFlowRow(20, "AutoGetMission = 21") + campaignFlowRow(21, "map = \"extra.alm\"\nAutoGetMission = 22") + campaignFlowRow(22, "map = \"extra.alm\""), []int{10, 20, 21, 22}, 30},
		{"removed", campaignFlowRow(20, "AutoGetMission = -1"), []int{10, 20}, 30},
		{"early town", campaignFlowRow(10, "AutoGetMission = -1"), []int{10}, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := campaignFlowRows(t, tc.source)
			f := campaignFlowFront(t, d)
			for i, n := range tc.missions {
				openMissionFlow(t, f, n)
				next := 0
				if i+1 < len(tc.missions) {
					next = tc.missions[i+1]
				}
				winMissionFlow(t, f, next)
			}
			if !f.Town.Open() || f.Town.Chapter() != tc.chapter || f.Offered != tc.chapter {
				t.Fatalf("live town: open %t chapter %d offered %d, want %d", f.Town.Open(), f.Town.Chapter(), f.Offered, tc.chapter)
			}
			wantParty := winRouteParty(f.Carried)
			wantGrants := maps.Clone(f.Town.heroGrants)
			wantDocuments := append([]Document(nil), f.Town.documents...)
			if tc.chapter == 30 {
				if _, ok := guestMember(f); !ok {
					t.Fatal("the reached chapter did not hand out its companion")
				}
			}
			raw := currentTownSave(t, f)
			g := campaignFlowFront(t, d)
			if _, town, err := g.RestoreOriginal(raw); err != nil || !town {
				t.Fatalf("town LOAD: %t %v", town, err)
			}
			if g.Town.Chapter() != tc.chapter || g.Offered != tc.chapter || winRouteParty(g.Carried) != wantParty ||
				!maps.Equal(g.Town.heroGrants, wantGrants) || !reflect.DeepEqual(g.Town.documents, wantDocuments) {
				t.Fatalf("town LOAD changed chapter, offer, hand-outs or party: %d/%d %s -> %d/%d %s", tc.chapter, f.Offered, wantParty, g.Town.Chapter(), g.Offered, winRouteParty(g.Carried))
			}
			if !bytes.Equal(raw, currentTownSave(t, g)) {
				t.Fatal("town LOAD changed the next SAV")
			}
		})
	}
}

func TestModMissionFlowWaitingPartyKeepsEveryField(t *testing.T) {
	for _, tc := range []struct {
		name, fields string
		number       int
		before       []int
		reload       bool
	}{
		{"hero before town", "guest = 22\nchapter = 30", 20, []int{10}, false},
		{"party after town", "party = [{ character = \"npc22\", level = 1 }]", 40, []int{10, 20, 30}, false},
		{"loaded hero before town", "guest = 22\nchapter = 30", 20, []int{10}, true},
		{"loaded party after town", "party = [{ character = \"npc22\", level = 1 }]", 40, []int{10, 20, 30}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := campaignFlowRows(t, campaignFlowRow(tc.number, tc.fields))
			f := campaignFlowFront(t, d)
			control := campaignFlowFront(t, mod.MissionData{})
			for _, n := range tc.before {
				openMissionFlow(t, f, n)
				openMissionFlow(t, control, n)
				next := 0
				if n == 10 {
					next = 20
				}
				winMissionFlow(t, f, next)
				winMissionFlow(t, control, next)
			}
			for i := range f.Carried {
				if f.Carried[i].Kept == nil || control.Carried[i].Kept == nil {
					t.Fatal("the preceding mission left no kept actor")
				}
				f.Carried[i].Kept.Deadline = 623
				control.Carried[i].Kept.Deadline = 623
			}
			waiting := mapload.CloneParty(f.Carried)
			if waiting[0].Kept == nil {
				t.Fatal("the waiting-state control holds no Kept actor")
			}
			openMissionFlow(t, f, tc.number)
			stash := f.Town.guest.stash
			restored := campaignFlowFront(t, d)
			if _, town, err := restored.RestoreOriginal(stash); err != nil || !town {
				t.Fatalf("waiting SAV LOAD: %t %v", town, err)
			}
			if !reflect.DeepEqual(restored.Carried[0].Kept, waiting[0].Kept) {
				t.Fatal("capture or LOAD lost Kept before campaign completion")
			}
			// Before the town exists, the control starts from the same waiting
			// SAV and wins the mission with the main party.
			if tc.number == 20 {
				if _, town, err := control.RestoreOriginal(stash); err != nil || !town {
					t.Fatalf("main-party control LOAD: %t %v", town, err)
				}
			}
			if tc.reload {
				raw := guestMissionSave(t, f)
				loaded := campaignFlowFront(t, d)
				open, town, err := loaded.RestoreOriginal(raw)
				if err != nil || town {
					t.Fatalf("guest LOAD: %t %v", town, err)
				}
				if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
					t.Fatal(err)
				}
				f = loaded
			}
			winMissionFlow(t, f, 0)
			for _, member := range waiting {
				found := false
				for _, after := range f.Carried {
					if after.ID == member.ID {
						found = true
						if winRouteParty([]mapload.PartyMember{member}) != winRouteParty([]mapload.PartyMember{after}) {
							t.Fatalf("waiting member %s changed: Kept %+v -> %+v", member.ID, member.Kept, after.Kept)
						}
					}
				}
				if !found {
					t.Fatalf("waiting member %s is missing", member.ID)
				}
			}
			openMissionFlow(t, control, tc.number)
			winMissionFlow(t, control, 0)
			if tc.number == 20 {
				for i := range control.Carried {
					if control.Carried[i].CompanionNPC == 22 {
						member, _ := guestMember(f)
						control.Carried[i] = mapload.CloneParty([]mapload.PartyMember{member})[0]
					}
				}
			}
			if a, b := currentTownSave(t, f), currentTownSave(t, control); !bytes.Equal(a, b) {
				da, db := engineSaveDocument(t, a), engineSaveDocument(t, b)
				t.Logf("head %+v / %+v; party %s / %s", da.Head, db.Head, winRouteParty(f.Carried), winRouteParty(control.Carried))
				ra, rb := reflect.ValueOf(da), reflect.ValueOf(db)
				for i := 0; i < ra.NumField(); i++ {
					if !reflect.DeepEqual(ra.Field(i).Interface(), rb.Field(i).Interface()) {
						t.Logf("different document field %s", ra.Type().Field(i).Name)
					}
				}
				campaignFlowValueDiff(t, "campaign", autoGetCampaign(t, a), autoGetCampaign(t, b))
				aa, _ := readCurrentActions(&da)
				ab, _ := readCurrentActions(&db)
				campaignFlowValueDiff(t, "actions", aa, ab)
				t.Fatal("guest completion and main-party completion write different town SAVs")
			}
			raw := currentTownSave(t, f)
			cold := campaignFlowFront(t, d)
			if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
				t.Fatalf("final town LOAD: %t %v", town, err)
			}
			if winRouteParty(cold.Carried) != winRouteParty(f.Carried) {
				t.Fatal("town LOAD changes the waiting party")
			}
			openMissionFlow(t, cold, cold.Town.currentMain())
			actor, _ := cold.live.world.Entity(cold.live.mission.ids[0])
			if !actor.ActionClock.Known || actor.ActionClock.End != 623 {
				t.Fatalf("the next mission lost the regeneration clock: %+v", actor.ActionClock)
			}
		})
	}
}

func TestModMissionFlowGuestPartyKeepsTownBuiltMembers(t *testing.T) {
	d := campaignFlowRows(t, campaignFlowRow(30, "party = [{ character = \"npc22\" }]"))
	f := campaignFlowFront(t, d)
	openMissionFlow(t, f, 10)
	winMissionFlow(t, f, 20)
	openMissionFlow(t, f, 20)
	winMissionFlow(t, f, 0)
	if len(f.Carried) != 2 || f.Carried[0].Kept == nil || f.Carried[1].Kept != nil {
		t.Fatal("the fixture must hold one kept and one town-built member")
	}
	before := winRouteParty(f.Carried)
	openMissionFlow(t, f, 30)
	winMissionFlow(t, f, 0)
	if winRouteParty(f.Carried) != before {
		t.Fatal("guest completion changed a field of the waiting party")
	}
	raw := currentTownSave(t, f)
	cold := campaignFlowFront(t, d)
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("town LOAD: %t %v", town, err)
	}
	if winRouteParty(cold.Carried) != before || !bytes.Equal(raw, currentTownSave(t, cold)) {
		t.Fatal("town LOAD changed the waiting members or their SAV")
	}
}

func campaignFlowValueDiff(t *testing.T, name string, a, b any) {
	t.Helper()
	x, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	y, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	i := 0
	for i < len(x) && i < len(y) && x[i] == y[i] {
		i++
	}
	lo := max(0, i-70)
	t.Logf("%s first difference at %d:\n%s\n%s", name, i, x[lo:min(len(x), i+140)], y[lo:min(len(y), i+140)])
}

func TestModMissionFlowDocumentationLoadsFromFolder(t *testing.T) {
	doc, err := os.ReadFile("../../docs/MODS.md")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(doc), "For example, the main party plays 10,")
	if start < 0 {
		t.Fatal("worked example heading missing")
	}
	block := strings.SplitN(string(doc[start:]), "```toml\n", 2)
	if len(block) != 2 {
		t.Fatal("worked example TOML missing")
	}
	source := strings.SplitN(block[1], "```", 2)[0]
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			mapBytes := synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "worked example"})
			dir := fileFormMod(t, map[string][]byte{
				"main.star":          []byte("def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n"),
				"data/missions.toml": []byte(source), "maps/detour.alm": mapBytes,
				"maps/detour.text/en/event01.txt": []byte("English detour"),
				"maps/detour.text/ru/event01.txt": []byte("Russian detour"),
			})
			entries, err := mod.Resolve(filepath.Dir(dir), []string{"file"})
			if err != nil {
				t.Fatal(err)
			}
			res, err := modrt.Load(entries, "rom1-"+lang, nil, modrt.Options{})
			if err != nil {
				t.Fatal(err)
			}
			wantText := "English detour"
			if lang == "ru" {
				wantText = "Russian detour"
			}
			m := res.Missions.Missions[1]
			if !bytes.Equal(m.MapBytes, mapBytes) || string(m.TextFiles["event01.txt"]) != wantText {
				t.Fatalf("ordinary mod load lost map or %s texts", lang)
			}
			f := campaignFlowFront(t, res.Missions)
			f.Table.Mods.Set = res.Set
			page, err := f.Archives.Containers.ReadFile("main/text/battle/m11/event01.txt")
			if err != nil || string(page) != wantText {
				t.Fatalf("mission page = %q, %v", page, err)
			}
			openMissionFlow(t, f, 10)
			winMissionFlow(t, f, 11)
			openMissionFlow(t, f, 11)
			winMissionFlow(t, f, 20)
			openMissionFlow(t, f, 20)
			winMissionFlow(t, f, 0)
			if f.Town.Gold() != 325 || f.Town.Chapter() != 30 {
				t.Fatalf("worked example ends with gold %d chapter %d", f.Town.Gold(), f.Town.Chapter())
			}
		})
	}
}

func TestModMissionFlowAddedMissionLossAndLeave(t *testing.T) {
	for _, outcome := range []string{"loss", "leave before town", "leave after town", "restart after town"} {
		t.Run(outcome, func(t *testing.T) {
			d := missionFlowData(t, false)
			if outcome == "loss" {
				payload := goalScript([]scriptNode{{label: "lose", opcode: 5, id: 2}},
					[]scriptNode{{label: "zero", opcode: 0x10002, id: 1, values: []uint32{0}, tags: []uint32{7}}},
					[]goalTrigger{{name: "failure", left: 1, right: 1, actions: []uint32{2}}})
				d.Missions[1].MapBytes = synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "loss", Type7Payload: payload})
			}
			f := campaignFlowFront(t, d)
			if strings.HasSuffix(outcome, "after town") {
				c := f.Campaign.Value()
				c.Offered = append([]int{10}, c.Offered...)
				f.Campaign, f.Town = resolved(c, nil), NewTown(c)
				f.arriveInTown()
			}
			openMissionFlow(t, f, 10)
			winMissionFlow(t, f, 11)
			beforeParty, beforeGold := winRouteParty(f.Carried), f.Town.Gold()
			openMissionFlow(t, f, 11)
			ms, mw := f.live.mission.state, f.live
			var dest ui.NoticeDest
			var open ui.MapOpener
			if outcome == "loss" {
				for i := 0; i < 128 && mw.world.Outcome() == sim.OutcomeUndecided; i++ {
					sim.Step(mw.world, nil)
				}
				mw.settleNotices(messageStep)
				if mw.world.Outcome() != sim.OutcomeLost {
					t.Fatal("synthetic mission did not lose")
				}
				dest, _, open = f.continuity(11, ms, mw.advanceNotice)(ui.NoticeExitMain)
				if dest != ui.NoticeToMenu {
					t.Fatalf("loss destination %v, want main menu", dest)
				}
			} else {
				action := ui.NoticeAbandon
				want := ui.NoticeStay
				if strings.HasSuffix(outcome, "after town") {
					want = ui.NoticeToTown
				}
				if strings.HasPrefix(outcome, "restart") {
					action, want = ui.NoticeRestart, ui.NoticeToMission
				}
				dest, _, open = f.continuity(11, ms, mw.advanceNotice)(action)
				if dest != want || (open != nil) != (action == ui.NoticeRestart) {
					t.Fatalf("leave destination %v opener %t, want %v", dest, open != nil, want)
				}
			}
			if f.Town.Done(11) || winRouteParty(f.Carried) != beforeParty || f.Town.Gold() != beforeGold {
				t.Fatal("loss or leave completed, paid or changed the waiting campaign")
			}
			if open != nil {
				if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
					t.Fatal(err)
				}
				if f.liveMission != 11 {
					t.Fatal("restart opened another mission")
				}
			}
		})
	}
}
