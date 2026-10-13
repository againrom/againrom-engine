package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"sort"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// The original editor's save form (ALM-SAVE-255, ALM-SAVE-256) leaves every
// installed map as it is: each map's bytes through alm.SaveForm equal the
// installed bytes. The control moves one unit's owner off the roster and the
// form must bring it back to player 1, so the comparison can differ.
func TestReleaseInstalledMapsKeepTheirBytesThroughTheSaveForm(t *testing.T) {
	x := installedInspector(t)
	same, controlled := 0, 0
	for _, key := range installedMapKeys(x) {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		out, err := alm.SaveForm(d.SourceBytes)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if bytes.Equal(out, d.SourceBytes) {
			same++
		} else {
			at := 0
			for at < len(out) && at < len(d.SourceBytes) && out[at] == d.SourceBytes[at] {
				at++
			}
			t.Errorf("%s: the save form differs from the installed bytes at offset %#x", key, at)
		}
		if controlled == 0 {
			m, err := alm.Open(d.SourceBytes)
			if err != nil || len(m.Units) == 0 || len(m.Groups) == 0 {
				continue
			}
			doc, err := alm.OpenDocument(d.SourceBytes)
			if err != nil {
				t.Fatal(err)
			}
			raw := doc.Write()
			off := 20
			for _, id := range doc.RecordTypeIDs() {
				size := int(binary.LittleEndian.Uint32(raw[off+8:]))
				if id == 6 {
					binary.LittleEndian.PutUint32(raw[off+20+0x14:], 99)
				}
				off += 20 + size
			}
			back, err := alm.SaveForm(raw)
			if mb, merr := alm.Open(back); err != nil || merr != nil || bytes.Equal(raw, back) || mb.Units[0].Owner != 1 {
				t.Errorf("%s: control: an owner off the roster is not bound back to player 1 (%v)", key, err)
			}
			controlled++
		}
	}
	t.Logf("%d of %d installed maps keep their bytes through the save form", same, len(installedMapKeys(x)))
}

// stockKey lists the unit-owned loot records of a map, one line per record
// (owner, position, gold, items), sorted.
func stockKey(t *testing.T, b []byte) []string {
	t.Helper()
	m, err := alm.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	l, err := m.Loot()
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range l.Records {
		if !r.Ground() {
			out = append(out, fmt.Sprint(r.Owner, r.X, r.Y, r.Gold, r.Elements))
		}
	}
	sort.Strings(out)
	return out
}

// DIV-3255: deleting one ground sack renumbers the units' loot links, so the
// saved map holds one ground sack fewer and every unit's stock keeps its owner,
// position and contents. Run on the first and the last ground sack of every
// installed map that has one.
func TestReleaseDeletingAGroundSackKeepsTheOtherSacks(t *testing.T) {
	x := installedInspector(t)
	maps, probes := 0, 0
	for _, key := range installedMapKeys(x) {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		base, err := alm.SaveForm(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(base)
		if err != nil {
			t.Fatal(err)
		}
		l, err := m.Loot()
		if err != nil {
			t.Fatal(err)
		}
		var ground []int
		for i, r := range l.Records {
			if r.Ground() {
				ground = append(ground, i)
			}
		}
		if len(ground) == 0 {
			continue
		}
		maps++
		want := stockKey(t, base)
		for _, at := range []int{ground[0], ground[len(ground)-1]} {
			ed, err := mapedit.New(d.SourceBytes)
			if err != nil {
				t.Fatal(err)
			}
			if err := ed.Delete(mapedit.KindLoot, at); err != nil {
				t.Fatalf("%s: %v", key, err)
			}
			out, err := ed.SaveBytes()
			if err != nil {
				t.Fatalf("%s: %v", key, err)
			}
			ol, err := func() (alm.Loot, error) {
				om, err := alm.Open(out)
				if err != nil {
					return alm.Loot{}, err
				}
				return om.Loot()
			}()
			if err != nil {
				t.Fatalf("%s: %v", key, err)
			}
			g := 0
			for _, r := range ol.Records {
				if r.Ground() {
					g++
				}
			}
			probes++
			if g != len(ground)-1 {
				t.Errorf("%s: deleting ground sack %d leaves %d ground sacks, want %d", key, at, g, len(ground)-1)
			}
			if got := stockKey(t, out); !slices.Equal(got, want) {
				t.Errorf("%s: deleting ground sack %d changed the units' stock", key, at)
			}
		}
	}
	if probes == 0 {
		t.Fatal("no installed map has a ground sack")
	}
	t.Logf("%d maps with a ground sack, %d deletions", maps, probes)
}
