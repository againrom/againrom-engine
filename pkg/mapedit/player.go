package mapedit

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/alm"
)

const (
	playerTypeID     = 5
	playerRecordSize = 76
	metaPlayerCount  = 0x1c
	playerLimit      = 16
	playerName       = 0x0c
	playerNameSize   = 32
	playerRelations  = 0x2c
)

var playerBaseFields = []Field{
	{Key: "name", Label: "Name", Offset: playerName, Width: playerNameSize, Text: &alm.PlayerNameText},
	{Key: "colour", Label: "Colour slot", Offset: 0, Width: 4, Min: 0, Max: 15},
	{Key: "control", Label: "Control (0 human, 1 computer)", Offset: 4, Width: 4, Min: 0, Max: 1},
}

func playerFields(e *Editor, _ []byte) []Field {
	out := append([]Field(nil), playerBaseFields...)
	for i := 0; i < min(e.Count(KindPlayer), playerLimit); i++ {
		out = append(out, Field{Key: fmt.Sprintf("relation%d", i), Label: fmt.Sprintf("Relation to player %d (0 neutral, 1 hostile, 2 ally)", i+1),
			Offset: playerRelations + 2*i, Width: 2, Min: 0, Max: 2, Toward: i + 1})
	}
	return out
}

// NewPlayerRecord builds the next roster row without changing the map.
func (e *Editor) NewPlayerRecord() ([]byte, error) {
	w, err := e.placementSection(KindPlayer, "add a player")
	if err != nil {
		return nil, err
	}
	if w.n() >= playerLimit {
		return nil, fmt.Errorf("mapedit: the roster already has %d players; maximum %d", w.n(), playerLimit)
	}
	rec := make([]byte, playerRecordSize)
	binary.LittleEndian.PutUint32(rec, uint32(min(w.n()+1, 15)))
	binary.LittleEndian.PutUint32(rec[4:], 1)
	binary.LittleEndian.PutUint32(rec[8:], 5000)
	copy(rec[playerName:playerName+playerNameSize], fmt.Sprintf("New Player %d", w.n()+1))
	for j := 0; j < playerLimit; j++ {
		value := uint16(1)
		if j == w.n() {
			value = 2
		}
		binary.LittleEndian.PutUint16(rec[playerRelations+2*j:], value)
	}
	return rec, nil
}

func validatePlayerPlacement(_ *Editor, w walk, rec []byte) error {
	if w.n() >= playerLimit {
		return fmt.Errorf("mapedit: the roster already has %d players; maximum %d", w.n(), playerLimit)
	}
	for _, f := range playerBaseFields {
		if f.Text == nil {
			if v := f.decode(rec[f.Offset:]); v < f.Min || v > f.Max {
				return fmt.Errorf("mapedit: %s %d is outside %d..%d", f.Label, v, f.Min, f.Max)
			}
		}
	}
	name := rec[playerName : playerName+playerNameSize]
	if at := bytes.IndexByte(name, 0); at >= 0 {
		name = name[:at]
	}
	if bytes.IndexByte(name, 0x98) >= 0 {
		return fmt.Errorf("mapedit: Name must fit Windows-1251")
	}
	for j := 0; j < playerLimit; j++ {
		if v := binary.LittleEndian.Uint16(rec[playerRelations+2*j:]); v > 2 {
			return fmt.Errorf("mapedit: relation to player %d is %d, outside 0..2", j+1, v)
		}
	}
	return nil
}

func playerDeleteParts(e *Editor, roster walk, index int) ([]part, error) {
	if index == 0 {
		return nil, fmt.Errorf("mapedit: cannot delete the first player")
	}
	slot := uint32(index + 1)
	var parts []part
	// Structure owners resolve through their low word; actor owners keep all bits.
	for _, owner := range []struct {
		kind   Kind
		offset int
		mask   uint32
	}{{KindUnit, 0x14, ^uint32(0)}, {KindStructure, 0x0e, 0xffff}} {
		if !roster.frame.carries(sections[owner.kind].typeID) {
			continue
		}
		w, err := e.walkSection(owner.kind, "check player references")
		if err != nil {
			return nil, err
		}
		for i := 0; i < w.n(); i++ {
			at, _ := w.at(i)
			p := at + owner.offset
			value := e.u32(p) & owner.mask
			if value == slot {
				return nil, fmt.Errorf("mapedit: cannot delete player slot %d: %s %d names it as owner", slot, w.sec.name, i)
			}
			if value > slot && uint64(value) <= uint64(roster.n()) {
				parts = append(parts, e.wordPart(p, -1))
			}
		}
	}
	if roster.frame.carries(scriptTypeID) {
		for _, kind := range []Kind{KindAction, KindCondition} {
			w, err := e.walkSection(kind, "check player references")
			if err != nil {
				return nil, err
			}
			for i := 0; i < w.n(); i++ {
				at, _ := w.at(i)
				for j := 0; j < nodeSlots; j++ {
					p := at + nodeValues + 4*j
					if e.u32(at+nodeTypes+4*j) != paramPlayer {
						continue
					}
					value := e.u32(p)
					if value == slot {
						return nil, fmt.Errorf("mapedit: cannot delete player slot %d: %s %d parameter %d names it", slot, w.sec.name, i, j)
					}
					if value > slot && uint64(value) <= uint64(roster.n()) {
						parts = append(parts, e.wordPart(p, -1))
					}
				}
			}
		}
	}
	if index >= playerLimit {
		return parts, nil
	}
	for i := 0; i < roster.n(); i++ {
		if i == index {
			continue
		}
		at, _ := roster.at(i)
		start, end := at+playerRelations+2*index, at+playerRecordSize
		next := append(cloneBytes(e.data[start+2:end]), 0, 0)
		parts = append(parts, replace(start, e.data[start:end], next))
	}
	return parts, nil
}
