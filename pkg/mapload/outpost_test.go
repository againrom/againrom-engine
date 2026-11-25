package mapload_test

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// outpostMap is a 40x40 map: owner 2's group 7 at cells 12,12 and 15,13 and
// the human's unit at 20,20, with the type-0 word gate and type-10 entries.
func outpostMap(gate uint32, entries ...[4]uint32) *alm.Map {
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{
		{X: 0x0C80, Y: 0x0C80, ClassID: 7, Owner: 2, GroupID: 7},
		{X: 0x0F80, Y: 0x0D80, ClassID: 7, Owner: 2, GroupID: 7},
		{X: 0x1480, Y: 0x1480, ClassID: 7, Owner: sim.SelfSlot, GroupID: 1},
	}}
	m.Meta.Word70 = gate
	for _, e := range entries {
		for _, v := range e {
			m.Extension[0] = binary.LittleEndian.AppendUint32(m.Extension[0], v)
		}
	}
	return m
}

func startOutpostMap(t *testing.T, m *alm.Map, second, spread bool) *sim.World {
	t.Helper()
	var s *sim.Script
	if second {
		var err error
		if s, err = sim.NewROM2Script(nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	tbl := cwTable(nil)
	tbl.OutpostSpread = spread
	w, _, err := mapload.StartMissionScripted(m, tbl, mapload.DifficultyNormal, nil, s)
	if err != nil {
		t.Fatalf("StartMissionScripted: %v", err)
	}
	return w
}

// The mission start runs the type-10 pass: a second-game map whose gate is
// open builds an Outpost for owner 2's group 7 with the entry's reload and,
// in the Russian build, the members' spread; a shut gate, a map without
// entries under a shut gate and a first-game World build none.
func TestMissionStartBuildsTheMapsOutposts(t *testing.T) {
	entry := [4]uint32{7, 5, 2, 0}
	w := startOutpostMap(t, outpostMap(4, entry), true, true)
	got := w.Outposts()
	if len(got) != 1 {
		t.Fatalf("Outposts = %+v, want owner 2's group 7 alone", got)
	}
	o := got[0]
	if o.Owner != 2 || o.Group != 7 || o.Reload != 5 || o.Countdown != 5 || o.Radius != 3 || len(o.Members) != 2 {
		t.Fatalf("Outpost = %+v", o)
	}
	if o := startOutpostMap(t, outpostMap(4, entry), true, false).Outposts(); len(o) != 1 || o[0].Radius != 0 {
		t.Fatalf("English build Outposts = %+v, want radius 0", o)
	}
	for _, c := range []struct {
		name string
		w    *sim.World
	}{
		{"shut gate", startOutpostMap(t, outpostMap(1, entry), true, true)},
		{"no entries, shut gate", startOutpostMap(t, outpostMap(1), true, true)},
		{"first-game World", startOutpostMap(t, outpostMap(4, entry), false, true)},
	} {
		if len(c.w.Outposts()) != 0 {
			t.Fatalf("%s built %+v", c.name, c.w.Outposts())
		}
	}
	// An open gate builds the defaults for a group no entry names.
	if o := startOutpostMap(t, outpostMap(4), true, true).Outposts(); len(o) != 1 || o[0].Reload != 120 || o[0].Radius != 3 {
		t.Fatalf("no-entry Outposts = %+v, want the defaults", o)
	}
}

// The type-10 records decode in file order; a payload that is not whole
// records gives none.
func TestOutpostsDecodeTheTypeTenEntries(t *testing.T) {
	got := mapload.Outposts(outpostMap(2, [4]uint32{7, 0xFFFFFFFC, 3, 9}, [4]uint32{8, 40, 0, 0}), &mapload.Table{OutpostSpread: true})
	want := []sim.OutpostEntry{{Group: 7, Percent: -4, Flags: 3}, {Group: 8, Percent: 40}}
	if !got.Runs || !got.Spread || len(got.Entries) != 2 || got.Entries[0] != want[0] || got.Entries[1] != want[1] {
		t.Fatalf("Outposts = %+v, want %+v", got, want)
	}
	m := outpostMap(2)
	m.Extension[0] = make([]byte, 15)
	if got := mapload.Outposts(m, nil); len(got.Entries) != 0 || got.Spread {
		t.Fatalf("a torn payload gave %+v", got)
	}
}
