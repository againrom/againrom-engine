package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestReleaseKeptTurtleOfferAvailabilitySurvivesCurrentSAV(t *testing.T) {
	root := os.Getenv("AGAINROM_TAVERN_REVIEW_DIR")
	if root == "" {
		t.Skip("AGAINROM_TAVERN_REVIEW_DIR is not set")
	}
	rel, err := filepath.Rel(reviewDirRoot("hotfix-tavern-completed"), root)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("tavern outputs are outside the review directory", root, err)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"TMP", "TEMP", "TMPDIR"} {
		t.Setenv(key, root)
	}
	for _, tc := range []struct {
		name, variable, hash string
		main, selected       uint32
		current              bool
		wonSource            bool
	}{
		{"absent_row", "AGAINROM_TAVERN_CLOSED_INPUT", "f3d5450e57f15857f902d8eab9ae480bdb37c3efe875fe325c0173a25da6d6ea", 60, 60, false, false},
		{"retained", "AGAINROM_TAVERN_PENDING_INPUT", "c94deca4da6982a7d0218c2befb24c441c09c5a2df004c92893e257502e37831", 50, 51, true, false},
		{"won51", "AGAINROM_TAVERN_WON_INPUT", "e3850e9cead43f47c423ffbca1d50008a1dd6c2802e9be0f0759edebbbb8c175", 50, 50, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := os.Getenv(tc.variable)
			if path == "" {
				t.Skip(tc.variable + " is not set")
			}
			raw, err := os.ReadFile(path)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != tc.hash {
				t.Fatal("preserved tavern input changed", path, err)
			}
			t.Cleanup(func() {
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(raw, after) {
					t.Error("preserved tavern input bytes changed", err)
				}
			})
			f := shopOrderFront(t)
			var app *ui.App
			if tc.wonSource {
				file, err := sav.Open(raw)
				if err != nil {
					t.Fatal(err)
				}
				won, lost, err := file.Counters()
				if err != nil || file.Head.Mission != 51 || won != 1 || lost != 0 {
					t.Fatal("source is not the won turtle mission", won, lost, err)
				}
				app, _ = openOriginalSAVAppAt(t, f, raw, "turtlekilled.sav", 640, 480)
				_, kind, shown := f.LiveNotice()
				if f.liveMission != 51 || f.Town.progress == nil || f.Town.progress.record(51) == nil || !shown || kind != ui.NoticeSuccess {
					t.Fatal("won source did not load the actual mission51 Victory", f.liveMission, kind, shown)
				}
				t.Logf("original won51: tick%d main%d selected%d Victory=%v", f.live.world.Tick(), f.Town.Chapter(), f.Town.selectedMission(), kind)
				if err := app.HeadlessActivate("notice"); err != nil {
					t.Fatal("actual Victory acknowledgment", err)
				}
				view := f.townUI.WorldMapView()
				if app.Screen() != ui.ScreenTown || !f.townUI.AtWorldMap() || !view.Returning || !view.HideScrolls {
					t.Fatal("Victory did not start the actual automatic return", app.Screen(), view)
				}
				for n := 0; !f.townUI.AtTownSquare() && n < 4000; n++ {
					if err := app.HeadlessStep(); err != nil {
						t.Fatal(err)
					}
				}
				if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() || f.Town.Chapter() != int(tc.main) || f.Town.selectedMission() != int(tc.selected) {
					t.Fatal("Victory did not return to the expected town", app.Screen(), f.Town.Chapter(), f.Town.selectedMission())
				}
			} else {
				checkTurtleCampaignWire(t, raw, tc.main, tc.selected, tc.current)
				app = turtleOpenTownApp(t, f, raw, tc.name+".sav")
			}
			t.Cleanup(app.StopAudio)
			body := checkTurtleTavernApp(t, f, app, tc.current)
			store := SaveStore{Dir: t.TempDir()}
			app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
			name := shopOrderF2Save(t, app, store)
			current, err := store.Read(name)
			if err != nil {
				t.Fatal(err)
			}
			checkTurtleCampaignWire(t, current, tc.main, tc.selected, tc.current)
			out := filepath.Join(root, filepath.Base(os.Getenv("AGAINROM_ASSETS")), tc.name+"-current.sav")
			if err := os.MkdirAll(filepath.Dir(out), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(out, current, 0600); err != nil {
				t.Fatal(err)
			}
			g := shopOrderFront(t)
			cold := turtleOpenTownApp(t, g, current, "current.sav")
			t.Cleanup(cold.StopAudio)
			if restored := checkTurtleTavernApp(t, g, cold, tc.current); restored != body {
				t.Fatal("cold LOAD changed the retained NPC32 body", body, restored)
			}
			t.Logf("source %s, current SAV %x: %s", tc.hash, sha256.Sum256(current), out)
		})
	}
}

func turtleOpenTownApp(t *testing.T, f *FrontEnd, raw []byte, name string) *ui.App {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
		t.Fatal(err)
	}
	app := openLocalTownSAV(t, f, dir, name)
	app.Layout(640, 480)
	return app
}

func checkTurtleCampaignWire(t *testing.T, raw []byte, main, selected uint32, current bool) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	campaign, ok, err := file.Campaign()
	if err != nil || !ok || file.Head.Mission != 0 || campaign.Main.Mission != main || campaign.SelectedMission != selected {
		t.Fatal("tavern SAV has different campaign identity", file.Head.Mission, campaign.Main.Mission, campaign.SelectedMission, err)
	}
	found := false
	for _, child := range campaign.Children {
		if child.Mission == 51 {
			found = true
			t.Logf("wire child51: age%d announced%v", child.Age, child.Announced)
		}
	}
	if found != current {
		t.Fatalf("wire retains child51=%v, want %v", found, current)
	}
}

func checkTurtleTavernApp(t *testing.T, f *FrontEnd, app *ui.App, current bool) [32]byte {
	t.Helper()
	if app.Screen() != ui.ScreenTown || f.Town.progress == nil || (f.Town.progress.record(51) != nil) != current {
		t.Fatal("actual LOAD did not retain the source campaign record population", app.Screen())
	}
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	if tip := s.TownSurface().Tip; tip.Showing() {
		p := ui.TipPanelCloseRect(tip.Rect).Min.Add(image.Pt(2, 2))
		turtleTap(t, app, p)
	}
	if s.room != roomTavern {
		t.Fatal("App did not enter the tavern", s.room)
	}
	index := -1
	var offer TownOffer
	for i, candidate := range s.tavernCandidates() {
		if candidate.key.kind == tavernCandidateOffer && candidate.offer.NPC == 32 && candidate.offer.Mission == 51 {
			index, offer = i, candidate.offer
			break
		}
	}
	if !current && f.Town.Done(51) {
		before := f.Town.progress.projection()
		gold := f.Town.Gold()
		if index >= 0 || app.HeadlessActivate("NPC 32") == nil || s.room != roomTavern ||
			len(s.innQueue) != 0 || f.Town.Gold() != gold || !reflect.DeepEqual(before, f.Town.progress.projection()) {
			t.Fatal("explicit completed turtle quest still exposes a talker or activation changes state")
		}
		if _, _, err := app.HeadlessFrame(); err != nil {
			t.Fatal("completed turtle tavern paint", err)
		}
		t.Logf("explicit Done51: no NPC32 cell, unchanged chapter%d campaign", f.Town.Chapter())
		return [32]byte{}
	}
	if index < 0 && !current && f.Town.Chapter() == 60 {
		if err := app.HeadlessActivate("NPC 32"); err == nil || s.room != roomTavern {
			t.Fatal("the later chapter manufactured an old NPC32 cell")
		}
		t.Logf("honest later-source control: no NPC32 row; roster %+v, record51 absent", f.Town.TavernRoster())
		return [32]byte{}
	}
	if index < 0 || !current && offer.Index >= 0 {
		t.Fatalf("source does not bind the kept NPC32/m51 route: %+v", f.Town.TavernRoster())
	}
	p := turtleCellPoint(t, s.TownSurface(), index)
	turtleTap(t, app, p)
	v := s.TownSurface()
	if !v.Cells[index].Selected || !v.Cells[index].Enabled || v.Cells[index].Picture == nil || v.CandidatePixels == nil {
		t.Fatal("NPC32 lost his selected cell or body")
	}
	body := bytes.Clone(v.CandidatePixels.Pix)
	before := f.Town.progress.projection()
	gold, queued := f.Town.Gold(), len(s.innQueue)
	if current {
		if !v.Buttons[tavernButtonTalk].Enabled || app.HeadlessActivate(f.Words.TavernTalk) != nil || s.room != roomTalk || s.dialogue.displayPart != 1 {
			t.Fatal("a retained active turtle record cannot replay its dialogue")
		}
		payload, err := f.Archives.Containers.ReadFile(mainPrefix + "text/inn/npc/npc32m51.txt")
		if err != nil || !bytes.Equal(s.dialogue.payload, payload) {
			t.Fatal("active turtle record opened different text", err)
		}
		for n := 0; s.room == roomTalk && n < 8; n++ {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		if s.room != roomTavern {
			t.Fatal("the active conversation did not return to the tavern")
		}
	} else {
		if v.Buttons[tavernButtonTalk].Enabled {
			t.Fatal("Talk remains enabled for kept NPC32 without a current mission51 record")
		}
		r := ui.TownSurfaceButtonRect(v.Kind, tavernButtonTalk)
		turtleTap(t, app, r.Min.Add(r.Max).Div(2))
		turtleTap(t, app, p)
		turtleTap(t, app, p)
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate("NPC 32"); err == nil {
			t.Fatal("semantic activation bypassed disabled Talk")
		}
		if s.room != roomTavern || len(s.innQueue) != queued {
			t.Fatal("an alternate App activation opened obsolete turtle text or queued its mission")
		}
	}
	v = s.TownSurface()
	if !v.Cells[index].Selected || v.CandidatePixels == nil || !bytes.Equal(body, v.CandidatePixels.Pix) || f.Town.Gold() != gold || !reflect.DeepEqual(before, f.Town.progress.projection()) {
		t.Fatal("hearing availability changed NPC32's body, purse or campaign")
	}
	if _, _, err := app.HeadlessFrame(); err != nil {
		t.Fatal("actual App town paint", err)
	}
	t.Logf("NPC32 index%d offerIndex%d current51=%v Done51=%v Talk=%v, body %x", index, offer.Index, current, f.Town.Done(51), v.Buttons[tavernButtonTalk].Enabled, sha256.Sum256(body))
	return sha256.Sum256(body)
}

func turtleCellPoint(t *testing.T, v ui.TownSurfaceView, index int) image.Point {
	t.Helper()
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			p := image.Pt(x, y)
			if c, ok := ui.TownSurfaceControlAt(v, p); ok && c.Kind == ui.TownSurfaceControlCell && c.Index == index {
				return p
			}
		}
	}
	t.Fatal("visible NPC32 cell has no pointer hit")
	return image.Point{}
}

func turtleTap(t *testing.T, app *ui.App, p image.Point) {
	t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
}
