package game

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// partySkillSlot is the skill this front-end's hero trains, and — since this
// story — it is a SETTING rather than a fixed fact about the game.
//
// Character generation is a screen this tree does not have, so until it
// exists something still has to answer "which skill". The default below is
// the BLADE slot this tree has always used: it is the slot the owner's own
// measured damage bands were taken on — which makes those four numbers a
// live check on this tree rather than a coincidence — and its literal is the
// one weapon of the five whose resolved figures research states outright. An
// unset program is therefore unchanged by this story.
//
// IT IS PACKAGE STATE, NARROWLY, and not a parameter threaded through
// LoadDefinitions, the Definitions struct, PartyHero and MissionParty. Those
// signatures belong to character generation, and another lane is inside
// them; this is one flag, read once at start-up, for a screen that
// front-ends this build without one still have to answer somehow. It is a
// FRONT-END setting and not simulation state: resolveStartingWeapon reads it
// during LoadDefinitions, before a mission — let alone pkg/sim — exists,
// and the hero it produces is handed onward as ordinary Go values. Nothing
// in pkg/sim reads this variable or anything derived from it at simulation
// time, so no determinism rule is in play, and the contract is that a caller
// sets it, if at all, before the front end loads and never again — nothing
// downstream re-reads it mid-mission.
var partySkillSlot int32 = data.SkillBlade

// PartySkillSlot is the skill the party's hero is currently set to train.
func PartySkillSlot() int32 { return partySkillSlot }

// SetPartySkill sets the skill a generated party hero trains in. It refuses
// a slot StartingWeaponName itself would train no weapon for — out of range,
// or slot 0, the General slot the character sheet does not show — rather
// than writing a second predicate for the same question, and it leaves the
// stored value untouched when it refuses, so a rejected call cannot leave
// the setting half-changed.
//
// IT VALIDATES AGAINST THE FIGHTER ARM (StartingWeaponName's mage flag
// false), never the mage's: this front end's own authored default is a
// fighter (defaultChargenAxes), and the mage arm answers one literal for
// EVERY slot, which would make this refusal accept a slot the fighter arm
// — the one this setting actually trains — still refuses.
func SetPartySkill(slot int32) error {
	if _, ok := StartingWeaponName(false, slot); !ok {
		return fmt.Errorf("skill slot %d trains no weapon", slot)
	}
	partySkillSlot = slot
	return nil
}

// PartySpread is the point spread this front-end's hero was generated with, and
// it is OURS.
//
// THE LEGAL SPACE IS DECODED AND THE POINT INSIDE IT IS NOT. Character
// generation admits any four statistics in [15, 45] whose costs sum to at most
// 140; nothing in the corpus says which of those a player's character should be,
// and no amount of further decoding will, because it is a question about a
// player and not about the game. So this is an AUTHORED verdict — absence
// established positively, disclosed, and named — rather than a hold.
//
// THE RULE THAT PRODUCED IT, so that a reader can disagree with the rule rather
// than with four magic numbers: only Body reaches damage and only Body and
// Reaction reach to-hit, so a fighter floors Mind and Spirit, maximises Body,
// then maximises Reaction with what is left. That is 43 / 26 / 15 / 15, costing
// 139 of 140.
//
// THE LAST POINT IS UNSPENDABLE AND NOT FORGOTTEN. No legal spread raises Body
// past 43, and with Body at 43 the next Reaction step costs two where one
// remains. Spending it on Mind or Spirit would buy sight, mana or a protection —
// none of which this tree derives, and none of which a blow reads. The test
// beside this file searches the legal space and asserts both maximalities, so
// the claim is checked rather than asserted.
//
// IT WAS WRITTEN AS THE SUBSTITUTION POINT, and 0119-chargen is the story
// that substitutes it — so this paragraph now says what happened rather than
// what was promised. ChargenParty (chargen.go) calls the SAME assembleParty
// this function's own caller (MissionParty, below) calls, over the player's
// own spread instead of this one's four numbers: nothing about HOW a party
// is built differs between the two, only which spread reaches the build.
// PartySpread itself does not move and did not need to — it is read in
// exactly one place now, MissionParty, as the default a mission opens with
// when no flag ever ran the screen. It stays a FUNCTION and not a package
// variable for MissionParty's own reason — a variable is writable from
// anywhere, and "every mission starts with the same hero" would then be true
// by mutation rather than by design.
func PartySpread() data.Spread {
	return data.Spread{Body: 43, Reaction: 26, Mind: 15, Spirit: 15}
}

// PartyHero is the character every mission this front end opens starts with: the
// authored spread above, trained in the authored slot above.
//
// The two are composed HERE and in one place, so the check line, the party and
// any later reader are looking at one hero rather than at three constructions of
// one that agree today.
func PartyHero() data.Hero { return data.NewHero(PartySpread(), PartySkillSlot()) }

// startingWeapons is the ORDINARY arm of the table character generation
// dispatches through: one weapon literal per trained skill slot, indexed by that
// slot, with slot 0 — the one the sheet does not show — naming none.
//
// It is a jump table on `slot - 1` in the image and an array here, and the
// strings are the image's own. The dispatch has TWO arms and this is the lower:
// a trained level above ten takes the other, whose five literals are beside
// this one. Which arm a shipped single-player campaign takes is research's
// **Medium** — it rests on no writer of the mode value being found rather than
// on the menu chain being read — so the ordinary arm is taken here and the
// grade is carried rather than resolved. The owner's own character sheet lists
// his weapon at 5-8, which is this arm's blade weapon, and that is corroboration
// arriving from the game rather than the reason.
//
// THE MAGE ARM IS NOT IN THIS TABLE, and that is no longer for want of a
// class axis: the axis exists now, StartingWeaponName's own leading flag,
// and it is the axis that decides which of this file's TWO weapon sources a
// trained slot reaches at all. What is still missing is the spell — see
// mageWeaponName's own doc, beside StartingWeaponName below. HERO-START-039.
var startingWeapons = [data.SkillSlots]string{
	data.SkillBlade:   "Iron Short Sword",
	data.SkillAxe:     "Uncommon Bronze Axe",
	data.SkillBludgen: "Uncommon Bronze Mace",
	data.SkillPike:    "Bronze Pike",
	data.SkillShoot:   "Uncommon Wood Short Bow",
}

// highStartingWeapons is the other arm, written down so the Medium above is one
// constant away from being tested rather than a re-decode. Nothing reads it yet.
var highStartingWeapons = [data.SkillSlots]string{
	data.SkillBlade:   "Uncommon Steel Two Handed Sword",
	data.SkillAxe:     "Uncommon Steel Axe",
	data.SkillBludgen: "Uncommon Steel Mace",
	data.SkillPike:    "Uncommon Steel Pike",
	data.SkillShoot:   "Uncommon Magic Wood Short Bow",
}

// mageWeaponName is the shipped mage arm's own literal, WHOLE (D-11, 0139
// D-13, cite `HERO-START-039`): ONE weapon, not a table indexed by slot —
// the corpus names exactly one mage literal, and five copies of it would
// assert a variation nothing states.
//
// THE ATTACHMENT IS CARRIED, AS OF 0139 — until this story it was dropped
// and disclosed here (spec 0134 R-4; plan 0134 D-11's own rejection),
// because no weapon in this tree carried a spell and there was nowhere for
// `{castSpell=Fire_Arrow:10}` to land. Something can now (data.Weapon's own
// SpellName and SpellPower, 0139 T1), so the constant becomes the shipped
// literal whole: data.ResolveWeapon already reads that suffix rather than
// only stripping it, and MissionPartyAs's own mage arm resolves this string
// through the SAME resolveWeaponForSlot every other slot's literal resolves
// through, so a mage's staff carries its spell by the one weapon resolution
// this file has, not by a second reading of the literal.
const mageWeaponName = "Wood Staff {castSpell=Fire_Arrow:10}"

// StartingWeaponName is the literal character generation hands a trained
// slot: mage's own ONE literal when mage is true, WHATEVER slot names —
// the class axis decides FIRST and the trained slot SECOND, so a mage reads
// no further than the flag — and the fighter arm's per-slot literal
// otherwise, false for a slot that arm trains no weapon for.
func StartingWeaponName(mage bool, slot int32) (string, bool) {
	if mage {
		return mageWeaponName, true
	}
	if slot <= data.SkillGeneral || slot >= data.SkillSlots {
		return "", false
	}
	n := startingWeapons[slot]
	return n, n != ""
}

// resolveWeaponForSlot is the ONE weapon resolution a trained slot goes
// through, whichever of this story's two callers reaches it: the front end's
// own authored slot at construction (resolveStartingWeapon, below), always
// the fighter arm, and a generated character's CHOSEN class and slot,
// resolved fresh at confirm time by ChargenParty (chargen.go) because there
// is no front-end field for "every slot's weapon" the way StartWeapon is one
// for the single slot this build authors.
//
// mage IS THE SAME LEADING FLAG StartingWeaponName TAKES (0134 D-11): it
// grows on THIS function rather than on a sibling beside it, so there stays
// exactly one resolution of a generated character's weapon, mage or
// fighter, rather than two that could disagree over how the fighter arm's
// own nine slots resolve.
//
// It reads StartingWeaponName's own literal and resolves it against the
// three item collections handed in — never a *databin.File, so a caller
// already holding the definition table's own parsed collections (a
// FrontEnd's Table, after construction) does not have to re-parse the file
// to reach them a second time.
func resolveWeaponForSlot(mage bool, shapes, materials data.ScaleTable, weapons data.Collection, slot int32) (*data.Weapon, error) {
	name, ok := StartingWeaponName(mage, slot)
	if !ok {
		return nil, fmt.Errorf("no starting weapon for skill slot %d", slot)
	}
	w, err := data.ResolveWeapon(name, shapes, materials, weapons)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// resolveStartingWeapon is resolveWeaponForSlot at FRONT-END CONSTRUCTION,
// off the *databin.File LoadDefinitions parsed the table from — the one
// caller that still holds one, because table.go keeps only the three
// resolved collections on Definitions.Table and lets the parsed file itself
// go once this call and the table's own collections are both built off it.
//
// IT IS ALWAYS THE FIGHTER ARM (mage false): this front end's own authored
// default, before any generation screen ever ran, is a fighter
// (defaultChargenAxes, below) — there is no class choice yet at this call,
// only the ordinary slot -skill may have set.
func resolveStartingWeapon(f *databin.File, slot int32) (*data.Weapon, error) {
	return resolveWeaponForSlot(false, f.Collection(databin.Shapes), f.Collection(databin.Materials), f.Collection(databin.Weapons), slot)
}

func partyBody(w *data.Weapon, list data.BodyList) (data.HeroBody, bool) {
	var eq data.Equipment
	if w != nil {
		eq.SetCode(1, w.Code)
	}
	return data.HeroBodyFor(list, eq)
}

// chargenProfile is the profile, the figure face and the resolved base row's
// own ten equipment cell strings a base-row search's own answer states —
// HumanDef.Profile(), the row's own Face column, and humans.EntryStrings(i)
// when a row resolved, the zero Profile, face 1 and no cells at all when
// none did. It is the ONE place both MissionParty and ChargenParty
// (chargen.go) read those facts off a data.ChargenBase result, so a later
// story cannot make the two party builders come to disagree about what "no
// base row" means.
//
// humans AND i ARE THE SEARCH'S OWN COLLECTION AND INDEX (0134 D-13): the
// equipment cells are not a HumanDef field at all, they are the resolved
// entry's own TRAILING STRINGS (data.Collection.EntryStrings), which only
// the collection and the index the search actually landed on can still
// answer — a second search by the archetype's own name could, after
// ChargenBase's own fallback, name a different row from the one base and ok
// came off (plan D-13's own rejection). i is meaningless when ok is false
// and is never read in that arm.
// class is the archetype axis the caller already asked ChargenBase for
// (true mage, false fighter, the same bool archetypeSlot indexes with): when
// no row resolves, DIV-1389 has this arm fall back to the requested
// archetype's own Fighter bit rather than Go's silent zero Profile, whose
// Fighter field would have named every unresolvable install a mage
// regardless of which axis was requested.
//
// HealthColumn IS NOW TRUE HERE TOO, which DIV-1389 left false on the R-3
// "less accurate" reading that no row means no column to read. HERO-HP-072's
// own health derivation reads Body and the class bit alone — no shipped row
// gates it — and a member built on this arm already carries a real,
// authored Body (PartySpread) and a resolved class bit, so the graph has
// everything it needs whether or not a row resolved. Before this story the
// one caller of this arm, `MissionParty(nil, nil, nil)`, minted its hero at
// the provisional constant (mapload.SpawnHP) instead of a derived maximum —
// the owner's own witnessed 100-vs-137 mismatch. ManaColumn stays false:
// this story's scope is the health maximum alone, and DIV-1390 records the
// derived number's own remaining gap against an original resave.
func chargenProfile(base data.HumanDef, ok bool, humans data.Collection, i int, class bool) (data.Profile, int, uint32, []string) {
	if !ok {
		return data.Profile{Fighter: !class, HealthColumn: true}, 1, 0, nil
	}
	return base.Profile(), int(base.Face), base.KnownSpells, humans.EntryStrings(i)
}

// equipmentFromSlots is the ONE conversion in this package from a
// [sim.EquipSlots]uint16 slot array to a data.Equipment (spec 0134 D-8): the
// same shape mapload.GeneratedWornSet answers below and sim.World.Equipped
// answers to a running mission's own live read (world.go's own
// currentEquipment), so a generated character's starting worn set and a
// placed subject's current one cannot come to read a slot array two
// different ways. world.go is a LATER task's boundary and is not pointed at
// this function here; the loop is written once so that task only has to
// call it.
func equipmentFromSlots(slots [sim.EquipSlots]uint16) data.Equipment {
	var eq data.Equipment
	for slot := 1; slot <= len(slots); slot++ {
		eq.SetCode(slot, data.ItemCode(slots[slot-1]))
	}
	return eq
}

// partyInputs is the ONE value both party builders hand assembleParty (spec
// 0134 D-8), replacing the growing positional list — eight arguments before
// this story, of which three were already booleans or near enough, which is
// exactly the shape in which a caller silently transposes two. Every field
// below is a fact one of the two builders already resolved through its own
// route; assembleParty invents none of them.
type partyInputs struct {
	Name   string
	Spread data.Spread
	Slot   int32

	Mage bool

	Profile data.Profile
	Dir     data.FigureDir
	Face    int
	Book    uint32

	// Weapon is what he holds, or nil for BARE HANDS — see MissionParty's
	// own doc on what a nil weapon means at the generated Body this front
	// end authors.
	Weapon *data.Weapon

	// Cells is the resolved base row's own ten equipment cell strings
	// (chargenProfile's fourth return), or nil for a character with no base
	// row. GeneratedWornSet is called over it EITHER WAY: a character with no
	// base row still goes through that call with no cells at all, which is what
	// keeps his handed weapon in slot 1 rather than requiring a base row to
	// have one.
	Cells []string

	// List is the shipped body list HeroAppearance reads the worn base name
	// from.
	List data.BodyList

	// Table is the three piece collections GeneratedWornSet resolves the
	// worn set's other nine cells against, or nil — GeneratedWornSet's own
	// nil-safe rule, on wearRow's, applies here unchanged.
	Table *mapload.Table

	// Documents asks for the campaign documents access item in the assembled
	// member's PACK. It is a field rather than something assembleParty does
	// unconditionally because this function builds a COMPANION as well as a
	// player character: addChapterCompanions (frontend.go) calls it and then
	// clears StartingHero, and an NPC who joined for one chapter is not the
	// person the campaign's documents belong to.
	//
	// IT IS AN OWNER DIRECTIVE OVER A POSITIVE RESEARCH FINDING, which is why
	// the field is named and not silent. ITEM-DOC-053 searched three producer
	// families — the .alm type-8 section, script instant 12 and image
	// immediates — and found zero placements of this code, DAT-DOC-021 found
	// no Quest cell in all 909 Humans equipment cells and no MagicItems arm in
	// the positional grammar, and SHOP-DOC-029 shows shop generation cannot
	// supply it at campaign start. DIV-305.
	Documents bool
}

// assembleParty is the ONE construction of a mission's one-member party,
// shared by MissionParty and ChargenParty (chargen.go) rather than composed
// by hand in each. Both resolve a spread, a trained slot, a class flag, a
// profile, a figure directory and face, a resolved base row's own equipment
// cells and a possibly-nil weapon by their own two different routes — one
// from this front end's own authored defaults, one from a player's confirmed
// choices — and hand all of it to this, packed into one partyInputs (plan
// D-8), rather than building a mapload.PartyMember literal a second time.
//
// THE WORN SET IS D-4's OWN RESOLUTION, reached here and not repeated: the
// handed weapon's own Name — the empty string for bare hands — replaces
// in.Cells' own weapon cell, and the whole ten go through
// mapload.GeneratedWornSet exactly as a placed person's row already does
// (mapload.wearRow, one tier down). Worn and Carried land on the member
// straight off that one call.
//
// THE BODY, THE DIRECTORY AND THE DRAWN CLASS ARE data.HeroAppearance's OWN
// ANSWER (plan D-1), never partyBody's or data.HeroBodyClass's any more:
// they are called over the worn set just resolved — converted to a
// data.Equipment through equipmentFromSlots — in.List, in.Mage, and false
// for dying, because this function derives no dying character (spec Out of
// scope). The fourth value HeroAppearance answers, whether the name and the
// directory were both actually produced rather than a law's own fallback,
// is discarded here on MissionParty's own precedent for HeroBodyClass's
// matched flag: a name or a directory that fell back is not a second error
// this function has an answer for.
func assembleParty(in partyInputs) []mapload.PartyMember {
	weaponName := ""
	if in.Weapon != nil {
		weaponName = in.Weapon.Name
	}
	worn, carried := mapload.GeneratedWornSet(in.Cells, weaponName, in.Table)
	carriedItems := make([]sim.ItemInstance, len(carried))
	for i, code := range carried {
		carriedItems[i] = mapload.ItemInstanceFromCode(code, in.Table)
	}

	body, dir, class, _ := data.HeroAppearance(in.List, equipmentFromSlots(worn), in.Mage, false)

	out := []mapload.PartyMember{{
		ID:              "hero",
		Name:            in.Name,
		PlayerCharacter: true,
		StartingHero:    true,
		Class:           class,
		Body:            string(body),
		BodyDir:         dir,
		Mage:            in.Mage,
		Profile:         in.Profile,
		FigureDir:       string(in.Dir),
		FigureFace:      in.Face,
		// HIS BOOK, off the same base row the profile and the face come off
		// (0127 FR-4a). It is the one loader input on this struct that DOES
		// reach the entity, and it has to: without it the character a player
		// actually commands knows no spell, so the whole cast affordance is
		// drawn empty for the only unit he can select. A base row that did not
		// resolve carries the empty book, exactly as it carries the empty
		// profile.
		KnownSpells:  in.Book,
		Hero:         data.NewHero(in.Spread, in.Slot),
		Weapon:       in.Weapon,
		Worn:         worn,
		Carried:      carried,
		CarriedItems: carriedItems,
	}}
	if in.Documents {
		// IN THE PACK, NOT WORN. DAT-DOC-021 establishes that the Humans
		// equipment grammar has no MagicItems arm, so there is no shipped
		// slot for it and inventing one would be inventing a fact;
		// data.EquipSlotFor refuses class 14 outright. Carried is the pack,
		// and it is where the item's own use gesture — a pack double-click —
		// can reach it (world.go's enqueueEquip). DIV-305.
		out[0].Carried = append(out[0].Carried, uint16(data.QuestDocumentCode))
		out[0].CarriedItems = append(out[0].CarriedItems,
			mapload.ItemInstanceFromCode(uint16(data.QuestDocumentCode), in.Table))
	}
	return out
}

// defaultChargenAxes is the archetype pair MissionParty searches humans for
// when no generation screen ran at all: no class flag and male — the axes
// PartySkillSlot's own blade training and PartyBodyDir's own no-armour,
// no-mage arm already assume, so a base row resolved for them describes the
// same hero those two already describe. It is a function rather than two
// package constants so there is exactly one statement of them for
// MissionParty's two reads of them — data.ChargenBase and
// data.FigureDirFor — to agree on without either repeating the two bools.
func defaultChargenAxes() (class, female bool) { return false, false }

// missionHumans is t's own Humans collection, and nil for a nil table — the
// same "no table" shape (*FrontEnd).tableWeapons (chargen.go) already gives
// a nil *mapload.Table, so a caller holding no table still reaches
// data.ChargenBase's own "no base row" answer rather than a panic on a nil
// field read.
func missionHumans(t *mapload.Table) data.Collection {
	if t == nil {
		return nil
	}
	return t.Humans
}

// MissionParty is the party every mission this front end opens starts with:
// exactly one member, wearing the worn set assembleParty resolves for the
// weapon he holds, carrying the body, the directory and the class key that
// worn set DERIVES, the generated hero and the weapon he was handed.
//
// AS OF 0119-CHARGEN IT ALSO SEARCHES t's Humans FOR A BASE ROW, at
// defaultChargenAxes' own pair, so the party this front end starts WITHOUT
// the generation flag carries a real profile too — that is the whole fix
// for the owner's 100-health defect on the path that never shows a screen at
// all. NOTHING ELSE THIS FUNCTION ANSWERED BEFORE MOVES: the spread is still
// PartySpread, the slot is still PartySkillSlot, and the body, the directory
// and the class key are still the weapon's and the base row's own
// derivations through assembleParty — which is what proves it, because it
// is the exact same construction ChargenParty (chargen.go) calls for a
// player's own choices, over a different four numbers and a different
// trained slot.
//
// t REPLACES THE BARE Humans COLLECTION THIS FUNCTION USED TO TAKE (0134
// D-9): resolving a generated character's starting clothes needs the three
// piece collections beside the Humans search, and t's Table already carries
// both off the one walk LoadDefinitions performs — a second parameter for
// them would be a second answer to "what table does this install carry".
// t is passed straight to assembleParty as well, for GeneratedWornSet's own
// use; missionHumans above is the one nil-safe read of its Humans field
// this function and ChargenParty's own f.Humans read do not have to
// duplicate a second time.
//
// THE CLASS KEY, THE BODY AND THE DIRECTORY ARE PRODUCED HERE AND NOWHERE
// ELSE. They used to be a constant this project chose, and the constant it
// chose named the roster's bare-handed human body — so the one figure the
// player controls was drawn as an unarmed man, by arithmetic nobody
// performed. All three now come out of the game's own name chain, at the
// single site that assembles a party, so there is one expression in the tree
// that answers "what does a party member wear, and what class does he carry"
// and no literal beside it to disagree.
//
// list is the shipped body list, read once at front-end construction and
// carried in beside the weapon: nothing in this tree can change what a
// character holds after a mission opens, so there is nothing to re-derive
// later. A REFUSED DERIVATION CARRIES THE EMPTY BODY NAME FORWARD, and that
// needs no handling of its own: an empty Body is already what a member drawn
// through its class record like any placed unit carries, so there is no
// second answer to write here.
//
// THE WEAPON IS AN ARGUMENT rather than something read here, because the mission
// tool starts the same party without holding a whole front end. A nil one is a
// BARE hero — a real state, in which he swings for `ftol(1.1^Body/20)`. At the
// generation START that was nothing at all, because the term is zero below Body
// 32; at the generated Body 43 it is a real blow, which is the clearest single
// consequence of this hero no longer being an initialiser.
//
// It is handed out as a fresh slice rather than held in a package variable, for
// the reason it always was: a variable carrying a slice is writable from
// anywhere, and "every mission starts with the same party" would then be true by
// mutation rather than by design.
//
// IT IS MissionPartyAs AT mage false, AS OF 0139: every word above still
// describes the party this call produces, because the fighter arm below
// takes neither of MissionPartyAs's two mage-only branches and w passes
// through exactly as it always has.
func MissionParty(w *data.Weapon, list data.BodyList, t *mapload.Table) []mapload.PartyMember {
	return MissionPartyAs(false, w, list, t)
}

// missionTableWeapons is tableWeapons' own guard (chargen.go), over the t
// argument MissionPartyAs is handed rather than over a FrontEnd's own field:
// a nil *mapload.Table cannot be field-accessed at all, so this is the same
// "zero triple" nil check restated for the one caller here that holds a bare
// *mapload.Table instead of a FrontEnd.
func missionTableWeapons(t *mapload.Table) (shapes, materials data.ScaleTable, weapons data.Collection) {
	if t == nil {
		return nil, nil, nil
	}
	return t.Shapes, t.Materials, t.Weapons
}

// fallbackHeroName names a default hero whose table carries no installed name
// for his picture: the male fighter's name in the shipped English install
// (TEXT-073, DIV-1508).
const fallbackHeroName = "Danath"

// defaultHeroName is the installed name of the picture a default hero is drawn
// as. Table.HeroNames holds the four in picture order, male fighter, male mage,
// female fighter, female mage, so his class and sex select one; a nil table, or
// one that read no names, names him fallbackHeroName.
func defaultHeroName(t *mapload.Table, mage, female bool) string {
	i := 0
	if mage {
		i++
	}
	if female {
		i += 2
	}
	if t != nil && t.HeroNames[i] != "" {
		return t.HeroNames[i]
	}
	return fallbackHeroName
}

// MissionPartyAs is MissionParty over the class axis a caller names: mage
// decides which of StartingWeaponName's two arms arms the party — the
// fighter arm, w exactly as every caller before this story already resolved
// it, or the mage arm's own ONE literal, resolved FRESH off t rather than
// taken from w.
//
// THE MAGE ARM IGNORES w AND RESOLVES ITS OWN WEAPON, on ChargenParty's own
// precedent (chargen.go, "THE WEAPON IS RESOLVED FRESH"): w is always
// resolved through StartingWeaponName's FIGHTER arm — resolveStartingWeapon
// is unconditionally mage=false — and cannot answer for a mage any more than
// f.StartWeapon could there. Handing a mage the fighter's own sword would be
// the wrong archetype's numbers wearing a mage's class flag, which is worse
// than the BARE mage a table this build cannot resolve against yields
// instead: resolveWeaponForSlot's own error is discarded here exactly as
// ChargenParty discards it, `w, _ :=` rather than a second refusal path this
// function's signature has nowhere to return.
//
// MissionParty IS THIS FUNCTION AT THE AUTHORED DEFAULT: mage false takes
// neither branch below, so w passes through unchanged and no existing
// caller's behaviour moves.
//
// THE HERO CARRIES THE INSTALLED NAME OF HIS PICTURE (TEXT-074, DIV-1508): the
// male fighter's, or the male mage's when mage is true. defaultHeroName reads
// it off t, so the same bytes the generator's name field opens with name a
// hero who never saw the field.
func MissionPartyAs(mage bool, w *data.Weapon, list data.BodyList, t *mapload.Table) []mapload.PartyMember {
	class, female := defaultChargenAxes()
	if mage {
		class = true
		shapes, materials, weapons := missionTableWeapons(t)
		w, _ = resolveWeaponForSlot(true, shapes, materials, weapons, PartySkillSlot())
	}
	humans := missionHumans(t)
	base, i, ok := data.ChargenBase(humans, class, female)
	profile, face, book, cells := chargenProfile(base, ok, humans, i, class)
	return assembleParty(partyInputs{
		Name:    defaultHeroName(t, class, female),
		Spread:  PartySpread(),
		Slot:    PartySkillSlot(),
		Mage:    class,
		Profile: profile,
		Dir:     data.FigureDirFor(class, female),
		Face:    face,
		Book:    book,
		Weapon:  w,
		Cells:   cells,
		List:    list,
		Table:   t,
		// THE DEFAULT HERO STARTS A CAMPAIGN TOO. This is the party a mission
		// opened without the generation screen gets — cmd/missionrun's own door,
		// and the front end's authored default — so a player who never sees
		// generation is still the person the campaign's documents belong to.
		Documents: true,
	})
}
