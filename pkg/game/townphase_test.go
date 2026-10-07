package game

import (
	"againrom/internal/synth"
	"image"
	"testing"

	"againrom/pkg/ui"
)

// The fixture uses a synthetic town-entry seam, not a decoded SAV witness.
func townPhaseFixture(t *testing.T, occupied bool) (*ui.App, *townScreen) {
	t.Helper()
	f := shellFrontEnd()
	campaign := shellCampaign()
	chapter := campaign.Chapters[30]
	chapter.InnNPC, chapter.Inn = nil, nil
	if occupied {
		for i := 0; i < 17; i++ {
			chapter.InnNPC = append(chapter.InnNPC, 9+i)
			chapter.Inn = append(chapter.Inn, 0)
		}
	} else {
		entry := campaign.Chapters[20]
		entry.EnableMercenary = nil
		campaign.Chapters[20] = entry
	}
	campaign.Chapters[30] = chapter
	f.Campaign = resolved(campaign, nil)
	f.Town = NewTown(campaign)
	f.Town.Won(20)
	f.Town.Arrive()
	f.Town.gold = 1000
	f.TownTavernArt = resolved(&ui.TownTavernArt{}, nil)
	app := f.App("town-phase-witness")
	app.Layout(640, 480)
	app.SetSaveSeams(nil,
		func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.sav", Label: "Town"}} },
		func(string) (ui.MapOpener, bool, error) { f.townUI.resetForNewGame(); return nil, true, nil })
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("Tavern"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || f.townUI.room != roomTavern {
		t.Fatal("fixture did not reach tavern")
	}
	f.townUI.CloseTip()
	return app, f.townUI
}

func TestTownFigureOccupiedCellSelectsOnDown(t *testing.T) {
	for _, target := range []struct {
		name  string
		point image.Point
		index int
	}{
		{"tender", image.Pt(300, 300), 14}, {"cauldron", image.Pt(430, 300), 17},
	} {
		t.Run(target.name, func(t *testing.T) {
			app, town := townPhaseFixture(t, true)
			view := town.TownSurface()
			if len(view.Cells) != 18 {
				t.Fatalf("fixture has %d cells, want18", len(view.Cells))
			}
			control, hit := ui.TownSurfaceControlAt(view, target.point)
			if !hit || control.Kind != ui.TownSurfaceControlCell || control.Index != target.index {
				t.Fatalf("fixture target resolves to %+v, hit%v", control, hit)
			}
			want := tavernCandidateKey{kind: tavernCandidateOffer, id: target.index - 1}
			if town.tavernSelection == want {
				t.Fatal("fixture already selects target")
			}
			if err := app.HeadlessPointer("press", target.point.X, target.point.Y); err != nil {
				t.Fatal(err)
			}
			if town.tavernSelection != want {
				t.Fatalf("occupied figure down leaves selection%+v; want%+v before release", town.tavernSelection, want)
			}
			if err := app.HeadlessPointer("release", 2, 2); err != nil {
				t.Fatal(err)
			}
			if town.tavernSelection != want {
				t.Fatal("outside release changed selected candidate")
			}
		})
	}
}

func TestTownFigureUnoccupiedPointsRemainInert(t *testing.T) {
	app, town := townPhaseFixture(t, false)
	if cells := town.TownSurface().Cells; len(cells) != 0 {
		t.Fatalf("fixture has %d cells, want0", len(cells))
	}
	before := town.tavernSelection
	for _, point := range []image.Point{image.Pt(300, 300), image.Pt(430, 300), image.Pt(200, 100)} {
		if err := app.HeadlessPointer("press", point.X, point.Y); err != nil {
			t.Fatal(err)
		}
		if town.tavernSelection != before {
			t.Fatal("unoccupied down changed selection")
		}
		if err := app.HeadlessPointer("release", point.X, point.Y); err != nil {
			t.Fatal(err)
		}
		if town.tavernSelection != before {
			t.Fatal("unoccupied up changed selection")
		}
	}
}

func TestTavernAppDoublePressSelectsBeforeHireDismissAndTalk(t *testing.T) {
	app, town := townPhaseFixture(t, true)
	press := func(p image.Point) {
		t.Helper()
		if err := app.HeadlessPointer("press", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	release := func(p image.Point) {
		t.Helper()
		if err := app.HeadlessPointer("release", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	merc := image.Pt(200, 450)
	beforeGold, beforeParty := town.sess.Town.Gold(), len(town.sess.Carried)
	price := town.tavernMercenaries()[0].Price
	press(merc)
	if town.sess.Town.Gold() != beforeGold || len(town.sess.Carried) != beforeParty {
		t.Fatal("single down hired squad")
	}
	release(merc)
	press(merc)
	if !town.sess.Town.MercenaryHired(3) || town.sess.Town.Gold() != beforeGold-price || len(town.sess.Carried) != beforeParty+2 {
		t.Fatal("second down did not atomically hire selected squad")
	}
	release(merc)
	if town.sess.Town.Gold() != beforeGold-price {
		t.Fatal("release applied hire twice")
	}
	press(merc)
	release(merc)
	press(merc)
	if town.sess.Town.MercenaryHired(3) || town.sess.Town.Gold() != beforeGold || len(town.sess.Carried) != beforeParty {
		t.Fatal("dismiss pair changed purse or retained squad")
	}
	release(merc)
	town.in.Archives = &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/inn/npc/npc22m30.txt", Data: []byte("<part=1>\nSelected talk candidate")}})}
	talk := image.Pt(300, 300)
	press(talk)
	release(talk)
	press(talk)
	if town.room != roomTalk || town.npc != 22 || town.tavernSelection != (tavernCandidateKey{kind: tavernCandidateOffer, id: 13}) {
		t.Fatal("talk double did not select candidate before opening its dialogue")
	}
}

func TestTavernAppUnaffordableDoublePressIsAtomic(t *testing.T) {
	app, town := townPhaseFixture(t, true)
	town.sess.Town.gold = 0
	party := len(town.sess.Carried)
	for _, edge := range []string{"press", "release", "press", "release"} {
		if err := app.HeadlessPointer(edge, 200, 450); err != nil {
			t.Fatal(err)
		}
	}
	if town.tavernSelection != (tavernCandidateKey{kind: tavernCandidateMercenary, id: 3}) || town.sess.Town.Gold() != 0 || town.sess.Town.MercenaryHired(3) || len(town.sess.Carried) != party {
		t.Fatal("unaffordable App double changed purse/party or lost selection")
	}
}
