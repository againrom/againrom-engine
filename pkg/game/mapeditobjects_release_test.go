package game

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// objectEdits is the host seam of a document the editor opened.
func objectEdits(t *testing.T, e *ui.MapEditor) *mapInspectionEdits {
	t.Helper()
	s, ok := e.Document().Edits.(*mapInspectionEdits)
	if !ok {
		t.Fatal("the document has no editing host")
	}
	return s
}

// objectRow is the list row for the nth drawable class.
func objectRow(t *testing.T, s *mapInspectionEdits, n int) ui.BrushObject {
	t.Helper()
	rows := s.BrushObjects()
	if len(rows) < 2 || rows[0].Code != 0 {
		t.Fatalf("the objects list %v lacks the clear row or any class", rows)
	}
	return rows[1+n%(len(rows)-1)]
}

// interiorCell is a cell whose radius-2 disc lies wholly inside the map.
func interiorCell(m *alm.Map) image.Point { return image.Pt(m.Width/2, m.Height/2) }

func TestReleaseMapEditorObjectListOffersOnlyDrawableClasses(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	rows := objectEdits(t, e).BrushObjects()
	if rows[0] != (ui.BrushObject{Code: 0}) {
		t.Fatalf("the first row is %v", rows[0])
	}
	offered := map[int]bool{}
	last := 0
	for _, r := range rows[1:] {
		c := x.statics.Classes[r.Code]
		if c == nil || c.Frame == nil || r.Code <= last {
			t.Fatalf("row %v is not a drawable class in placement order", r)
		}
		offered[r.Code], last = true, r.Code
	}
	hidden := 0
	for b, c := range x.statics.Classes {
		if c != nil && !offered[b] {
			hidden++
			if c.Frame != nil {
				t.Fatalf("class %d draws but is not offered", b)
			}
		}
	}
	t.Logf("the objects list holds the clear row and %d classes; %d registry classes have no art and stay off it", len(rows)-1, hidden)
	for _, r := range rows {
		t.Logf("row %3d %s", r.Code, r.Name)
	}
	if _, err := objectEdits(t, e).Paint(ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: 255, Cells: []image.Point{{X: 20, Y: 20}}}); err == nil && !offered[255] {
		t.Fatal("a code the list does not offer was painted")
	}
}

// A stroke on an installed map changes the saved file in the painted type-3
// cells only.
func TestReleaseMapEditorObjectStrokeChangesOnlyTheTypeThreeCells(t *testing.T) {
	x := installedInspector(t)
	source, err := x.archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	before, err := alm.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := objectEdits(t, e)
	row := objectRow(t, s, 3)
	centre := interiorCell(before)
	const radius = 2
	d, err := s.Paint(ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: row.Code, Radius: radius, Cells: []image.Point{centre}})
	if err != nil || d == nil {
		t.Fatalf("stroke: %v", err)
	}
	path := filepath.Join(t.TempDir(), "painted.alm")
	if _, err := s.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil || len(saved) != len(source) {
		t.Fatalf("saved %d bytes (%v), source %d", len(saved), err, len(source))
	}
	after, err := alm.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	var cells []int
	for i := range before.Overlay {
		cx, cy := i%before.Width-centre.X, i/before.Width-centre.Y
		in := cx*cx+cy*cy <= radius*radius
		want := before.Overlay[i]
		if in {
			want = byte(row.Code)
		}
		if after.Overlay[i] != want {
			t.Fatalf("overlay cell %d is %d, want %d", i, after.Overlay[i], want)
		}
		if after.Overlay[i] != before.Overlay[i] {
			cells = append(cells, i)
		}
	}
	if len(cells) == 0 {
		t.Fatal("the stroke changed no cell; pick a class other than the map's own")
	}
	var diffs []int
	for i := range source {
		if source[i] != saved[i] {
			diffs = append(diffs, i)
		}
	}
	if len(diffs) != len(cells) {
		t.Fatalf("%d bytes differ, %d overlay cells changed", len(diffs), len(cells))
	}
	base := diffs[0] - cells[0]
	for k, at := range diffs {
		if at-base != cells[k] {
			t.Fatalf("differing byte %d is not the type-3 cell %d", at, cells[k])
		}
	}
	t.Logf("class %d %q over a radius-%d disc at %v changed %d bytes, all type-3 cells (payload at %d)", row.Code, row.Name, radius, centre, len(diffs), base)

	// Undo restores the source bytes; the clear row empties the same disc.
	if _, err := s.Undo(); err != nil || !bytes.Equal(s.model.Bytes(), source) {
		t.Fatalf("Undo: %v, restored %v", err, bytes.Equal(s.model.Bytes(), source))
	}
	if _, err := s.Paint(ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: row.Code, Radius: radius, Cells: []image.Point{centre}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Paint(ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: 0, Radius: radius, Cells: []image.Point{centre}}); err != nil {
		t.Fatal(err)
	}
	cleared := alm.Map{}
	if m, err := alm.Open(s.model.Bytes()); err != nil {
		t.Fatal(err)
	} else {
		cleared = *m
	}
	for i := range cleared.Overlay {
		cx, cy := i%before.Width-centre.X, i/before.Width-centre.Y
		if cx*cx+cy*cy <= radius*radius && cleared.Overlay[i] != 0 {
			t.Fatalf("the clear row left cell %d at %d", i, cleared.Overlay[i])
		}
	}
	if bytes.Equal(s.model.Bytes(), source) {
		t.Fatal("clearing a disc of an installed map should not restore it")
	}
}

// A painted object blocks ground movement as the installed map's objects do,
// and the map report judges the drop cell and the units against it.
func TestReleaseMapEditorPaintedObjectBlocksLikeAnInstalledOne(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := objectEdits(t, e)
	m, err := alm.Open(s.model.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	// An installed object of each class blocks the ground in the derived plane.
	plain := mapload.Passability(m)
	seen := 0
	for i, b := range m.Overlay {
		if b != 0 {
			seen++
			if plain[i]&1 == 0 {
				t.Fatalf("installed object %d at cell %d does not block", b, i)
			}
		}
	}
	if seen == 0 {
		t.Fatal("the map places no object")
	}
	// Paint over open ground and read the plane of the saved map.
	centre := interiorCell(m)
	for plain[centre.Y*m.Width+centre.X] != 0 {
		centre.X++
	}
	row := objectRow(t, s, 5)
	if _, err := s.Paint(ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: row.Code, Cells: []image.Point{centre}}); err != nil {
		t.Fatal(err)
	}
	p, err := alm.Open(s.model.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := mapload.Passability(p)[centre.Y*p.Width+centre.X]; got&1 == 0 {
		t.Fatalf("the painted object %q at %v leaves the cell open (%d)", row.Name, centre, got)
	}
	// The report's unit and drop-cell rules read the same plane: put the drop
	// cell on the painted object and the map report names it.
	if _, err := s.SetDropCell(centre); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range s.Report() {
		found = found || f.Rule == "drop-cell" && f.Text != ""
	}
	if !found {
		t.Fatal("a drop cell on a painted object is not reported")
	}
}

// A band of painted objects across the party's way is loaded by the mission
// build at its cells, blocks them, and is walked around.
func TestReleaseMapEditorPaintedObjectsLoadAsAModMissionAndBlockTheWalk(t *testing.T) {
	x := installedInspector(t)
	plain := guestMods(t, guestModID)
	saveEditedGuestMap(t, x, plain, func(*ui.MapEditor) {})
	f := guestFrontWorld(t, plain)
	leader, partyStart := leaderOf(t, f)
	plainMap := f.live.mission.state.Map
	route := openRoute(t, plainMap, partyStart)
	goal := route.at(8, 0)
	var band []image.Point
	bandCell := map[image.Point]bool{}
	for j := -3; j <= 3; j++ {
		band = append(band, route.at(4, j))
		bandCell[route.at(4, j)] = true
	}
	if trail := walk(f, leader, route.start); len(trail) == 0 || trail[len(trail)-1] != route.start {
		t.Fatalf("the leader never reaches %v", route.start)
	}
	plainTrail := walk(f, leader, goal)
	crossed := false
	for _, c := range plainTrail {
		crossed = crossed || bandCell[c]
	}
	if len(plainTrail) == 0 || plainTrail[len(plainTrail)-1] != goal || !crossed {
		t.Fatalf("loss control: the unedited walk does not run through the band: %v", plainTrail)
	}

	dir := guestMods(t, guestModID)
	var row ui.BrushObject
	saveEditedGuestMap(t, x, dir, func(e *ui.MapEditor) {
		rows := objectEdits(t, e).BrushObjects()
		at := 1 + 21%(len(rows)-1)
		row = rows[at]
		e.ShowCell(route.at(4, 0))
		brushWindow(t, e, "objects", at)
		pointerStroke(e, band)
		if !e.Dirty() {
			t.Fatal("the stroke left the document clean")
		}
		if dir := os.Getenv("AGAINROM_WITNESS_DIR"); dir != "" {
			pic, _, err := e.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.Create(filepath.Join(dir, "objects-stroke.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, pic); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}
	})
	fp := guestFrontWorld(t, dir)
	pm := fp.live.mission.state.Map
	blocked := mapload.Passability(pm)
	for _, c := range band {
		if got := pm.Overlay[c.Y*pm.Width+c.X]; int(got) != row.Code {
			t.Fatalf("band cell %v holds object %d after loading, want %d (%s)", c, got, row.Code, row.Name)
		}
		if blocked[c.Y*pm.Width+c.X]&1 == 0 {
			t.Fatalf("band cell %v is open ground", c)
		}
	}
	changed := 0
	for i := range plainMap.Overlay {
		if plainMap.Overlay[i] != pm.Overlay[i] {
			changed++
			if !bandCell[image.Pt(i%pm.Width, i/pm.Width)] {
				t.Fatalf("overlay cell %d changed away from the band", i)
			}
		}
	}
	t.Logf("%q painted over %d cells changed %d overlay cells", row.Name, len(band), changed)
	id, _ := leaderOf(t, fp)
	if trail := walk(fp, id, route.start); len(trail) == 0 || trail[len(trail)-1] != route.start {
		t.Fatal("the leader never reaches the stretch's start on the painted map")
	}
	trail := walk(fp, id, goal)
	if len(trail) == 0 || trail[len(trail)-1] != goal {
		t.Fatalf("the leader never reaches %v around the painted objects: %v", goal, trail)
	}
	around := false
	for _, c := range trail {
		if bandCell[c] {
			t.Fatalf("the leader stands on painted object %v", c)
		}
		around = around || route.across(c) >= 4
	}
	if !around || len(trail) <= len(plainTrail) {
		t.Fatalf("the leader did not walk around the band: %d cells against %d", len(trail), len(plainTrail))
	}
}
