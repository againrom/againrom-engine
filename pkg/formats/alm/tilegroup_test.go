package alm

import "testing"

func TestTileGroupMaskRule(t *testing.T) {
	for _, c := range []struct {
		name  string
		words []uint16
		want  uint32
	}{
		{"grass group 0", []uint16{0x11}, 1},
		{"water group 8 gains 0xf00", []uint16{0x211}, 0x1<<8 | 0xf00},
		{"two groups", []uint16{0x11, 0x91}, 1 | 1<<2},
		{"flag bits above the index are ignored", []uint16{0x2011}, 1},
		{"every install group is bounded", []uint16{0x11, 0x51, 0x91, 0xd1, 0x111, 0x151, 0x191, 0x1d1, 0x211, 0x311}, 0x1fff},
	} {
		if got := TileGroupMask(c.words); got != c.want {
			t.Errorf("%s: %#x, want %#x (ALM-SAVE-238)", c.name, got, c.want)
		}
	}
	if TileGroupMask(nil) != 0 {
		t.Error("an empty plane has no group")
	}
}

// A new map writes the mask through the one function.
func TestNewMapBitmaskIsTheTileGroupMask(t *testing.T) {
	for _, tile := range []uint16{0x14, 0x11, 0x91, 0x211, 0x1d1} {
		s := DefaultNewMap()
		s.Tile = tile
		d := mustNewMap(t, s)
		m, err := d.Map()
		if err != nil {
			t.Fatal(err)
		}
		got := m.Meta.Bitmask
		if want := TileGroupMask([]uint16{tile}); got != want {
			t.Errorf("tile %#x: +0x18 = %#x, want %#x", tile, got, want)
		}
	}
}

// The format's height range and the dialog's slider are two limits: the
// slider stops one below the top the height tool and the maps reach.
func TestHeightLimitsAreTwoClaims(t *testing.T) {
	if MaxHeight != 127 {
		t.Errorf("MaxHeight = %d, want 127 (TERR-245)", MaxHeight)
	}
	if NewMapMaxAltitude != 126 {
		t.Errorf("NewMapMaxAltitude = %d, want 126 (TERR-246)", NewMapMaxAltitude)
	}
	s := DefaultNewMap()
	s.Altitude = NewMapMaxAltitude
	if err := s.Validate(); err != nil {
		t.Errorf("the dialog accepts 126: %v", err)
	}
	s.Altitude = MaxHeight
	if err := s.Validate(); err == nil {
		t.Error("the dialog refuses 127 (TERR-246)")
	}
}

// A mod group in use keeps its own bit above the install groups' bound, so a
// map of groups 13 and 31 names both to the loader (DIV-4412); a word past
// the loader's 32 groups names none.
func TestTileGroupMaskKeepsModGroups(t *testing.T) {
	grass, snow, ice := uint16(0x11), uint16(13<<6|0x11), uint16(31<<6|0x21)
	if got, want := TileGroupMask([]uint16{grass, snow, ice}), uint32(1|1<<13|1<<31); got != want {
		t.Errorf("mask %#x, want %#x", got, want)
	}
	if got := TileGroupMask([]uint16{0x2000 | snow}); got != 1<<13 {
		t.Errorf("impassable snow: %#x, want bit 13", got)
	}
	if got := TileGroupMask([]uint16{32 << 6, 127 << 6}); got != 0 {
		t.Errorf("groups past the loader: %#x, want 0", got)
	}
	if TileGroup(ice) != 31 || TileGroup(0x2000|grass) != 0 {
		t.Error("TileGroup splits the word as TERR-TILE-291")
	}
}

func TestTileGroupBitsIsTheGroupMask(t *testing.T) {
	if TileGroupBits != 0x1fff {
		t.Fatalf("TileGroupBits = %#x (ALM-SAVE-238)", TileGroupBits)
	}
}
