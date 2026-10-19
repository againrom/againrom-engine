package game

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

type usablePlacement struct {
	mission, id int
	kind        uint16
}

func shippedUsablePlacements(t *testing.T, f *FrontEnd) []usablePlacement {
	t.Helper()
	first := map[uint16]usablePlacement{}
	counts := map[uint16]int{}
	maps := 0
	for mission := 1; mission <= 160; mission++ {
		path, _ := MissionMap(mission)
		raw, err := f.Archives.Containers.ReadFile(path)
		if err != nil {
			continue
		}
		m, err := alm.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		maps++
		for id, o := range m.Objects {
			kind := uint16(byte(o.Kind))
			c := f.Structures.Classes[kind]
			if c == nil || !c.Usable {
				continue
			}
			if counts[kind] == 0 {
				first[kind] = usablePlacement{mission, id, kind}
			}
			counts[kind]++
		}
	}
	var placed []usablePlacement
	for _, kind := range []uint16{15, 16, 28, 29, 34, 35, 39, 40, 66} {
		maximum := 0
		if kind == 15 || kind == 16 {
			maximum = 3
		} else if kind == 28 || kind == 29 {
			maximum = 1
		}
		params := f.Table.Buildings.EntryParams(int(kind))
		if len(params) <= 3 || params[3] != int32(maximum) {
			t.Fatalf("class%d maximum=%v", kind, params)
		}
		t.Logf("census missions1..160 readable=%d class%d placements=%d first=%+v maximum=%d", maps, kind, counts[kind], first[kind], maximum)
		if counts[kind] > 0 {
			placed = append(placed, first[kind])
		}
	}
	return placed
}

func usableHoverProof(t *testing.T, f *FrontEnd, a *ui.App, kind uint16, id uint32) {
	t.Helper()
	if cursor, ok := a.HeadlessMapCursor(); !ok || cursor != "town" {
		t.Fatalf("class%d cursor=%q,%v", kind, cursor, ok)
	}
	pic, _, lines, err := a.HeadlessStructureReadout()
	if err != nil {
		t.Fatal(err)
	}
	s := f.live.world.Structures()[id]
	current, maximum := int(s.Field42), int(s.MaxHealth)
	if maximum == 0 {
		current, maximum = 1000, 1000
	}
	if lines[2] != fmt.Sprintf("%d/%d", current, maximum) {
		t.Fatalf("class%d readout=%q", kind, lines)
	}
	t.Logf("hover class%d structure%d tick%d cursor=town readout=%q", kind, id, f.live.world.Tick(), lines)
	if dir := os.Getenv("AGAINROM_USABLE_STRUCTURE_WITNESS_DIR"); dir != "" {
		dir, err = effectRimOutputPath(os.Getenv("AGAINROM_ASSETS"), dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		cursor, _, err := a.HeadlessMissionCursor()
		if err != nil {
			t.Fatal(err)
		}
		frame := image.NewRGBA(image.Rect(0, 0, max(pic.Bounds().Dx(), cursor.Bounds().Dx())+16, pic.Bounds().Dy()+cursor.Bounds().Dy()+24))
		draw.Draw(frame, pic.Bounds().Add(image.Pt(8, 8)), pic, pic.Bounds().Min, draw.Src)
		draw.Draw(frame, cursor.Bounds().Add(image.Pt(8, pic.Bounds().Dy()+16)), cursor, cursor.Bounds().Min, draw.Over)
		path := filepath.Join(dir, fmt.Sprintf("%s-class%d-hover.png", filepath.Base(os.Getenv("AGAINROM_ASSETS")), kind))
		out, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(out, frame)
		closeErr := out.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
}
