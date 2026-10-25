package data

// HumanSpeakerFaces is a speaker record for every Humans row at its server id,
// the row's figure in what outfit says it wears, plus the player's character
// at key me when positive. Server ids above 10000 give colliding keys
// (R2-ENGINE-072) and no record.
func HumanSpeakerFaces(c Collection, me int32, outfit func(row int) Equipment) map[int32]NPCFace {
	out := map[int32]NPCFace{}
	for i := 1; i < c.Len(); i++ {
		p := c.EntryParams(i)
		key, ok := HumanServerID(p)
		if !ok || key <= 0 || key > 10000 {
			continue
		}
		d, err := NewHumanDef(c.EntryName(i), p)
		if err != nil {
			continue
		}
		dir, face := FigureFor(d.TypeID, d.Face, d.Gender)
		tokens := NPCTokens(NPCTokenHuman) | NPCTokens(NPCTokenFace) | NPCTokens(NPCTokenNotMage) | NPCTokens(NPCTokenNotFemale)
		if dir.Mage() {
			tokens ^= NPCTokens(NPCTokenNotMage) | NPCTokens(NPCTokenMage)
		}
		if dir.Female() {
			tokens ^= NPCTokens(NPCTokenNotFemale) | NPCTokens(NPCTokenFemale)
		}
		rec := NPCFace{Kind: NPCFigure, Dir: dir, Face: face, Tokens: tokens}
		if outfit != nil {
			rec.Worn = outfit(i)
		}
		out[key] = rec
	}
	if me > 0 {
		out[me] = NPCFace{Kind: NPCNoPicture, Tokens: NPCTokens(NPCTokenHero) | NPCTokens(NPCTokenMe)}
	}
	return out
}
