package game

import (
	"errors"
	"math"
	"reflect"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
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
	// fieldKinds are the kinds that list and set fields while fields is off.
	fieldKinds    kindSet
	unitNoRowWord string
	unitNoRowID   func(alm.Unit) int64
	structureName func(*MapInspector, alm.Object) string
	// paints are the brush planes a stroke writes, each with its rule; nil:
	// every plane, any value. A plane outside it takes refusal.
	paints map[ui.BrushPlane]planeRule
	// randomTerrain: the whole-map terrain command redraws tile words. The
	// second game's is off: a look-alike word may hold another tile class
	// and terrain cost there (R2-ENGINE-1665).
	randomTerrain bool
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
	// kindTable says the install holds the table structureKind reads.
	kindTable func(*MapInspector) bool
	// allRules: the report runs every rule; off, only the rules marked Second.
	allRules bool
	// guestTest: test play plays a guest mission, which the campaign may refuse.
	guestTest bool
	// testMission is the mission row the test mod plays the map under.
	testMission func(x *MapInspector, source string) (mod.Mission, error)
	// seams are the optional host seams the editor may use; nil: all.
	seams []reflect.Type
	// templatePlacing: a new unit or structure is a copy of one the map holds.
	templatePlacing bool
	// centredCells: a placed record sits at the centre of its cell.
	centredCells bool
	// admissions are the format's cell admission rows; nil: the shared ones.
	admissions map[admissionKey]cellAdmission
	// The inspection of the first game's authoring model.
	unitOverrides  bool // a unit's lines show the overrides of its definition
	spellSets      bool // cell spells group into sets, and go with their object
	playerRecords  bool // the Players window lists the player's fields, not a raw record
	structureOwner bool // a structure record has an owner field
	storedKeys     bool // each record has a key the model keeps
	playableRing   bool // a new record is refused in the border ring
	workspace      bool // the map has pages, labels and a save gate beside it
}

// editRows is the one table of rows, keyed by the map's format.
var editRows = map[alm.Format]*editRow{
	alm.FirstFormat: {
		wholeScripts: true, fields: true, randomTerrain: true, manyValues: true,
		unitNoRowWord: "editor.unit.nodef",
		unitNoRowID: func(u alm.Unit) int64 {
			if u.DefID != 0 {
				return int64(u.DefID)
			}
			return int64(u.ClassID)
		},
		structureKind: (*MapInspector).checkStructureKind,
		kindTable:     func(x *MapInspector) bool { return x.structures != nil },
		allRules:      true,
		guestTest:     true,
		testMission:   func(x *MapInspector, _ string) (mod.Mission, error) { return x.testPlayGuestMission() },
		unitOverrides: true, spellSets: true, playerRecords: true, structureOwner: true, storedKeys: true, playableRing: true, workspace: true,
	},
	alm.SecondFormat: {
		moves: placedKinds,
		// A new node holds zero in the words no compiler read names
		// (R2-ENGINE-1728); a new trigger is the first game's 184 bytes.
		adds: kindSet{ui.KindUnit: true, ui.KindStructure: true, ui.KindSack: true, ui.KindPlayer: true,
			ui.KindTrigger: true, ui.KindAction: true, ui.KindCondition: true},
		// A node goes as the first game's does; the type-7 rows keep +0x28.
		deletes: kindSet{ui.KindUnit: true, ui.KindStructure: true, ui.KindSack: true, ui.KindAction: true, ui.KindCondition: true},
		sets:    kindSet{ui.KindAction: true, ui.KindCondition: true, ui.KindTrigger: true},
		// A trigger's 184 bytes are the first game's fields (R2-ENGINE-1316).
		fieldKinds:     kindSet{ui.KindUnit: true, ui.KindStructure: true, ui.KindSack: true, ui.KindStock: true, ui.KindPlayer: true, ui.KindTrigger: true},
		unitNoRowWord:  "editor.unit.norow",
		unitNoRowID:    func(u alm.Unit) int64 { return int64(u.ServerID) },
		structureName:  (*MapInspector).structureCatalogLine,
		playerRecords:  true,
		structureOwner: true,
		refusal:        "editor.map.secondtool", scriptRefusal: "editor.map.secondscript", undecoded: "editor.map.secondundecoded",
		newSteps:      map[string]bool{"identity": true, "kind": true, "item": true}, // a new player takes the roster's defaults
		structureKind: checkBuildingKind,
		kindTable:     func(x *MapInspector) bool { return x.table != nil },
		testMission:   (*MapInspector).testPlayReplaceMission,
		seams: []reflect.Type{
			reflect.TypeFor[ui.InspectionChecker](), reflect.TypeFor[ui.InspectionErrorSave](), reflect.TypeFor[ui.InspectionTestPlay](),
			reflect.TypeFor[ui.InspectionSaveFolder](), reflect.TypeFor[ui.InspectionInPlace](),
			reflect.TypeFor[ui.InspectionNewRecords](), reflect.TypeFor[ui.InspectionBrush](),
			reflect.TypeFor[ui.InspectionBrushFill](), reflect.TypeFor[ui.InspectionBrushIcons](),
			reflect.TypeFor[ui.InspectionFieldSet](), reflect.TypeFor[ui.InspectionTextFields](),
			reflect.TypeFor[ui.InspectionSelection](), reflect.TypeFor[ui.InspectionGroups](),
			reflect.TypeFor[ui.InspectionItemLook](),
			reflect.TypeFor[ui.InspectionTriggerCopy](), reflect.TypeFor[ui.InspectionTriggerWizard](),
			reflect.TypeFor[ui.InspectionWholeMap](),
		},
		paints:          secondPaints,
		templatePlacing: true, centredCells: true, admissions: secondAdmissions,
	},
}

// planeRule is what a brush plane writes on a format: values are the stroke
// values it takes, nil any, and refusal the word of another value.
type planeRule struct {
	values  []int
	refusal string
}

// secondPaints are the second game's brush planes. The type-1 word's bit
// 0x2000 and the type-2 height plane are read by the game (R2-ENGINE-1340,
// R2-ENGINE-1666); which object a type-3 byte shows is not, so the object
// brush only clears (R2-ENGINE-1667), and no game reader of the type-4 bridge
// element is read (R2-ASSET-262).
var secondPaints = map[ui.BrushPlane]planeRule{
	ui.BrushPlaneTerrain:     {},
	ui.BrushPlaneHeight:      {},
	ui.BrushPlanePassability: {},
	ui.BrushPlaneObjects:     {values: []int{0}, refusal: "editor.map.secondobjects"},
	ui.BrushPlaneBridge:      {values: []int{}, refusal: "editor.map.secondbridge"},
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
