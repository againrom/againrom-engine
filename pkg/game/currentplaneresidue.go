package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

const (
	currentPlaneCost = 1 << iota
	currentPlaneStatic
	currentPlaneDynamic
)

type currentCellPlaneResidue struct {
	Cell                  uint16
	Fields                uint8
	Cost, Static, Dynamic uint8
	Anchor                [16]byte
}

type currentCellPlaneResidues []currentCellPlaneResidue

func (rows currentCellPlaneResidues) MarshalJSON() ([]byte, error) {
	if err := rows.validate(); err != nil {
		return nil, err
	}
	var raw []byte
	if rows != nil {
		raw = make([]byte, 22*len(rows))
	}
	for i, row := range rows {
		at := i * 22
		binary.LittleEndian.PutUint16(raw[at:], row.Cell)
		raw[at+2], raw[at+3], raw[at+4], raw[at+5] = row.Fields, row.Cost, row.Static, row.Dynamic
		copy(raw[at+6:], row.Anchor[:])
	}
	return json.Marshal(raw)
}

func (rows *currentCellPlaneResidues) UnmarshalJSON(data []byte) error {
	var raw []byte
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw)%22 != 0 || len(raw)/22 > 65536 {
		return fmt.Errorf("current cell plane residue has invalid framing")
	}
	var next currentCellPlaneResidues
	if raw != nil {
		next = make(currentCellPlaneResidues, len(raw)/22)
	}
	for i := range next {
		at := i * 22
		r := &next[i]
		r.Cell = binary.LittleEndian.Uint16(raw[at:])
		r.Fields, r.Cost, r.Static, r.Dynamic = raw[at+2], raw[at+3], raw[at+4], raw[at+5]
		copy(r.Anchor[:], raw[at+6:at+22])
	}
	if err := next.validate(); err != nil {
		return err
	}
	*rows = next
	return nil
}

func (rows currentCellPlaneResidues) validate() error {
	if len(rows) > 65536 {
		return fmt.Errorf("current cell plane residue exceeds its keyspace")
	}
	for i, row := range rows {
		if row.Fields == 0 || row.Fields&^7 != 0 || i > 0 && rows[i-1].Cell >= row.Cell ||
			row.Fields&currentPlaneCost == 0 && row.Cost != 0 || row.Fields&currentPlaneStatic == 0 && row.Static != 0 || row.Fields&currentPlaneDynamic == 0 && row.Dynamic != 0 {
			return fmt.Errorf("invalid current cell plane residue %04x", row.Cell)
		}
	}
	return nil
}

type currentPlaneOrdinary struct {
	base          []byte
	cells, blocks map[uint16][]byte
}

func currentPlaneOrdinaryFields(doc *sav.DocumentData) currentPlaneOrdinary {
	p := currentPlaneOrdinary{cells: map[uint16][]byte{}, blocks: map[uint16][]byte{}}
	p.base = binary.LittleEndian.AppendUint32(nil, doc.World.TerrainIdentity)
	p.base = binary.LittleEndian.AppendUint32(p.base, uint32(len(doc.Head.MapName)))
	p.base = append(p.base, doc.Head.MapName...)
	for _, cell := range doc.World.Cells {
		payload := cell.Payload()
		p.cells[cell.Cell] = append(p.cells[cell.Cell], payload[:]...)
	}
	for _, block := range doc.World.Blocks {
		p.blocks[block.Cell] = append(p.blocks[block.Cell], block.Dyn, block.Static)
	}
	return p
}

func (p currentPlaneOrdinary) anchor(cell uint16) [16]byte {
	raw := append([]byte(nil), p.base...)
	raw = binary.LittleEndian.AppendUint16(raw, cell)
	raw = binary.LittleEndian.AppendUint32(raw, uint32(len(p.cells[cell])))
	raw = append(raw, p.cells[cell]...)
	raw = binary.LittleEndian.AppendUint32(raw, uint32(len(p.blocks[cell])))
	raw = append(raw, p.blocks[cell]...)
	hash := sha256.Sum256(raw)
	return [16]byte(hash[:16])
}

func (f *FrontEnd) finalizeCurrentPlaneResidue(doc *sav.DocumentData, s Snapshot) error {
	if doc.World == nil || len(s.World) == 0 {
		return nil
	}
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		return err
	}
	var w sim.World
	if err := w.UnmarshalBinary(s.World); err != nil {
		return err
	}
	planes, present := w.SavedCellPlanes()
	if !present {
		return nil
	}
	base, err := readOriginalCellPlanes(f.live.mission.state.Map, f.Archives.Containers)
	if err != nil || base == nil {
		return err
	}
	ordinary := currentPlaneOrdinaryFields(doc)
	_, cells, _, _ := w.SavedActorMotions()
	hasCell := map[uint16]bool{}
	for _, cell := range cells {
		hasCell[cell.Cell] = true
	}
	a.CellPlaneResidue = nil
	for key := range 65536 {
		cell := uint16(key)
		row := currentCellPlaneResidue{Cell: cell}
		if !hasCell[cell] && len(ordinary.cells[cell]) == 0 && planes.Cost[cell] != base.Cost[cell] {
			row.Fields |= currentPlaneCost
			row.Cost = planes.Cost[cell]
		}
		if len(ordinary.blocks[cell]) == 0 {
			if planes.Static[cell] != base.Static[cell] {
				row.Fields |= currentPlaneStatic
				row.Static = planes.Static[cell]
			}
			if planes.Dynamic[cell] != base.Dynamic[cell] {
				row.Fields |= currentPlaneDynamic
				row.Dynamic = planes.Dynamic[cell]
			}
		}
		if row.Fields != 0 {
			row.Anchor = ordinary.anchor(cell)
			a.CellPlaneResidue = append(a.CellPlaneResidue, row)
		}
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, raw)
}

func matchingCurrentPlaneResidue(doc *sav.DocumentData, rows currentCellPlaneResidues) ([]sim.CurrentCellPlaneResidue, error) {
	if err := rows.validate(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	if doc == nil || doc.World == nil {
		return nil, fmt.Errorf("current cell plane residue lacks ordinary world")
	}
	ordinary := currentPlaneOrdinaryFields(doc)
	var out []sim.CurrentCellPlaneResidue
	for _, row := range rows {
		if row.Anchor != ordinary.anchor(row.Cell) {
			continue
		}
		if row.Fields&currentPlaneCost != 0 && len(ordinary.cells[row.Cell]) != 0 || row.Fields&(currentPlaneStatic|currentPlaneDynamic) != 0 && len(ordinary.blocks[row.Cell]) != 0 {
			return nil, fmt.Errorf("current plane residue conflicts with ordinary cell %04x", row.Cell)
		}
		value := sim.CurrentCellPlaneResidue{Cell: row.Cell}
		if row.Fields&currentPlaneCost != 0 {
			value.Cost = &row.Cost
		}
		if row.Fields&currentPlaneStatic != 0 {
			value.Static = &row.Static
		}
		if row.Fields&currentPlaneDynamic != 0 {
			value.Dynamic = &row.Dynamic
		}
		out = append(out, value)
	}
	return out, nil
}
