package game

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

func ringPlacements(m *alm.Map) []image.Point {
	var out []image.Point
	for _, u := range m.Units {
		x, y := int(u.X>>8), int(u.Y>>8)
		if x < 8 || y < 8 || x >= int(m.Width)-8 || y >= int(m.Height)-8 {
			out = append(out, image.Pt(x, y))
		}
	}
	return out
}

// The production opener leaves no entity on the border ring of the three maps
// that author one there (TERR-PLACE-208); the plain build is the loss control.
func TestReleaseMissionStartRefusesBorderRingPlacements(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	for _, tc := range []struct {
		mission int
		ring    []image.Point
	}{
		{80, []image.Point{{77, 136}}},
		{90, []image.Point{{54, 136}}},
		{100, []image.Point{{43, 136}, {53, 137}}},
	} {
		t.Run(fmt.Sprintf("mission %d", tc.mission), func(t *testing.T) {
			addr, _ := MissionMap(tc.mission)
			raw, err := f.Archives.Containers.ReadFile(addr)
			if err != nil {
				t.Fatal(err)
			}
			m, err := alm.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			authored := len(m.Units)
			if got := ringPlacements(m); fmt.Sprint(got) != fmt.Sprint(tc.ring) {
				t.Fatalf("authored ring placements = %v, want %v", got, tc.ring)
			}

			control, err := mapload.FromALMWith(m, f.Table, mapload.DifficultyNormal)
			if err != nil {
				t.Fatal(err)
			}
			onRing := 0
			for _, e := range control.Entities() {
				if x, y := int(e.X), int(e.Y); x < 8 || y < 8 || x >= int(m.Width)-8 || y >= int(m.Height)-8 {
					onRing++
				}
			}
			if onRing != len(tc.ring) {
				t.Fatalf("loss control: %d entities on the ring without the withdrawal, want %d", onRing, len(tc.ring))
			}

			app := f.App("border ring placements")
			app.Layout(1024, 768)
			t.Cleanup(app.StopAudio)
			party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
			if err := app.OpenMission(f.MissionOpenerWith(tc.mission, party)); err != nil {
				t.Fatal(err)
			}
			entities := f.live.world.Entities()
			for _, e := range entities {
				if x, y := int(e.X), int(e.Y); x < 8 || y < 8 || x >= int(m.Width)-8 || y >= int(m.Height)-8 {
					t.Errorf("entity %d (owner %d, class %d) stands on ring cell (%d,%d)", e.ID, e.Owner, e.Class, x, y)
				}
			}
			if want := authored - len(tc.ring) + len(party); len(entities) != want {
				t.Errorf("%d entities, want %d authored placements less %d ring placements plus %d party members",
					len(entities), want, len(tc.ring), len(party))
			}
		})
	}
}

// Mission 90's world-map object has no picture and builds no marker
// (TERR-GMAP-206); mission 100's object is the control that does.
func TestReleaseKargallasWorldMapObjectBuildsNoMarker(t *testing.T) {
	f := releaseFront(t)
	a := f.worldMapAssets()
	if a == nil || a.data == nil {
		t.Fatal("no world-map registry")
	}
	src := f.Archives.Containers
	object, ok := a.data.Missions[90]
	if !ok {
		t.Fatal("mission 90 has no world-map object")
	}
	row := a.data.Objects[object]
	if row.Picture != "" || row.Point != image.Pt(478, 298) {
		t.Fatalf("mission 90 object %d = picture %q point %v, want no picture at (478,298)", object, row.Picture, row.Point)
	}
	if m := a.marker(src, 90, row); m != nil {
		t.Fatalf("mission 90 built a %v marker without a picture", m.Bounds())
	}
	if path, built := nativeCityMarkerPicture(f, 90); built {
		t.Fatalf("mission 90 resolved marker path %q", path)
	}
	control := a.data.Objects[a.data.Missions[100]]
	if control.Picture == "" || a.marker(src, 100, control) == nil {
		t.Fatalf("control mission 100 object %+v built no marker", control)
	}
}

// Mission 90's castle is placed whole inside the playable columns, and the
// highest camera position at every view x is above its top (TERR-STRUCT-207).
func TestReleaseKargallasCastleIsNeverCutByTheCamera(t *testing.T) {
	f := releaseFront(t)
	addr, _ := MissionMap(90)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	var castle *alm.Object
	for i := range m.Objects {
		if col, row := terrain.AnchorCell(m.Objects[i].X, m.Objects[i].Y); col == 104 && row == 9 {
			castle = &m.Objects[i]
		}
	}
	if castle == nil {
		t.Fatal("no structure anchored at (104,9)")
	}
	g := terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles),
		Altitudes: m.Altitudes, Overlay: m.Overlay, Structures: StructureRecords([]alm.Object{*castle})}
	proj := terrain.Project(g.Altitudes, g.Width, g.Height)
	places, _, _ := terrain.StructurePlacements(g, f.Structures, proj.Altitude, proj.MinV)
	if len(places) != 55 {
		t.Fatalf("castle placed %d entries, want 11 columns x 5 rows", len(places))
	}
	top, left, right := places[0].Rect().Min.Y, places[0].Rect().Min.X, places[0].Rect().Max.X
	for _, p := range places {
		r := p.Rect()
		top, left, right = min(top, r.Min.Y), min(left, r.Min.X), max(right, r.Max.X)
		if p.Cell.X < 104 || p.Cell.X > 114 || p.Cell.Y < 9 || p.Cell.Y > 13 {
			t.Fatalf("castle entry on cell %v, outside columns 104..114 rows 9..13", p.Cell)
		}
	}
	if left < 8*camera.CellSize || right > 136*camera.CellSize {
		t.Fatalf("castle spans world x %d..%d, outside the playable columns", left, right)
	}

	mv, err := LoadMapViewer(f.Tiles, raw, addr, Markers{},
		StaticLayer{Set: f.Statics, Art: true}, StructureLayer{Set: f.Structures, Art: true})
	if err != nil {
		t.Fatal(err)
	}
	mv.Viewer.Layout(1024, 768)
	cam := mv.Viewer.Camera()
	for x := float64(left) - 1000; x <= float64(right); x += 32 {
		cam.X = x
		cam.Clamp()
		cam.Y = -1e9
		cam.Clamp()
		if cam.X < float64(right)-float64(cam.ViewW) && cam.X+float64(cam.ViewW) > float64(left) && cam.Y > float64(top) {
			t.Fatalf("at view x %.0f the highest camera y %.0f is below the castle top %d", cam.X, cam.Y, top)
		}
	}
}
