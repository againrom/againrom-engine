package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// GroundArms is zero exactly where PassabilityWith leaves a ground mover free,
// names each terrain arm of a closed cell, and names the structure alone on a
// cell a footprint attached. Its control is the same map with no table: the
// footprint is not applied, the cells it opened read closed by their arms
// again, and so the comparison is sensitive to the structure stage.
func TestGroundArmsNameThePlaneTheWorldIsBuiltFrom(t *testing.T) {
	t.Parallel()
	const w, row = 40, 20
	tbl := bldTable(map[int][]int32{1: bldRow(4, 1, 0b0011, 0b1111)})
	m := passMap(w, w)
	m.Tiles[row*w+20] = 600             // water, under the footprint
	m.Tiles[row*w+21] = 7<<6 | 1<<4 | 1 // mountain, under the footprint
	m.Overlay[row*w+22] = 3             // an object, under the footprint
	m.Tiles[10*w+10] = 600 | 0x2000     // water and the flag
	m.Tiles[10*w+11] = 0x11 | 0x2000    // the flag on grass
	m.Tiles[10*w+12] = 7<<6 | 1<<4 | 1  // mountain
	m.Overlay[10*w+13] = 5              // an object
	m.Objects = []alm.Object{place(1, 20, row, nil)}

	arms := mapload.GroundArms(m, tbl)
	plane := mapload.PassabilityWith(m, tbl)
	if len(arms) != len(plane) {
		t.Fatalf("%d arms for %d cells", len(arms), len(plane))
	}
	for i := range plane {
		if (arms[i] != 0) != (plane[i]&1 != 0) {
			t.Fatalf("cell %d,%d: arms %#x, plane %#x", i%w, i/w, arms[i], plane[i])
		}
	}
	for _, c := range []struct {
		x, y int
		want mapload.GroundArm
	}{
		{0, 0, mapload.GroundBorder},
		{10, 10, mapload.GroundWater | mapload.GroundFlag},
		{11, 10, mapload.GroundFlag},
		{12, 10, mapload.GroundMountain},
		{13, 10, mapload.GroundObject},
		{14, 10, 0},
		{20, row, mapload.GroundStructure},
		{21, row, mapload.GroundStructure},
		{22, row, 0},
		{23, row, 0},
	} {
		if got := arms[c.y*w+c.x]; got != c.want {
			t.Errorf("cell %d,%d: arms %#x, want %#x", c.x, c.y, got, c.want)
		}
	}

	bare := mapload.GroundArms(m, nil)
	if bare[row*w+22] != mapload.GroundObject || bare[row*w+23] != 0 || bare[row*w+20] != mapload.GroundWater {
		t.Fatalf("with no table the footprint cells read %#x %#x %#x", bare[row*w+20], bare[row*w+22], bare[row*w+23])
	}
	if mapload.GroundArms(nil, tbl) != nil {
		t.Fatal("a nil map has arms")
	}
}
