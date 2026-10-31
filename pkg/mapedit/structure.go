package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"

	"againrom/pkg/formats/alm"
)

func structureLength(_ *Editor, b []byte) int {
	if len(b) < alm.ObjectSize {
		return -1
	}
	if binary.LittleEndian.Uint32(b[alm.ObjectKind:]) != alm.ObjectExtKind {
		return alm.ObjectSize
	}
	return fixedLength(b, alm.ObjectSize+alm.ObjectExtSize)
}

// reshapeStructure gives a record whose kind became 0x21 a zero extension and
// drops the extension of one whose kind left 0x21. Zero override bytes select
// the table footprint (TERR-STRUCT-090), so the added extension changes no
// footprint.
func reshapeStructure(rec []byte) []byte {
	ext := binary.LittleEndian.Uint32(rec[alm.ObjectKind:]) == alm.ObjectExtKind
	switch {
	case ext && len(rec) == alm.ObjectSize:
		return append(rec, make([]byte, alm.ObjectExtSize)...)
	case !ext && len(rec) > alm.ObjectSize:
		return rec[:alm.ObjectSize]
	}
	return rec
}

var structureKindField = Field{Key: "kind", Label: "Kind (structures.reg ID)", Offset: alm.ObjectKind, Width: 4, Min: 1, Max: 255, Check: CheckStructureKind}

var structureFieldsBase = []Field{
	structureKindField,
	{Key: "owner", Label: "Owner slot", Offset: alm.ObjectOwner, Width: 2, Min: 1, Max: 16, Check: CheckRosterSlot},
	{Key: "identity", Label: "Script target id", Offset: alm.ObjectIdentity, Width: 2, Min: 1, Max: math.MaxInt16, Check: CheckStructureIdentity},
	{Key: "placement", Label: "Placement word (signed raw)", Offset: alm.ObjectPlacement, Width: 2, Signed: true, Min: math.MinInt16, Max: math.MaxInt16},
}

var structureExtFields = []Field{
	{Key: "width", Label: "Footprint width (0 = table)", Offset: alm.ObjectWidth, Width: 1, Min: 0, Max: 255},
	{Key: "height", Label: "Footprint height (0 = table)", Offset: alm.ObjectHeight, Width: 1, Min: 0, Max: 255},
}

func structureFields(_ *Editor, rec []byte) []Field {
	if len(rec) == alm.ObjectSize {
		return append([]Field(nil), structureFieldsBase...)
	}
	return append(append([]Field{structureKindField}, structureExtFields...), structureFieldsBase[1:]...)
}

func (e *Editor) structureIdentityParts(index int, old, value int64) ([]part, error) {
	w, err := e.walkSection(KindStructure, "set a structure identity")
	if err != nil {
		return nil, err
	}
	for i := 0; i < w.n(); i++ {
		at, _ := w.at(i)
		id := int64(binary.LittleEndian.Uint16(e.data[at+alm.ObjectIdentity:]))
		if i != index && (id == value || id == old) {
			return nil, fmt.Errorf("mapedit: structure identity %d is ambiguous or %d is already used", old, value)
		}
	}
	if !e.locate().carries(alm.TypeScript) {
		return nil, nil
	}
	var out []part
	for _, k := range []Kind{KindAction, KindCondition} {
		nodes, err := e.walkSection(k, "retarget a structure identity")
		if err != nil {
			return nil, err
		}
		for n := 0; n < nodes.n(); n++ {
			at, _ := nodes.at(n)
			for j := 0; j < alm.ScriptParams; j++ {
				p := at + alm.ScriptNodeValues + 4*j
				if e.u32(at+alm.ScriptNodeTypes+4*j) != paramStructure {
					continue
				}
				if int64(e.u32(p)) == value && value != old {
					return nil, fmt.Errorf("mapedit: structure identity %d is reserved by a script", value)
				}
				if int64(e.u32(p)) == old {
					var b [4]byte
					binary.LittleEndian.PutUint32(b[:], uint32(value))
					out = append(out, replace(p, e.data[p:p+4], b[:]))
				}
			}
		}
	}
	return out, nil
}

func (e *Editor) NewStructureRecord(x, y, kind uint32, owner uint16) ([]byte, error) {
	w, err := e.placementSection(KindStructure, "build a structure")
	if err != nil {
		return nil, err
	}
	if kind < 1 || kind > 255 || owner < 1 || owner > 16 {
		return nil, fmt.Errorf("mapedit: invalid structure kind %d or owner %d", kind, owner)
	}
	rec := make([]byte, alm.ObjectSize)
	binary.LittleEndian.PutUint32(rec, x)
	binary.LittleEndian.PutUint32(rec[4:], y)
	binary.LittleEndian.PutUint32(rec[8:], kind)
	binary.LittleEndian.PutUint16(rec[alm.ObjectPlacement:], 1)
	binary.LittleEndian.PutUint16(rec[alm.ObjectOwner:], owner)
	if err := renumberStructure(e, w, rec); err != nil {
		return nil, err
	}
	return reshapeStructure(rec), nil
}

func renumberStructure(e *Editor, w walk, rec []byte) error {
	highest := uint32(0)
	for i := 0; i < w.n(); i++ {
		at, _ := w.at(i)
		highest = max(highest, uint32(binary.LittleEndian.Uint16(e.data[at+alm.ObjectIdentity:])))
	}
	scripted, err := e.highestScriptTarget(paramStructure, math.MaxUint32)
	if err != nil {
		return err
	}
	highest = max(highest, scripted)
	if highest >= math.MaxInt16 {
		return fmt.Errorf("mapedit: no positive signed structure identity above %d", highest)
	}
	binary.LittleEndian.PutUint16(rec[alm.ObjectIdentity:], uint16(highest+1))
	return nil
}
