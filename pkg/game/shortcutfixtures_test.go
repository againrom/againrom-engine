package game

import "againrom/pkg/ui"

func spellSlots(ids ...uint32) (slots ui.ShortcutSlots) {
	for i, id := range ids {
		slots[i].Spell = id
	}
	return
}
