package game

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// cellRoute is the edit, or the map check, that puts a record on a cell.
type cellRoute uint8

const (
	admitAdd   cellRoute = iota // a new record: the Add window, add_record and each add call
	admitMove                   // Move and Clone of one record
	admitField                  // a field edit of one record
	admitGroup                  // the selection's move, delete-and-place and paste
	admitCheck                  // the map check's reading of a placed record
)

// cellAdmission is how one record kind on one route admits a cell. bounds
// refuses a cell outside the map; ring a cell of the border ring, which the
// game does not show and where it removes a placed unit (TERR-PLACE-208);
// arms a cell that one of these ground arms closes, naming every arm that
// closes it; domain judges a unit off the ground by its movement domain
// instead; footprint has checkPlacement judge the record's footprint.
type cellAdmission struct {
	bounds, ring bool
	arms         mapload.GroundArm
	domain       bool
	footprint    bool
}

type admissionKey struct {
	kind  string
	route cellRoute
}

// anyArm is every ground arm.
const anyArm = ^mapload.GroundArm(0)

// ringDrop is the kind a drop cell is admitted as.
const ringDrop = "Drop"

// cellAdmissions is the editor's one admission table (owner): a unit, a
// structure or a drop cell may not stand on the border ring; a sack may not
// be added or grouped onto a cell the ground arms close as far as the row
// says. A terrain, height, scenery or spell cell may.
var cellAdmissions = map[admissionKey]cellAdmission{
	{"Unit", admitAdd}:          {bounds: true, ring: true, footprint: true},
	{"Unit", admitMove}:         {bounds: true, ring: true, footprint: true},
	{"Unit", admitField}:        {footprint: true},
	{"Unit", admitGroup}:        {bounds: true, ring: true, arms: anyArm, domain: true, footprint: true},
	{"Unit", admitCheck}:        {ring: true},
	{"Structure", admitAdd}:     {bounds: true, ring: true, footprint: true},
	{"Structure", admitMove}:    {bounds: true, ring: true, footprint: true},
	{"Structure", admitField}:   {footprint: true},
	{"Structure", admitGroup}:   {bounds: true, ring: true, footprint: true},
	{"Sack", admitAdd}:          {bounds: true, arms: mapload.GroundBorder},
	{"Sack", admitGroup}:        {bounds: true, arms: anyArm},
	{"Enchantment", admitField}: {bounds: true},
	{ringDrop, admitAdd}:        {bounds: true, ring: true},
}

// defaultAdmissions is the row of a kind the table does not list on a route.
var defaultAdmissions = map[cellRoute]cellAdmission{
	admitAdd: {bounds: true}, admitMove: {bounds: true}, admitGroup: {bounds: true},
}

func admissionOf(kind string, route cellRoute) cellAdmission {
	if a, ok := cellAdmissions[admissionKey{kind, route}]; ok {
		return a
	}
	return defaultAdmissions[route]
}

// cellVerdict is an admission's judgement of a cell's bounds and ring.
type cellVerdict uint8

const (
	cellOpen cellVerdict = iota
	cellOutside
	cellInRing
)

func (a cellAdmission) verdict(view *alm.Map, cell image.Point) cellVerdict {
	if a.bounds && !cell.In(image.Rect(0, 0, int(view.Width), int(view.Height))) {
		return cellOutside
	}
	if a.ring && mapload.InBorderRing(view, cell.X, cell.Y) {
		return cellInRing
	}
	return cellOpen
}

// refusal words the verdict on cell; what names the cell.
func (a cellAdmission) refusal(view *alm.Map, cell image.Point, what string) error {
	switch a.verdict(view, cell) {
	case cellOutside:
		return fmt.Errorf("%s %d,%d is outside the map", what, cell.X, cell.Y)
	case cellInRing:
		return fmt.Errorf("%s %d,%d is in the border ring; the playable area is %s", what, cell.X, cell.Y, ui.PlayableText(mapload.Playable(view)))
	}
	return nil
}

// closedArms is what of the ground arms closing a cell refuses a record of
// the row, zero to admit it: plane is the cell's passability, d the record's
// movement domain.
func (a cellAdmission) closedArms(arms mapload.GroundArm, plane byte, d sim.Domain) mapload.GroundArm {
	if arms&a.arms == 0 {
		return 0
	}
	if a.domain && d != sim.DomainGround {
		if !d.BlockedBy(plane) {
			return 0
		}
		if b := arms & mapload.GroundBorder; b != 0 {
			return b
		}
	}
	return arms
}

// admitCell is the admission of a record of kind that route puts on cell of
// m, judged on m before the edit; what names the cell in a bounds or ring
// refusal. Its arms are judged on m too: a sack does not shape them. A
// refused arm is named as a group edit names it, the record by the index the
// new record would take.
func (s *mapInspectionEdits) admitCell(m *mapedit.Editor, cell image.Point, what, kind string, route cellRoute) error {
	view, err := m.Map()
	if err != nil {
		return err
	}
	a := admissionOf(kind, route)
	if err := a.refusal(view, cell, what); err != nil {
		return err
	}
	if a.arms == 0 {
		return nil
	}
	arms := mapload.GroundArms(view, s.inspector.table)
	c := cell.Y*int(view.Width) + cell.X
	if c < 0 || c >= len(arms) {
		return nil
	}
	if closed := a.closedArms(arms[c], 0, sim.DomainGround); closed != 0 {
		return selectionRefusal(s.closedNote(ui.RecordRef{Kind: kind, Index: m.Count(editKinds[kind])}, cell, closed))
	}
	return nil
}

// closedNote is the refusal of ref on cell, which arms close.
func (s *mapInspectionEdits) closedNote(ref ui.RecordRef, cell image.Point, arms mapload.GroundArm) string {
	return fmt.Sprintf(s.inspector.word("editor.selection.closed"), s.recordName(ref), cell.X, cell.Y, strings.Join(s.armWords(arms), ", "))
}

// playableAreaError refuses, at its first step, a record whose row refuses
// the ring, on a map whose ring covers every cell.
func playableAreaError(view *alm.Map, kind string, route cellRoute) error {
	if admissionOf(kind, route).ring && mapload.Playable(view).Empty() {
		return fmt.Errorf("the map has no playable area: its border ring covers every cell")
	}
	return nil
}
