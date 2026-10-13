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
		{"highest index group is bounded", []uint16{0x3ff}, 0x1fff},
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

func TestTileGroupBitsIsTheGroupMask(t *testing.T) {
	if TileGroupBits != 0x1fff {
		t.Fatalf("TileGroupBits = %#x (ALM-SAVE-238)", TileGroupBits)
	}
}
