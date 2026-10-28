package data

// HumanSpeakerFaces is a speaker record for every Humans row at its server id,
// the row's figure in what outfit says it wears, plus the player's character
// at key me when positive. Server ids above 10000 give colliding keys
// (R2-ENGINE-072) and no record, except a row companions names: companions
// maps a row's server id to the identity key the party member built from it
// carries. That record is a hero, never the player's own, and fills its key
// only where no row's own server id is that key (DIV-4199).
func HumanSpeakerFaces(c Collection, me int32, companions map[int32]int32, outfit func(row int) Equipment) map[int32]NPCFace {
	out := map[int32]NPCFace{}
	joined := map[int32]NPCFace{}
	for i := 1; i < c.Len(); i++ {
		p := c.EntryParams(i)
		key, ok := HumanServerID(p)
		identity, companion := companions[key]
		if companion {
			key = identity
		}
		if !ok || key <= 0 || key > 10000 {
			continue
		}
		d, err := NewHumanDef(c.EntryName(i), p)
		if err != nil {
			continue
		}
		dir, face := FigureFor(d.TypeID, d.Face, d.Gender)
		tokens := NPCTokens(NPCTokenHuman) | NPCTokens(NPCTokenFace) | NPCTokens(NPCTokenNotMage) | NPCTokens(NPCTokenNotFemale)
		if companion {
			tokens ^= NPCTokens(NPCTokenHuman) | NPCTokens(NPCTokenHero) | NPCTokens(NPCTokenNotMe)
		}
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
		if companion {
			joined[key] = rec
		} else {
			out[key] = rec
		}
	}
	for key, rec := range joined {
		if _, held := out[key]; !held {
			out[key] = rec
		}
	}
	if me > 0 {
		out[me] = NPCFace{Kind: NPCNoPicture, Tokens: NPCTokens(NPCTokenHero) | NPCTokens(NPCTokenMe)}
	}
	return out
}

// FallbackSpeakerFace is the record of the actor the second game builds for a
// speaker no live unit carries (R2-ENGINE-583): its flags 0x56 draw the
// paperdoll of a female mage with face byte npc-0x50, wearing nothing. A key
// whose face byte falls outside 1..255 has none (DIV-4088).
func FallbackSpeakerFace(npc int32) (NPCFace, bool) {
	face := npc - fallbackSpeakerBase
	if face < 1 || face > 0xff {
		return NPCFace{}, false
	}
	tokens := NPCTokens(NPCTokenHuman) | NPCTokens(NPCTokenFace) | NPCTokens(NPCTokenMage) | NPCTokens(NPCTokenFemale)
	return NPCFace{Kind: NPCFigure, Dir: FigureDirWomanMage, Face: int(face), Tokens: tokens}, true
}

const (
	fallbackSpeakerBase = 0x50
	// FallbackSpeakerFirst and FallbackSpeakerLast are the first and last
	// keys FallbackSpeakerFace answers.
	FallbackSpeakerFirst int32 = fallbackSpeakerBase + 1
	FallbackSpeakerLast  int32 = fallbackSpeakerBase + 0xff
)
