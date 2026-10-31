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
