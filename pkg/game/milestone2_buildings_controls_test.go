package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestBuilding1145WitnessDetectsLostState(t *testing.T) {
	f := structureFront1114(t)
	raw := structureSave1114(t, 0, false)
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	wants, links, err := buildings1145Expected(source)
	if err != nil {
		t.Fatal(err)
	}
	// Three archive roots include one alias. Six cell writes reduce to four,
	// including a final null that replaces an earlier nonzero Building link.
	if len(wants) != 2 || len(links) != 4 || links[0x0c0c] != 0 || links[0x0c0d] != 701 || links[0x0c0e] != 700 {
		t.Fatalf("independent roster/cell read: %d %+v", len(wants), links)
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	sources, cells, present := ms.World.SavedStructures()
	live := ms.World.Structures()
	if differences := buildings1145Differences(wants, links, live, sources, cells, present); len(differences) != 0 {
		t.Fatal(differences)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*[]sim.Structure, *[]sim.SavedStructure, *[]sim.SavedStructureCell, *bool)
	}{
		{"lost roster", func(l *[]sim.Structure, s *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			*l, *s = (*l)[:1], (*s)[:1]
		}},
		{"duplicate live ID", func(l *[]sim.Structure, _ *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*l)[1].ID = (*l)[0].ID
		}},
		{"duplicate retained ID", func(_ *[]sim.Structure, s *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*s)[1].ID = (*s)[0].ID
		}},
		{"health", func(l *[]sim.Structure, _ *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*l)[0].Field42++
		}},
		{"kind", func(l *[]sim.Structure, _ *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*l)[0].Kind++
		}},
		{"maximum", func(l *[]sim.Structure, _ *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*l)[0].MaxHealth++
		}},
		{"geometry", func(l *[]sim.Structure, _ *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*l)[0].Width++
		}},
		{"attachment", func(l *[]sim.Structure, _ *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*l)[0].Attach ^= 1
		}},
		{"raw base", func(_ *[]sim.Structure, s *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*s)[0].Base52[0]++
		}},
		{"Token", func(_ *[]sim.Structure, s *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*s)[0].Token0E++
		}},
		{"fine position", func(_ *[]sim.Structure, s *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*s)[0].Position[4]++
		}},
		{"archive binding", func(_ *[]sim.Structure, s *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*s)[0].ArchiveIndex++
		}},
		{"identity", func(_ *[]sim.Structure, s *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, _ *bool) {
			(*s)[0].SourceKey++
		}},
		{"cell identity", func(_ *[]sim.Structure, _ *[]sim.SavedStructure, c *[]sim.SavedStructureCell, _ *bool) {
			(*c)[1].ID = 0
		}},
		{"lost null cell", func(_ *[]sim.Structure, _ *[]sim.SavedStructure, c *[]sim.SavedStructureCell, _ *bool) { *c = (*c)[1:] }},
		{"lost presence", func(_ *[]sim.Structure, _ *[]sim.SavedStructure, _ *[]sim.SavedStructureCell, p *bool) { *p = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l, s, c, p := append([]sim.Structure(nil), live...), append([]sim.SavedStructure(nil), sources...), append([]sim.SavedStructureCell(nil), cells...), present
			tc.mutate(&l, &s, &c, &p)
			if differences := buildings1145Differences(wants, links, l, s, c, p); len(differences) == 0 {
				t.Fatal("instrument accepted deliberately lost or changed state")
			}
		})
	}
}

func TestBuilding1145WitnessReadsSubclassTails(t *testing.T) {
	f := structureFixtureFront(t, 51, 52, 53)
	rows := structureFixtureRows()
	rows[0].Class, rows[0].Tavern9C = "Tavern", 0x99112233
	rows[1].Class, rows[1].Shop70 = "Shop", 0x33445566
	rows[2].Class, rows[2].OutpostWords = "Outpost", [4]uint32{0x84, 0x88, 0x80, 0x8c}
	rows[2].OutpostRecords = [][8]byte{{1, 2, 3, 4, 5, 6, 7, 8}}
	raw := structureFixtureSave(rows)
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	wants, links, err := buildings1145Expected(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(wants) != 3 || wants[0].source.Tavern9C != 0x99112233 || wants[1].source.Shop70 != 0x33445566 ||
		wants[2].source.OutpostWords != rows[2].OutpostWords || len(wants[2].source.OutpostRecords) != 1 ||
		wants[2].source.OutpostRecords[0] != rows[2].OutpostRecords[0] {
		t.Fatal("literal subclass tails changed")
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	sources, cells, present := ms.World.SavedStructures()
	if differences := buildings1145Differences(wants, links, ms.World.Structures(), sources, cells, present); len(differences) != 0 {
		t.Fatal(differences)
	}
	for i := range sources {
		changed := append([]sim.SavedStructure(nil), sources...)
		switch changed[i].Class {
		case sim.SavedTavern:
			changed[i].Tavern9C++
		case sim.SavedShop:
			changed[i].Shop70++
		case sim.SavedOutpost:
			changed[i].OutpostWords[0]++
		}
		if len(buildings1145Differences(wants, links, ms.World.Structures(), changed, cells, present)) == 0 {
			t.Fatal("subclass mutation escaped")
		}
	}
	// Exercise the independent reader's extended count and truncation bounds.
	body := make([]byte, 107)
	binary.LittleEndian.PutUint16(body[93:], 0xffff)
	binary.LittleEndian.PutUint32(body[95:], 1)
	copy(body[99:], []byte{1, 2, 3, 4, 5, 6, 7, 8})
	want, err := building1145Raw(body, 0, "Outpost", 1)
	if err != nil || len(want.source.OutpostRecords) != 1 || want.source.OutpostRecords[0] != rows[2].OutpostRecords[0] {
		t.Fatalf("wide count: %+v %v", want, err)
	}
	for _, n := range []int{76, 80, 94, 98, 106} {
		if _, err := building1145Raw(body[:n], 0, "Outpost", 1); err == nil {
			t.Fatalf("accepted truncated Outpost at %d", n)
		}
	}
}

func TestBuilding1145CellPopulationComesFromBytes(t *testing.T) {
	for _, wide := range []bool{false, true} {
		source, err := sav.Open(structureSave1114(t, 0, false))
		if err != nil {
			t.Fatal(err)
		}
		p := source.World.CellRecOff
		if binary.LittleEndian.Uint16(source.Body[p:]) != 6 {
			t.Fatal("fixture count changed")
		}
		if wide {
			// A legal extended count shifts the payload by four bytes. Leave
			// the decoded payload locator stale as a second negative control.
			body := append([]byte(nil), source.Body[:p]...)
			body = binary.LittleEndian.AppendUint16(body, 0xffff)
			body = binary.LittleEndian.AppendUint32(body, 6)
			source.Body = append(body, source.Body[p+2:]...)
		}
		// Review D1: trusting this count drops the final null overwrite and
		// can agree with a live importer that lost the same last record.
		source.World.CellRecCount = 5
		_, links, err := buildings1145Expected(source)
		if err != nil {
			t.Fatal(err)
		}
		if len(links) != 4 || links[0x0c0c] != 0 || links[0x0c0d] != 701 || links[0xffff] != 701 {
			t.Errorf("wide=%v: trusted decoder bookkeeping instead of six raw writes: %+v", wide, links)
		}
	}
}
