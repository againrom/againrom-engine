package data

import (
	"testing"

	"againrom/internal/synth"
)

// A speaker record resolved to a picture request, over synthetic registries —
// npc_test.go's own fixtures for the other reader of the same file.
//
// EVERY SHAPE BELOW IS ONE THE SHIPPED REGISTRY ACTUALLY CARRIES, measured over
// the 26 records the campaign's dialogue names: a male fighter, a male mage, a
// woman fighter, a monster carrying BOTH keys, and a record carrying neither.
func TestNPCFacesResolvesTheThreeKinds(t *testing.T) {
	faces := LoadNPCFaces(parseReg(t, []synth.RegNode{
		// `npc25`: Hero,Face,!Female,!Mage with Face 1.
		regDir("npc25", regStr("Flags", "Hero,Face,!Female,!Mage"), regInt("Face", 1)),
		// `npc26`: a male mage.
		regDir("npc26", regStr("Flags", "Hero,Face,Mage,!Female"), regInt("Face", 4)),
		// `npc51`: a woman fighter — the shape a reader that ignored `Female`
		// would draw as a man.
		regDir("npc51", regStr("Flags", "Human,Female,!Mage,Face"), regInt("Face", 8)),
		// `npc103`: a monster. It carries BOTH keys and the flag decides.
		regDir("npc103", regStr("Flags", "!Human,Picture,Face"),
			regInt("Picture", 64), regInt("Face", 9)),
		// `npc21`: the player's own character — neither key.
		regDir("npc21", regStr("Flags", "Hero,Me,Start")),
	}))

	for _, tc := range []struct {
		id   int32
		want NPCFace
	}{
		{25, NPCFace{Kind: NPCFigure, Dir: FigureDirManFighter, Face: 1,
			Tokens: tokens(NPCTokenHero, NPCTokenFace, NPCTokenNotFemale, NPCTokenNotMage)}},
		{26, NPCFace{Kind: NPCFigure, Dir: FigureDirManMage, Face: 4,
			Tokens: tokens(NPCTokenHero, NPCTokenFace, NPCTokenMage, NPCTokenNotFemale)}},
		{51, NPCFace{Kind: NPCFigure, Dir: FigureDirWomanFighter, Face: 8,
			Tokens: tokens(NPCTokenHuman, NPCTokenFemale, NPCTokenNotMage, NPCTokenFace)}},
		// THE FACE IS THE TIER DIGIT ON THIS ARM (`REG-NPC-088`), so it is
		// carried and not dropped: a reader that dropped it showed every
		// creature in every dialogue at its first tier.
		{103, NPCFace{Kind: NPCPortrait, Class: 64, HasClass: true, Face: 9,
			Tokens: tokens(NPCTokenNotHuman, NPCTokenPicture, NPCTokenFace)}},
		// `Start` is not one of the seventeen terms the speaker search reads,
		// so it contributes no token.
		{21, NPCFace{Kind: NPCNoPicture, Tokens: tokens(NPCTokenHero, NPCTokenMe), Archetype: true, ArchetypeFaces: [4]int{1, 1, 1, 1}}},
	} {
		got, ok := faces[tc.id]
		if !ok {
			t.Errorf("npc%d resolved to no record at all", tc.id)
			continue
		}
		if got != tc.want {
			t.Errorf("npc%d = %+v, want %+v", tc.id, got, tc.want)
		}
	}

	if _, ok := faces[99]; ok {
		t.Error("a subscript with no section resolved to a record")
	}
}

// tokens is the set of the named tokens, so a want value states the tokens it
// means rather than a bitmask literal that has to be recomputed whenever a
// constant is inserted.
func tokens(list ...NPCToken) NPCTokens {
	var out NPCTokens
	for _, tok := range list {
		out |= NPCTokens(tok)
	}
	return out
}

// EVERY ONE OF THE SEVENTEEN TOKENS `DLG-SPEAKER-023` NAMES IS READ, and
// nothing else is. The routine combines one term per token present, in the
// order below, and `Start` — which the shipped lists do carry (`REG-NPC-058`) —
// is not among them.
func TestNPCFacesReadsEverySpeakerToken(t *testing.T) {
	const all = "Me,Mage,Female,Hero,Human,MySex,MyClass," +
		"!Me,!Mage,!Female,!Hero,!Human,!MySex,!MyClass,Platoon,Face,Picture,Start"
	faces := LoadNPCFaces(parseReg(t, []synth.RegNode{
		regDir("npc1", regStr("Flags", all), regInt("Face", 1)),
		regDir("npc2", regStr("Flags", "Start"), regInt("Face", 1)),
	}))

	want := tokens(NPCTokenMe, NPCTokenMage, NPCTokenFemale, NPCTokenHero, NPCTokenHuman,
		NPCTokenMySex, NPCTokenMyClass, NPCTokenNotMe, NPCTokenNotMage, NPCTokenNotFemale,
		NPCTokenNotHero, NPCTokenNotHuman, NPCTokenNotMySex, NPCTokenNotMyClass,
		NPCTokenPlatoon, NPCTokenFace, NPCTokenPicture)
	if got := faces[1].Tokens; got != want {
		t.Errorf("all seventeen tokens read as %#x, want %#x", got, want)
	}
	for _, tok := range []NPCToken{NPCTokenMe, NPCTokenPicture, NPCTokenPlatoon} {
		if !want.Has(tok) {
			t.Errorf("Has(%#x) is false over the full set", tok)
		}
	}
	if got := faces[2].Tokens; got != 0 {
		t.Errorf("`Start` alone read as %#x, want no token at all", got)
	}
}

// THE RECORD'S `Picture` IS CARRIED ON EVERY ARM (0160 plan Step 1). It selects
// the flat portrait on the NPCPortrait arm and is the right-hand side of the
// `Picture` comparison on any arm; a record stating the key on the figure arm
// therefore keeps its value, and one stating no key answers HasClass false.
func TestNPCFacesCarriesPictureOffThePortraitArm(t *testing.T) {
	faces := LoadNPCFaces(parseReg(t, []synth.RegNode{
		regDir("npc7", regStr("Flags", "Human,Picture,Face"),
			regInt("Picture", 70), regInt("Face", 2)),
		regDir("npc8", regStr("Flags", "Human,Face"), regInt("Face", 2)),
	}))
	if got := faces[7]; got.Kind != NPCFigure || !got.HasClass || got.Class != 70 {
		t.Errorf("npc7 = %+v, want the figure arm carrying Picture 70", got)
	}
	if got := faces[8]; got.HasClass || got.Class != 0 {
		t.Errorf("npc8 = %+v, want no Picture at all", got)
	}
}

// THE FLAG DECIDES BEFORE THE KEY, and this is the case that makes it matter:
// all 33 shipped records carrying `Picture` carry `Face` too, and all 33 are
// `!Human`. A reader that took whichever key it saw first would compose a
// person out of a dragon's tier digit.
func TestNPCFacesLetsTheFlagBeatTheKey(t *testing.T) {
	faces := LoadNPCFaces(parseReg(t, []synth.RegNode{
		regDir("npc1", regStr("Flags", "!Human,Picture,Face"), regInt("Picture", 71), regInt("Face", 3)),
		// The mirror: a Human record carrying a Picture is a person, and the
		// picture is ignored.
		regDir("npc2", regStr("Flags", "Human,Face"), regInt("Picture", 71), regInt("Face", 3)),
	}))
	if got := faces[1]; got.Kind != NPCPortrait || got.Class != 71 {
		t.Errorf("npc1 = %+v, want a portrait of class 71", got)
	}
	if got := faces[2]; got.Kind != NPCFigure || got.Face != 3 {
		t.Errorf("npc2 = %+v, want a figure at face 3", got)
	}
}

// A NEGATED TOKEN IS AN ABSENT ONE. `!Mage` must not select the mage directory,
// which is what a reader doing a plain substring search over the flag list would
// do — and 15 distinct combinations ship, most of them carrying negations.
func TestNPCFacesReadsNegationAsAbsence(t *testing.T) {
	faces := LoadNPCFaces(parseReg(t, []synth.RegNode{
		regDir("npc1", regStr("Flags", "Human,!Mage,!Female,Face"), regInt("Face", 2)),
		regDir("npc2", regStr("Flags", "Human,Mage,Female,Face"), regInt("Face", 2)),
		// Spacing and case are the registry's business, not this reader's.
		regDir("npc3", regStr("Flags", " HUMAN , mage , Female "), regInt("Face", 2)),
	}))
	if got := faces[1].Dir; got != FigureDirManFighter {
		t.Errorf("a doubly negated record chose %q", got)
	}
	if got := faces[2].Dir; got != FigureDirWomanMage {
		t.Errorf("a mage woman chose %q", got)
	}
	if got := faces[3].Dir; got != FigureDirWomanMage {
		t.Errorf("a spaced, upper-cased flag list chose %q", got)
	}
}

// THE PANE'S CROP IS THE RECORD'S OWN, and it belongs to the record and not to
// either arm: `REG-NPC-089` reads `PortraitX1`/`PortraitY1` as the top-left
// corner of the window the dialogue pane cuts out of whatever picture the
// speaker resolves to, and 48 of the 105 shipped sections state one — 33 of them
// creatures, 13 people and two carrying neither picture key.
func TestNPCFacesCarriesThePaneWindow(t *testing.T) {
	faces := LoadNPCFaces(parseReg(t, []synth.RegNode{
		// A person who states one: `npc51`'s own shipped values.
		regDir("npc51", regStr("Flags", "Human,Female,!Mage,Face"), regInt("Face", 8),
			regInt("PortraitX1", 40), regInt("PortraitY1", 10)),
		// A creature who states one, with the dead second pair beside it. Those
		// two are never read: the record constructor copies all four as a RECT
		// and the single reader touches only `left` and `top`.
		regDir("npc104", regStr("Flags", "!Human,Picture,Face"),
			regInt("Picture", 65), regInt("Face", 1),
			regInt("PortraitX1", 26), regInt("PortraitY1", 23),
			regInt("PortraitX2", 52), regInt("PortraitY2", 19)),
		// A record stating NEITHER — the ordinary case, 57 of the 105.
		regDir("npc25", regStr("Flags", "Hero,Face,!Female,!Mage"), regInt("Face", 1)),
		// HALF A CORNER IS NOT A CORNER. No shipped section states one without
		// the other; a registry that did would otherwise cut a window at an
		// origin it never named.
		regDir("npc60", regStr("Flags", "Human,Face"), regInt("Face", 2),
			regInt("PortraitX1", 40)),
		// ZERO IS A LEGAL ORIGIN, which is why the presence flag exists rather
		// than a sentinel coordinate: the shipped domain starts at 0.
		regDir("npc61", regStr("Flags", "Human,Face"), regInt("Face", 2),
			regInt("PortraitX1", 0), regInt("PortraitY1", 0)),
	}))
	for _, tc := range []struct {
		id      int32
		wantX   int
		wantY   int
		wantHas bool
	}{
		{51, 40, 10, true},
		{104, 26, 23, true},
		{25, 0, 0, false},
		{60, 0, 0, false},
		{61, 0, 0, true},
	} {
		got := faces[tc.id]
		if got.WindowX != tc.wantX || got.WindowY != tc.wantY || got.HasWindow != tc.wantHas {
			t.Errorf("npc%d window = (%d,%d) has=%v, want (%d,%d) has=%v",
				tc.id, got.WindowX, got.WindowY, got.HasWindow, tc.wantX, tc.wantY, tc.wantHas)
		}
	}
}

// A registry that will not open is no faces and not a panic — the answer the
// window above needs, because the pane it feeds has always been allowed to be
// empty.
func TestNPCFacesIsTotalOverAnAbsentRegistry(t *testing.T) {
	if got := LoadNPCFaces(nil); len(got) != 0 {
		t.Errorf("a nil registry answered %d records", len(got))
	}
	if got := LoadNPCFaces(parseReg(t, nil)); len(got) != 0 {
		t.Errorf("an empty registry answered %d records", len(got))
	}
	// A section that is not a per-npc one contributes nothing, which is how the
	// four archetype blocks and the multiplayer face lists stay out.
	if got := LoadNPCFaces(parseReg(t, []synth.RegNode{
		regDir("MaleFighter", regInt("Face", 5)),
		regDir("Multiplayer", regInt("Face", 5)),
	})); len(got) != 0 {
		t.Errorf("the archetype blocks contributed %d records", len(got))
	}
}
