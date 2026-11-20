package game

import (
	"errors"
	"math"
	"reflect"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// kindSet is a set of inspection record kinds; nil holds every kind.
type kindSet map[ui.RecordKind]bool

func (k kindSet) has(kind ui.RecordKind) bool { return k == nil || k[kind] }

// editRow is what the map editor's one host may do on a map of one format.
// The first game's row has every capability on; the second game's turns off
// what the format notes do not yet read. The record sections and the script
// profile of a format are the row of the same format in pkg/mapedit.
type editRow struct {
	// The record kinds a tool moves, adds (a clone adds), deletes and sets.
	moves, adds, deletes, sets kindSet
	// refusal is the word of the refusal a tool outside the row gives.
	refusal string
	// fields: the record panels list the settable fields of a record.
	fields bool
	// planes: the brush paints every plane, not the terrain alone.
	planes bool
	// newSteps are the keys of a new record the palettes offer; nil: all.
	newSteps map[string]bool
	// manyValues: one new record takes several values.
	manyValues bool
	// wholeScripts: a script node or trigger takes every field edit; off, an
	// edit writes the parameter words of a decoded node or moves a trigger.
	wholeScripts bool
	// scriptRefusal and undecoded are the words of that narrower write.
	scriptRefusal, undecoded string
	// structureKind refuses a structure kind the format has no building for.
	structureKind func(*MapInspector, int64) error
	// seams are the optional host seams the editor may use; nil: all.
	seams []reflect.Type
	// templatePlacing: a new unit or structure is a copy of one the map holds.
	templatePlacing bool
	// centredCells: a placed record sits at the centre of its cell.
	centredCells bool
	// admissions are the format's cell admission rows; nil: the shared ones.
	admissions map[admissionKey]cellAdmission
	// The inspection of the first game's authoring model.
	unitOverrides bool // a unit's lines show the overrides of its definition
	spellSets     bool // cell spells group into sets, and go with their object
	playerRecords bool // the player records are the first game's rows
	storedKeys    bool // each record has a key the model keeps
	playableRing  bool // a new record is refused in the border ring
	workspace     bool // the map has pages, labels and a save gate beside it
}

// editRows is the one table of rows, keyed by the map's format.
var editRows = map[alm.Format]*editRow{
	alm.FirstFormat: {
		wholeScripts: true, fields: true, planes: true, manyValues: true,
		structureKind: (*MapInspector).checkStructureKind,
		unitOverrides: true, spellSets: true, playerRecords: true, storedKeys: true, playableRing: true, workspace: true,
	},
	alm.SecondFormat: {
		moves:   placedKinds,
		adds:    placedKinds,
		deletes: placedKinds,
		sets:    kindSet{ui.KindAction: true, ui.KindCondition: true, ui.KindTrigger: true},
		refusal: "editor.map.secondtool", scriptRefusal: "editor.map.secondscript", undecoded: "editor.map.secondundecoded",
		newSteps:      map[string]bool{"identity": true, "kind": true, "item": true},
		structureKind: checkBuildingKind,
		seams: []reflect.Type{
			reflect.TypeFor[ui.InspectionSaveFolder](), reflect.TypeFor[ui.InspectionInPlace](),
			reflect.TypeFor[ui.InspectionNewRecords](), reflect.TypeFor[ui.InspectionBrush](),
			reflect.TypeFor[ui.InspectionBrushFill](), reflect.TypeFor[ui.InspectionBrushIcons](),
			reflect.TypeFor[ui.InspectionFieldSet](),
			reflect.TypeFor[ui.InspectionSelection](), reflect.TypeFor[ui.InspectionGroups](),
		},
		templatePlacing: true, centredCells: true, admissions: secondAdmissions,
	},
}

// placedKinds are the records the second game's palettes place: units,
// structures and ground sacks.
var placedKinds = kindSet{ui.KindUnit: true, ui.KindStructure: true, ui.KindSack: true}

// checkBuildingKind refuses a structure kind the installed building table
// does not name.
func checkBuildingKind(x *MapInspector, kind int64) error {
	if _, _, ok := mapload.BuildingName(x.table, uint32(kind)); !ok || kind > math.MaxUint8 {
		return wordErr("editor.err.structurekind", kind)
	}
	return nil
}

// rowFor is the row of m's format.
func rowFor(m *mapedit.Editor) *editRow {
	if m != nil {
		if r, ok := editRows[m.Format()]; ok {
			return r
		}
	}
	return editRows[alm.FirstFormat]
}

func (s *mapInspectionEdits) row() *editRow { return rowFor(s.model) }

// refuse is the error of a tool the row leaves off.
func (s *mapInspectionEdits) refuse() error {
	return errors.New(s.inspector.word(s.row().refusal))
}

// Offers reports whether the editor may use the optional seam on this map;
// the editor reads it through ui.Seam.
func (s *mapInspectionEdits) Offers(seam reflect.Type) bool {
	seams := s.row().seams
	return seams == nil || slices.Contains(seams, seam)
}
