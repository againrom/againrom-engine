package mapedit

// CellRole names the cell a type-9 record's coordinate field belongs to and
// its axis: the cell the record stands on (trigger), the cell its effect
// reads (source) and the cell it moves a unit to (target). CellNone marks
// every other field.
type CellRole uint8

const (
	CellNone CellRole = iota
	CellTriggerX
	CellTriggerY
	CellSourceX
	CellSourceY
	CellTargetX
	CellTargetY
)

// cellRoleKeys are the keys both formats address the role's field by.
var cellRoleKeys = [...]string{"", "triggerx", "triggery", "sourcex", "sourcey", "targetx", "targety"}

// Key returns the key of the field that holds the role.
func (r CellRole) Key() string { return cellRoleKeys[r] }

// Trigger reports that r is a coordinate of the cell the record stands on.
func (r CellRole) Trigger() bool { return r == CellTriggerX || r == CellTriggerY }

// IsTrapKind reports that a type-9 record of kind stands on a cell of its
// own, as a unit spell or a casting structure (kind 4 and up) stands on an
// object.
func IsTrapKind(kind uint16) bool { return kind < 4 }
