package ui

import "encoding/json"

type Shortcut struct {
	Spell uint32 `json:",omitempty"`
	Item  string `json:",omitempty"`
}

type ShortcutSlots [9]Shortcut

func (s *Shortcut) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] != '{' {
		return json.Unmarshal(b, &s.Spell)
	}
	type plain Shortcut
	return json.Unmarshal(b, (*plain)(s))
}

// SetQuickSpells attaches the session's spell-or-item slots in shortcut order.
// An empty slot is unbound. The owner outlives this viewer; no actor owns these slots.
// A standalone viewer without this seam cannot change session bindings.
func (v *Viewer) SetQuickSpells(slots *ShortcutSlots) { v.quickSpells = slots }

// QuickSpellState reports a value snapshot for diagnostics and headless
// witnesses. It does not expose the session's writable storage.
func (v *Viewer) QuickSpellState() (slots ShortcutSlots, current uint32, armed bool) {
	if v.quickSpells != nil {
		slots = *v.quickSpells
	}
	return slots, v.selectedSpell, v.spellModeLive()
}

// quickSpell binds or invokes slots.
// Items use the inventory route.
func (v *Viewer) quickSpell(slot int, ctrl bool, x, y int) {
	if v.quickSpells == nil || slot < 0 || slot >= len(v.quickSpells) {
		return
	}
	if ctrl {
		id := v.selectedSpell
		if id == 0 {
			x, y = v.windowToFrame(x, y)
			if idx, ok := v.spellbookEntryAt(x, y); ok {
				id = v.spellbook[idx].ID
			}
		}
		if id != 0 {
			v.bindShortcut(slot, Shortcut{Spell: id})
		}
	} else if s := v.quickSpells[slot]; s.Item != "" {
		if v.inventoryEligible() {
			for i, key := range v.invSubject.PackKeys {
				if key == s.Item {
					v.invEquipRequest = i + 1
					break
				}
			}
		}
		return
	} else if s.Spell != 0 {
		v.selectedSpell = s.Spell
	}
	if id := v.quickSpells[slot].Spell; id != 0 && !ctrl {
		for _, entry := range v.spellbook {
			if entry.ID == id && entry.Unavailable && entry.ItemCast != nil {
				v.ArmItemCast(v.spellbookOwner, entry.ItemCast.Index, id, entry.ItemCast.Key)
				return
			}
		}
	}
	// Ctrl also reaches this test. A closed book never opens as a side effect.
	if id := v.quickSpells[slot].Spell; id != 0 && !v.hudShown(hudPanelBook) && v.spellAvailable(id) &&
		canArmAttack(v.sel, v.entities, v.localOwner) && v.commandCastActive() {
		// The same one-shot arm as C (AI-QUICKINVOKE-279).
		v.armSpell()
		v.spellNeedsBook, v.castOnce = false, true
	}
}

func (v *Viewer) bindShortcutItem(slot, x, y int) {
	if v.quickSpells == nil || slot < 0 || slot >= len(v.quickSpells) {
		return
	}
	x, y = v.windowToFrame(x, y)
	i, ok := v.inventoryPackCellAt(x, y)
	if !ok || i >= len(v.invSubject.PackKeys) || v.invSubject.PackKeys[i] == "" {
		return
	}
	v.bindShortcut(slot, Shortcut{Item: v.invSubject.PackKeys[i]})
}

func (v *Viewer) bindShortcut(slot int, s Shortcut) {
	for i, other := range v.quickSpells {
		if other == s {
			v.quickSpells[i] = Shortcut{}
		}
	}
	v.quickSpells[slot] = s
}

func (v *Viewer) spellAvailable(id uint32) bool {
	for _, entry := range v.spellbook {
		if entry.ID == id {
			return !entry.Unavailable
		}
	}
	return false
}

// castKey is the C key on the map (MENU-054, MENU-055). It reports whether the
// key was consumed. An empty selection and a selection with the foreign or
// structure bit set are not consumed: the original's second dispatch returns 0
// for C, and no other handler of this build takes the key. A nonempty selection
// with no spell-capable object is consumed with no message, sound or state
// change. With no spell selected C is a no-op (owner-observed, DIV-2265).
// Otherwise Cast mode is armed with the selected spell and the closed book
// opens.
func (v *Viewer) castKey() bool {
	if len(presentSelected(v.sel, v.entities)) == 0 {
		return false
	}
	summary := v.selectionSummary()
	if summary&selSummaryNeutral != 0 {
		return false
	}
	if summary&selSummarySpell != 0 && v.selectedSpell != 0 {
		// A book that C itself opens belongs to this one cast; a book the
		// player already had open is left as he opened it (DIV-2265).
		once := v.castOnce || !v.hudShown(hudPanelBook)
		v.armCast()
		v.showSpellBar()
		v.castOnce = once
	}
	return true
}

// endCastOnce spends a C-armed cast: the armed hook drops to the book-chosen
// way (the spell stays selected, live only while the book is shown) and the
// book that C opened closes, through the same switch the icon and Space read,
// so no panel disagrees with its icon afterwards.
func (v *Viewer) endCastOnce() {
	v.spellArmed, v.spellNeedsBook, v.castOnce = true, true, false
	v.cmdOverlayHidden = false
	if v.hudShown(hudPanelBook) {
		v.toggleHudPanel(hudPanelBook)
	}
}

// showSpellBar opens a closed spell bar and leaves an open one, so a second C
// is idempotent. It selects no spell (MENU-064, MENU-065, DIV-2076).
func (v *Viewer) showSpellBar() {
	if !v.hudShown(hudPanelBook) {
		v.toggleHudPanel(hudPanelBook)
	}
}

// armCast raises Cast mode from the key or the panel cell. The mode stands
// with the book closed: only a spell chosen in the book needs the book shown.
func (v *Viewer) armCast() {
	v.armSpell()
	v.spellNeedsBook = false
}

// spellModeLive: a spell chosen in the book casts only while the book is
// shown (AI-SPELLGUARD-289, AI-CURSOR-226); a key-armed spell stands without it.
func (v *Viewer) spellModeLive() bool {
	return v.spellArmed && (!v.spellNeedsBook || v.hudShown(hudPanelBook))
}

// armBookSpell is the book-chosen arm: a click on a book cell and a LOAD of a
// saved current spell, whose SAV state names no other mode.
func (v *Viewer) armBookSpell() {
	v.armSpell()
	v.spellNeedsBook = true
}

func (v *Viewer) armSpell() {
	v.spellArmed, v.castOnce = true, false
	v.armed, v.attackHeld, v.aimed = false, false, commandNone
	v.cmdOverlayHidden = false
}

// Book producers filter per accepted object, independently of the selection's
// union availability and Cast capability (AI-SPELLPOP-287). Item casts bypass
// this producer entirely. Custom viewers with no projection retain their seam.
func bookCasters(present []MapEntity, spell uint32) []MapEntity {
	if spell == 0 {
		return nil // AI-SPELLGUARD-289: a negative original current emits nothing.
	}
	casters := make([]MapEntity, 0, min(len(present), 253))
	for _, e := range present {
		if e.SpellStateKnown && (spell >= 32 || e.KnownSpells&(uint32(1)<<spell) == 0) {
			continue
		}
		casters = append(casters, e)
		if len(casters) == 253 {
			break
		}
	}
	return casters
}
