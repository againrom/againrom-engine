//go:build release

package game

import (
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// Two units and a structure of a second-game map are copied, pasted on
// another cell, given a group, saved and loaded again.
func TestReleaseSecondMapCopiesPastesAndGroups(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	cellOf := func(f secondFacts, u int) image.Point { return image.Pt(int(f.ux(u)>>8), int(f.uy(u)>>8)) }
	pick, ua, ub, st := -1, 0, 0, 0
	for i, raw := range raws {
		f := readSecondFacts(t, raw)
		if f.width < 32 || f.height < 32 {
			continue
		}
		s := -1
		for j := range f.structs {
			if c := image.Pt(int(f.structX[j]>>8), int(f.structY[j]>>8)); c.X > 10 && c.Y > 10 && c.X < f.width-12 && c.Y < f.height-12 {
				s = j
				break
			}
		}
		inside := func(u int) bool {
			c := cellOf(f, u)
			return c.X > 0 && c.Y > 0 && c.X < f.width-3 && c.Y < f.height-3
		}
		a := -1
		for u := range f.units {
			if !inside(u) {
				continue
			}
			if a < 0 {
				a = u
				continue
			}
			owner := binary.LittleEndian.Uint32
			if owner(f.units[u][alm.SecondUnitOwner:]) == owner(f.units[a][alm.SecondUnitOwner:]) && s >= 0 && (f.link(a) != 0 || f.link(u) != 0 || pick < 0) {
				pick, ua, ub, st = i, a, u, s
				if f.link(a) != 0 || f.link(u) != 0 {
					break
				}
			}
		}
		if pick == i && (f.link(ua) != 0 || f.link(ub) != 0) {
			break
		}
	}
	if pick < 0 {
		t.Fatal("no installed map holds two units of one owner and a structure inside its edge")
	}
	source := sources[pick]
	was := readSecondFacts(t, raws[pick])
	d, err := x.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	if _, ok := ui.Seam[ui.InspectionSelection](host); !ok {
		t.Fatal("the second game's row offers no selection")
	}
	if _, ok := ui.Seam[ui.InspectionGroups](host); !ok {
		t.Fatal("the second game's row offers no groups")
	}
	refs := []ui.RecordRef{{Kind: ui.KindUnit, Index: ua}, {Kind: ui.KindUnit, Index: ub}, {Kind: ui.KindStructure, Index: st}}
	clip, _, err := host.CopyRecords(refs)
	if err != nil {
		t.Fatal(err)
	}
	origin := image.Pt(1<<30, 1<<30)
	for _, u := range []int{ua, ub} {
		origin = image.Pt(min(origin.X, cellOf(was, u).X), min(origin.Y, cellOf(was, u).Y))
	}
	if c := (image.Pt(int(was.structX[st]>>8), int(was.structY[st]>>8))); true {
		origin = image.Pt(min(origin.X, c.X), min(origin.Y, c.Y))
	}
	target := origin.Add(image.Pt(1, 1))
	out, err := host.PasteRecords(clip, target)
	if err != nil || out.Doc == nil || len(out.Selected) != 3 || len(out.Refusals) != 0 {
		t.Fatalf("paste: %+v, %v", out, err)
	}
	var units []ui.RecordRef
	for _, r := range out.Selected {
		if r.Kind == ui.KindUnit {
			units = append(units, r)
		}
	}
	group := host.FreeGroup()
	if group < 1 {
		t.Fatalf("free group %d", group)
	}
	if _, err := host.SetUnitsGroup(units, group); err != nil {
		t.Fatal(err)
	}
	path, err := host.SaveAs(filepath.Join(t.TempDir(), "pasted.alm"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := readSecondFacts(t, saved)
	checkSecondFacts(t, "saved", got)
	if len(got.units) != len(was.units)+2 || len(got.structs) != len(was.structs)+1 {
		t.Fatalf("units %d structures %d, was %d %d", len(got.units), len(got.structs), len(was.units), len(was.structs))
	}
	sacks := b2i(was.link(ua) != 0) + b2i(was.link(ub) != 0)
	if sacks == 0 {
		t.Fatal("no copied unit holds a sack")
	}
	if len(got.sackOwner) != len(was.sackOwner)+sacks {
		t.Errorf("sacks %d, want %d", len(got.sackOwner), len(was.sackOwner)+sacks)
	}
	le := binary.LittleEndian
	for k, src := range []int{ua, ub} {
		n := len(was.units) + k
		c := cellOf(was, src).Add(target.Sub(origin))
		if got.ux(n) != uint32(c.X)<<8|0x80 || got.uy(n) != uint32(c.Y)<<8|0x80 {
			t.Errorf("pasted unit %d at %d,%d, want cell %v", n, got.ux(n), got.uy(n), c)
		}
		if g := le.Uint32(got.units[n][alm.SecondUnitGroup:]); g != uint32(group) {
			t.Errorf("pasted unit %d group %d, want %d", n, g, group)
		}
		if (was.link(src) != 0) != (got.link(n) != 0) {
			t.Errorf("pasted unit %d sack link %d, source %d", n, got.link(n), was.link(src))
		}
	}
	sc := image.Pt(int(was.structX[st]>>8), int(was.structY[st]>>8)).Add(target.Sub(origin))
	last := len(got.structs) - 1
	if got.structX[last] != uint32(sc.X)<<8|0x80 || got.structY[last] != uint32(sc.Y)<<8|0x80 {
		t.Errorf("pasted structure at %d,%d, want cell %v", got.structX[last], got.structY[last], sc)
	}
	members := 0
	for u := range got.units {
		if le.Uint32(got.units[u][alm.SecondUnitGroup:]) == uint32(group) {
			members++
		}
	}
	if members != 2 {
		t.Errorf("group %d holds %d units", group, members)
	}
	reloaded, err := x.Open(path)
	if err != nil {
		t.Fatalf("the saved map does not reload: %v", err)
	}
	count := map[ui.RecordKind]int{}
	for _, r := range reloaded.Records {
		count[r.Kind]++
	}
	if count[ui.KindUnit] != len(got.units) || count[ui.KindStructure] != len(got.structs) {
		t.Errorf("reloaded units %d structures %d", count[ui.KindUnit], count[ui.KindStructure])
	}
	t.Logf("PASS: %s pasted two units and a structure by %v, grouped them as %d, saved and reloaded (%s)", source, target.Sub(origin), group, installLanguage(x))
}

// A structure of a second-game map pasted or moved onto a cell within 8 cells
// of the edge is refused, as the single placement refuses it; the cells just
// inside stay allowed, and the saved map holds no refused structure.
func TestReleaseSecondMapStructureEdgeRuleHoldsForPasteAndMove(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	pick, st := -1, -1
	for i, source := range sources {
		if filepath.Base(source) != "havec_2.alm" {
			continue
		}
		f := readSecondFacts(t, raws[i])
		for j := range f.structs {
			if c := image.Pt(int(f.structX[j]>>8), int(f.structY[j]>>8)); c.X > 10 && c.Y > 10 && c.X < f.width-12 && c.Y < f.height-12 {
				pick, st = i, j
				break
			}
		}
	}
	if pick < 0 {
		t.Skip("no map havec_2.alm with a structure inside its edge")
	}
	was := readSecondFacts(t, raws[pick])
	w, h := was.width, was.height
	home := image.Pt(int(was.structX[st]>>8), int(was.structY[st]>>8))
	var cells []image.Point
	for _, v := range []int{7, 8, w - 10, w - 9, w - 8} {
		cells = append(cells, image.Pt(v, home.Y))
	}
	for _, v := range []int{7, 8, h - 10, h - 9, h - 8} {
		cells = append(cells, image.Pt(home.X, v))
	}
	allowed := func(c image.Point) bool { return c.X >= 8 && c.Y >= 8 && c.X+1 < w-8 && c.Y+1 < h-8 }
	for _, c := range cells {
		for _, how := range []string{"paste", "move"} {
			d, err := x.Open(sources[pick])
			if err != nil {
				t.Fatal(err)
			}
			host := d.Edits.(*mapInspectionEdits)
			ref := ui.RecordRef{Kind: ui.KindStructure, Index: st}
			var out ui.SelectionEdit
			if how == "paste" {
				clip, _, err := host.CopyRecords([]ui.RecordRef{ref})
				if err != nil {
					t.Fatal(err)
				}
				out, err = host.PasteRecords(clip, c)
				if err != nil && allowed(c) {
					t.Fatalf("%s to %v: %v", how, c, err)
				}
			} else {
				out, err = host.MoveRecords([]ui.RecordRef{ref}, c.Sub(home))
				if err != nil && allowed(c) {
					t.Fatalf("%s to %v: %v", how, c, err)
				}
			}
			refused := err != nil || len(out.Refusals) != 0 || len(out.Selected) == 0
			if refused == allowed(c) {
				t.Errorf("%s to %v of %dx%d: refused %v, want %v (refusals %v, err %v)", how, c, w, h, refused, !allowed(c), out.Refusals, err)
			}
			path, err := host.SaveAs(filepath.Join(t.TempDir(), "edge.alm"))
			if err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got := readSecondFacts(t, saved)
			checkSecondFacts(t, "saved", got)
			for j := range got.structs {
				at := image.Pt(int(got.structX[j]>>8), int(got.structY[j]>>8))
				if j >= len(was.structs) || at != image.Pt(int(was.structX[j]>>8), int(was.structY[j]>>8)) {
					if !allowed(at) {
						t.Errorf("%s to %v: the saved map holds structure %d at %v within 8 cells of the edge", how, c, j, at)
					}
				}
			}
		}
	}
}
