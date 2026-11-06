package ui

// DisplayName is the character display name's name-table stage: own, else
// row classIndex of the unit name table; a structure takes no class row.
func DisplayName(kind InspectionKind, own string, classIndex int, words *Words) (string, bool) {
	if own != "" {
		return own, true
	}
	if kind == InspectionStructure || words == nil || classIndex < 0 || classIndex >= len(words.UnitNames) || words.UnitNames[classIndex] == "" {
		return "", false
	}
	return words.UnitNames[classIndex], true
}

// NPCName is the display name's NPC-word stage: line word-1 of the NPC name
// table, and false for a zero word, a word outside the table or an empty line.
func NPCName(word int, words *Words) (string, bool) {
	line := word - 1
	if words == nil || line < 0 || line >= len(words.NPCNames) || words.NPCNames[line] == "" {
		return "", false
	}
	return words.NPCNames[line], true
}
