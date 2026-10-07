package game

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseLocalizedMissionUnitNames(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("localized enemy names")
	defer a.StopAudio()
	if err := a.OpenMission(f.MissionOpener(111)); err != nil {
		t.Fatal(err)
	}
	a.Layout(1024, 768)
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16 && a.HeadlessNoticeOpen(); i++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	live := f.live
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	for _, tc := range []struct{ id, nameIndex uint32 }{{0, 27}, {1, 24}, {4, 66}, {5, 64}} {
		want := f.Words.UnitNames[tc.nameIndex]
		if want == "" {
			t.Fatal("installed unit label missing")
		}
		e := releaseEntityByID(t, live.world, sim.EntityID(tc.id))
		inspectionCentre(live, int(e.X), int(e.Y))
		live.push()
		releaseHoverInspection(t, a, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: tc.id})
		rows, ok := live.view.PanelStatement()
		if !ok || len(rows) == 0 || !strings.Contains(rows[0], want) {
			t.Fatalf("actor%d hover name: %v, want %q", tc.id, rows, want)
		}
		t.Logf("mission111 actor%d hover=%q", tc.id, rows[0])
	}
}

func TestReleaseVampiricStaffTrainsWaterAtFullHealth(t *testing.T) {
	path, _ := groundCorpusFile(t, "2026-09-15/mission111-portrait-input.ags",
		"522d5cd01dea60a6d64637884f0427b58888c329be3458259c14d17bce5afe35")
	f := releaseFront(t)
	a := f.App("vampiric staff")
	defer a.StopAudio()
	_, list, load := agsSaveSeams(f, SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
	a.SetSaveSeams(nil, list, load)
	groundAppLoad(t, a, list, filepath.Base(path))
	caster := releaseEntityByID(t, f.live.world, 84)
	for _, e := range f.live.entityDraws() {
		if e.ID == uint32(caster.ID) && e.Name != "Fergard" {
			t.Fatalf("localized type replaced the saved hero's name: %q", e.Name)
		}
	}
	if caster.WeaponSpell != 11 || caster.HP != caster.MaxHP {
		t.Fatal("owner fixture lost its full-health vampiric staff wielder")
	}
	// Keep the owner's actual equipment, skills and source arithmetic, placing
	// that actor opposite a passive target so a full attack cycle is measurable.
	victim := sim.Entity{ID: 1, X: caster.X + 3, Y: caster.Y, Owner: 3, HP: 300, MaxHP: 300, XPValue: 300}
	var stock []sim.Stock
	for _, holding := range f.live.world.Stock() {
		if holding.ID == caster.ID {
			stock = append(stock, holding)
		}
	}
	w, err := sim.NewStockedSpelledWorld(7, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{caster, victim}, nil, sim.Relations{}, nil, stock, f.live.world.Spells())
	if err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(w)
	for tick := 0; tick < 240; tick++ {
		var cmds []sim.Command
		if tick == 0 {
			cmds = []sim.Command{{Kind: sim.KindAttack, Entity: caster.ID, X: int32(victim.ID)}}
		}
		for _, event := range sim.StepObserved(w, cmds) {
			if event.Caster != caster.ID || !event.Weapon || event.Spell != 11 {
				continue
			}
			got := releaseEntityByID(t, w, caster.ID)
			if event.School != 2 || got.SkillXP[2] <= caster.SkillXP[2] || got.HP != got.MaxHP {
				t.Fatalf("full-health staff release: event%+v HP%d/%d XP%v -> %v", event, got.HP, got.MaxHP, caster.SkillXP, got.SkillXP)
			}
			for i := range caster.SkillXP {
				if i != 2 && got.SkillXP[i] != caster.SkillXP[i] {
					t.Fatalf("staff trained unrelated slot%d", i)
				}
			}
			t.Logf("owner mage84 staff11: full HP%d, Water XP+%d after%d attack ticks", got.HP, got.SkillXP[2]-caster.SkillXP[2], tick+1)
			return
		}
	}
	t.Fatal("owner's staff never released through its ordinary attack cycle")
}

type softwareTerrainSource struct {
	terrain.EntrySource
	paths []string
}

func (s *softwareTerrainSource) ReadFile(path string) ([]byte, error) {
	s.paths = append(s.paths, path)
	return s.EntrySource.ReadFile(path)
}

func TestReleaseSoftwareTerrainFamily(t *testing.T) {
	f := releaseFront(t)
	source := &softwareTerrainSource{EntrySource: f.Archives.Containers}
	set := terrain.LoadTileset(source)
	if set.Loaded != 52 || set.Dirt == nil {
		t.Fatalf("ordinary terrain population: %d strips, dirt%t", set.Loaded, set.Dirt != nil)
	}
	for _, path := range source.paths {
		if !strings.HasPrefix(path, "graphics/terrain/") {
			t.Fatalf("software renderer loaded another family: %s", path)
		}
	}
	different := 0
	for g := 1; g <= 4; g++ {
		for v := 0; v < 16; v++ {
			strip := set.Slot(terrain.SlotIndex(g, v))
			if strip == nil {
				continue
			}
			path := fmt.Sprintf("graphics/terrain/tile%d-%02d.bmp", g, v)
			original, err := f.Archives.Containers.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			bmp, err := terrain.DecodeBMP8(original)
			if err != nil || !bytes.Equal(strip.SubCells[0].Pix, bmp.Pix[:32*32]) {
				t.Fatalf("loaded strip differs from ordinary resource %s: %v", path, err)
			}
			threeD, err := f.Archives.Containers.ReadFile(strings.Replace(path, "/terrain/", "/terrain.3d/", 1))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(original, threeD) {
				different++
			}
		}
	}
	if different != 52 {
		t.Fatalf("paired family contrast covers only%d strips", different)
	}
	t.Log("52 ordinary terrain strips plus dirt loaded; all 52 paired 3D strips differ")
}
