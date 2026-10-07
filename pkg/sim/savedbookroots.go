package sim

import (
	"cmp"
	"fmt"
	"slices"

	"againrom/pkg/rules"
)

// BookSlotValues resolves legacy rules without changing the actor. Explicit
// book instances keep their current literals, including unusual byte values.
func BookSlotValues(r Rules, e Entity, rules []SpellRule) [28]SourceItemSpell {
	var out [28]SourceItemSpell
	if e.Book.State == BookLegacy {
		e.Book = Spellbook{State: BookPresent}
		for _, rule := range rules {
			if rule.ID < 1 || rule.ID > 28 || e.KnownSpells&(1<<rule.ID) == 0 {
				continue
			}
			v := &e.Book.Slots[rule.ID-1]
			v.ManaCost = uint16(rule.ManaCost)
			if rule.Defensive {
				v.Defensive = 1
			}
		}
		RefreshBook(r, &e, rules)
	}
	if !e.Book.HasInstances() {
		return out
	}
	for slot, value := range e.Book.Slots {
		if e.KnownSpells&(1<<uint(slot+1)) != 0 {
			out[slot] = SourceItemSpell{Present: true, ID: uint8(slot + 1), Range: value.Range, Defensive: value.Defensive, ManaCost: value.ManaCost}
		}
	}
	return out
}

// A book mutation changes only its own incoming edge. It may update a unique
// Spell in place, but cannot change a weapon or another book through an alias.
func (w *World) refreshSavedBookRoots(index int) {
	if w.savedObjects == nil || index < 0 || index >= len(w.entities) {
		return
	}
	e := w.entities[index]
	position, found := slices.BinarySearchFunc(w.savedObjects.BookRoots, e.ID, func(root SavedBookRoot, id EntityID) int {
		return cmp.Compare(root.Entity, id)
	})
	if !found {
		return
	}
	values := BookSlotValues(w.rules, e, w.spells)
	root := w.savedObjects.BookRoots[position]
	changed := false
	for slot, id := range root.Slots {
		if id == 0 {
			changed = changed || values[slot].Present
		} else if child := w.savedObjects.spell(id); child == nil || child.Value != values[slot] {
			changed = true
		}
	}
	if !changed {
		return
	}
	r := w.savedObjects.Clone()
	uses := map[SavedObjectID]uint64{}
	for _, item := range r.Items {
		if !item.Retired {
			uses[item.Spell]++
		}
	}
	for _, book := range r.BookRoots {
		for _, id := range book.Slots {
			uses[id]++
		}
	}
	for slot, value := range values {
		id := root.Slots[slot]
		child := r.spell(id)
		if child != nil && child.Value == value {
			continue
		}
		if !value.Present {
			root.Slots[slot] = 0
			continue
		}
		if child != nil && uses[id] == 1 && child.ExternalReferences == 0 {
			child.Value = value
			continue
		}
		// Zero is an explicit unbound native slot. Even an exhausted identity
		// namespace must retain the actor's new value and unrelated aliases.
		root.Slots[slot] = 0
		if len(r.Items)+len(r.Effects)+len(r.Spells)+len(r.Sacks)+len(r.Containers)+len(r.ItemRoots)+len(r.BookRoots) >= MaxSavedObjects {
			continue
		}
		fresh, err := r.mint()
		if err != nil {
			continue
		}
		root.Slots[slot] = fresh
		r.Spells = append(r.Spells, SavedSpellObject{ID: fresh, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated},
			Value: value, Coverage: SavedObjectCoverage{Unknown: SavedUnknownSpellInitialization}})
	}
	r.BookRoots[position] = root
	r.RefreshChildLiveness()
	w.savedObjects = r
}

func (w *World) validateSavedBookRoots() error {
	if w.savedObjects == nil {
		return nil
	}
	for _, root := range w.savedObjects.BookRoots {
		index := indexOfEntity(w.entities, root.Entity)
		if index < 0 {
			return fmt.Errorf("sim: saved book root has no current actor")
		}
		e := w.entities[index]
		values := BookSlotValues(w.rules, e, w.spells)
		if e.Book.State == BookLegacy && !w.rules.HasSpellFormulas() {
			for _, rule := range w.spells {
				if rule.ID >= 1 && rule.ID <= 28 && values[rule.ID-1].Present {
					var level int32
					if rule.School < skillSlots {
						level = e.Skill[rule.School]
					}
					values[rule.ID-1].Range = originalSpellRange(rule, level, e.Mind)
				}
			}
		}
		for slot, id := range root.Slots {
			if id != 0 {
				child := w.savedObjects.spell(id)
				if child == nil || !w.bookRootValueMatches(child.Value, values[slot]) {
					return fmt.Errorf("sim: saved book root differs from current actor")
				}
			}
		}
	}
	return nil
}

// A legacy range may lie between its ordinary projection and the live power
// bound. Every other instance operand must match.
func (w *World) bookRootValueMatches(saved, derived SourceItemSpell) bool {
	if saved == derived {
		return true
	}
	if !saved.Present || !derived.Present || saved.ID != derived.ID {
		return false
	}
	rule, ok := w.findSpell(uint32(saved.ID))
	if !ok {
		return false
	}
	bottom, top := int64(derived.Range), spellRange(rule, spellPowerMax)
	if w.rules.SpellFormulaTable(rules.FormulaRange, rule.ID) != nil {
		bottom, top = w.rangeSpan(rule)
	}
	if top > 255 || int64(saved.Range) < bottom || int64(saved.Range) > top {
		return false
	}
	saved.Range = derived.Range
	return saved == derived
}
