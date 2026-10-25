package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const newMapModID = "newmap"

// newMapMod writes, in a fresh mods folder, a mod that replaces campaign
// mission 20 with a map file it does not hold yet. It returns the mods folder.
func newMapMod(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"mod.toml": "id = \"" + newMapModID + "\"\ntitle = \"New map\"\nversion = \"1.0.0\"\nauthor = \"Againrom\"\n" +
			"description = \"Mission 20 on a new map.\"\napi = 1\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n",
		"main.star":          "def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n",
		"data/missions.toml": "[[mission]]\nkind = \"replace\"\nkey = \"fresh\"\nnumber = 20\nmap = \"mission/map.alm\"\n",
	}
	for name, body := range files {
		p := filepath.Join(root, newMapModID, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// creatureClass is a creature class key of the installed units.
func creatureClass(t *testing.T, x *MapInspector) int16 {
	t.Helper()
	raw, err := x.archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Units {
		if u.ClassID >= 0x1a && mapload.Resolve(u, x.table).Found() {
			return u.ClassID
		}
	}
	t.Fatal("mission 10 places no creature that resolves")
	return 0
}

func TestReleaseMapEditorListsAndOpensEveryNewMapSize(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	rows := newMapRows()
	if len(rows) != len(newMapSides) || rows[0].Key != "new/32x32" {
		t.Fatalf("new map rows %+v", rows)
	}
	for _, side := range newMapSides {
		e := x.Editor()
		if err := e.Open(newMapSource(side, side)); err != nil {
			t.Fatalf("%dx%d: %v", side, side, err)
		}
		d := e.Document()
		if d.Width != side || d.Height != side || e.Dirty() {
			t.Fatalf("%dx%d opened as %dx%d, dirty %v", side, side, d.Width, d.Height, e.Dirty())
		}
		groups, units := 0, 0
		for _, r := range d.Records {
			switch r.Kind {
			case "Player":
				groups++
			case "Unit":
				units++
			}
		}
		if groups != 2 || units != 0 {
			t.Fatalf("%dx%d holds %d roster entries and %d units", side, side, groups, units)
		}
		// The drop cell at the centre is a start point; nothing wins the map
		// yet, which is its one finding.
		rows := d.Edits.(ui.InspectionChecker).Report()
		if len(rows) != 1 || rows[0].Error || rows[0].Rule != ui.RuleVictory {
			t.Errorf("%dx%d: report %+v", side, side, rows)
		}
	}
	for _, bad := range []string{"new/31x32", "new/32", "new/512x512", "new/32x32x"} {
		if _, err := x.Open(bad); err == nil {
			t.Errorf("%s opened", bad)
		}
	}
}

func TestReleaseMapEditorNewMapPlaysAsModMission(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	root := newMapMod(t)
	modDir := filepath.Join(root, newMapModID)
	x := installedInspector(t)
	if err := x.SetMod(modDir); err != nil {
		t.Fatal(err)
	}
	class := creatureClass(t, x)
	target := filepath.Join(modDir, "mission", "map.alm")
	const side = 32
	drop, foe := image.Pt(16, 16), image.Pt(22, 22)

	e := x.Editor()
	if err := e.Open(newMapSource(side, side)); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	s := e.Document().Edits.(*mapInspectionEdits)

	// The new map's drop cell is its centre.
	if m, err := s.model.Map(); err != nil || !slices.Equal(mapload.DropCells(m), []mapload.Cell{{X: 16, Y: 16}}) {
		t.Fatalf("the new map's drop cell: %v", err)
	}
	if d, err := s.AddUnit(foe, int64(class)); err != nil || d == nil {
		t.Fatalf("add a unit: %v", err)
	}
	// A second unit is placed and undone; the history holds the edits.
	if d, err := s.AddUnit(image.Pt(24, 24), int64(class)); err != nil || d == nil {
		t.Fatalf("add a second unit: %v", err)
	}
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("the map exists before it was saved")
	}
	if err := pastNotice(e, "saveas"); err != nil || e.Dirty() {
		t.Fatalf("Save into the mod: %v", err)
	}
	saved, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(saved, s.model.Bytes()) {
		t.Fatalf("the mod holds %d bytes, the model %d (%v)", len(saved), len(s.model.Bytes()), err)
	}
	m, err := alm.Open(saved)
	if err != nil || len(m.Units) != 1 || m.Units[0].ClassID != class || m.Units[0].Owner != 2 ||
		int(m.Units[0].X>>8) != foe.X || int(m.Units[0].Y>>8) != foe.Y {
		t.Fatalf("the saved map holds %v (%v)", m.Units, err)
	}
	if cells := mapload.DropCells(m); len(cells) != 1 || cells[0] != (mapload.Cell{X: 16, Y: 16}) {
		t.Fatalf("drop cells %v", cells)
	}
	if m.Width != side || m.Height != side || len(m.Groups) != 2 {
		t.Fatalf("saved map %dx%d with %d roster entries", m.Width, m.Height, len(m.Groups))
	}

	// The saved map opens in the editor as an installed map does, and Save As
	// writes the same bytes again.
	e2 := x.Editor()
	if err := e2.Open(target); err != nil {
		t.Fatal(err)
	}
	units := 0
	for _, r := range e2.Document().Records {
		if r.Kind == "Unit" {
			units++
		}
	}
	if units != 1 || e2.Dirty() || !bytes.Equal(e2.Document().SourceBytes, saved) {
		t.Fatalf("reopened: %d units, dirty %v", units, e2.Dirty())
	}
	for _, f := range e2.Document().Edits.(ui.InspectionChecker).Report() {
		if f.Rule != ui.RuleVictory {
			t.Errorf("the saved map reports %s", f.Line())
		}
	}
	x2, err := NewMapInspector(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	e3 := x2.Editor()
	if err := e3.Open(target); err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(t.TempDir(), "copy.alm")
	_ = e3.Key("saveas")
	e3.TextInput(copyPath)
	if err := e3.Key("enter"); err != nil {
		t.Fatalf("Save As of the reopened map: %v", err)
	}
	if got, err := os.ReadFile(copyPath); err != nil || !bytes.Equal(got, saved) {
		t.Fatalf("Save As of the reopened map wrote other bytes (%v)", err)
	}

	// Play: the party stands at the drop cell, the unit stands where it was
	// placed, and a move order is carried out.
	f := guestFront(t, root, newMapModID)
	app := f.App("new map")
	app.Layout(1024, 768)
	party := f.ChargenParty(ui.ChargenResult{Name: "Fresh", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	if live.mission.state.Map.Width != side || len(live.mission.state.Map.Units) != 1 {
		t.Fatal("mission 20 was not built from the new map")
	}
	hero := live.mission.ids[0]
	he, _ := live.entity(hero)
	if he.X != int32(drop.X) || he.Y != int32(drop.Y) || he.Owner != 1 {
		t.Fatalf("the party stands at %d,%d owner %d, want the drop cell %v", he.X, he.Y, he.Owner, drop)
	}
	var enemies []sim.Entity
	for _, en := range live.world.Entities() {
		if en.MapUnitID == 1 {
			enemies = append(enemies, en)
		}
	}
	if len(enemies) != 1 || enemies[0].X != int32(foe.X) || enemies[0].Y != int32(foe.Y) || enemies[0].Owner != 2 || enemies[0].Class != int32(class) {
		t.Fatalf("the placed unit plays as %+v", enemies)
	}

	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	// A ground point whose cell is open ground inside the border, a few
	// cells from the party and clear of the placed unit.
	gx, gy, cx, cy := -1, -1, 0, 0
	for y := 4; y < 768 && gx < 0; y += 4 {
		for x := 4; x < 1024-160 && gx < 0; x += 4 {
			c, r, err := app.HeadlessDropCell(x, y)
			if err != nil || c < 10 || r < 10 || c > 21 || r > 21 {
				continue
			}
			if d := attackMoveDist(he, int32(c), int32(r)); d >= 4 && d <= 6 && attackMoveDist(enemies[0], int32(c), int32(r)) >= 4 {
				gx, gy, cx, cy = x, y, c, r
			}
		}
	}
	if gx < 0 {
		t.Fatal("no ground point 4 to 6 cells from the party")
	}
	for _, act := range []string{"press", "release"} {
		if err := app.HeadlessPointer(act, gx, gy); err != nil {
			t.Fatal(err)
		}
	}
	if !live.commanded[hero] {
		t.Fatal("the click queued no order")
	}
	start := attackMoveDist(he, int32(cx), int32(cy))
	arrived := false
	for n := 0; n < 3000 && !arrived; n++ {
		if err := attackMoveStep(app); err != nil {
			t.Fatal(err)
		}
		at, _ := live.entity(hero)
		arrived = at.Alive() && attackMoveDist(at, int32(cx), int32(cy)) <= attackMoveArrival
	}
	at, _ := live.entity(hero)
	if !arrived {
		t.Fatalf("the party never reached %d,%d (from %d away); it stands at %d,%d", cx, cy, start, at.X, at.Y)
	}
	t.Logf("the party walked from the drop cell %v to %d,%d (%d cells)", drop, at.X, at.Y, start)

	// Loss control: without the mod the same route builds the installed map.
	plain := guestFront(t, "")
	plain.App("installed").Layout(1024, 768)
	if err := plain.App("installed 20").OpenMission(plain.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	if plain.live.mission.state.Map.Width == side {
		t.Fatal("the installed mission 20 is the new map")
	}
}

func TestReleaseMapEditorBlankWriteIsTheConstructorsBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	path := filepath.Join(t.TempDir(), "blank.alm")
	got, findings, err := x.WriteNewMap(newMapSource(32, 32), path)
	if err != nil || got == "" {
		t.Fatalf("write: %v", err)
	}
	// The constructor's map in single-player mode with its drop cell at the
	// centre (DIV-3597).
	want, err := mapedit.NewMap(alm.NewMapSpec{Width: 32, Height: 32, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	if err := want.SetWord70(ui.MapModeSingle); err != nil {
		t.Fatal(err)
	}
	if err := want.SetDropCell(16, 16); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(got)
	if err != nil || !bytes.Equal(written, want.Bytes()) {
		t.Fatalf("the file is not the constructor's bytes with the mode and the drop cell (%v)", err)
	}
	if len(findings) != 1 || findings[0].Rule != ui.RuleVictory {
		t.Fatalf("findings %+v", findings)
	}
	if _, _, err := x.WriteNewMap(newMapSource(32, 32), path); err == nil {
		t.Fatal("a second write replaced the file")
	}
}

// Loss control: the new-map door changes nothing for an installed map. It
// opens clean, a Save As writes its installed bytes, and the writer refuses
// the source it came from.
func TestReleaseMapEditorInstalledMapStillWritesItsInstalledBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	installed, err := x.archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, installed) {
		t.Fatal("the installed map opened changed")
	}
	target := filepath.Join(t.TempDir(), "copy.alm")
	_ = e.Key("saveas")
	e.TextInput(target)
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, installed) {
		t.Fatalf("Save As wrote other bytes than the installed map's (%v)", err)
	}
}
