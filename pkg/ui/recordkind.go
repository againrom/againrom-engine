package ui

// RecordKind is the word id of an editor record kind, the Kind of an
// InspectionRecord and of a RecordRef. Its row in recordKinds states what the
// kind can do; the editor and its host read those capabilities and compare a
// kind only with the constants below, never with a string.
type RecordKind string

const (
	KindUnit        RecordKind = "Unit"
	KindStructure   RecordKind = "Structure"
	KindSack        RecordKind = "Sack"
	KindStock       RecordKind = "Stock"
	KindScenery     RecordKind = "Scenery"
	KindSection     RecordKind = "Section"
	KindPlayer      RecordKind = "Player"
	KindTrigger     RecordKind = "Trigger"
	KindCondition   RecordKind = "Condition"
	KindAction      RecordKind = "Action"
	KindMessage     RecordKind = "Message"
	KindScript      RecordKind = "Script"
	KindEnchantment RecordKind = "Enchantment"
	KindMod         RecordKind = "Mod"
	// KindDrop is the cell the script drops the party on; the host admits a
	// cell for it as for a record, but the map holds no record of it.
	KindDrop RecordKind = "Drop"
)

// recordKindRow is what one record kind can do.
//
//   - owned: the record carries an owner slot (InspectionRecord.Player).
//   - placed: the record stands on the map and a drag, a selection, a clone or
//     a move takes it; rank orders the placed kinds when one press hits several.
//   - scripted: the record is a node of the map's script.
//   - listed: a list filter shows the record by its kind; filter is that
//     filter's number.
//   - editable: the window clones, deletes and sets fields of the record.
//   - fielded: the window sets fields of the record.
//   - addressed: scripts name the record by its ScriptID.
type recordKindRow struct {
	owned, placed, scripted, listed bool
	editable, fielded, addressed    bool
	rank, filter                    int
}

// recordKinds holds every record kind the editor knows: the one place a
// kind's abilities are written.
var recordKinds = map[RecordKind]recordKindRow{
	KindUnit:        {owned: true, placed: true, listed: true, editable: true, fielded: true, addressed: true, rank: 1, filter: 1},
	KindSack:        {placed: true, listed: true, editable: true, fielded: true, rank: 2, filter: 3},
	KindStructure:   {owned: true, placed: true, listed: true, editable: true, fielded: true, addressed: true, rank: 3, filter: 2},
	KindStock:       {listed: true, editable: true, fielded: true, filter: 3},
	KindScenery:     {listed: true, filter: 4},
	KindPlayer:      {listed: true, fielded: true, filter: editorPlayersFilter},
	KindTrigger:     {scripted: true, fielded: true},
	KindCondition:   {scripted: true, fielded: true},
	KindAction:      {scripted: true, fielded: true},
	KindEnchantment: {fielded: true},
	KindSection:     {},
	KindMessage:     {},
	KindScript:      {},
	KindMod:         {},
	KindDrop:        {},
}

// Owned reports a kind whose records carry an owner slot.
func (k RecordKind) Owned() bool { return recordKinds[k].owned }

// Placed reports a kind whose records stand on the map and move with a drag.
func (k RecordKind) Placed() bool { return recordKinds[k].placed }

// Scripted reports a kind whose records are nodes of the map's script.
func (k RecordKind) Scripted() bool { return recordKinds[k].scripted }

// Listed reports a kind a list filter shows by its kind.
func (k RecordKind) Listed() bool { return recordKinds[k].listed }

// Editable reports a kind the window clones, deletes and sets fields of.
func (k RecordKind) Editable() bool { return recordKinds[k].editable }

// Fielded reports a kind whose fields the window sets.
func (k RecordKind) Fielded() bool { return recordKinds[k].fielded }

// Addressed reports a kind whose records scripts name by ScriptID.
func (k RecordKind) Addressed() bool { return recordKinds[k].addressed }

// DragRank orders the placed kinds when one press hits several; zero for a
// kind a drag does not move.
func (k RecordKind) DragRank() int { return recordKinds[k].rank }

// Filter is the number of the list filter that shows the kind, zero for none.
func (k RecordKind) Filter() int { return recordKinds[k].filter }

// String is the kind's word id.
func (k RecordKind) String() string { return string(k) }

// placedKindOf is the placed kind a list filter shows; none for a filter that
// shows no placed kind. Each placed kind has its own filter.
func placedKindOf(filter int) (RecordKind, bool) {
	for k, row := range recordKinds {
		if row.placed && row.filter == filter {
			return k, true
		}
	}
	return "", false
}

// kindWords are the word ids of kinds, for a command schema's choices.
func kindWords(kinds ...RecordKind) []string {
	out := make([]string, len(kinds))
	for i, k := range kinds {
		out[i] = k.String()
	}
	return out
}

// ScriptNodeKind is the kind of a script's condition node or action node.
func ScriptNodeKind(condition bool) RecordKind {
	if condition {
		return KindCondition
	}
	return KindAction
}
