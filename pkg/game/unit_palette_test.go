package game

import (
	"fmt"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/pal"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func TestUnitPaletteBranchesUseSharedSingleAndUnitTables(t *testing.T) {
	integer := func(name string, value int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 2, Int: value}
	}
	colours := make([]color.RGBA, 256)
	colours[55] = color.RGBA{R: 19, G: 31, B: 47}
	source := chargenSource{
		UnitRegistry: synth.UnitsReg([]string{`test\sprites`},
			[]synth.RegNode{integer("ID", 1), integer("File", 0), integer("Palette", 0)},
			[]synth.RegNode{integer("ID", 2), integer("File", 0), integer("Palette", 1)},
			[]synth.RegNode{integer("ID", 3), integer("File", 0), integer("Palette", 4)}),
		graphicsPrefix + "units/test/sprites.256": synth.Sheet256(synth.Sheet256Options{
			Palette: colours,
			Frames: []synth.Frame256{{Width: 1, Height: 1,
				Pixels: []synth.Pixel256{{Index: 55, Opaque: true}}}},
		}),
		UnitOwnerPalette: make([]byte, 16*1024),
	}
	for table := 0; table < 16; table++ {
		e := source[UnitOwnerPalette][table*1024+55*4:]
		e[0], e[1], e[2] = byte(90+table), byte(60+table), byte(30+table)
	}
	for table := 1; table <= 4; table++ {
		colours[55] = color.RGBA{R: byte(100 + table), G: byte(120 + table), B: byte(140 + table)}
		raw := make([]byte, pal.TableOffset)
		raw[0], raw[1] = 'B', 'M'
		name := "palette.pal"
		if table > 1 {
			name = fmt.Sprintf("palette%d.pal", table)
		}
		source[graphicsPrefix+"units/test/"+name] = append(raw, synth.Palette256(colours)...)
	}
	set, err := LoadUnits(source)
	if err != nil {
		t.Fatal(err)
	}
	mw := &mapWorld{units: set}
	for _, tc := range []struct {
		class int32
		tier  int
		owner uint32
		want  color.RGBA
	}{
		{1, 0, 2, color.RGBA{R: 32, G: 62, B: 92, A: 255}},
		{1, 3, 9, color.RGBA{R: 39, G: 69, B: 99, A: 255}},
		{2, 0, 2, color.RGBA{R: 101, G: 121, B: 141, A: 255}},
		{2, 1, 9, color.RGBA{R: 101, G: 121, B: 141, A: 255}},
		{2, 4, 2, color.RGBA{R: 101, G: 121, B: 141, A: 255}},
		{3, 1, 2, color.RGBA{R: 101, G: 121, B: 141, A: 255}},
		{3, 4, 9, color.RGBA{R: 104, G: 124, B: 144, A: 255}},
	} {
		c := set.Classes[tc.class]
		frame := mw.ownerFrame(c, c.TierFrames(tc.tier)[0], tc.owner, false)
		if got := frame.RGBA().RGBAAt(0, 0); got != tc.want {
			t.Errorf("class=%d tier=%d owner=%d: got %v, want %v", tc.class, tc.tier, tc.owner, got, tc.want)
		}
	}
}

func TestStonePaletteUsesTableZeroBeforeBrightness(t *testing.T) {
	for _, ownerShaded := range []bool{false, true} {
		t.Run(fmt.Sprint(ownerShaded), func(t *testing.T) {
			class := stoneOwnerClass()
			class.OwnerShaded = ownerShaded
			set := stoneOwnerSet(class)
			set.OwnerPalettes[0][55] = color.RGBA{R: 200, G: 100, B: 20, A: 255}
			first := ownerShadeFrame()
			first.Palette[55] = color.RGBA{R: 200, G: 100, B: 20, A: 255}
			last := ownerShadeFrame()
			last.Palette[55] = color.RGBA{B: 240, A: 255}
			class.Tiers = [][]*terrain.StaticFrame{{first}, {last}}
			_, mw := stoneOwnerWorld(t, set, 2, false)
			mw.tiers = map[sim.EntityID]int{2: 2}
			draw := stoneOwnerTarget(t, mw)
			for row, value := range map[int]byte{0: 213, 8: 106, 15: 13} {
				want := color.RGBA{R: value, G: value, B: value, A: 255}
				if got := draw.Frame.RGBALit([3]uint8{99, 27, 51}, row).RGBAAt(0, 0); got != want {
					t.Errorf("owner arm=%t row=%d: got %v, want %v from source table 0 with tint off", ownerShaded, row, got, want)
				}
			}
		})
	}
}

func TestStonePaletteCacheKeepsDistinctTypeSources(t *testing.T) {
	frame := ownerShadeFrame()
	other := ownerShadeFrame()
	other.Palette[55] = color.RGBA{R: 60, G: 30, B: 9, A: 255}
	owner := &terrain.UnitClass{OwnerShaded: true}
	typed := &terrain.UnitClass{Tiers: [][]*terrain.StaticFrame{{other}}}
	set := &terrain.UnitSet{HasOwnerPalettes: true}
	set.OwnerPalettes[0][55] = color.RGBA{R: 200, G: 100, B: 20, A: 255}
	mw := &mapWorld{units: set}
	for _, c := range []*terrain.UnitClass{owner, typed, owner, typed} {
		want := byte(106)
		if c == typed {
			want = 33
		}
		got := mw.ownerFrame(c, frame, 2, true).RGBA().RGBAAt(0, 0)
		if got != (color.RGBA{R: want, G: want, B: want, A: 255}) {
			t.Fatalf("class %p reused another mode-5 source: %v", c, got)
		}
	}
	if frame.Grayscale || frame.Palette[55].R != 1 || other.Grayscale {
		t.Fatal("mode-5 selection mutated a source frame")
	}
}
