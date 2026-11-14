package ui

func spellSlots(ids ...uint32) (slots ShortcutSlots) {
	for i, id := range ids {
		slots[i].Spell = id
	}
	return
}
