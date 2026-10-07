package game

import (
	"maps"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

func configuredGateColdTown(t *testing.T, store SaveStore, c Campaign, overlay string) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	f.Campaign = resolved(c, nil)
	f.Options = OptionsStore{}
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
	useOfferTextOverlay(t, f, overlay)
	a := f.App("Configured main cold SAV")
	a.Layout(640, 480)
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	for _, target := range []string{"load game", "@first"} {
		if err := a.HeadlessActivate(target); err != nil {
			t.Fatal("actual cold LOAD SAV", target, err, a.HeadlessMessage())
		}
	}
	s := f.TownScreen().(*townScreen)
	s.CloseTip()
	t.Cleanup(a.StopAudio)
	if a.Screen() != ui.ScreenTown || !s.AtTownSquare() {
		t.Fatal("cold LOAD did not reach square", a.Screen(), s.room)
	}
	return f, a, s
}

func TestReleaseConfiguredGateMainSAVContinuation(t *testing.T) {
	for _, building := range []TownBuilding{TownShop, TownTavern} {
		for _, coldBefore := range []bool{false, true} {
			label := map[TownBuilding]string{TownShop: "shop", TownTavern: "tavern"}[building] + map[bool]string{false: "-fresh", true: "-restored"}[coldBefore]
			t.Run(label, func(t *testing.T) {
				f := releaseFront(t)
				c := f.Campaign.Value()
				c.Chapters = maps.Clone(c.Chapters)
				ch := c.Chapters[30]
				ch.Shop, ch.Inn, ch.InnNPC, ch.School = nil, nil, nil, []int{31}
				if building == TownShop {
					ch.Shop = []int{40}
				} else {
					ch.Inn, ch.InnNPC = []int{40}, []int{22}
				}
				c.Chapters[30] = ch
				f.Campaign, f.Town = resolved(c, nil), NewTown(c)
				f.Options = OptionsStore{}
				f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
				f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Configured main", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
				for _, mission := range []int{10, 20} {
					if _, ok := f.Town.Won(mission); !ok {
						t.Fatal("controlled prologue", mission)
					}
				}
				f.arriveInTown()
				overlay := offerTextOverlay(t, f, []synth.File{
					{Path: "text/shop/npc31m40.txt", Data: []byte("<part=1 npc=31>\r\nConfigured shop main\r\n")},
					{Path: "text/inn/npc/npc22m40.txt", Data: []byte("<part=1 npc=22>\r\nConfigured tavern main\r\n")},
				})
				a, s := exteriorApp(t, f)
				t.Cleanup(a.StopAudio)
				unaccepted, before := gateRetentionSave(t, f, a, "configured-"+label+"-unaccepted")
				if before.Main.Mission != 30 || before.Main.Announced || len(f.Town.Available()) != 0 {
					t.Fatal("unaccepted future main has a latch", before)
				}
				g, b, v := configuredGateColdTown(t, unaccepted, c, overlay)
				if onMap, line := gateRetentionGateResult(t, g, b, v); onMap || !line {
					t.Fatal("unaccepted cold SAV did not open npc35", onMap, line)
				}
				gateRetentionBackToSquare(t, b, v)
				if coldBefore {
					f, a, s = g, b, v
				} else if err := a.HeadlessGameMenuAction("return"); err != nil || a.Screen() != ui.ScreenTown || !s.AtTownSquare() {
					t.Fatal("return to fresh town after F2", err, a.Screen(), s.room)
				}
				gold := f.Town.Gold()
				if building == TownShop {
					if err := a.HeadlessActivate("SHOP"); err != nil {
						t.Fatal("App configured shop", err)
					}
				} else {
					roomExitEnter(t, a, s, "TAVERN", roomTavern)
					if err := a.HeadlessActivate("NPC 22"); err != nil {
						t.Fatal("App configured tavern conversation", err)
					}
				}
				for n := 0; s.room == roomTalk && n < 8; n++ {
					if err := a.HeadlessKey("escape"); err != nil {
						t.Fatal(err)
					}
				}
				if building == TownTavern && f.Town.currentMain() != 30 {
					t.Fatal("hearing loaded or accepted future main before leave")
				}
				if err := a.HeadlessKey("escape"); err != nil || !s.AtTownSquare() {
					t.Fatal("App configured building leave", err, s.room)
				}
				if f.Town.currentMain() != 40 || f.Town.gateMission() != 40 || f.Town.Gold() != gold || !f.Town.taken[offerRef{30, building, 0}] {
					t.Fatal("App handover lost record/latch/provenance or paid reward", f.Town.currentMain(), f.Town.Available(), f.Town.taken, f.Town.Gold())
				}
				if onMap, line := gateRetentionGateResult(t, f, a, s); !onMap || line {
					t.Fatal("accepted configured main cannot pass actual App gate", onMap, line)
				}
				gateRetentionBackToSquare(t, a, s)
				acceptedTaken := maps.Clone(f.Town.taken)
				accepted, emitted := gateRetentionSave(t, f, a, "configured-"+label+"-accepted")
				if emitted.Main.Mission != 40 || !emitted.Main.Announced || emitted.SelectedMission != 40 {
					t.Fatal("current F2 SAV lost accepted main", emitted)
				}
				h, d, w := configuredGateColdTown(t, accepted, c, overlay)
				if h.Town.Gold() != gold || h.Town.currentMain() != 40 || h.Town.progress.record(31).age != 1 {
					t.Fatal("cold SAV lost gold/main/retained age", h.Town.Gold(), h.Town.currentMain(), h.Town.progress)
				}
				if !maps.Equal(acceptedTaken, h.Town.taken) || !h.Town.taken[offerRef{30, building, 0}] {
					t.Fatalf("cold SAV lost consumption provenance: accepted=%+v restored=%+v", acceptedTaken, h.Town.taken)
				}
				projection := h.Town.progress.projection()
				taken, beforeGold := maps.Clone(h.Town.taken), h.Town.Gold()
				registered := h.Town.registerOfferedMission(40)
				after := h.Town.progress.projection()
				if !registered || !reflect.DeepEqual(projection, after) || h.Town.Gold() != beforeGold || !maps.Equal(taken, h.Town.taken) {
					t.Fatalf("cold repeated registration changed state: registered=%v projectionBefore=%+v projectionAfter=%+v takenBefore=%+v takenAfter=%+v goldBefore=%d goldAfter=%d", registered, projection, after, taken, h.Town.taken, beforeGold, h.Town.Gold())
				}
				if onMap, line := gateRetentionGateResult(t, h, d, w); !onMap || line {
					t.Fatal("next actual App gate after cold accepted SAV", onMap, line)
				}
				gateRetentionBackToSquare(t, d, w)
				if _, ok := h.Town.Won(40); !ok || h.Town.Gold() != gold+c.Reward(40) || h.Town.progress.record(31) != nil {
					t.Fatal("post-LOAD completion reward or age2 expiry")
				}
				paid := h.Town.Gold()
				if _, ok := h.Town.Won(40); ok || h.Town.Gold() != paid {
					t.Fatal("post-LOAD completion paid twice")
				}
				t.Logf("%s: installed root=%s; synthetic configured chapter/text; actual App %v handover; two F2 SAVs and cold menu LOADs; next gate main40; no duplicate consumption/reward; retained31 age1 then controlled Won40 expires it", label, f.Archives.Root, building)
			})
		}
	}
}
