package alm

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
)

func mustNewMap(t *testing.T, s NewMapSpec) *Document {
	t.Helper()
	d, err := NewMap(s)
	if err != nil {
		t.Fatalf("NewMap(%+v): %v", s, err)
	}
	return d
}

// The values below are written out from ALM-NEWMAP-236, ALM-NEWMAP-237,
// TERR-246 and ALM-SAVE-238, not read back from the constructor's table.
func TestNewMapDefaultsMatchTheClaims(t *testing.T) {
	d := mustNewMap(t, DefaultNewMap())
	m, err := d.Map()
	if err != nil {
		t.Fatal(err)
	}
	if m.Width != 80 || m.Height != 80 {
		t.Fatalf("default size %dx%d, want 80x80 (ALM-NEWMAP-237)", m.Width, m.Height)
	}
	if got, want := d.RecordTypeIDs(), []uint32{0, 1, 2, 3, 5, 4, 9, 8, 6, 7}; !slices.Equal(got, want) {
		t.Fatalf("record order %v, want %v", got, want)
	}
	meta := d.RecordPayload(0)
	if len(meta) != 632 {
		t.Fatalf("type-0 size %d, want 632", len(meta))
	}
	word := func(off int) uint32 { return binary.LittleEndian.Uint32(meta[off:]) }
	for _, c := range []struct {
		off  int
		want uint32
	}{
		{0x08, 0x3f490fda}, {0x0c, 360}, {0x10, 21}, {0x14, 63},
		{0x18, 1}, // Grass 0x11: 1 << ((0x11 & 0x3ff) >> 6)
		{0x1c, 2}, {0x20, 0}, {0x24, 0}, {0x28, 0}, {0x2c, 0}, {0x70, 4}, {0x74, 1},
	} {
		if got := word(c.off); got != c.want {
			t.Errorf("type-0 +%#x = %#x, want %#x", c.off, got, c.want)
		}
	}
	if got := string(bytes.TrimRight(meta[0x30:0x70], "\x00")); got != "Multiplayer Map" {
		t.Errorf("name %q, want Multiplayer Map", got)
	}
	if !bytes.Equal(meta[0x78:], make([]byte, 512)) {
		t.Errorf("the text region is not zero")
	}
	cells := 80 * 80
	if p := d.RecordPayload(1); len(p) != 2*cells || binary.LittleEndian.Uint16(p) != 0x11 || binary.LittleEndian.Uint16(p[len(p)-2:]) != 0x11 {
		t.Errorf("tile plane is not %d cells of 0x11 (TERR-246)", cells)
	}
	if p := d.RecordPayload(2); !bytes.Equal(p, bytes.Repeat([]byte{64}, cells)) {
		t.Errorf("height plane is not %d cells of 64 (TERR-246)", cells)
	}
	if p := d.RecordPayload(3); !bytes.Equal(p, make([]byte, cells)) {
		t.Errorf("scenery plane is not zero (TERR-246)")
	}
	for id, size := range map[uint32]int{4: 0, 5: 152, 6: 0, 7: 12, 8: 0, 9: 4} {
		i := slices.Index(d.RecordTypeIDs(), id)
		if got := len(d.RecordPayload(i)); got != size {
			t.Errorf("type-%d size %d, want %d (ALM-NEWMAP-237)", id, got, size)
		}
	}
	raw := d.Write()
	if got := binary.LittleEndian.Uint32(raw[0x08:]); got != uint32(4*cells+0x48) {
		t.Errorf("dataSize %d, want 4*W*H+0x48 (ALM-NEWMAP-237)", got)
	}
	if got := binary.LittleEndian.Uint32(raw[0x10:]); got != 990 {
		t.Errorf("formatVersion %d, want 990", got)
	}
	if len(m.Units) != 0 || len(m.Objects) != 0 {
		t.Errorf("a new map holds %d units and %d structures", len(m.Units), len(m.Objects))
	}
}

func TestNewMapGroupMaskFollowsTheTileWord(t *testing.T) {
	for _, c := range []struct {
		tile uint16
		want uint32
	}{
		{0x11, 1}, {0x14, 1}, {0x91, 1 << 2}, {0x51, 1 << 1}, {0x191, 1 << 6},
		{0x211, 0x100 | 0xf00}, // above 0xff gains 0xf00 (ALM-SAVE-238)
	} {
		s := DefaultNewMap()
		s.Tile = c.tile
		meta := mustNewMap(t, s).RecordPayload(0)
		if got := binary.LittleEndian.Uint32(meta[0x18:]); got != c.want {
			t.Errorf("tile %#x: +0x18 = %#x, want %#x", c.tile, got, c.want)
		}
	}
}

func TestNewMapEverySizeOpensAndRoundTrips(t *testing.T) {
	for _, side := range []int{32, 40, 48, 64, 80, 96, 128, 256} {
		d := mustNewMap(t, NewMapSpec{Width: side, Height: side, Tile: 0x11, Altitude: 64})
		raw := d.Write()
		m, err := Open(raw)
		if err != nil || m.Width != side || m.Height != side {
			t.Fatalf("side %d: open %v, %dx%d", side, err, m.Width, m.Height)
		}
		again, err := OpenDocument(raw)
		if err != nil || !bytes.Equal(again.Write(), raw) {
			t.Fatalf("side %d: the document does not write back its bytes (%v)", side, err)
		}
	}
	d := mustNewMap(t, NewMapSpec{Width: 32, Height: 48, Tile: 0x11, Altitude: 64})
	if m, err := d.Map(); err != nil || m.Width != 32 || m.Height != 48 {
		t.Fatalf("a 32x48 map opens as %v", err)
	}
}

func TestNewMapRefusesWhatItCannotBuild(t *testing.T) {
	good := DefaultNewMap()
	for name, mutate := range map[string]func(*NewMapSpec){
		"width below the bound":   func(s *NewMapSpec) { s.Width = 24 },
		"width above the bound":   func(s *NewMapSpec) { s.Width = 264 },
		"height off the step":     func(s *NewMapSpec) { s.Height = 70 },
		"height zero":             func(s *NewMapSpec) { s.Height = 0 },
		"tile bit the save drops": func(s *NewMapSpec) { s.Tile = 0x411 },
		"altitude above 126":      func(s *NewMapSpec) { s.Altitude = 127 },
	} {
		s := good
		mutate(&s)
		if d, err := NewMap(s); err == nil || d != nil {
			t.Errorf("%s: built a document", name)
		}
	}
}

// rosterProblems lists how a type-5 payload departs from ALM-NEWMAP-258: two
// 76-byte records with id words 1 and 2, side words 0 and 1, scalar 0, the
// names Self and Enemy zero past the NUL, and a sixteen-word relation row
// holding 2 at the record's own slot and 1 elsewhere.
func rosterProblems(p []byte) []string {
	var out []string
	if len(p) != 152 {
		return []string{"the roster is not two 76-byte records"}
	}
	for i, want := range []struct {
		id, side uint32
		name     string
	}{{1, 0, "Self"}, {2, 1, "Enemy"}} {
		r := p[76*i : 76*(i+1)]
		le := binary.LittleEndian
		if le.Uint32(r[0:]) != want.id || le.Uint32(r[4:]) != want.side || le.Uint32(r[8:]) != 0 {
			out = append(out, "id, side or scalar word")
		}
		if !bytes.Equal(r[0x0c:0x2c], append([]byte(want.name), make([]byte, 32-len(want.name))...)) {
			out = append(out, "name")
		}
		for k := 0; k < 16; k++ {
			w := uint16(1)
			if k == i {
				w = 2
			}
			if le.Uint16(r[0x2c+2*k:]) != w {
				out = append(out, "relation word")
			}
		}
	}
	return out
}

// ALM-NEWMAP-258: the two players of a new map. The control is the roster the
// constructor wrote before the claim: scalar 5000, no first name, zeros in the
// relation row.
func TestNewMapPlayersMatchTheClaim(t *testing.T) {
	d := mustNewMap(t, DefaultNewMap())
	if got := rosterProblems(d.RecordPayload(slices.Index(d.RecordTypeIDs(), 5))); len(got) != 0 {
		t.Errorf("roster departs from ALM-NEWMAP-258: %v", got)
	}
	old := make([]byte, 152)
	binary.LittleEndian.PutUint32(old[0:], 1)
	binary.LittleEndian.PutUint32(old[8:], 5000)
	copy(old[0x2c:], []byte{2, 0, 1, 0})
	binary.LittleEndian.PutUint32(old[76:], 2)
	binary.LittleEndian.PutUint32(old[80:], 1)
	binary.LittleEndian.PutUint32(old[84:], 5000)
	copy(old[76+0x0c:], "Enemy")
	if len(rosterProblems(old)) == 0 {
		t.Error("control: the earlier roster passes the check")
	}
}

// ALM-NEWMAP-257: the original leaves heap contents after the name's NUL and
// in the text region; the constructor writes zeros there.
func TestNewMapUnwrittenBytesAreZero(t *testing.T) {
	meta := mustNewMap(t, DefaultNewMap()).RecordPayload(0)
	if !bytes.Equal(meta[0x30+len("Multiplayer Map"):0x70], make([]byte, 64-len("Multiplayer Map"))) ||
		!bytes.Equal(meta[0x78:0x78+512], make([]byte, 512)) {
		t.Error("a byte after the name or in the text region is not zero")
	}
}
