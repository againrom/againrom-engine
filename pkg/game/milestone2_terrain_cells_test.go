package game

import (
	"encoding/binary"
	"fmt"
	"sort"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func terminalTerrainAcceptance(rows []terrainAcceptanceRecord, doc sav.DocumentData) []terrainAcceptanceRecord {
	terminal := map[uint32]bool{}
	for _, index := range doc.DeadActors {
		r := &doc.Objects[index-1]
		hp, _ := savedStructureValue(r, "Health")
		stage, _ := savedStructureValue(r, "Stage")
		key, _ := savedStructureValue(r, "Identity")
		if int16(hp) <= -10 && stage >= 2 && key != 0 {
			terminal[key] = true
		}
	}
	cells := map[uint16]sav.DocumentCellData{}
	for _, cell := range doc.World.Cells {
		cells[cell.Cell] = cell
	}
	out := append([]terrainAcceptanceRecord(nil), rows...)
	for i := range out {
		cell := cells[out[i].cell]
		if terminal[cell.GroundActor] {
			out[i].dynamic &^= 0x40
		}
		if terminal[cell.AirActor] {
			out[i].dynamic &^= 0x80
		}
	}
	return out
}

func TestTerminalTerrainAcceptanceOnlyReleasesExactDeadSlots(t *testing.T) {
	doc := sav.DocumentData{World: &sav.DocumentWorldData{Cells: []sav.DocumentCellData{
		{Cell: 0x0808, GroundActor: 1, AirActor: 2}, {Cell: 0x0809, GroundActor: 2, AirActor: 1},
		{Cell: 0x080a, GroundActor: 1}, {Cell: 0x080a, GroundActor: 3}, {Cell: 0x080b, GroundActor: 4},
	}}}
	for i, tuple := range []struct {
		hp    int16
		stage uint32
	}{{-40, 3}, {0, 1}, {20, 0}, {-1000, 1}} {
		r := mustNewRecord("Unit")
		mustSetValue(&r, "Identity", uint32(i+1))
		mustSetValue(&r, "Health", uint32(uint16(tuple.hp)))
		mustSetValue(&r, "Stage", tuple.stage)
		doc.Objects = append(doc.Objects, r)
		doc.DeadActors = append(doc.DeadActors, uint16(i+1))
	}
	wants := []terrainAcceptanceRecord{{0x0808, 0x27, 0xe7}, {0x0809, 0x27, 0xe7}, {0x080a, 0x27, 0xe7}, {0x080b, 0x27, 0xe7}, {0x080c, 0x27, 0xe7}}
	got := terminalTerrainAcceptance(wants, doc)
	for i, want := range []byte{0xa7, 0x67, 0xe7, 0xe7, 0xe7} {
		if got[i].dynamic != want || got[i].static != 0x27 || wants[i].dynamic != 0xe7 {
			t.Fatal("terminal oracle altered a nonterminal layer, static bits or source", got)
		}
	}
}

type terrainAcceptanceRecord struct {
	cell            uint16
	static, dynamic byte
}

type cellAcceptanceRecord struct {
	cell                 uint16
	cost, static, layers byte
}

// SAV-TERRKEY-056: counted four-byte block deltas, counted 54-byte cell
// records, then the terrain identity. SAV-758 admits the extended count even
// for a small value. Only the first structure's locator comes from sav.Open;
// neither decoded counts nor payload offsets determine this population.
func terrainCellsRaw(body []byte, start int) ([]terrainAcceptanceRecord, []cellAcceptanceRecord, error) {
	p := start
	table := func(stride int) ([]byte, error) {
		if p < 0 || p > len(body)-2 {
			return nil, fmt.Errorf("missing count at %d", p)
		}
		n := uint32(binary.LittleEndian.Uint16(body[p:]))
		p += 2
		if n == 0xffff {
			if p > len(body)-4 {
				return nil, fmt.Errorf("missing extended count at %d", p)
			}
			n = binary.LittleEndian.Uint32(body[p:])
			p += 4
		}
		span := uint64(n) * uint64(stride)
		if span > uint64(len(body)-p) {
			return nil, fmt.Errorf("%d records of %d bytes exceed body at %d", n, stride, p)
		}
		data := body[p : p+int(span)]
		p += int(span)
		return data, nil
	}
	blockBytes, err := table(4)
	if err != nil {
		return nil, nil, fmt.Errorf("terrain blocks: %w", err)
	}
	cellBytes, err := table(54)
	if err != nil {
		return nil, nil, fmt.Errorf("cell records: %w", err)
	}
	if p > len(body)-4 {
		return nil, nil, fmt.Errorf("missing terrain identity at %d", p)
	}
	blocks := make([]terrainAcceptanceRecord, len(blockBytes)/4)
	for i := range blocks {
		b := blockBytes[4*i:]
		blocks[i] = terrainAcceptanceRecord{cell: binary.LittleEndian.Uint16(b[2:]), static: b[0], dynamic: b[1]}
	}
	cells := make([]cellAcceptanceRecord, len(cellBytes)/54)
	for i := range cells {
		b := cellBytes[54*i:]
		// SAV-CELLLOAD-111's first three payload bytes follow the u16 key.
		cells[i] = cellAcceptanceRecord{cell: binary.LittleEndian.Uint16(b), cost: b[2], static: b[3], layers: b[4]}
	}
	return blocks, cells, nil
}

func terrainAcceptanceDifferences(wants []terrainAcceptanceRecord, planes *sim.SavedCellPlanes) []string {
	if planes == nil {
		return []string{"missing saved terrain planes"}
	}
	var differences []string
	last := -1
	for _, want := range wants {
		// SAV-BLOCK-011's raw stream invariants.
		if want.cell < 0x807 || int(want.cell) >= 0x807+0xe5e7 || want.dynamic <= 0x0f || int(want.cell) <= last {
			differences = append(differences, fmt.Sprintf("terrain cell %#04x violates window, dynamic gate or increasing order", want.cell))
		}
		last = int(want.cell)
		if planes.Static[want.cell] != want.static {
			differences = append(differences, fmt.Sprintf("terrain cell %#04x Static: live=%#02x file=%#02x", want.cell, planes.Static[want.cell], want.static))
		}
		if planes.Dynamic[want.cell] != want.dynamic {
			differences = append(differences, fmt.Sprintf("terrain cell %#04x Dynamic: live=%#02x file=%#02x", want.cell, planes.Dynamic[want.cell], want.dynamic))
		}
	}
	return differences
}

func cellAcceptanceFinal(rows []cellAcceptanceRecord) map[uint16]cellAcceptanceRecord {
	final := make(map[uint16]cellAcceptanceRecord, len(rows))
	for _, row := range rows {
		final[row.cell] = row // SAV-CELLLOAD-109: the final saved overwrite wins.
	}
	return final
}

// Compare the saved-record carriers, not the whole ALM-constructed cell hash.
// Each distinct saved key must occur once in both carriers. Baselines and
// layer counts cover this milestone row; object references have other audits.
func cellAcceptanceDifferences(wants []cellAcceptanceRecord, baselines []sim.SavedStructureCell, layers []sim.SavedCellRecord, present bool) []string {
	final := cellAcceptanceFinal(wants)
	var differences []string
	add := func(format string, args ...any) { differences = append(differences, fmt.Sprintf(format, args...)) }
	if !present || len(baselines) != len(final) || len(layers) != len(final) {
		add("cell population: present=%v baseline=%d layer=%d unique file=%d", present, len(baselines), len(layers), len(final))
	}
	byBaseline := make(map[uint16]sim.SavedStructureCell, len(baselines))
	for _, row := range baselines {
		if _, exists := byBaseline[row.Cell]; exists {
			add("duplicate live baseline cell %#04x", row.Cell)
		}
		if _, exists := final[row.Cell]; !exists {
			add("unexpected live baseline cell %#04x", row.Cell)
		}
		byBaseline[row.Cell] = row
	}
	byLayer := make(map[uint16]sim.SavedCellRecord, len(layers))
	for _, row := range layers {
		if _, exists := byLayer[row.Cell]; exists {
			add("duplicate live layer cell %#04x", row.Cell)
		}
		if _, exists := final[row.Cell]; !exists {
			add("unexpected live layer cell %#04x", row.Cell)
		}
		byLayer[row.Cell] = row
	}
	keys := make([]int, 0, len(final))
	for key := range final {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)
	for _, key := range keys {
		want := final[uint16(key)]
		baseline, found := byBaseline[want.cell]
		if !found {
			add("missing live baseline cell %#04x", want.cell)
		} else {
			if baseline.BaselineCost != want.cost {
				add("cell %#04x BaselineCost: live=%d file=%d", want.cell, baseline.BaselineCost, want.cost)
			}
			if baseline.BaselineStatic != want.static {
				add("cell %#04x BaselineStatic: live=%#02x file=%#02x", want.cell, baseline.BaselineStatic, want.static)
			}
		}
		layer, found := byLayer[want.cell]
		if !found || layer.LayerCount != want.layers {
			add("cell %#04x LayerCount: present=%v live=%d file=%d", want.cell, found, layer.LayerCount, want.layers)
		}
	}
	return differences
}

func TestTerrainCellAcceptanceCountsAndOverwrites(t *testing.T) {
	for _, wide := range []bool{false, true} {
		t.Run(fmt.Sprintf("extended=%v", wide), func(t *testing.T) {
			count := func(b []byte, n uint16) []byte {
				if wide {
					return binary.LittleEndian.AppendUint32(append(b, 0xff, 0xff), uint32(n))
				}
				return binary.LittleEndian.AppendUint16(b, n)
			}
			// Literal cells 0x0908 and 0x0909; the second record for 0x0908
			// deliberately changes all compared fields and clears the layer.
			body := count([]byte{0xaa, 0xbb, 0xcc}, 2)
			body = append(body, 1, 0x21, 8, 9, 4, 0x54, 9, 9)
			body = count(body, 3)
			for _, head := range [][5]byte{{8, 9, 11, 0x21, 3}, {9, 9, 12, 0x24, 2}, {8, 9, 13, 0x25, 0}} {
				body = append(body, head[:]...)
				body = append(body, make([]byte, 49)...)
			}
			body = append(body, 0x44, 0x33, 0x22, 0x11)
			blocks, cells, err := terrainCellsRaw(body, 3)
			if err != nil || len(blocks) != 2 || len(cells) != 3 || len(cellAcceptanceFinal(cells)) != 2 {
				t.Fatalf("raw population: %d/%d %v", len(blocks), len(cells), err)
			}
			planes := &sim.SavedCellPlanes{}
			planes.Static[0x0908], planes.Dynamic[0x0908] = 1, 0x21
			planes.Static[0x0909], planes.Dynamic[0x0909] = 4, 0x54
			baselines := []sim.SavedStructureCell{{Cell: 0x0908, BaselineCost: 13, BaselineStatic: 0x25}, {Cell: 0x0909, BaselineCost: 12, BaselineStatic: 0x24}}
			layers := []sim.SavedCellRecord{{Cell: 0x0908, LayerCount: 0}, {Cell: 0x0909, LayerCount: 2}}
			if diff := terrainAcceptanceDifferences(blocks, planes); len(diff) != 0 {
				t.Fatal(diff)
			}
			if diff := cellAcceptanceDifferences(cells, baselines, layers, true); len(diff) != 0 {
				t.Fatal(diff)
			}
			for _, tc := range []struct {
				name string
				edit func(*sim.SavedCellPlanes, []sim.SavedStructureCell, []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord)
			}{
				{"wrong Static only", func(p *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					p.Static[0x0909] ^= 1
					return b, l
				}},
				{"wrong Dynamic only", func(p *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					p.Dynamic[0x0909] ^= 1
					return b, l
				}},
				{"wrong BaselineCost only", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					b[1].BaselineCost++
					return b, l
				}},
				{"wrong BaselineStatic only", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					b[1].BaselineStatic ^= 1
					return b, l
				}},
				{"wrong LayerCount only", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					l[1].LayerCount++
					return b, l
				}},
				{"lost final terrain delta", func(p *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					p.Static[0x0909], p.Dynamic[0x0909] = 0, 0
					return b, l
				}},
				{"lost baseline", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					return b[:1], l
				}},
				{"lost layer", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					return b, l[:1]
				}},
				{"lost final overwrite", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					b[0].BaselineCost, b[0].BaselineStatic, l[0].LayerCount = 11, 0x21, 3
					return b, l
				}},
				{"wrong key with same count", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					b[1].Cell, l[1].Cell = 0x0910, 0x0910
					return b, l
				}},
				{"duplicate live key", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					b[1], l[1] = b[0], l[0]
					return b, l
				}},
				{"extra live key", func(_ *sim.SavedCellPlanes, b []sim.SavedStructureCell, l []sim.SavedCellRecord) ([]sim.SavedStructureCell, []sim.SavedCellRecord) {
					return append(b, sim.SavedStructureCell{Cell: 0x0910}), append(l, sim.SavedCellRecord{Cell: 0x0910})
				}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					p := *planes
					b, l := tc.edit(&p, append([]sim.SavedStructureCell(nil), baselines...), append([]sim.SavedCellRecord(nil), layers...))
					if len(terrainAcceptanceDifferences(blocks, &p))+len(cellAcceptanceDifferences(cells, b, l, true)) == 0 {
						t.Fatal("corruption escaped acceptance")
					}
				})
			}
		})
	}
}

func TestTerrainCellAcceptanceRejectsTruncatedCounts(t *testing.T) {
	for _, body := range [][]byte{nil, {0}, {0xff, 0xff}, {0xff, 0xff, 1, 0, 0}, {0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, {1, 0, 0}, {0, 0, 0xff, 0xff}, {0, 0, 1, 0, 0}, {0, 0, 0, 0}} {
		if _, _, err := terrainCellsRaw(body, 0); err == nil {
			t.Fatalf("accepted truncated table %x", body)
		}
	}
	if _, _, err := terrainCellsRaw(make([]byte, 8), -1); err == nil {
		t.Fatal("accepted negative offset")
	}
	if b, c, err := terrainCellsRaw(make([]byte, 8), 0); err != nil || len(b) != 0 || len(c) != 0 {
		t.Fatalf("empty tables: %v", err)
	}
}
