package sav

import (
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"slices"

	"againrom/pkg/formats/reg"
)

// A Token row is one byte in the first game's records (T0C) and a word in
// the second game's, whose Humans and Buildings rows run past 255
// (R2-ENGINE-739, R2-ENGINE-209). An engine document in the first game's records writes a
// wider row's low byte in the record and the row in its AgainromRows leaf:
// version 1, then the archive index and the row of each such record, in
// index order. A document whose rows all fit a byte writes no leaf. Decoding
// gives each record its row back and leaves the leaf out of the state.
const tokenRowsRefusal = "sav: AgainromRows requires version 1 and ascending archive index, row pairs above 255"

type tokenRow struct {
	index uint16
	row   uint32
}

// wideRowClasses are the classes whose row the rows leaf carries: a Human
// and each Building class.
var wideRowClasses = []string{"Human", "Building", "Outpost", "Tavern", "Shop"}

// wideTokenRow reports a row the first game's one-byte Token row cannot
// hold and the rows leaf carries.
func wideTokenRow(class string, m Member, v uint32) bool {
	return slices.Contains(wideRowClasses, class) && m.Name == "T0C" && m.Kind == KindU8 && v > 0xff && v <= 0xffff
}

func validateTokenRows(b []byte) error {
	_, err := decodeTokenRows(b)
	return err
}

func decodeTokenRows(b []byte) ([]tokenRow, error) {
	le := binary.LittleEndian
	if len(b) < 12 || (len(b)-4)%8 != 0 || le.Uint32(b) != 1 {
		return nil, errors.New(tokenRowsRefusal)
	}
	rows := make([]tokenRow, 0, (len(b)-4)/8)
	prior := uint32(0)
	for at := 4; at < len(b); at += 8 {
		index, row := le.Uint32(b[at:]), le.Uint32(b[at+4:])
		if index <= prior || index > 0xffff || row <= 0xff || row > 0xffff {
			return nil, errors.New(tokenRowsRefusal)
		}
		prior = index
		rows = append(rows, tokenRow{uint16(index), row})
	}
	return rows, nil
}

func encodeTokenRows(rows []tokenRow) []byte {
	b := binary.LittleEndian.AppendUint32(nil, 1)
	for _, r := range rows {
		b = binary.LittleEndian.AppendUint32(b, uint32(r.index))
		b = binary.LittleEndian.AppendUint32(b, r.row)
	}
	return b
}

// withTokenRows is state holding rows in the rows leaf, and no such leaf when
// rows is empty. A state that already agrees is returned unchanged.
func withTokenRows(state *cityState, rows []tokenRow) (*cityState, error) {
	if state == nil {
		if len(rows) == 0 {
			return state, nil
		}
		return nil, fmt.Errorf("sav: a Token row above 255 needs a state store")
	}
	record, present, err := state.leafIndex().read(nativeRowsLeaf.path)
	if err != nil {
		return nil, errors.New(nativeRowsLeaf.refusals.ambiguous)
	}
	if !present && len(rows) == 0 {
		return state, nil
	}
	slices.SortFunc(rows, func(a, b tokenRow) int { return int(a.index) - int(b.index) })
	out := &cityState{rootKind: state.rootKind, directoryKinds: state.directoryKinds, values: maps.Clone(state.values)}
	path := nativeRowsLeaf.path
	if present {
		path = record.Path
		delete(out.values, path)
	}
	if len(rows) != 0 {
		if _, ok := state.directoryKinds["/"+supplementDirectory]; !ok {
			return nil, fmt.Errorf("sav: a Token row above 255 needs the %s directory", supplementDirectory)
		}
		out.values[path] = cityStateValue{kind: nativeRowsLeaf.kind, bytes: encodeTokenRows(rows)}
	}
	return out, nil
}

// takeTokenRows reads the rows leaf and answers state without it.
func takeTokenRows(state *cityState) (*cityState, []tokenRow, error) {
	if state == nil {
		return state, nil, nil
	}
	record, present, err := state.leafIndex().read(nativeRowsLeaf.path)
	if err != nil {
		return nil, nil, errors.New(nativeRowsLeaf.refusals.ambiguous)
	}
	if !present {
		return state, nil, nil
	}
	if record.Value.Kind != nativeRowsLeaf.kind || record.Value.Int32 != 0 {
		return nil, nil, errors.New(tokenRowsRefusal)
	}
	rows, err := decodeTokenRows(record.Value.Bytes)
	if err != nil {
		return nil, nil, err
	}
	out := &cityState{rootKind: state.rootKind, directoryKinds: state.directoryKinds, values: maps.Clone(state.values)}
	delete(out.values, record.Path)
	return out, rows, nil
}

// fileTokenRows reads the rows leaf of a file's state store.
func (f *File) fileTokenRows() ([]tokenRow, error) {
	node, present, err := f.StateLeaves().ReadLeaf(nativeRowsLeaf.path)
	if err != nil {
		return nil, errors.New(nativeRowsLeaf.refusals.ambiguous)
	}
	if !present {
		return nil, nil
	}
	if node.Dir || node.Type != reg.TypeIntArray {
		return nil, errors.New(tokenRowsRefusal)
	}
	b := make([]byte, 0, 4*len(node.Ints))
	for _, v := range node.Ints {
		b = binary.LittleEndian.AppendUint32(b, uint32(v))
	}
	return decodeTokenRows(b)
}

// applyFileRows gives the records the walk has read their rows from the
// file's rows leaf. A complete walk has read every record a row may name.
func (w *walker) applyFileRows(f *File, complete bool) error {
	rows, err := f.fileTokenRows()
	if err != nil || len(rows) == 0 {
		return err
	}
	if !complete {
		rows = slices.DeleteFunc(rows, func(r tokenRow) bool { return r.index >= w.next })
	}
	return applyTokenRows(rows, func(i uint16) *Record { return w.objects[i] })
}

// applyTokenRows gives each record of rows its row. The record must hold the
// row's low byte in a one-byte T0C.
func applyTokenRows(rows []tokenRow, record func(uint16) *Record) error {
	for _, r := range rows {
		rec := record(r.index)
		if rec == nil || !slices.Contains(wideRowClasses, rec.Class) || rec.Value == nil || rec.Value["T0C"] != r.row&0xff || !tokenRowByte(rec) {
			return fmt.Errorf("sav: AgainromRows names archive object %d, which holds no Token row %d", r.index, r.row)
		}
		rec.Value["T0C"] = r.row
	}
	return nil
}

// tokenRowByte reports whether r's class writes T0C in one byte.
func tokenRowByte(r *Record) bool {
	prog, ok := r.profile().programmes[r.Class]
	return ok && programmeHolds(r.profile(), prog, func(m Member) bool { return m.Name == "T0C" && m.Kind == KindU8 })
}

func programmeHolds(p *Profile, prog []step, match func(Member) bool) bool {
	for _, s := range prog {
		switch s.op {
		case stpRun:
			if slices.ContainsFunc(s.members, match) {
				return true
			}
		case stpClass:
			if programmeHolds(p, p.programmes[s.class], match) {
				return true
			}
		}
	}
	return false
}
