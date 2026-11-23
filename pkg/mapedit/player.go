package mapedit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"againrom/pkg/formats/alm"
)

const (
	playerLimit = 16
)

var playerBaseFields = []Field{
	{Key: "name", Label: "Name", Offset: alm.GroupName, Width: alm.GroupNameLen, Text: &alm.PlayerNameText},
	{Key: "colour", Label: "Colour slot", Offset: alm.GroupColour, Width: 4, Min: 0, Max: 15},
	{Key: "control", Label: "Control (0 human, 1 computer)", Offset: alm.GroupControl, Width: 4, Min: 0, Max: 1},
}

func playerFields(e *Editor, _ []byte) []Field {
	return append(append([]Field(nil), playerBaseFields...), playerRelationFields(e)...)
}

// playerRelationFields is one relation field per player of the roster.
func playerRelationFields(e *Editor) []Field {
	var out []Field
	for i := 0; i < min(e.Count(KindPlayer), playerLimit); i++ {
		out = append(out, Field{Key: fmt.Sprintf("relation%d", i), Label: fmt.Sprintf("Relation to player %d (0 neutral, 1 hostile, 2 ally)", i+1),
			Offset: alm.GroupRelations + 2*i, Width: 2, Min: 0, Max: 2, Toward: i + 1})
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
	return w.sec.blank(w.n()), nil
}

// blankPlayer is the first game's roster row n+1: hostile to every player and
// allied to itself.
func blankPlayer(n int) []byte {
	return newPlayerRow(n, 2)
}

// newPlayerRow is a roster row for player n+1 with the diagonal word self.
func newPlayerRow(n int, self uint16) []byte {
	rec := make([]byte, alm.GroupSize)
	binary.LittleEndian.PutUint32(rec, uint32(min(n+1, 15)))
	binary.LittleEndian.PutUint32(rec[4:], 1)
	binary.LittleEndian.PutUint32(rec[8:], 5000)
	copy(rec[alm.GroupName:alm.GroupName+alm.GroupNameLen], fmt.Sprintf("New Player %d", n+1))
	for j := 0; j < playerLimit; j++ {
		value := uint16(1)
		if j == n {
			value = self
		}
		binary.LittleEndian.PutUint16(rec[alm.GroupRelations+2*j:], value)
	}
	return rec
}

// The second game's roster row has the first game's layout; the loader forces
// the diagonal word to 0x12, no claim reads the +4 and +8 words as a control
// or a score, and the relation byte is held to the shipped 0, 1 and 2.
const secondSelfRelation = 0x12

func blankSecondPlayer(n int) []byte {
	return newPlayerRow(n, secondSelfRelation)
}

var secondPlayerBaseFields = []Field{
	playerBaseFields[0],
	{Key: "colour", Label: "Colour slot", Offset: alm.GroupColour, Width: 4, Min: 1, Max: 15},
	{Key: "flags", Label: "Word +4 (unread by claims)", Offset: alm.GroupControl, Width: 4, Min: 0, Max: math.MaxUint32},
	{Key: "unread", Label: "Word +8 (not read by the loader)", Offset: alm.GroupScalar, Width: 4, Min: 0, Max: math.MaxUint32},
}

func secondPlayerFields(e *Editor, rec []byte) []Field {
	out := append([]Field(nil), secondPlayerBaseFields...)
	return append(out, playerRelationFields(e)...)
}

// secondPlayerLocked holds the two unread words and the diagonal word, which
// the loader overwrites.
func secondPlayerLocked(i int, f Field) bool {
	return f.Key == "flags" || f.Key == "unread" || f.Toward == i+1
}

func validatePlayerName(rec []byte) error {
	name := rec[alm.GroupName : alm.GroupName+alm.GroupNameLen]
	if at := bytes.IndexByte(name, 0); at >= 0 {
		name = name[:at]
	}
	if bytes.IndexByte(name, 0x98) >= 0 {
		return fmt.Errorf("mapedit: Name must fit Windows-1251")
	}
	return nil
}

func validatePlayerPlacement(_ *Editor, w walk, rec []byte) error {
	return validatePlayerRow(w, rec, playerBaseFields, -1)
}

func validateSecondPlayerPlacement(_ *Editor, w walk, rec []byte) error {
	return validatePlayerRow(w, rec, secondPlayerBaseFields, w.n())
}

// validatePlayerRow checks a new roster row against fields; a row with a
// diagonal word holds it at secondSelfRelation.
func validatePlayerRow(w walk, rec []byte, fields []Field, diagonal int) error {
	if w.n() >= playerLimit {
		return fmt.Errorf("mapedit: the roster already has %d players; maximum %d", w.n(), playerLimit)
	}
	for _, f := range fields {
		if f.Text == nil && f.Key != "unread" && f.Key != "flags" {
			if v := f.decode(rec[f.Offset:]); v < f.Min || v > f.Max {
				return fmt.Errorf("mapedit: %s %d is outside %d..%d", f.Label, v, f.Min, f.Max)
			}
		}
	}
	if err := validatePlayerName(rec); err != nil {
		return err
	}
	for j := 0; j < playerLimit; j++ {
		v := binary.LittleEndian.Uint16(rec[alm.GroupRelations+2*j:])
		if j == diagonal && v == secondSelfRelation {
			continue
		}
		if v > 2 {
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
	}{{KindUnit, alm.UnitOwner, ^uint32(0)}, {KindStructure, alm.ObjectOwner, 0xffff}} {
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
	if roster.frame.carries(alm.TypeScript) {
		for _, kind := range []Kind{KindAction, KindCondition} {
			w, err := e.walkSection(kind, "check player references")
			if err != nil {
				return nil, err
			}
			for i := 0; i < w.n(); i++ {
				at, _ := w.at(i)
				for j := 0; j < alm.ScriptParams; j++ {
					p := at + alm.ScriptNodeValues + 4*j
					if e.u32(at+alm.ScriptNodeTypes+4*j) != paramPlayer {
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
		start, end := at+alm.GroupRelations+2*index, at+alm.GroupSize
		next := append(cloneBytes(e.data[start+2:end]), 0, 0)
		parts = append(parts, replace(start, e.data[start:end], next))
	}
	return parts, nil
}
