package game

import (
	"bytes"
	"fmt"
	"image"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleasePortraitInputHotfix(t *testing.T) {
	t.Run("shop-statistics-equip", func(t *testing.T) {
		for _, dest := range []string{"pack", "table"} {
			t.Run(dest, func(t *testing.T) {
				a, s := releaseShopApp(t)
				defer a.StopAudio()
				s.CloseTip()
				before := *s.shopWornSlots(s.shopMemberIndex())
				gold := s.sess.Town.Gold()
				// Move the equipped weapon through the visible doll and actual
				// pack/table. Then return it through the statistics view.
				x, y, err := a.HeadlessShopPoint("doll", 1)
				if err != nil {
					t.Fatal(err)
				}
				tx, ty, err := a.HeadlessShopPoint(dest, 0)
				if err != nil {
					t.Fatal(err)
				}
				portraitInputDrag(t, a, image.Pt(x, y), image.Pt(tx, ty))
				if s.shopWornSlots(s.shopMemberIndex())[0] != 0 {
					t.Fatal("weapon did not leave the doll")
				}
				if err = a.HeadlessKey("tab"); err != nil {
					t.Fatal(err)
				}
				if !s.ShopScreen().Character.Statistics {
					t.Fatal("town Tab no longer switches to statistics")
				}
				if dest == "pack" {
					found := false
					for i, c := range s.shopPackStacks() {
						if uint16(c.Code) == before[0] {
							tx, ty, err = a.HeadlessShopPoint("pack", i+1-s.packBase)
							found = true
							break
						}
					}
					if !found || err != nil {
						t.Fatal("weapon not found in pack", err)
					}
				}
				portraitInputDrag(t, a, image.Pt(tx, ty), image.Pt(560, 350))
				if got := *s.shopWornSlots(s.shopMemberIndex()); got[0] != before[0] || s.sess.Town.Gold() != gold {
					t.Fatalf("statistics drop lost item or charged own goods: worn=%v gold=%d", got, s.sess.Town.Gold())
				}
				if !s.ShopScreen().Character.Statistics {
					t.Fatal("drop switched the card mode")
				}
				for _, kind := range []string{"press", "release"} {
					if err = a.HeadlessPointer(kind, 560, 350); err != nil {
						t.Fatal(err)
					}
				}
				if s.shopWornSlots(s.shopMemberIndex())[0] != before[0] {
					t.Fatal("clicking statistics unequipped the item")
				}
				if err = a.HeadlessKey("tab"); err != nil {
					t.Fatal(err)
				}
				if s.ShopScreen().Character.Statistics {
					t.Fatal("town Tab did not restore doll")
				}
			})
		}
	})
	for _, fixture := range []struct {
		name, hash     string
		mission, count int
	}{
		{"mission110-portrait-input.ags", "d6f93dcb979fe9a1cd5506b727e286f9ba45709691c94c0b873d2da6f1d1345e", 110, 41},
		{"mission111-portrait-input.ags", "522d5cd01dea60a6d64637884f0427b58888c329be3458259c14d17bce5afe35", 111, 38},
	} {
		t.Run(fmt.Sprintf("mission%d-hover-and-tab", fixture.mission), func(t *testing.T) {
			path, _ := groundCorpusFile(t, "2026-09-15/"+fixture.name, fixture.hash)
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			a := f.App("portrait input")
			defer a.StopAudio()
			_, list, load := agsSaveSeams(f, SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
			a.SetSaveSeams(nil, list, load)
			groundAppLoad(t, a, list, filepath.Base(path))
			a.Layout(1024, 768)
			if err := a.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			mw := f.live
			// Expose only presentation fog. The saved world and orders remain
			// untouched while the pointer visits every targetable foreign actor
			// and every building with a health pool in these two owner saves.
			for i := range mw.fog.visible {
				mw.fog.visible[i], mw.fog.explored[i] = 1, 1
			}
			mw.push()
			hero := mw.mission.ids[0]
			if e, ok := mw.entity(hero); ok {
				inspectionCentre(mw, int(e.X), int(e.Y))
			}
			if err := a.HeadlessSelectEntity(uint32(hero)); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessPointer("hover", -1, -1); err != nil {
				t.Fatal(err)
			}
			before := mw.world.Hash()
			pending := len(mw.pending)
			inventory := mw.invSubject
			pane, stats, err := a.HeadlessCharacterPane()
			if err != nil || stats {
				t.Fatal("initial figure", err)
			}
			card, err := a.HeadlessMissionCard()
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"tab", "shift-tab"} {
				if err = a.HeadlessKey(key); err != nil {
					t.Fatal(err)
				}
				got, stats, err := a.HeadlessCharacterPane()
				if err != nil || stats || !bytes.Equal(got.Pix, pane.Pix) {
					t.Fatal("mission Tab changed figure", err)
				}
				gotCard, err := a.HeadlessMissionCard()
				if err != nil || !bytes.Equal(gotCard.Pix, card.Pix) {
					t.Fatal("mission Tab changed lower stats", err)
				}
			}
			count := 0
			check := func(ref ui.InspectionSubject, col, row, hp, maxHP int, want *image.RGBA) {
				t.Helper()
				if want == nil {
					t.Fatalf("missing source figure %+v", ref)
				}
				inspectionCentre(mw, col, row)
				releaseHoverInspection(t, a, mw, ref)
				s, ok := mw.view.InspectionPanel()
				if !ok || s.Kind != ref.Kind || s.ID != ref.ID || s.HP != hp || s.MaxHP != maxHP {
					t.Fatalf("incorrect live card %+v for %+v", s, ref)
				}
				pic, stats, err := a.HeadlessCharacterPane()
				if err != nil || stats {
					t.Fatal("hover lost figure", err)
				}
				checkInspectionFigure(t, pic, want, fmt.Sprint(ref))
				if _, err = a.HeadlessMissionCard(); err != nil {
					t.Fatal("hover lost lower stats", err)
				}
				if selected, ok := mw.view.SelectedUnit(); !ok || selected != uint32(hero) {
					t.Fatal("hover changed selected hero")
				}
				count++
			}
			for _, e := range mw.world.Entities() {
				if !e.OrdinaryTargetable() || e.Owner == sim.SelfSlot {
					continue
				}
				check(ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(e.ID)}, int(e.X), int(e.Y), int(e.HP), int(e.MaxHP), mw.inspectionUnitPicture(uint32(e.ID)))
			}
			for _, s := range mw.world.Structures() {
				if s.MaxHealth == 0 {
					continue
				}
				c := f.Structures.Classes[mw.mission.state.Map.Objects[s.ID].Kind]
				if c == nil {
					t.Fatal("unknown structure", s.ID)
				}
				check(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(s.ID)}, int(s.Col), int(s.Row), int(int16(s.Field42)), int(s.MaxHealth), c.Portrait)
			}
			if count != fixture.count || mw.world.Hash() != before || len(mw.pending) != pending || !reflect.DeepEqual(mw.invSubject, inventory) {
				t.Fatalf("hover census count=%d want=%d or mutated state", count, fixture.count)
			}
			t.Logf("mission%d: %d pointer targets retained live figures and health; Tab preserved both panels", fixture.mission, count)
		})
	}
}

func portraitInputDrag(t *testing.T, a *ui.App, from, to image.Point) {
	t.Helper()
	for _, step := range []struct {
		kind string
		p    image.Point
	}{{"press", from}, {"hover", to}, {"release", to}} {
		if err := a.HeadlessPointer(step.kind, step.p.X, step.p.Y); err != nil {
			t.Fatal(err)
		}
	}
}
