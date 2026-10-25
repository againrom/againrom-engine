package game

import (
	"image"
	"os"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// recordPicks are the installed map's records the release test edits: the
// structure M moves from Old to New, the structure D is deleted, and a sack
// of Item is added later at a cell next to the hero.
type recordPicks struct {
	M, D     int
	Old, New image.Rectangle
	Kind     uint16
	DAnchor  image.Point
	Item     uint16
}

func footprintRect(s sim.Structure) image.Rectangle {
	return image.Rect(int(s.Col), int(s.Row), int(s.Col)+int(s.Width), int(s.Row)+int(s.Height))
}

// blockingCells are the cells of s's footprint whose Blocking bit is set.
func blockingCells(s sim.Structure, at image.Point) []image.Point {
	var out []image.Point
	for dy := 0; dy < int(s.Height); dy++ {
		for dx := 0; dx < int(s.Width); dx++ {
			if s.Blocking>>uint((dy*int(s.Width)+dx)&31)&1 != 0 {
				out = append(out, at.Add(image.Pt(dx, dy)))
			}
		}
	}
	return out
}

// pickRecords chooses, by rule, the smallest structure whose footprint
// blocks, stands on open ground and has no other structure within two cells
// and no unit or sack within four, and the
// nearest shift along X that lands it on open ground no record touches; and
// another plain structure with a unique anchor to delete.
func pickRecords(t *testing.T, m *alm.Map, table *mapload.Table) recordPicks {
	t.Helper()
	structs := mapload.Structures(m, table)
	plane := mapload.Passability(m)
	taken := map[image.Point]bool{}
	for _, u := range m.Units {
		taken[image.Pt(int(u.X>>8), int(u.Y>>8))] = true
	}
	if l, err := m.Loot(); err == nil {
		for _, r := range l.Records {
			taken[image.Pt(int(r.CellX()), int(r.CellY()))] = true
		}
	}
	inner := image.Rect(2, 2, m.Width-2, m.Height-2)
	open := func(r image.Rectangle, skip int) bool {
		if !r.In(inner) {
			return false
		}
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if plane[y*m.Width+x]&1 != 0 {
					return false
				}
			}
		}
		for c := range taken {
			if c.In(r.Inset(-4)) {
				return false // no unit or sack within four cells
			}
		}
		for j, s := range structs {
			if j != skip && s.Width > 0 && footprintRect(s).Overlaps(r.Inset(-2)) {
				return false
			}
		}
		return true
	}
	p := recordPicks{M: -1, D: -1}
	best := 1 << 30
	for i, s := range structs {
		o := m.Objects[i]
		area := int(s.Width) * int(s.Height)
		if o.Kind == 0x21 || o.Kind == 34 || o.Kind == 35 || area == 0 || area >= best || len(blockingCells(s, image.Point{})) == 0 || !open(footprintRect(s), i) {
			continue
		}
		for dx := int(s.Width) + 3; dx < int(s.Width)+16; dx++ {
			if r := footprintRect(s).Add(image.Pt(dx, 0)); open(r, i) {
				p.M, p.Old, p.New, p.Kind, best = i, footprintRect(s), r, s.Kind, area
				break
			}
		}
	}
	anchors := map[image.Point]int{}
	for _, s := range structs {
		anchors[image.Pt(int(s.Col), int(s.Row))]++
	}
	for i, s := range structs {
		o := m.Objects[i]
		a := image.Pt(int(s.Col), int(s.Row))
		if i != p.M && o.Kind != 0x21 && o.Kind != 34 && o.Kind != 35 && s.Width > 0 && anchors[a] == 1 && !a.In(p.New) {
			p.D, p.DAnchor = i, a
			break
		}
	}
	if p.M < 0 || p.D < 0 {
		t.Fatalf("no structure to move (%d) or delete (%d)", p.M, p.D)
	}
	if l, err := m.Loot(); err == nil && len(l.Records) > 0 && len(l.Records[0].Elements) > 0 {
		p.Item = l.Records[0].Elements[0].ItemCode()
	}
	if p.Item == 0 {
		var codes []int
		for c := range table.Names {
			codes = append(codes, int(c))
		}
		slices.Sort(codes)
		p.Item = uint16(codes[0])
	}
	return p
}

// openGuestHero opens the mod's guest mission over the mods folder dir and
// returns the front end and its starting hero.
func openGuestHero(t *testing.T, dir string) (*FrontEnd, sim.EntityID) {
	t.Helper()
	f := guestFront(t, dir, guestModID)
	app := f.App("edited guest mission")
	party := f.ChargenParty(ui.ChargenResult{Name: "Guest", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	if err := app.OpenMission(f.MissionOpenerWith(guestMissionNo, party)); err != nil {
		t.Fatal(err)
	}
	return f, equipmentReturnHero(t, f)
}

// sackCell is the nearest cell to the hero, ring by ring, on open ground
// that no entity, sack or footprint in avoid touches.
func sackCell(t *testing.T, f *FrontEnd, hero sim.EntityID, m *alm.Map, avoid []image.Rectangle) image.Point {
	t.Helper()
	h, _ := f.live.world.Entity(hero)
	plane := mapload.Passability(m)
	busy := map[image.Point]bool{}
	for _, e := range f.live.world.Entities() {
		busy[image.Pt(int(e.X), int(e.Y))] = true
	}
	for _, s := range f.live.world.Sacks() {
		busy[image.Pt(int(s.X), int(s.Y))] = true
	}
	for _, s := range f.live.world.Structures() {
		avoid = append(avoid, footprintRect(s))
	}
	for r := 2; r < 8; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				c := image.Pt(int(h.X)+dx, int(h.Y)+dy)
				if !c.In(image.Rect(1, 1, m.Width-1, m.Height-1)) || plane[c.Y*m.Width+c.X]&1 != 0 || busy[c] {
					continue
				}
				if !slices.ContainsFunc(avoid, func(a image.Rectangle) bool { return c.In(a) }) {
					return c
				}
			}
		}
	}
	t.Fatal("no open cell near the hero")
	return image.Point{}
}

// walksTo orders hero to c and runs ordinary ticks until it stands there or
// its route has ended and it has stood still for 40 ticks; it reports
// whether the hero stands on c.
func walksTo(f *FrontEnd, hero sim.EntityID, c image.Point) bool {
	f.live.pending = append(f.live.pending, sim.MoveTo(hero, sim.CellPoint{X: int32(c.X), Y: int32(c.Y)}))
	var last sim.Entity
	still := 0
	for tick := 0; tick < 4000 && still < 40; tick++ {
		f.live.tick()
		e, _ := f.live.world.Entity(hero)
		if e.X == int32(c.X) && e.Y == int32(c.Y) {
			return true
		}
		if e.X == last.X && e.Y == last.Y && len(f.live.world.Route(hero)) == 0 {
			still++
		} else {
			still = 0
		}
		last = e
	}
	return false
}

func structureAt(ss []sim.Structure, a image.Point) []sim.Structure {
	var out []sim.Structure
	for _, s := range ss {
		if int(s.Col) == a.X && int(s.Row) == a.Y {
			out = append(out, s)
		}
	}
	return out
}

func TestReleaseMapEditorStructuresAndSacksPlayAsModMission(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	plain := guestMods(t, guestModID)
	edited := guestMods(t, guestModID)
	x, err := NewMapInspector(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := pickRecords(t, m, x.table)

	// Loss control: the unedited map through the same route.
	saveEditedGuestMap(t, x, plain, func(*ui.MapEditor) {})
	f, hero := openGuestHero(t, plain)
	before := f.live.world.Structures()
	moved := structureAt(before, p.Old.Min)
	if len(moved) != 1 || moved[0].Kind != p.Kind || len(structureAt(before, p.DAnchor)) != 1 {
		t.Fatalf("loss control: structures at %v %v and %v", p.Old.Min, moved, structureAt(before, p.DAnchor))
	}
	cell := sackCell(t, f, hero, m, []image.Rectangle{p.Old, p.New})
	if groundAt(f.live.world.Sacks(), int32(cell.X), int32(cell.Y)) != nil {
		t.Fatal("loss control: a sack already lies on the chosen cell")
	}
	for _, c := range blockingCells(moved[0], p.Old.Min) {
		if walksTo(f, hero, c) {
			t.Fatalf("loss control: the unedited structure does not block %v", c)
		}
	}
	for y := p.New.Min.Y; y < p.New.Max.Y; y++ {
		for x := p.New.Min.X; x < p.New.Max.X; x++ {
			if !walksTo(f, hero, image.Pt(x, y)) {
				t.Fatalf("loss control: destination cell %d,%d is closed before the move", x, y)
			}
		}
	}
	t.Logf("picks %+v sack cell %v", p, cell)

	saveEditedGuestMap(t, x, edited, func(e *ui.MapEditor) {
		docs := e.Document().Edits
		apply := func(d *ui.InspectionDocument, err error) {
			t.Helper()
			if err != nil || d == nil {
				t.Fatalf("edit refused: %v", err)
			}
		}
		apply(docs.Move("Structure", p.M, p.New.Min))
		apply(docs.Delete("Structure", p.D))
		apply(docs.(*mapInspectionEdits).AddSack(cell, int64(p.Item)))
	})
	f, hero = openGuestHero(t, edited)
	after := f.live.world.Structures()
	now := structureAt(after, p.New.Min)
	if len(now) != 1 || now[0].Kind != p.Kind || footprintRect(now[0]) != p.New {
		t.Fatalf("moved structure at %v: %+v", p.New.Min, now)
	}
	if len(structureAt(after, p.Old.Min)) != 0 || len(structureAt(after, p.DAnchor)) != 0 || len(after) != len(before)-1 {
		t.Fatalf("world holds %d structures (want %d), at old anchor %v, at deleted anchor %v",
			len(after), len(before)-1, structureAt(after, p.Old.Min), structureAt(after, p.DAnchor))
	}
	s := groundAt(f.live.world.Sacks(), int32(cell.X), int32(cell.Y))
	if s == nil || len(s.ItemInstances) != 1 || s.ItemInstances[0].Code != p.Item {
		t.Fatalf("added sack %+v, want item %#x at %v", s, p.Item, cell)
	}
	carried := func() int {
		items, _ := f.live.world.CarriedItems(hero)
		n := 0
		for _, it := range items {
			if it.Code == p.Item {
				n++
			}
		}
		return n
	}
	had := carried()
	packStackPickUp(t, f, hero, int32(cell.X), int32(cell.Y))
	if carried() != had+1 {
		t.Fatalf("the hero carries %d of item %#x after the pickup, want %d", carried(), p.Item, had+1)
	}
	for _, c := range blockingCells(now[0], p.New.Min) {
		if walksTo(f, hero, c) {
			t.Fatalf("the moved structure does not block %v", c)
		}
	}
	for y := p.Old.Min.Y; y < p.Old.Max.Y; y++ {
		for x := p.Old.Min.X; x < p.Old.Max.X; x++ {
			if !walksTo(f, hero, image.Pt(x, y)) {
				t.Fatalf("old cell %d,%d is still closed after the move", x, y)
			}
		}
	}
}
