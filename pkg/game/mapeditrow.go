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
	// unitBookWord is the word the unit window states for the unit's spell
	// book and maximum mana and the refusal of a book edit; "" offers both.
	unitBookWord  string
	structureName func(*MapInspector, alm.Object) string
	// randomTerrain: the whole-map terrain command redraws tile words. The
	// second game's is off: a look-alike word may hold another tile class
	// and terrain cost there (R2-ENGINE-1665).
	randomTerrain bool
	// newSteps are the keys of a new record the palettes offer; nil: all.
	newSteps map[string]bool
	// sceneryLabel and sceneryDetail word a cell's type-3 byte record; ""
	// names the byte as an object class.
	sceneryLabel, sceneryDetail string
	// trapWord words a trap's view from its spell, power, source and target;
	// "" words it from the kinds its mode names.
	trapWord string
	// manyValues: one new record takes several values; off, a sack still
	// takes its items and a type-9 record its template and spell.
	manyValues bool
	// wholeScripts: a script node or trigger takes every field edit; off, an
	// edit writes a node's label and texts, the operation and parameter
	// words of a decoded node, or moves a trigger.
	wholeScripts bool
	// scriptRefusal, undecoded and unstated are the words of that narrower
	// write; unstated refuses an operation whose slots no claim states.
	scriptRefusal, undecoded, unstated string
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
	// eventText is the one text file of #key sections the map's events show,
	// by the map's campaign mission, 0 for none; nil: the map's pages are
	// files of their own. scopes are the workspace scopes the row offers;
	// nil: all.
	eventText func(mission int) textAddress
	scopes    []string
	marks     mapload.MarkTable
	// passNoteWord is the word given when the passability view is switched on,
	// and passClosedWord the hover's word for a cell with arms; "": the plain ones.
	passNoteWord, passClosedWord string
	// passBrush are the passability brush's rows; nil: block and unblock.
	passBrush []ui.PassBrushRow
	// rawHeights: the height numbers show the stored height.
	rawHeights bool
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
		marks: mapload.FirstMarks,
	},
	alm.SecondFormat: {
		moves: placedKinds,
		// A new node holds zero in the words no compiler read names
		// (R2-ENGINE-1728); a new trigger is the first game's 184 bytes. A
		// type-9 record is the cell move its field table admits.
		adds: kindSet{ui.KindUnit: true, ui.KindStructure: true, ui.KindSack: true, ui.KindPlayer: true,
			ui.KindTrigger: true, ui.KindAction: true, ui.KindCondition: true, ui.KindEnchantment: true},
		// A node goes as the first game's does; the type-7 rows keep +0x28.
		deletes: kindSet{ui.KindUnit: true, ui.KindStructure: true, ui.KindSack: true, ui.KindAction: true, ui.KindCondition: true, ui.KindEnchantment: true},
		sets:    kindSet{ui.KindAction: true, ui.KindCondition: true, ui.KindTrigger: true, ui.KindEnchantment: true},
		// A trigger's 184 bytes are the first game's fields (R2-ENGINE-1316).
		fieldKinds:     kindSet{ui.KindUnit: true, ui.KindStructure: true, ui.KindSack: true, ui.KindStock: true, ui.KindPlayer: true, ui.KindTrigger: true, ui.KindEnchantment: true},
		unitNoRowWord:  "editor.unit.norow",
		unitNoRowID:    func(u alm.Unit) int64 { return int64(u.ServerID) },
		structureName:  (*MapInspector).structureCatalogLine,
		playerRecords:  true,
		structureOwner: true,
		refusal:        "editor.map.secondtool", scriptRefusal: "editor.map.secondscript", undecoded: "editor.map.secondundecoded",
		unstated: "editor.map.secondunstated",
		// A new player takes the roster's defaults; a type-9 record takes the
		// steps its field table names.
		newSteps: map[string]bool{"identity": true, "kind": true, "item": true, "template": true, "spell": true, "link": true, "power": true, "range": true},
		// A type-3 byte blocks a walker inside the band; what a value shows
		// is unread (R2-ENGINE-1667).
		sceneryLabel: "editor.second.scenery", sceneryDetail: "editor.second.scenery.detail",
		// The mode-below-4 arm reads no mode bit; the spell picks the
		// entering unit or the target cell (R2-ENGINE-1698).
		trapWord:      "editor.cast.secondtrap",
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
			reflect.TypeFor[ui.InspectionWholeMap](), reflect.TypeFor[ui.InspectionWorkspace](),
		},
		// A campaign mission's events read its own file, any other map's
		// quest.txt (R2-ENGINE-049, R2-ENGINE-047).
		eventText: func(mission int) textAddress {
			if mission > 0 {
				return editorMissionText.at(mission)
			}
			return editorQuestText.at()
		},
		scopes:          []string{"pages"},
		templatePlacing: true, centredCells: true, admissions: secondAdmissions,
		// The unit record holds no mana word, and what the mode-8 record
		// gives a unit is unread (R2-ENGINE-1292, R2-SESSION-162).
		unitBookWord: "editor.map.secondbook",
		// The arms are the ring (R2-ENGINE-1341) and the cells the terrain
		// build marks (R2-ENGINE-1340); the value they leave closes move
		// kinds by the route test (R2-ENGINE-1805, Medium). The brush sets the
		// values its file bytes reach: 1 and 0; which type-3 value stores 5
		// with no object drawn is unread (R2-ENGINE-1667).
		marks: mapload.SecondMarks, passNoteWord: "editor.view.secondpass", passClosedWord: "editor.pass.marked",
		passBrush:  []ui.PassBrushRow{{Value: 1, Word: "editor.brush.secondclose"}, {Value: 0, Word: "editor.brush.secondopen"}},
		rawHeights: true,
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

// holdsWorkspace: the edits keep a workspace and its history beside the map.
func (r *editRow) holdsWorkspace() bool { return r.workspace || r.eventText != nil }

// offersScope reports whether the row's workspace offers scope.
func (r *editRow) offersScope(scope string) bool {
	return r.scopes == nil || slices.Contains(r.scopes, scope)
}

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
