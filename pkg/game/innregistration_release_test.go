package game

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/render/frame"
	"againrom/pkg/ui"
)

func hearInnOfferIndex(t *testing.T, app *ui.App, s *townScreen, mission, index int) {
	t.Helper()
	cell := -1
	for i, candidate := range s.tavernCandidates() {
		if candidate.offer.Index == index && candidate.offer.Mission == mission {
			cell = i
			break
		}
	}
	if cell < 0 {
		t.Fatalf("no mission %d at stable index %d", mission, index)
	}
	surface := s.TownSurface()
	point, found := image.Point{}, false
	for y := 0; y < 480 && !found; y++ {
		for x := 0; x < 640; x++ {
			if c, ok := ui.TownSurfaceControlAt(surface, image.Pt(x, y)); ok && c.Kind == ui.TownSurfaceControlCell && c.Index == cell {
				point, found = image.Pt(x, y), true
				break
			}
		}
	}
	if !found {
		t.Fatal("inn cell has no pointer target", cell)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, point.X, point.Y); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomTalk {
		r := ui.TownSurfaceButtonRect(surface.Kind, tavernButtonTalk)
		p := r.Min.Add(r.Max).Div(2)
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	if s.room != roomTalk {
		t.Fatal("installed inn text opened no dialogue", mission, index, s.offer)
	}
	for n := 0; s.room == roomTalk && n < 64; n++ {
		page := s.said
		button := dialogueButtonPoint(t, s)
		for _, edge := range []struct {
			action string
			point  image.Point
		}{
			{"release", button}, {"press", button}, {"release", image.Pt(0, 0)}, {"release", button},
		} {
			if err := app.HeadlessPointer(edge.action, edge.point.X, edge.point.Y); err != nil {
				t.Fatal(err)
			}
			if s.room != roomTalk || s.said != page {
				t.Fatalf("%s advanced installed town dialogue before an owned release", edge.action)
			}
		}
		dialogueKeyPress(t, app, s, "click")
	}
	if s.room != roomTavern {
		t.Fatal("installed inn dialogue did not return to tavern")
	}
}

func saveInnApp(t *testing.T, f *FrontEnd, app *ui.App) []byte {
	t.Helper()
	dir := t.TempDir()
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(dir, "Inn current", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "Inn current.sav"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func coldInnApp(t *testing.T, raw []byte) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app, s := releaseTavernApp(t, f, store)
	return f, app, s
}

func TestReleaseInnRegistrationSAVAndNextAction(t *testing.T) {
	for _, main := range []int{30, 50} {
		for _, hearings := range []int{1, 2} {
			t.Run(fmt.Sprintf("main=%d/hearings=%d", main, hearings), func(t *testing.T) {
				f := currentTown(t, nil, nil)
				c := f.Campaign.Value()
				ch := c.Chapters[main]
				npc := 0
				for i, mission := range ch.Inn {
					if mission == main && i < len(ch.InnNPC) {
						npc = ch.InnNPC[i]
						break
					}
				}
				if npc == 0 {
					t.Fatal("chapter lacks main inn speaker", main, ch.Inn, ch.InnNPC)
				}
				ch.Inn, ch.InnNPC = []int{main, main + 1, main, 0}, []int{npc, npc, npc, 90}
				c.Chapters[main] = ch
				f.Campaign = resolved(c, nil)
				gold := f.Town.Gold()
				f.Town = NewTown(c)
				f.Town.main = main
				f.Town.records = newCampaignRecords(c, main)
				f.Town.Arrive()
				f.Town.gold = gold
				f.Town.takeAddHeroes(main)
				f.Town.selectMission(main + 1)
				initial := currentTownSave(t, f)
				if main == 50 && hearings == 2 {
					d, err := sav.DecodeDocumentData(initial)
					if err != nil {
						t.Fatal(err)
					}
					picture, ok := nativeCityMarkerPicture(f, main)
					if !ok {
						t.Fatal("missing marker picture control")
					}
					d.Campaign.Markers = []sav.CityCampaignMarkerData{{Value: uint32(main), Text: append([]byte(picture), 0), Tail: [8]byte{17, 0, 0, 0, 29, 0, 0, 0}}}
					initial, err = sav.EncodeDocumentData(d)
					if err != nil {
						t.Fatal(err)
					}
				}
				f, app, s := coldInnApp(t, initial)
				for n := 0; n < hearings; n++ {
					hearInnOfferIndex(t, app, s, main, 2)
				}
				if len(s.innQueue) != hearings || f.Town.selectedMission() != main+1 {
					t.Fatal("hearing registered early or lost a duplicate", s.innQueue, f.Town.selectedMission())
				}
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				want := []TownOffer{{Index: 1, Mission: main + 1, NPC: npc}, {Index: 3, NPC: 90}}
				if hearings == 1 {
					want = []TownOffer{{Index: 1, Mission: main + 1, NPC: npc}, {Index: 2, Mission: main, NPC: npc}, {Index: 3, NPC: 90}}
				}
				if got := f.Town.Offers(TownTavern); !reflect.DeepEqual(got, want) || f.Town.selectedMission() != main || !f.Town.progress.main.announced || f.Town.Gold() != gold {
					t.Fatal("inn registration state", got, want, f.Town.selectedMission(), f.Town.Gold())
				}
				raw := saveInnApp(t, f, app)
				projection := autoGetCampaign(t, raw)
				wantMissions, wantNPCs := []uint16{uint16(main + 1), 0}, []uint16{uint16(npc), 90}
				if hearings == 1 {
					wantMissions, wantNPCs = []uint16{uint16(main + 1), uint16(main), 0}, []uint16{uint16(npc), uint16(npc), 90}
				}
				if !reflect.DeepEqual(projection.InnMission, wantMissions) || !reflect.DeepEqual(projection.InnNPC, wantNPCs) || projection.SelectedMission != uint32(main) || !projection.Main.Announced {
					t.Fatal("SAVE lost paired offers, selection or announce", projection.InnMission, projection.InnNPC, projection.SelectedMission)
				}
				markerCount := 0
				for _, m := range projection.Markers {
					if m.Value == uint32(main) {
						markerCount++
						if main == 50 && hearings == 2 && (m.Field0 != 17 || m.Field1 != 29) {
							t.Fatal("registration replaced existing marker payload", m)
						}
					}
				}
				_, hasPicture := nativeCityMarkerPicture(f, main)
				if main == 50 && !hasPicture {
					t.Fatal("picture-bearing control changed")
				}
				if markerCount != 0 && !hasPicture || hasPicture && markerCount != 1 {
					t.Fatal("marker identity guard", markerCount, hasPicture)
				}
				g, gapp, gs := coldInnApp(t, raw)
				if !reflect.DeepEqual(g.Town.Offers(TownTavern), want) || g.Town.selectedMission() != main || !g.Town.progress.main.announced || g.Town.Gold() != gold {
					t.Fatal("cold LOAD lost registered state")
				}
				if !reflect.DeepEqual(g.Town.progress.projection().Markers, projection.Markers) {
					t.Fatal("cold LOAD lost marker payload")
				}
				if _, ok := g.Town.Take(TownTavern, 0); ok {
					t.Fatal("direct Take accepted removed stable index 0")
				}
				if hearings == 1 {
					hearInnOfferIndex(t, gapp, gs, main, 2)
					if err := gapp.HeadlessKey("escape"); err != nil {
						t.Fatal(err)
					}
					if got := g.Town.Offers(TownTavern); !reflect.DeepEqual(got, []TownOffer{{Index: 1, Mission: main + 1, NPC: npc}, {Index: 3, NPC: 90}}) {
						t.Fatal("next city action did not remove remaining identity", got)
					}
				} else {
					keptIndex := 0
					found := false
					for _, candidate := range gs.tavernCandidates() {
						if candidate.offer.Mission == main && candidate.offer.Index < 0 {
							keptIndex, found = candidate.offer.Index, true
							break
						}
					}
					if !found {
						t.Fatal("kept speaker missing", main)
					}
					for n := 0; n < 2; n++ {
						hearInnOfferIndex(t, gapp, gs, main, keptIndex)
					}
					if len(gs.innQueue) != 0 || !reflect.DeepEqual(g.Town.Offers(TownTavern), want) {
						t.Fatal("kept speaker changed paired offers or queued registration")
					}
				}
				if g.Town.Gold() != gold {
					t.Fatal("next action changed gold")
				}
				losses := []string{"identity", "index", "selected"}
				if hasPicture {
					losses = append(losses, "marker")
				}
				for _, loss := range losses {
					mutant, err := sav.DecodeDocumentData(raw)
					if err != nil {
						t.Fatal(err)
					}
					switch loss {
					case "marker":
						mutant.Campaign.Markers = nil
					case "identity":
						mutant.Campaign.Arrays[2], mutant.Campaign.Arrays[3] = []uint16{uint16(npc), 90}, []uint16{uint16(main), 0}
					case "selected":
						mutant.Campaign.Scalars[0] = uint32(main + 1)
					case "index":
						a, err := readCurrentActions(&mutant)
						if err != nil {
							t.Fatal(err)
						}
						a.Session.OfferLabels.Buildings[TownTavern][0].Index = 0
						payload, err := json.Marshal(a)
						if err != nil {
							t.Fatal(err)
						}
						if err := sav.SetNativeActions(&mutant.State, payload); err != nil {
							t.Fatal(err)
						}
					}
					changed, err := sav.EncodeDocumentData(mutant)
					if err != nil {
						t.Fatal(err)
					}
					lost := currentTownReload(t, changed)
					if reflect.DeepEqual(lost.Town.Offers(TownTavern), want) && lost.Town.selectedMission() == main && reflect.DeepEqual(lost.Town.progress.projection().Markers, projection.Markers) {
						t.Fatal("loss control was not discriminated", loss)
					}
				}
				t.Logf("SAV/cold LOAD/next action: %d hearings; paired offers %v; selected %d; markers %v; gold %d", hearings, want, projection.SelectedMission, projection.Markers, gold)
			})
		}
	}
}

func TestReleaseMissionDialoguePointerCancellation(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("mission dialogue pointer")
	app.Layout(640, 480)
	if err := app.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	for n := 0; ; n++ {
		if _, kind, open := f.LiveNotice(); open && kind == ui.NoticeDialogue {
			break
		}
		if n == 1500 {
			t.Fatal("no mission dialogue within 1500 ticks")
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	before, kind, open := f.LiveNotice()
	if !open || kind != ui.NoticeDialogue {
		t.Fatal("no installed mission dialogue")
	}
	view := f.live.view
	viewport := view.NoticeViewportSize()
	noticePlace := frame.FitDown(640, 480, viewport.X, viewport.Y)
	inside := image.Point{}
	found := false
	for dy := -3; dy <= 3 && !found; dy++ {
		for dx := -3; dx <= 3 && !found; dx++ {
			x, y, ok := noticePlace.FrameToWindow(image.Pt(316+dx, 309+dy))
			if ok {
				x, y, found = view.Placement().FrameToWindow(image.Pt(x, y))
				inside = image.Pt(x, y)
			}
		}
	}
	if !found {
		t.Fatal("no window pixel at mission dialogue button centre")
	}
	for _, e := range []struct {
		action string
		x, y   int
	}{{"release", inside.X, inside.Y}, {"press", inside.X, inside.Y}, {"release", 0, 0}, {"release", inside.X, inside.Y}, {"press", 0, 0}, {"release", inside.X, inside.Y}, {"right-press", inside.X, inside.Y}, {"right-release", inside.X, inside.Y}} {
		if err := app.HeadlessPointer(e.action, e.x, e.y); err != nil {
			t.Fatal(err)
		}
		after, k, shown := f.LiveNotice()
		if !shown || k != kind || after != before {
			t.Fatal("unowned input changed mission dialogue", e.action, after, before)
		}
	}
	if err := app.HeadlessPointer("press", inside.X, inside.Y); err != nil {
		t.Fatal(err)
	}
	if after, _, _ := f.LiveNotice(); after != before {
		t.Fatal("down advanced mission dialogue")
	}
	if err := app.HeadlessPointer("release", inside.X, inside.Y); err != nil {
		t.Fatal(err)
	}
	if after, _, shown := f.LiveNotice(); shown && after == before {
		t.Fatal("owned release did not advance mission dialogue")
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
}
