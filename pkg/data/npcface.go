package data

import (
	"strings"

	"againrom/pkg/formats/reg"
)

// What a dialogue speaker looks like: which of the two kinds of picture an NPC
// record calls for, and the columns that select it.
//
// THE KEYS WERE DECODED AFTER THIS FILE WAS WRITTEN, AND THE READING SURVIVED.
// 0141 wrote what is below against `REG-NPC-058`, which measures `Flags`
// (105 of 105 sections, a comma list over eleven tokens with `!` negation),
// `Face` (86 sections, domain 1..30) and `Picture` (33 sections, domain 64..80)
// exactly and then grades what those keys INDEX as Unknown. `REG-NPC-088` now
// names it from the image: the record's `Picture` is stored into the synthesised
// dialogue actor's typeID and its `Face` into that actor's face byte, and the
// portrait that follows is `UNIT-PICT-036`'s existing formatter over the class's
// own `InfoPicture`. So both arms below are the engine's, not this project's —
// and one detail was wrong and is corrected here.
//
// THE READING, NOW WITH ITS CLAIM. A record whose flags carry `Human` uses
// `Face` and composes a figure — `Mage` and `Female` being the two axes a figure
// directory is chosen by (`HERO-DOLL-078`, and `REG-NPC-088` reads those same
// two tokens into bits 1 and 2 of the actor's `+0x18c`) and `Face` the sheet
// inside it. A record whose flags carry `!Human` uses `Picture`, whose value is
// a unit class id whose class has a flat portrait (`UNIT-PICT-035`,
// `UNIT-PICT-036`). A record with neither key names the player's own character,
// `Me` being one of the eleven tokens.
//
// WHAT WAS WRONG: `Face` IS READ ON BOTH ARMS. It is the sheet number on the
// figure arm and the TIER DIGIT on the portrait arm — the formatter is
// `graphics\infowindow\%s%d` with the digit dropped at 1, and the digit is this
// key (`REG-NPC-088`). Every one of the 33 shipped `Picture` records carries a
// `Face` of 1..4 and the 33 fall in (class, tier) order, so a reader taking the
// portrait arm without it showed the FIRST tier of the right creature for all
// four — a green goblin where the file names the red one.
//
// THE TAG THAT REACHES THIS TABLE IS DECODED TOO. `DLG-NPCTAG-018` follows the
// `<npc=N>` number from the `sscanf` to the array subscript with no arithmetic
// in any frame, which is what 0141 could only measure: 337 of 337 campaign tags
// naming a section of this registry, against 53 for the reading that the number
// names a placement's own npc subscript. The measurement eliminated the
// alternative and the claim establishes this one.

// The three flag tokens the PICTURE ARM is chosen by, of the eleven `Flags`
// carries. Presence of the KEY is what decides which picture, and a token
// agreeing with it adds nothing that choice could act on.
//
// The whole token list is read too, into Tokens below, and it is read for a
// different question: WHICH LIVE ACTOR this record is about
// (`DLG-SPEAKER-023`). These three names are shared by both readings.
const (
	npcFlagHuman  = "human"
	npcFlagMage   = "mage"
	npcFlagFemale = "female"
)

// NPCToken is one token of a speaker record's `Flags` list, as the original's
// own speaker search reads it (`DLG-SPEAKER-023`).
//
// THE SEVENTEEN ARE THE ROUTINE'S OWN, in the routine's own order: `Me`,
// `Mage`, `Female`, `Hero`, `Human`, `MySex`, `MyClass`, then the seven
// negations, then `Platoon`, `Face` and `Picture`. Each contributes ONE TERM
// when it is present and nothing at all when it is absent, so a record's
// tokens are a set and not a state per axis — which is why a negation is its
// own token here rather than a third value of the token it negates. That
// differs from npcFlags below, which reads a negation as absence because the
// picture arm has no use for one; both readings are of the same list and
// neither can serve the other's question.
//
// `Start` is not among them. It appears in the shipped lists (`REG-NPC-058`)
// and the speaker search reads no term from it.
type NPCToken uint32

const (
	NPCTokenMe NPCToken = 1 << iota
	NPCTokenMage
	NPCTokenFemale
	NPCTokenHero
	NPCTokenHuman
	NPCTokenMySex
	NPCTokenMyClass
	NPCTokenNotMe
	NPCTokenNotMage
	NPCTokenNotFemale
	NPCTokenNotHero
	NPCTokenNotHuman
	NPCTokenNotMySex
	NPCTokenNotMyClass
	NPCTokenPlatoon
	NPCTokenFace
	NPCTokenPicture
)

// NPCTokens is the set of tokens one record states.
//
// It is a bitset and not a map, so an NPCFace stays comparable and copyable —
// the same property figureID relies on one tier up — and so a record stating
// nothing is the zero value rather than an allocation.
type NPCTokens uint32

// Has reports whether this record states tok.
func (t NPCTokens) Has(tok NPCToken) bool { return t&NPCTokens(tok) != 0 }

// npcTokenNames maps the seventeen token spellings, lowercased, to their bits.
// The negations are spelled with the `!` the registry itself writes.
var npcTokenNames = map[string]NPCToken{
	"me": NPCTokenMe, "mage": NPCTokenMage, "female": NPCTokenFemale,
	"hero": NPCTokenHero, "human": NPCTokenHuman,
	"mysex": NPCTokenMySex, "myclass": NPCTokenMyClass,
	"!me": NPCTokenNotMe, "!mage": NPCTokenNotMage, "!female": NPCTokenNotFemale,
	"!hero": NPCTokenNotHero, "!human": NPCTokenNotHuman,
	"!mysex": NPCTokenNotMySex, "!myclass": NPCTokenNotMyClass,
	"platoon": NPCTokenPlatoon, "face": NPCTokenFace, "picture": NPCTokenPicture,
}

// npcTokens is one record's token set: every spelling npcTokenNames knows,
// with everything else — `Start` among them — silently ignored, exactly as a
// term the search reads nothing from is.
func npcTokens(list string) NPCTokens {
	var out NPCTokens
	for _, tok := range strings.Split(list, ",") {
		if bit, ok := npcTokenNames[strings.ToLower(strings.TrimSpace(tok))]; ok {
			out |= NPCTokens(bit)
		}
	}
	return out
}

// The keys, beside npcDefinitionKey in npc.go.
//
// `PortraitX1` and `PortraitY1` are the top-left corner of the window the
// dialogue pane cuts out of this speaker's picture, in the picture's own
// top-down pixels (`REG-NPC-089`, `REG-NPC-091`). `PortraitX2` and `PortraitY2`
// are NOT read and are not named here: the record constructor copies all four as
// a Win32 `RECT` and the single reader uses `left` and `top` four and five times
// while `right` and `bottom` are never read again — an exhaustive read of four
// adjacent locals in one function body. Ten shipped sections carry the second
// pair and editing it changes nothing the original draws.
const (
	npcFlagsKey   = "Flags"
	npcFaceKey    = "Face"
	npcPictureKey = "Picture"
	npcWindowXKey = "PortraitX1"
	npcWindowYKey = "PortraitY1"

	// npcStartKey gates the four speaker-conditional markup arms.
	npcStartKey = "Start"
)

// hasKey reports whether a record states a key of this name, whatever its type
// and whether or not it is itself a directory.
//
// IT IS A PRESENCE TEST AND NOT AN ACCESSOR. reg's typed getters answer false
// for a key of the wrong type, which would read here as a record that does not
// state the key at all, and the gate this feeds asks only whether it is stated.
func hasKey(sec *reg.Node, key string) bool {
	for _, k := range sec.Children {
		if nameEqualFold(k.Name, key) {
			return true
		}
	}
	return false
}

// NPCKind is which of the two pictures a speaker record calls for.
type NPCKind int

const (
	// NPCNoPicture names neither picture key. Its Start token selects an
	// archetype only after the live speaker search finds nobody (DLG-SYNTH-042).
	NPCNoPicture NPCKind = iota
	// NPCFigure is a `Human` record: compose a figure from Dir and Face.
	NPCFigure
	// NPCPortrait is a `!Human` record: load the flat portrait of the unit
	// class Class names.
	NPCPortrait
)

// NPCFace is one speaker record resolved to a picture request.
//
// It carries the ANSWER and not the columns, so a consumer neither re-reads a
// flag list nor decides which key wins. Dir is meaningful for NPCFigure, Class
// for NPCPortrait, and FACE FOR BOTH — the sheet number on one arm and the tier
// digit on the other (`REG-NPC-088`).
//
// WindowX, WindowY and HasWindow are the pane's own crop, and they belong to
// the RECORD rather than to either arm: a section may state one whichever kind
// of picture it names, and 48 of the 105 do — 33 portraits, 13 figures and two
// carrying neither picture key. HasWindow is a third field rather than a
// sentinel coordinate because 0 is a legal origin and the shipped domain starts
// there.
// Class IS THE RECORD'S `Picture` VALUE ON BOTH ARMS, and HasClass says
// whether the key was present. It selects the flat portrait on the NPCPortrait
// arm; on every arm it is also the right-hand side of `DLG-SPEAKER-023`'s
// `Picture` comparison, which reads the same key against a live actor's type
// id. Storing it only where it selects a picture would leave the predicate
// re-reading the registry for a key this table already parsed.
//
// Tokens is the whole `Flags` list, negations included, for the speaker
// predicate. It is carried and not interpreted here: which of the seventeen
// terms narrows a candidate is a fact about an ACTOR, and this package holds
// none.
type NPCFace struct {
	Kind           NPCKind
	Dir            FigureDir
	Face           int
	Class          int32
	HasClass       bool
	Tokens         NPCTokens
	WindowX        int
	WindowY        int
	HasWindow      bool
	Archetype      bool
	ArchetypeFaces [4]int

	// Start is whether this record carries a KEY named `Start`.
	//
	// It is what gates the four speaker-conditional arms of a dialogue's markup:
	// a record without it skips all four, so a part tagged for a female speaker
	// is accepted whoever the speaker is.
	//
	// IT IS THE KEY AND NOT THE `Flags` TOKEN. The two are different lookups and
	// they disagree on shipped data: no shipped section carries a key of this
	// name, and four carry `Start` inside their `Flags` list. Reading the token
	// here would make the four speaker arms live for exactly those four records.
	// Which of the two the original does is not established, so this takes the
	// narrower reading, and lifting it is this one assignment.
	Start bool
}

// SynthesisedFigure applies the Start token's archetype selection (DLG-SYNTH-042).
func (v NPCFace) SynthesisedFigure(primary FigureDir, hasPrimary bool) (FigureDir, int, bool, bool) {
	if !v.Archetype {
		return "", 0, false, false
	}
	t := v.Tokens
	if !hasPrimary && (t.Has(NPCTokenMySex) || t.Has(NPCTokenNotMySex) || t.Has(NPCTokenMyClass) || t.Has(NPCTokenNotMyClass)) {
		return "", 0, false, false
	}
	mage := t.Has(NPCTokenMage) || t.Has(NPCTokenMyClass) && primary.Mage() || t.Has(NPCTokenNotMyClass) && !primary.Mage()
	female := t.Has(NPCTokenFemale) || t.Has(NPCTokenMySex) && primary.Female() || t.Has(NPCTokenNotMySex) && !primary.Female()
	index := 0
	if mage {
		index |= 1
	}
	if female {
		index |= 2
	}
	return FigureDirFor(mage, female), v.ArchetypeFaces[index], t.Has(NPCTokenHero), true
}

// npcFlags is one record's flag set, lowercased, with a negated token counted
// as ABSENT.
//
// `!X` is how the registry writes "not X" and 15 distinct combinations ship
// (`REG-NPC-058`). Reading a negation as absence rather than as a third state is
// what lets the three tests below be plain membership: no shipped record carries
// both `X` and `!X`, and one that did would be a registry saying two things.
func npcFlags(list string) map[string]bool {
	out := make(map[string]bool)
	for _, tok := range strings.Split(list, ",") {
		tok = strings.ToLower(strings.TrimSpace(tok))
		if tok == "" || strings.HasPrefix(tok, "!") {
			continue
		}
		out[tok] = true
	}
	return out
}

// LoadNPCFaces reads every `npc<n>` section of r and resolves it to a picture
// request — the same walk LoadNPCDefs makes over the same file, for three
// different keys.
//
// IT IS A SECOND WALK AND NOT A WIDER LOOKUP, deliberately: npc.go's own doc
// refuses to carry keys that "belong to screens", on the ground that one table
// serving three consumers is a table three consumers keep in agreement. This is
// the screen's table, and the placement's stays what it was.
//
// A section carrying NEITHER key still gets an entry, because "this speaker has
// no picture in the registry" is an answer a window acts on — it is the player's
// own character — and it is a different answer from "this number names no
// section at all".
func LoadNPCFaces(r *reg.Reg) map[int32]NPCFace {
	out := make(map[int32]NPCFace)
	if r == nil || r.Root == nil {
		return out
	}
	for _, sec := range r.Root.Children {
		id, ok := npcSubscript(sec.Name)
		if !ok {
			continue
		}
		flags := map[string]bool{}
		var tokens NPCTokens
		if s, ok := r.GetString(sec.Name, npcFlagsKey); ok {
			flags, tokens = npcFlags(s), npcTokens(s)
		}
		face, hasFace := r.GetInt(sec.Name, npcFaceKey)
		pic, hasPic := r.GetInt(sec.Name, npcPictureKey)

		v := NPCFace{Tokens: tokens, Start: hasKey(sec, npcStartKey)}
		if flags["start"] {
			v.Archetype = true
			for i, section := range []string{"MaleFighter", "MaleMage", "FemaleFighter", "FemaleMage"} {
				v.ArchetypeFaces[i] = 1
				if face, ok := r.GetInt(section, npcFaceKey); ok {
					v.ArchetypeFaces[i] = int(face)
				}
			}
		}
		if hasPic {
			v.Class, v.HasClass = pic, true
		}
		if x, okX := r.GetInt(sec.Name, npcWindowXKey); okX {
			if y, okY := r.GetInt(sec.Name, npcWindowYKey); okY {
				// BOTH OR NEITHER, because the pane's window needs a corner and
				// not a coordinate: `REG-NPC-089`'s branch tests `top` alone
				// against the loader's key-absent default, but every one of the
				// 48 shipped sections that states either states both, so a
				// record with one of them is outside the corpus and takes the
				// default rather than half of a window.
				v.WindowX, v.WindowY, v.HasWindow = int(x), int(y), true
			}
		}
		if hasFace {
			// THE FACE IS READ BEFORE THE ARM IS CHOSEN, because both arms use
			// it (`REG-NPC-088`): the sheet number inside a figure directory,
			// and the tier digit appended to a class's `InfoPicture`.
			v.Face = int(face)
		}
		switch {
		// THE FLAG DECIDES BEFORE THE KEY, and it has to: 33 shipped records
		// carry BOTH keys, and all 33 are `!Human` creatures whose `Face` value
		// would otherwise compose a person out of a tier digit.
		case !flags[npcFlagHuman] && hasPic:
			v.Kind = NPCPortrait
		case hasFace:
			v.Kind = NPCFigure
			v.Dir = FigureDirFor(flags[npcFlagMage], flags[npcFlagFemale])
		}
		out[id] = v
	}
	return out
}
