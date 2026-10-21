package ui

import "image"

// InspectionTriggerWizard is implemented by edits whose host writes a
// trigger the wizard built, as one edit: its name, once switch, condition
// pairs and actions, with every node a row names created, rewritten or kept.
type InspectionTriggerWizard interface {
	WriteTrigger(t WizardTrigger) (*InspectionDocument, error)
}

// WizardNode is one node a wizard row names. Keep names node ID as it
// stands. Otherwise the node becomes Opcode with Types and Values: node ID
// is rewritten in place when no other slot of any trigger names it, else a
// new node is made; ID 0 always makes a new node.
type WizardNode struct {
	ID            uint32
	Keep          bool
	Opcode        uint32
	Types, Values [10]uint32
}

// WizardPair is one condition row: its two sides and their comparison code
// (TRIG-CMP-006).
type WizardPair struct {
	Left, Right WizardNode
	Cmp         uint32
}

// WizardTrigger is the trigger the wizard writes: Index -1 adds a trigger.
// A node the trigger named before and no trigger names after is deleted
// where deleting it moves no variable register.
type WizardTrigger struct {
	Index   int
	Name    string
	Once    uint32
	Pairs   []WizardPair
	Actions []WizardNode
}

// InspectionDropZones is implemented by edits whose host adds, moves and
// deletes drop zones, each as one edit. Zone z is the z-th drop action in
// file order, the z-th of InspectionDocument.Drops.
type InspectionDropZones interface {
	AddDropZone(cell image.Point) (*InspectionDocument, error)
	MoveDropZone(zone int, cell image.Point) (*InspectionDocument, error)
	DeleteDropZone(zone int) (*InspectionDocument, error)
}
