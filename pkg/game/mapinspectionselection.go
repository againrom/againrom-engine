package game

import (
	"errors"
	"fmt"
	"image"
	"math"
	"slices"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// selectionClip is a copied set of records: a model of the map they came
// from as it stood at the copy, that map's source name, each record's address
// and cell in it, and the top-left of those cells, which a paste puts on the
// target cell. An area copy also holds the terrain patch of its rectangle,
// and the origin is the rectangle's top-left cell.
type selectionClip struct {
	source  *mapedit.Editor
	from    string
	records []clipRecord
	origin  image.Point
	patch   *mapedit.Patch
}

type clipRecord struct {
	ref  ui.RecordRef
	kind mapedit.Kind
	cell image.Point
}

// Len is how many things a paste places: the records, and a terrain patch
// as one.
func (c *selectionClip) Len() int {
	if c.patch != nil {
		return len(c.records) + 1
	}
	return len(c.records)
}

// selectionKind is the model section of a kind the group selection takes.
func selectionKind(kind string) (mapedit.Kind, error) {
	switch kind {
	case "Unit", "Structure", "Sack":
		return editKinds[kind], nil
	}
	return 0, fmt.Errorf("a %s record cannot be in a group", strings.ToLower(kind))
}

// recordName is the status note's name of a record.
func (s *mapInspectionEdits) recordName(ref ui.RecordRef) string {
	switch ref.Kind {
	case "Unit":
		return fmt.Sprintf(s.inspector.word("editor.selection.unit"), ref.Index)
	case "Structure":
		return fmt.Sprintf(s.inspector.word("editor.selection.structure"), ref.Index)
	}
	return fmt.Sprintf(s.inspector.word("editor.selection.sack"), ref.Index)
}

// selectionRefusal is a refusal already worded for the status note.
type selectionRefusal string

func (r selectionRefusal) Error() string { return string(r) }

// refusalNote is the note of a record an edit refused for err.
func (s *mapInspectionEdits) refusalNote(ref ui.RecordRef, err error) string {
	var r selectionRefusal
	if errors.As(err, &r) {
		return string(r)
	}
	return fmt.Sprintf(s.inspector.word("editor.selection.refused"), s.recordName(ref), err)
}

// selectionNote is the lead of a group edit's status note: what it did to how
// many of total records, then the extra parts. The window adds the refusals.
func selectionNote(format string, done, total int, extra []string) string {
	return strings.Join(append([]string{fmt.Sprintf(format, done, total)}, extra...), "; ")
}

// CopyRecords copies the records: a model of the current map and each
// record's cell.
func (s *mapInspectionEdits) CopyRecords(refs []ui.RecordRef) (ui.InspectionClip, string, error) {
	source, err := mapedit.New(s.model.Bytes())
	if err != nil {
		return nil, "", err
	}
	clip := &selectionClip{source: source, from: s.source}
	for _, ref := range refs {
		k, err := selectionKind(ref.Kind)
		if err != nil {
			return nil, "", err
		}
		x, y, err := source.Position(k, ref.Index)
		if err != nil {
			return nil, "", err
		}
		cell := image.Pt(int(x>>8), int(y>>8))
		if len(clip.records) == 0 {
			clip.origin = cell
		}
		clip.origin = image.Pt(min(clip.origin.X, cell.X), min(clip.origin.Y, cell.Y))
		clip.records = append(clip.records, clipRecord{ref: ref, kind: k, cell: cell})
	}
	if len(clip.records) == 0 {
		return nil, "", fmt.Errorf("no records to copy")
	}
	return clip, fmt.Sprintf(s.inspector.word("editor.selection.copied"), len(clip.records)), nil
}

// groupPlacement is one record of a move or a paste: the cell it is to stand
// on and the call that puts it there, which answers its index afterwards.
type groupPlacement struct {
	ref   ui.RecordRef
	kind  mapedit.Kind
	cell  image.Point
	apply func(m *mapedit.Editor) (int, error)
}

// placeGroup applies every placement to m as one edit and answers the records
// it placed and a note per record it refused. A record is refused when its
// cell is outside the map, when its call fails, when a structure's footprint
// leaves the map, or when the passability plane of the edited map closes a
// unit's cell to the unit's movement domain or a sack's cell to a ground
// mover. Units and sacks do not shape that plane, so one more pass without
// the closed ones settles the edit.
func (s *mapInspectionEdits) placeGroup(m *mapedit.Editor, items []groupPlacement) ([]ui.RecordRef, []string, error) {
	view, err := m.Map()
	if err != nil {
		return nil, nil, err
	}
	bounds := image.Rect(0, 0, int(view.Width), int(view.Height))
	mark := m.Mark()
	closed := map[int]string{}
	for {
		var refusals []string
		placed := map[int]int{}
		for n, it := range items {
			if why, ok := closed[n]; ok {
				refusals = append(refusals, why)
				continue
			}
			if !it.cell.In(bounds) {
				refusals = append(refusals, fmt.Sprintf(s.inspector.word("editor.selection.outside"), s.recordName(it.ref), it.cell.X, it.cell.Y))
				continue
			}
			before := m.Mark()
			i, err := it.apply(m)
			if err == nil {
				err = s.checkPlacement(it.kind, m, i)
			}
			if err != nil {
				for m.Mark() > before {
					m.Undo()
				}
				refusals = append(refusals, s.refusalNote(it.ref, err))
				continue
			}
			placed[n] = i
		}
		after, err := m.Map()
		if err != nil {
			return nil, nil, err
		}
		arms := mapload.GroundArms(after, s.inspector.table)
		plane := mapload.PassabilityWith(after, s.inspector.table)
		again := false
		for n, it := range items {
			i, ok := placed[n]
			if !ok || it.kind == mapedit.KindStructure || len(arms) != bounds.Dx()*bounds.Dy() || len(plane) != len(arms) {
				continue
			}
			c := it.cell.Y*bounds.Dx() + it.cell.X
			a := arms[c]
			if it.kind == mapedit.KindUnit && i < len(after.Units) {
				d, err := mapload.UnitDomain(after.Units[i], s.inspector.table)
				if err == nil && d != sim.DomainGround {
					a = 0
					if d.BlockedBy(plane[c]) {
						a = arms[c] & mapload.GroundBorder
						if a == 0 {
							a = arms[c]
						}
					}
				}
			}
			if a != 0 {
				closed[n] = fmt.Sprintf(s.inspector.word("editor.selection.closed"), s.recordName(it.ref), it.cell.X, it.cell.Y, strings.Join(s.armWords(a), ", "))
				again = true
			}
		}
		if !again {
			var refs []ui.RecordRef
			for n, it := range items {
				if i, ok := placed[n]; ok {
					refs = append(refs, ui.RecordRef{Kind: it.ref.Kind, Index: i})
				}
			}
			m.Squash(mark)
			return refs, refusals, nil
		}
		for m.Mark() > mark {
			m.Undo()
		}
	}
}

// armWords are the passability words of the arms that close a cell.
func (s *mapInspectionEdits) armWords(a mapload.GroundArm) []string {
	pairs := [...]struct {
		arm mapload.GroundArm
		id  string
	}{{mapload.GroundBorder, "editor.pass.border"}, {mapload.GroundMountain, "editor.pass.mountain"}, {mapload.GroundWater, "editor.pass.water"},
		{mapload.GroundFlag, "editor.pass.flag"}, {mapload.GroundObject, "editor.pass.object"}, {mapload.GroundStructure, "editor.pass.structure"}}
	var out []string
	for _, p := range pairs {
		if a&p.arm != 0 {
			out = append(out, s.inspector.word(p.id))
		}
	}
	return out
}

// MoveRecords moves every record by offset cells, each keeping its fraction
// within the cell, as one edit.
func (s *mapInspectionEdits) MoveRecords(refs []ui.RecordRef, offset image.Point) (ui.SelectionEdit, error) {
	if offset == (image.Point{}) || len(refs) == 0 {
		return ui.SelectionEdit{}, nil
	}
	var items []groupPlacement
	for _, ref := range refs {
		k, err := selectionKind(ref.Kind)
		if err != nil {
			return ui.SelectionEdit{}, err
		}
		x, y, err := s.model.Position(k, ref.Index)
		if err != nil {
			return ui.SelectionEdit{}, err
		}
		index := ref.Index
		cell := image.Pt(int(x>>8), int(y>>8)).Add(offset)
		items = append(items, groupPlacement{ref: ref, kind: k, cell: cell, apply: func(m *mapedit.Editor) (int, error) {
			nx, ny := cellWords(cell, x, y)
			return index, m.Move(k, index, nx, ny)
		}})
	}
	return s.groupEdit(items, s.inspector.word("editor.selection.moved"), nil, refs)
}

// PasteRecords copies every record of the clip so that the clip's top-left
// cell lands on cell, each through the clone of its kind, as one edit. A
// record whose owner slot the target map's roster lacks is refused. Into the
// map the clip was copied from, each record is what one Clone of it writes.
// Into another map, each group a pasted unit holds, group 0 aside, takes a
// group id no unit and no script of that map uses, the same one for every
// unit of the group, so no script of that map names a pasted unit. A clip
// with a terrain patch writes it first, its top-left cell on cell and cut at
// the map's edge, and the records then stand on the pasted terrain; patch
// and records are one edit.
func (s *mapInspectionEdits) PasteRecords(c ui.InspectionClip, cell image.Point) (ui.SelectionEdit, error) {
	clip, ok := c.(*selectionClip)
	if !ok || clip == nil || clip.Len() == 0 {
		return ui.SelectionEdit{}, fmt.Errorf("nothing to paste")
	}
	same := clip.from == s.source
	groups, err := s.pasteGroups(clip, same)
	if err != nil {
		return ui.SelectionEdit{}, err
	}
	offset := cell.Sub(clip.origin)
	var items []groupPlacement
	for _, r := range clip.records {
		r := r
		target := r.cell.Add(offset)
		items = append(items, groupPlacement{ref: r.ref, kind: r.kind, cell: target, apply: func(m *mapedit.Editor) (int, error) {
			if err := s.pasteOwner(m, clip.source, r); err != nil {
				return -1, err
			}
			x, y, err := clip.source.Position(r.kind, r.ref.Index)
			if err != nil {
				return -1, err
			}
			nx, ny := cellWords(target, x, y)
			if same {
				return m.CloneFromCopy(clip.source, r.kind, r.ref.Index, nx, ny)
			}
			i, err := m.CloneFrom(clip.source, r.kind, r.ref.Index, nx, ny)
			if err != nil || r.kind != mapedit.KindUnit {
				return i, err
			}
			g, err := clip.source.FieldValue(mapedit.KindUnit, r.ref.Index, "group")
			if err != nil {
				return i, err
			}
			if fresh, ok := groups[g]; ok {
				err = m.SetField(mapedit.KindUnit, i, "group", fresh)
			}
			return i, err
		}})
	}
	recipes := s.model.Count(mapedit.KindEnchantment)
	var terrain func(*mapedit.Editor) (bool, error)
	if clip.patch != nil {
		terrain = func(m *mapedit.Editor) (bool, error) {
			return m.PastePatch(*clip.patch, mapedit.Cell{X: cell.X, Y: cell.Y})
		}
	}
	return s.groupEditOn(terrain, items, s.inspector.word("editor.selection.pasted"), func(m *mapedit.Editor) []string {
		var out []string
		if clip.patch != nil {
			out = append(out, fmt.Sprintf(s.inspector.word("editor.selection.terrain"), clip.patch.W, clip.patch.H))
		}
		if n := m.Count(mapedit.KindEnchantment) - recipes; n > 0 {
			out = append(out, fmt.Sprintf(s.inspector.word("editor.selection.recipes"), n))
		}
		if n := pastedGroups(m, groups); n > 0 {
			out = append(out, fmt.Sprintf(s.inspector.word("editor.selection.regrouped"), n))
		}
		return out
	}, nil)
}

// pasteGroups maps each non-zero group id the clip's units hold to a fresh
// group id of the target map, in the clip's order, for a paste into another
// map; nil for a paste into the map the clip came from.
func (s *mapInspectionEdits) pasteGroups(clip *selectionClip, same bool) (map[int64]int64, error) {
	if same {
		return nil, nil
	}
	next, err := s.model.HighestGroup()
	if err != nil {
		return nil, err
	}
	groups := map[int64]int64{}
	for _, r := range clip.records {
		if r.kind != mapedit.KindUnit {
			continue
		}
		g, err := clip.source.FieldValue(mapedit.KindUnit, r.ref.Index, "group")
		if err != nil {
			return nil, err
		}
		if _, ok := groups[g]; ok || g == 0 {
			continue
		}
		if next >= math.MaxUint32 {
			return nil, fmt.Errorf("this map has no free group id left")
		}
		next++
		groups[g] = int64(next)
	}
	return groups, nil
}

// pastedGroups is how many of the fresh group ids a unit of m holds.
func pastedGroups(m *mapedit.Editor, groups map[int64]int64) int {
	if len(groups) == 0 {
		return 0
	}
	view, err := m.Map()
	if err != nil {
		return 0
	}
	held := map[int64]bool{}
	for _, u := range view.Units {
		held[int64(u.GroupID)] = true
	}
	n := 0
	for _, fresh := range groups {
		if held[fresh] {
			n++
		}
	}
	return n
}

// pasteOwner refuses a unit or a structure whose owner slot the roster of
// its own map holds and the roster of m does not. A slot neither roster holds
// is not a paste's doing, so it is copied as a clone copies it.
func (s *mapInspectionEdits) pasteOwner(m *mapedit.Editor, source *mapedit.Editor, r clipRecord) error {
	if r.kind != mapedit.KindUnit && r.kind != mapedit.KindStructure {
		return nil
	}
	owner, err := source.FieldValue(r.kind, r.ref.Index, "owner")
	if err != nil {
		return err
	}
	from, err := source.Map()
	if err != nil {
		return err
	}
	view, err := m.Map()
	if err != nil {
		return err
	}
	if rosterSlotError(from, uint32(owner)) == nil && rosterSlotError(view, uint32(owner)) != nil {
		return selectionRefusal(fmt.Sprintf(s.inspector.word("editor.selection.noplayer"), s.recordName(r.ref), owner))
	}
	return nil
}

// groupEdit runs placeGroup as one edit and words its outcome in format.
// extra adds note parts read off the edited model; keep, when nothing
// changed, is the group the window keeps.
func (s *mapInspectionEdits) groupEdit(items []groupPlacement, format string, extra func(*mapedit.Editor) []string, keep []ui.RecordRef) (ui.SelectionEdit, error) {
	return s.groupEditOn(nil, items, format, extra, keep)
}

// groupEditOn is groupEdit after first, when it is not nil, has written the
// map; first and the placements are one edit.
func (s *mapInspectionEdits) groupEditOn(first func(*mapedit.Editor) (bool, error), items []groupPlacement, format string, extra func(*mapedit.Editor) []string, keep []ui.RecordRef) (ui.SelectionEdit, error) {
	var placed []ui.RecordRef
	var refusals, more []string
	d, err := s.change(func(m *mapedit.Editor) (bool, error) {
		mark, wrote := m.Mark(), false
		if first != nil {
			var err error
			if wrote, err = first(m); err != nil {
				return false, err
			}
		}
		var err error
		placed, refusals, err = s.placeGroup(m, items)
		if err == nil && extra != nil {
			more = extra(m)
		}
		m.Squash(mark)
		return wrote || len(placed) > 0, err
	})
	if err != nil {
		return ui.SelectionEdit{}, err
	}
	out := ui.SelectionEdit{Doc: d, Selected: placed, Note: selectionNote(format, len(placed), len(items), more), Refusals: refusals}
	if d == nil {
		out.Selected = keep
	}
	return out, nil
}

// DeleteRecords removes every record as one edit, the highest index of each
// kind first so the lower indices still name their records.
func (s *mapInspectionEdits) DeleteRecords(refs []ui.RecordRef) (ui.SelectionEdit, error) {
	order := slices.Clone(refs)
	slices.SortStableFunc(order, func(a, b ui.RecordRef) int {
		if a.Kind != b.Kind {
			return strings.Compare(a.Kind, b.Kind)
		}
		return b.Index - a.Index
	})
	var refusals []string
	deleted := 0
	d, err := s.change(func(m *mapedit.Editor) (bool, error) {
		refusals, deleted = nil, 0
		mark := m.Mark()
		for _, ref := range order {
			k, err := selectionKind(ref.Kind)
			if err == nil {
				err = m.Delete(k, ref.Index)
			}
			if err != nil {
				refusals = append(refusals, s.refusalNote(ref, err))
				continue
			}
			deleted++
		}
		m.Squash(mark)
		return deleted > 0, nil
	})
	if err != nil {
		return ui.SelectionEdit{}, err
	}
	out := ui.SelectionEdit{Doc: d, Note: selectionNote(s.inspector.word("editor.selection.deleted"), deleted, len(refs), nil), Refusals: refusals}
	if d == nil {
		out.Selected = refs
	}
	return out, nil
}
