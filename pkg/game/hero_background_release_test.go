package game

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseHeroBackground(t *testing.T) {
	t.Run("generator", func(t *testing.T) {
		f := releaseFront(t)
		for sex := 0; sex < 2; sex++ {
			for class := 0; class < 2; class++ {
				choices := make([]int, 3)
				choices[chargenChoiceSex], choices[chargenChoiceClass] = sex, class
				result := ui.ChargenResult{Name: "Hero background", Choices: choices, Stats: []int{25, 25, 25, 25}}
				member := f.ChargenParty(result)[0]
				dir, face := memberFigure(member)
				expected, _ := composeUnitFigure(f.Archives.Containers, equipmentFromSlots(member.Worn), figureID{Dir: dir, Face: face})
				if !dir.Mage() {
					expected = heroBackgroundExpected(t, f.Archives.Containers, dir, expected)
				}
				preview := f.ChargenPreview(result)
				if preview.Doll == nil || !imagesEqual(preview.Doll, expected) {
					t.Fatalf("generator sex%d class%d background differs", sex, class)
				}
			}
		}
	})
	for _, fixture := range []struct {
		name, hash string
		mission    int
	}{
		{"mission110-portrait-input.ags", "d6f93dcb979fe9a1cd5506b727e286f9ba45709691c94c0b873d2da6f1d1345e", 110},
		{"mission111-portrait-input.ags", "522d5cd01dea60a6d64637884f0427b58888c329be3458259c14d17bce5afe35", 111},
	} {
		t.Run(fmt.Sprint(fixture.mission), func(t *testing.T) {
			path, _ := groundCorpusFile(t, "2026-09-15/"+fixture.name, fixture.hash)
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			a := f.App("hero figure background")
			defer a.StopAudio()
			_, list, load := agsSaveSeams(f, SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
			a.SetSaveSeams(nil, list, load)
			groundAppLoad(t, a, list, filepath.Base(path))
			a.Layout(1024, 768)
			if err := a.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			mw := f.live
			before := mw.world.Hash()
			warriors, mages := 0, 0
			for i, p := range mw.mission.state.Party {
				id := mw.mission.ids[i]
				dir, face := memberFigure(p)
				eq := mw.equipmentOf(id)
				bare, bareMask := composeUnitFigure(f.Archives.Containers, eq, figureID{Dir: dir, Face: face})
				if bare == nil {
					t.Fatal("missing ordinary figure", p.Name)
				}
				want := bare
				if p.PlayerCharacter && !p.Hired() && !dir.Mage() {
					want = heroBackgroundExpected(t, f.Archives.Containers, dir, bare)
					if bytes.Equal(want.Pix, bare.Pix) {
						t.Fatal("fixture hides entire hero background", p.Name)
					}
					warriors++
				} else {
					mages++
				}
				actual := mw.unitFigure(id)
				if actual == nil || !bytes.Equal(actual.Pix, want.Pix) {
					t.Fatalf("%s: large unit doll missing or altering hero background", p.Name)
				}
				if !reflect.DeepEqual(mw.figureMasks[figureCacheKey{fig: mw.figures[id], eq: eq}], bareMask) {
					t.Fatal("hero decoration became an equipment target", p.Name)
				}
				town := composeMemberPortrait(f.Archives.Containers, f.Units, uint32(id), eq, p)
				if town.Figure == nil || !bytes.Equal(town.Figure.Pix, want.Pix) {
					t.Fatal("town doll differs", p.Name)
				}
				rawInventory, _ := composeInventorySubject(f.Archives.Containers, uint32(id), eq, dir, face)
				if !reflect.DeepEqual(town.SlotMask, rawInventory.SlotMask) || !reflect.DeepEqual(town.HoverSlotMask, rawInventory.HoverSlotMask) {
					t.Fatal("town equipment masks changed", p.Name)
				}
				entity, _ := mw.entity(id)
				inspectionCentre(mw, int(entity.X), int(entity.Y))
				if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
					t.Fatal(err)
				}
				if err := a.HeadlessPointer("hover", -1, -1); err != nil {
					t.Fatal(err)
				}
				pane, stats, err := a.HeadlessCharacterPane()
				if err != nil || stats {
					t.Fatal("large equipment pane unavailable", err)
				}
				checkInspectionFigure(t, pane, want, p.Name)
				// Force the normal equipment-cache refresh without changing the world.
				mw.invFigureEquipment.SetCode(1, 0)
				mw.refreshEquipment()
				if !bytes.Equal(mw.invSubject.Figure.Pix, want.Pix) {
					t.Fatal("equipment refresh lost background", p.Name)
				}
				suppressed := mw.suppressedDollSubject(1)
				withoutWeapon := eq
				withoutWeapon.SetCode(1, 0)
				suppressedWant, _ := composeUnitFigure(f.Archives.Containers, withoutWeapon, figureID{Dir: dir, Face: face})
				if !dir.Mage() {
					suppressedWant = heroBackgroundExpected(t, f.Archives.Containers, dir, suppressedWant)
				}
				if !bytes.Equal(suppressed.Figure.Pix, suppressedWant.Pix) {
					t.Fatal("drag preview lost background", p.Name)
				}
				// Same face and equipment, different actor identity: neither the
				// background nor a cached background may leak onto a regular human.
				ordinary := p
				ordinary.PlayerCharacter = false
				ordinary.StartingHero = false
				ordinary.Body = ""
				ordinary.Class = 3
				if dir.Mage() {
					ordinary.Class = 0x17
				}
				figures := partyFigures(nil, []mapload.PartyMember{p, ordinary}, []sim.EntityID{1, 2})
				resolver := speakerResolver{src: f.Archives.Containers}
				for _, test := range []struct {
					id   sim.EntityID
					want *image.RGBA
				}{{1, want}, {2, bare}, {1, want}} {
					got := resolver.speakerFigure(figures[test.id], eq)
					if got == nil || !bytes.Equal(got.Pix, test.want.Pix) {
						t.Fatal("speaker figure cache mixed hero and regular human", p.Name)
					}
				}
			}
			if warriors != 3 || mages != 2 || mw.world.Hash() != before {
				t.Fatalf("unexpected subjects or simulation changed: warriors=%d mages=%d", warriors, mages)
			}
			t.Logf("mission%d: three warrior capes, two unchanged mages, equipment and hover masks preserved", fixture.mission)
		})
	}
}

func heroBackgroundExpected(t *testing.T, src entrySource, dir data.FigureDir, front *image.RGBA) *image.RGBA {
	t.Helper()
	leaf := "backm.256"
	if dir.Female() {
		leaf = "backf.256"
	}
	sheets := sheetCache{src: src, decoded: map[string]*spr256.Sprite{}, converted: map[string][]*terrain.StaticFrame{}}
	back := figureRGBA(&sheets, "interface/heroback/"+leaf)
	if back == nil || back.Bounds() != image.Rect(0, 0, 160, 240) {
		t.Fatal("missing original hero background", leaf)
	}
	out := image.NewRGBA(front.Bounds())
	draw.Draw(out, out.Bounds(), back, back.Bounds().Min, draw.Src)
	draw.Draw(out, out.Bounds(), front, front.Bounds().Min, draw.Over)
	return out
}
