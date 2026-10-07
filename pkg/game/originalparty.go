package game

import (
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// RestoredParty is what a loaded original save restored, counted.
//
// IT IS THE DIVERGENCE IN THE READER'S UNITS. The resume report this type feeds
// used to say "POSITION only", which was true of the file and read as "your
// positions are carried" — and the positions a player cares about were the ones
// being dropped. So every number here is a count of CHARACTERS, ITEMS or SPELLS
// rather than of records, offsets or references: a player can check "5
// characters, 12 worn pieces, 3 in the pack" against what he is looking at.
type RestoredParty struct {
	// Characters counts distinct restored persistent characters, not raw archive
	// reference occurrences. Fallback reports that the party is NextParty's
	// after all — a save whose walk read no character at
	// all opens with the fresh party rather than with an empty one, because a
	// mission with nobody in it is not a resume, it is a crash to look at.
	Characters int
	Temporary  int
	Fallback   bool
	// SourceOffsets follow the restored, leader-first party. They are transient
	// file-local joins, never campaign identities or native save fields.
	SourceOffsets []int

	// Statistics is how many characters had their four statistics applied,
	// Pools how many had both pool pairs applied, and Definitions how many
	// resolved the Humans row their own head names.
	Statistics, Pools, Definitions int

	// Worn is how many equipment pieces were placed in a slot and WornSpare
	// how many the file records as worn whose own code names no slot — those
	// go into the pack rather than being dropped. Carried is how many pack
	// items were carried across.
	Worn, WornSpare, Carried int

	// Weapons is how many characters resolved the weapon their own worn slot 1
	// names, out of Armed, how many the file records as holding one.
	Weapons, Armed int

	// SpellbookChars counts present books, including empty ones. SpellRecords
	// counts non-null slots whose membership was restored, not array capacity.
	// SpellParameters counts restored instance range, Defensive and mana cost.
	SpellbookChars, SpellRecords int
	SpellParameters              int

	// JournalChars is how many carry a journal and JournalEntries the total
	// length of its two arrays — a bare count kept for this report; the typed
	// content (sav.Diary) is carried separately as sim.SavedDiary
	// (originaldiaries.go), NOT through RestoredParty: this tree has no journal
	// content itself. What a decoded entry's Count/Remaining pair means beyond
	// its own shape is still Unknown (SAV-DIARY-042, SAV-668).
	JournalChars, JournalEntries int

	// Capacity and Unknowns count restored capacity and own-weight/load words
	// (Unknowns keeps its legacy report field name). Skills is
	// how many members restored their own definition-borne skill set and their
	// own character-scoped experience scalar.
	Capacity, Unknowns, Skills int
	UnsupportedItemEffects     int

	// MapUnitIDs are the map unit ids the restored characters carry, the zero
	// ones dropped. A character the map placed and the player then took into
	// his own group — a hired mercenary is the case the corpus shows — keeps
	// the id of the record the map placed him from, and the map still holds
	// that record.
	//
	// THEY ARE HERE SO THE PLACEMENT CAN BE WITHDRAWN. Left in, the map would
	// build an entity for the record AND the party would add one for the
	// character, and the player would be looking at the same person twice.
	MapUnitIDs []uint16

	// Withdrawn is how many of those placements the map actually held and
	// gave up. It is separate from len(MapUnitIDs) because a character can
	// carry an id no map record answers to, and the difference between "the
	// map had it" and "the character claimed it" is what says whether a
	// duplicate was possible at all.
	Withdrawn int

	// Rebound is how many restored characters carry a map unit id the mission's
	// compiled script will bind to their own entity. It is len(MapUnitIDs) by
	// construction and is reported separately because the two answer different
	// questions: MapUnitIDs is what was WITHDRAWN from the map, and this is
	// what was BOUND to the party in its place.
	Rebound int

	// LeadFrom is the file position the party's leader came from, and -1 where
	// no ordering rule applied and the file's own order was kept. Zero is a
	// real answer — the leader was already first — so it is written on
	// every path rather than left at the zero value.
	LeadFrom int

	// LeadRule is which of the three rules produced that answer. It is reported
	// because two of the three are this tree's own choice and a divergence a
	// reader cannot see is a divergence that is not disclosed.
	LeadRule LeadRule

	// Err is the walk's own error where it had one. A walk that stopped
	// halfway still restored the characters before it, and this is what says
	// so instead of the count silently being short.
	Err error
}

// WithdrawRestored removes from the map every unit record a restored character
// claims, and answers how many it removed.
//
// IT PREVENTS THE SAME PERSON APPEARING TWICE. The file puts a hired mercenary
// in the human participant's own group AND the map still carries the record he
// was placed from, so a resume that restored him as a party member without
// withdrawing the record would build two entities for one person: two figures on
// one cell with the same statistics, one commandable and one not.
//
// IT RUNS IN THE ONE WINDOW WHERE THE MAP'S UNITS ARE STILL RECORDS — after the
// decode and before the start — which is the same window applyOriginalPositions
// writes in and for the same reason.
//
// THE SCRIPT'S REFERENCE FOLLOWS THE PERSON, not the record. The
// id-to-entity table the compile builds takes a second source of entries: a
// party member whose Saved names a unit binds that id to his own entity, so
// a Target_Unit parameter naming a withdrawn record measures the restored
// character where he actually stands. mapload.ScriptUnits is where that is
// done and Saved.MapUnitID is what carries it.
//
// 0147 SHIPPED THIS AS A DISCLOSED DIVERGENCE and the disclosure was wrong about
// its own cost. It said such an arm "finds no entity", which read as an arm that
// does nothing. An unresolved check is still built and still takes its register,
// and returns before writing anything — so the register keeps its initial zero
// and a proximity comparison against a small constant HOLDS. Mission 20 resumed
// from an original save declared victory at tick 16 on that arm, and mission 10
// handed the witch over a second time on another.
//
// WHAT IT STILL COSTS: a restored character is no longer a map unit, so an arm
// that acts on the RECORD — a group command naming the group it belonged to —
// reaches a figure the player now commands. Nothing in the corpus was observed
// to need that. The count above is what makes the choice visible.
func WithdrawRestored(m *alm.Map, ids []uint16) int {
	if m == nil || len(ids) == 0 {
		return 0
	}
	claimed := make(map[uint16]bool, len(ids))
	for _, id := range ids {
		claimed[id] = true
	}
	kept := m.Units[:0]
	for _, u := range m.Units {
		if claimed[u.UnitID] {
			continue
		}
		kept = append(kept, u)
	}
	n := len(m.Units) - len(kept)
	m.Units = kept
	return n
}

// Native mission saves retain the original entry party's Saved markers. They
// identify the placements removed on original import, not new party members
// adopted during this mission. Zero means no authored placement to withdraw.
func withdrawSavedPartyPlacements(m *alm.Map, party []mapload.PartyMember) int {
	var ids []uint16
	for _, member := range party {
		if member.Saved != nil && member.Saved.MapUnitID != 0 {
			ids = append(ids, member.Saved.MapUnitID)
		}
	}
	return WithdrawRestored(m, ids)
}

// RestoreParty is the party a loaded original save opens with: the human
// participant's own characters, each carrying the state the file records.
//
// bodies, humans and the three piece tables are handed in rather than reached
// for, on MissionParty's own terms one file over: this function opens no
// install and reads no field of a front end, so it can be decided against a
// hand-built save with no install anywhere near it.
//
// A SAVE WHOSE WALK REACHES NO CHARACTER FALLS BACK TO fresh, and says so. That
// is not a silent substitution — Fallback is a field, the report prints it, and
// the load window's own caveat is written for the case where it is false. The
// alternative is a mission opened with an empty party, which is a world the
// player cannot play and cannot diagnose.
func RestoreParty(sf *sav.File, fresh []mapload.PartyMember, bodies data.BodyList,
	t *mapload.Table) ([]mapload.PartyMember, RestoredParty) {

	chars, err := sf.Party()
	return restorePartyCharacters(chars, err, fresh, bodies, t)
}

// restorePartyCharacters is shared by original import and native provenance
// verification. No fields from a persisted PartyMember baseline enter it.
func restorePartyCharacters(chars []sav.Character, err error, fresh []mapload.PartyMember,
	bodies data.BodyList, t *mapload.Table) ([]mapload.PartyMember, RestoredParty) {
	r := RestoredParty{Err: err, LeadFrom: -1}
	if len(chars) == 0 {
		r.Fallback = true
		return fresh, r
	}
	persistent := chars[:0]
	for _, c := range chars {
		if persistentOriginalCharacter(c, t) {
			persistent = append(persistent, c)
		} else {
			r.Temporary++
		}
	}
	chars = persistent
	r.Characters = len(chars)
	if len(chars) == 0 {
		r.Fallback = true
		return fresh, r
	}
	chars, r.LeadFrom, r.LeadRule = leadFirst(chars)
	out := make([]mapload.PartyMember, 0, len(chars))
	for _, c := range chars {
		r.SourceOffsets = append(r.SourceOffsets, c.Off)
		if c.MapUnitID != 0 {
			r.MapUnitIDs = append(r.MapUnitIDs, c.MapUnitID)
			r.Rebound++
		}
		out = append(out, restoredMember(c, bodies, t, &r))
	}
	// SAV records carry their Humans row, not this engine's CompanionNPC.
	// Resolve it against the registry and the saved starting hero's axes.
	// A row alone cannot name Reniesta: row 28 also constructs Fergard.
	if t != nil && r.LeadRule == LeadNamed {
		heroDir := data.FigureDir(out[0].FigureDir)
		for i, c := range chars {
			if c.Hero || out[i].Hired() {
				continue
			}
			if npc, ok := t.NPC.CampaignCompanionForRow(t.Humans, int(c.DefRow), heroDir.Mage(), heroDir.Female()); ok {
				out[i].CompanionNPC = int(npc)
				out[i].ID = fmt.Sprintf("npc:%d", npc)
			}
		}
	}
	return out, r
}

// persistentOriginalCharacter keeps the primary character, every Humans
// definition explicitly authored as a PC_ row, and a hired mercenary's own
// actor record (SAV-616/617). That data identity includes joined player
// characters such as the paladin and excludes ordinary named mission allies
// without relying on a localized display name or a growing row-number
// allow-list.
func persistentOriginalCharacter(c sav.Character, t *mapload.Table) bool {
	if c.Hero {
		return true
	}
	if _, ok := originalSiegeHire(c, t); ok {
		return true
	}
	if c.Class != "Human" || t == nil || t.Humans == nil {
		return false
	}
	i := int(c.DefRow)
	if i > 0 && i < t.Humans.Len() && strings.HasPrefix(t.Humans.EntryName(i), "PC_") {
		return true
	}
	_, ok := originalMercenaryType(c, t)
	return ok
}

// originalSiegeHire names the hired siege engine a saved Unit actor is: the
// Catapult or Ballista row of the Units table (MERC-LEVEL-005). A legacy Human
// actor with definition row 0 is not one and stays dropped; the hire flag and
// the working pool rebuild it.
func originalSiegeHire(c sav.Character, t *mapload.Table) (int, bool) {
	if c.Hero || c.Class != "Unit" || t == nil || t.Units == nil || c.DefRow == 0 || int(c.DefRow) >= t.Units.Len() {
		return 0, false
	}
	switch t.Units.EntryName(int(c.DefRow)) {
	case siegeHireName(1):
		return 1, true
	case siegeHireName(2):
		return 2, true
	}
	return 0, false
}

func originalMercenaryType(c sav.Character, t *mapload.Table) (int, bool) {
	if c.Hero || c.Class != "Human" || t == nil || t.Humans == nil || c.DefRow == 0 || int(c.DefRow) >= t.Humans.Len() {
		return 0, false
	}
	definition := t.Humans.EntryName(int(c.DefRow))
	typ := uint8(c.DisplayBacking)
	if typ != 0 {
		return int(typ), typ >= 3 && typ <= 15 && strings.Contains(definition, "NPC")
	}
	if c.DisplayBacking == 0 && c.Name != "" && c.Name == definition {
		return mercenaryHireTypeFromName(c.Name)
	}
	return 0, false
}

func ordinaryActorName(member mapload.PartyMember, table *mapload.Table) string {
	if typ, ok := mercenaryHireTypeFromName(member.Name); ok && typ == int(member.MercenaryType) && table != nil && table.Humans != nil {
		row := nativeCityDefRow(member, mapload.PartyMember{}, table)
		if row != 0 && table.Humans.EntryName(int(row)) == member.Name {
			return ""
		}
	}
	return member.Name
}

// mercenaryHireTypeFromName recognizes a hired mercenary's own actor record
// by name (SAV-616: class Human, classKey 58, typeWord 0x000a, worn 6, three
// per file, health 120 at hire; SAV-617: one new group under the Player).
// Neither claim reads the Name field itself, so this is not a wire-format
// fact -- it is this tree's own read-side counterpart of the exact template
// string buildMercenarySquad (tavern.go) already writes at hire time,
// fmt.Sprintf("NPC%02d_%d", typ, level), reused verbatim rather than
// re-derived by a second parser. The full round-trip equality (not just a
// prefix or a partial Sscanf count) rejects a name that merely starts the
// same way, including one with no zero-padded type digit.
func mercenaryHireTypeFromName(name string) (int, bool) {
	var typ, level int
	if n, err := fmt.Sscanf(name, "NPC%02d_%d", &typ, &level); n != 2 || err != nil {
		return 0, false
	}
	if name != fmt.Sprintf("NPC%02d_%d", typ, level) {
		return 0, false
	}
	// buildMercenarySquad routes typ<=2 to buildSiegeSquad instead (Catapult/
	// Ballista, a Units-table Unit(0x198) record, MERC-LEVEL-005) -- a
	// different wire class SAV-608's retraction does not cover. Only 3..15
	// name a Human(0x1e8) mercenary this function's caller may treat as
	// persistent.
	if typ < 3 || typ > 15 {
		return 0, false
	}
	return typ, true
}

// LeadRule names which rule put a character at the head of the party.
type LeadRule uint8

const (
	// LeadKept is no rule at all: nothing resolved and the file's own order
	// was left exactly as written.
	LeadKept LeadRule = iota
	// LeadNamed is the file's own field. SAV-HERO-059: Player+0x34 is an
	// identity key naming the participant's own starting character, resolved
	// through the identity map on load.
	LeadNamed
	// LeadRuntimeID is the fallback, and it is this tree's inference rather
	// than a field: the actor carrying runtime creation-order id 1.
	LeadRuntimeID
)

// String is the rule in the reader's own units, for the resume report.
func (r LeadRule) String() string {
	switch r {
	case LeadNamed:
		return "the character the file names as the player's own"
	case LeadRuntimeID:
		return "the character carrying the participant's runtime id -- the file named none"
	default:
		return "NO RULE APPLIED -- the file's own order was kept"
	}
}

// heroRuntimeID is the runtime creation-order id the human participant's own
// character carries. SAV-ID-015 reads the allocator behind it: the id is the
// actor's own +0x04 field, assigned at tick-list insert as the lowest free bit
// of a bitmap with id 0 pre-marked, so the participant's character — created
// first — holds 1 while nothing has been freed.
//
// IT IS THE FALLBACK NOW AND NOT THE RULE. SAV-ID-015 grades this use Medium
// as a law: a corpse decay frees its bitmap bit and the next spawn reuses
// the lowest free one, so a save with mid-session churn could carry no id 1
// or, through a reuse this corpus does not show, two. It is kept because it
// is measured to agree with the decoded field on all 18 distinct corpus
// saves, so it costs nothing and covers a file whose named character does
// not resolve to anyone the walk reached.
const heroRuntimeID = 1

// leadFirst puts the human participant's own character at the head of the
// party, and answers the file position he came from and the rule that found him.
//
// THE FILE'S OWN ORDER RANKS NOTHING. SAV-GRPORD-058 reads both arms of the
// actor-list serializer: the store arm is a head-to-tail CObList walk with no
// comparison in it, so the order is the order of appends and is runtime state.
// The same five actors appear in three different orders across one session. A
// party built in file order is led by whoever was appended first, which on
// mission 10's own save is the witch who joins rather than the player's hero.
//
// THE RULE IS THE FILE'S OWN FIELD (SAV-HERO-059), and 0148's runtime-id rule
// is its fallback. Both are total and neither reorders on a guess: a file that
// resolves neither is left exactly as written, and LeadFrom is -1 so the report
// says so.
//
// ONLY THE LEADER MOVES. Every other character keeps the order the file's actor
// list holds. Reordering the rest would need a ranking, and SAV-GRPORD-058 is
// the finding that there is none to read.
func leadFirst(chars []sav.Character) ([]sav.Character, int, LeadRule) {
	if at, ok := onlyOne(chars, func(c sav.Character) bool { return c.Hero }); ok {
		return moveToFront(chars, at), at, LeadNamed
	}
	if at, ok := onlyOne(chars, func(c sav.Character) bool {
		return c.RuntimeID == heroRuntimeID
	}); ok {
		return moveToFront(chars, at), at, LeadRuntimeID
	}
	return chars, -1, LeadKept
}

// onlyOne answers the index of the single character satisfying want, and
// whether there was exactly one. Two matches is not a tie to break: it is a
// file this tree has no rule for, and both callers treat it as no match.
func onlyOne(chars []sav.Character, want func(sav.Character) bool) (int, bool) {
	at := -1
	for i, c := range chars {
		if !want(c) {
			continue
		}
		if at >= 0 {
			return -1, false
		}
		at = i
	}
	return at, at >= 0
}

// moveToFront returns chars with the character at at first and every other in
// the order the file holds.
func moveToFront(chars []sav.Character, at int) []sav.Character {
	if at <= 0 {
		return chars
	}
	out := make([]sav.Character, 0, len(chars))
	out = append(out, chars[at])
	out = append(out, chars[:at]...)
	return append(out, chars[at+1:]...)
}

// restoredMember is one character, as a party member.
func restoredMember(c sav.Character, bodies data.BodyList, t *mapload.Table,
	r *RestoredParty) mapload.PartyMember {

	// THE FOUR STATISTICS ARE THE FILE'S, and they are the only statistic
	// axis that is: they are the four words SAV-UNITFLD-049 grades High off
	// UNIT-STREAM-001's own slots 0 to 3, and they are exactly the four
	// numbers character generation produces — so they go on Hero and the
	// whole derived-stat graph folds them the way it folds a generated
	// character's.
	//
	// THE SKILL LEVELS AND PER-SKILL EXPERIENCE COME FROM THIS CHARACTER'S OWN
	// save record (SAV-HEROSKILL-064). The Humans row still supplies the profile,
	// figure and class inputs that are not serialized as character progress.
	if typ, ok := originalSiegeHire(c, t); ok {
		if member, ok := siegeHireMember(t.Units, typ); ok {
			r.Statistics++
			r.Definitions++
			return member
		}
	}
	profile, figure, face, mage, template, suppressCorpseLoot, typeID, rowRotationSpeed := restoredDefinition(c, t, r)
	if c.Basis != nil && c.Class != "Unit" {
		profile.Fighter = c.Basis.Human.Fighter
	}
	hero := template
	hero.Body = int32(c.Stat(sav.StatBody))
	hero.Reaction = int32(c.Stat(sav.StatReaction))
	hero.Mind = int32(c.Stat(sav.StatMind))
	hero.Spirit = int32(c.Stat(sav.StatSpirit))
	for i := range hero.Skill {
		hero.Skill[i] = int32(c.SkillLevels[i])
	}
	repairHeroSkills(&hero.Skill, c.SkillLevels, c.SkillXP)
	r.Statistics++
	r.Skills++
	r.Capacity++
	r.Unknowns += 2

	wornItems, carriedItems := restoredItemLoadout(c, r, t)

	mercType, hired := 0, false
	if !c.Hero {
		mercType, hired = originalMercenaryType(c, t)
	}
	if hired {
		wornItems = restoredHireWornEffects(wornItems, c.DefRow, t)
	}
	worn, carried := itemLoadoutCodes(wornItems, carriedItems)

	// HIS WEAPON IS THE ONE HIS OWN SLOT 1 NAMES, resolved through the SAME
	// code-to-weapon resolver a generated character's is (data.WeaponFromCode).
	// It matters beyond the picture: the weapon ASSIGNS the swing cadence, so a
	// restored character whose weapon did not resolve would arrive swinging at
	// bare-handed speed while wearing a sword.
	var weapon *data.Weapon
	if code, ok := wornWeapon(worn); ok {
		r.Armed++
		if t != nil {
			if w, err := data.WeaponFromCode(code, t.Shapes, t.Materials, t.Weapons); err == nil {
				weapon = &w
				r.Weapons++
			}
			// Current item effects precede the code-only legacy staff template.
			withoutCode := wornItems[0]
			withoutCode.Code = 0
			if spell, power, present := wornItems[0].CastSpell(); present {
				if weapon != nil {
					weapon.SpellPower = power
					if t.Spells != nil && spell != 0 && int(spell) < t.Spells.Len() {
						weapon.SpellName = strings.ReplaceAll(t.Spells.EntryName(int(spell)), " ", "_")
					}
				}
			} else if mage && withoutCode.Empty() {
				if staff, err := data.ResolveWeapon(mageWeaponName, t.Shapes, t.Materials, t.Weapons); err == nil && staff.Code == code {
					weapon = &staff
				}
			}
		}
	}
	weapon = canonicalMageWeapon(weapon, mage, t)

	// HIS PROFILE IS THE ROW HIS OWN HEAD NAMES. The head's +0x0c byte is a
	// row index into the collection the class picks — Humans for a Human
	// record — and the row is re-derived at every load rather than stored
	// (ITEM-DEF-002). So the three inputs the derived-stat graph reads that
	// are not a statistic, a skill or an item come off the same row the
	// original streamed him from, rather than off this tree's own default
	// archetype.

	// The map body comes from visible equipment. Character sex does not affect
	// that body (HERO-APPEAR-045), but the class axis does. The resolved Humans
	// row supplies the class axis used by FigureFor, so the body and inventory
	// figure use the same class.
	//
	// A RECOGNIZED HIRE IS NOT COMPOSED. Restoring a recognized hire through
	// HeroAppearance anyway would draw him in the cloaked-hero art the live
	// hire itself never wore -- tavern.go's own comment measures this exactly
	// as NPC14_1 reaching a mission as body "clubman_" under "heroes" -- so the
	// load side skips the same composition the hire side skips, for the same
	// member.
	var body data.HeroBody
	dir := ""
	class := typeID
	if !hired {
		body, dir, class, _ = data.HeroAppearance(bodies, equipmentFromSlots(worn), mage, false)
	}

	carry := restoredCarry(c, wornItems, carriedItems, hired, t)
	var hiredRotationSpeed int32
	if hired {
		hiredRotationSpeed = rowRotationSpeed
	}
	name, definition := c.Name, uint8(0)
	if hired {
		definition = c.DefRow
		if name == "" {
			name = t.Humans.EntryName(int(c.DefRow))
		}
	}
	return mapload.PartyMember{
		ID:                 restoredPartyID(c),
		Name:               name,
		DefinitionRow:      definition,
		Temporary:          hired,
		PlayerCharacter:    !hired,
		StartingHero:       c.Hero,
		MercenaryType:      uint8(mercType),
		Class:              class,
		Body:               string(body),
		BodyDir:            dir,
		Mage:               mage,
		Profile:            profile,
		FigureDir:          string(figure),
		FigureFace:         face,
		Hero:               hero,
		Weapon:             weapon,
		Worn:               worn,
		WornItems:          wornItems,
		Carried:            carried,
		CarriedItems:       carriedItems,
		Carry:              &carry,
		Saved:              restoredSaved(c, r),
		KnownSpells:        c.KnownSpells(),
		Book:               importedSpellbook(c.HasSpellbook, c.Spells),
		SpellbookRestored:  true,
		SpellbookPresent:   c.HasSpellbook,
		SuppressCorpseLoot: suppressCorpseLoot,
		HiredRotationSpeed: hiredRotationSpeed,
	}
}

// restoredCarry copies the six per-skill experience fields from this saved
// character. Unit+0x130 is the separately serialized aggregate and equals the
// sum of these fields on the measured corpus (SAV-HEROXP-063); it does not
// select a slot.
func restoredCarry(c sav.Character, worn [sim.EquipSlots]sim.ItemInstance, items []sim.ItemInstance, hired bool, tables ...*mapload.Table) mapload.Carry {
	var xp [data.SkillSlots]int32
	for i := range xp {
		xp[i] = int32(c.SkillXP[i])
	}
	codes, carried := itemLoadoutCodes(worn, items)
	carry := mapload.Carry{SkillXP: xp, Equipped: codes, Items: carried,
		EquippedItems: cloneOriginalEquipment(worn), ItemInstances: cloneOriginalItems(items)}
	if c.LoadState.Present && !hired {
		carry.LiveLoad = originalActorLoad(c.LoadState, int32(c.LoadState.Speed))
		carry.LiveLoad.Inventory.Source = originalActorBasis(c.Basis)
		for _, piece := range c.Items {
			carry.OrderedStacks = append(carry.OrderedStacks, sim.StackItem(originalItemInstance(piece, nil, tables...), uint32(piece.Stack)))
		}
	}
	return carry
}

func restoredPartyID(c sav.Character) string {
	if c.Hero {
		return "hero"
	}
	return ""
}

// restoredSaved is the cell and the two pool pairs, which are the values the
// mint must PUT IN rather than fold (mapload.Saved).
func restoredSaved(c sav.Character, r *RestoredParty) *mapload.Saved {
	r.Pools++
	if c.HasSpellbook {
		r.SpellbookChars++
		r.SpellRecords += len(c.Spells)
		r.SpellParameters += len(c.Spells)
	}
	if c.HasDiary {
		r.JournalChars++
		r.JournalEntries += c.JournalLen + c.JournalWords
	}
	return &mapload.Saved{
		// THE CELL IS THE SAVE'S OWN, and the column and the row are that way
		// round: a Cell is (X, Y) and the save packs (row << 8) | col, so
		// crossing them would put every restored character at the transpose of
		// where he stood — visible on a square map only as a party in the wrong
		// place, and on a rectangular one as a party off the edge.
		Cell: mapload.Cell{X: int32(c.Col()), Y: int32(c.Row())},

		// THE MAP UNIT ID TRAVELS WITH HIM. The resume withdraws the record this
		// id names so one person does not become two entities, and the mission's
		// script still names that id — so it is carried on the member, and
		// ScriptUnits binds it to the entity he becomes.
		MapUnitID: c.MapUnitID,

		HP:                int32(c.Stat(sav.StatHealth)),
		MaxHP:             int32(c.Stat(sav.StatHealthMax)),
		Mana:              int32(c.Stat(sav.StatMana)),
		MaxMana:           int32(c.Stat(sav.StatManaMax)),
		HealthRegenPeriod: int32(c.Stat(sav.StatHealthRegen)),
		ManaRegenPeriod:   int32(c.Stat(sav.StatManaRegen)),
	}
}

// restoredDefinition returns the profile and inventory figure stated by the
// character's Humans row. A missing row keeps the pre-existing zero-profile,
// male-fighter fallback. The trailing int32 pair is the row's own TypeID and
// RotationSpeed columns: a recognized hired mercenary's Class comes from
// TypeID rather than from HeroAppearance's worn-equipment composition
// (restoredMember), and its HiredRotationSpeed comes from this same row
// rather than the live cadence any Class-derived source would otherwise
// imply, so the caller needs both raw columns even when the composed body is
// skipped.
func restoredDefinition(c sav.Character, t *mapload.Table, r *RestoredParty) (data.Profile, data.FigureDir, int, bool, data.Hero, bool, int32, int32) {
	if t == nil || t.Humans == nil || c.Class != "Human" {
		return data.Profile{}, data.FigureDirManFighter, 1, false, data.Hero{}, false, 0, 0
	}
	i := int(c.DefRow)
	if i <= 0 || i >= t.Humans.Len() {
		return data.Profile{}, data.FigureDirManFighter, 1, false, data.Hero{}, false, 0, 0
	}
	templateName := t.Humans.EntryName(i)
	def, err := data.NewHumanDef(templateName, t.Humans.EntryParams(i))
	if err != nil {
		return data.Profile{}, data.FigureDirManFighter, 1, false, data.Hero{}, false, 0, 0
	}
	figure, face := data.FigureFor(def.TypeID, def.Face, def.Gender)
	mage := figure == data.FigureDirManMage || figure == data.FigureDirWomanMage
	r.Definitions++
	// Learned membership is saved character state, not a definition default.
	return def.Profile(), figure, face, mage, def.Hero(),
		mapload.SuppressesCorpseLoot(templateName), def.TypeID, def.RotationSpeed
}

// restoredLoadout is what the character has on and what he is carrying.
//
// EVERY PIECE IS ROUTED BY THE SLOT ITS OWN CODE CARRIES and not by the site the
// file read it from. The file writes fourteen equipment references — the weapon
// at +0x74, the shield at +0x78 and the twelve armour slots — and each one's
// appearance word already states which of the twelve slots it belongs to, field
// B being both the slot and the class at once (data.EquipSlotFor). So there is
// no site-to-slot table here to get wrong: the code says where it goes.
//
// A PACK ITEM KEEPS ITS OWN SLOT CODE AND STAYS IN THE PACK. Being in the
// container is what says an item is not worn; 73 of the 98 pack items measured
// across the owner's saves carry a real equipment slot in their code, and
// equipping them here would arm a character with what he took off.
func restoredLoadout(c sav.Character, r *RestoredParty) ([sim.EquipSlots]uint16, []uint16) {
	items, carriedItems := restoredItemLoadout(c, r)
	return itemLoadoutCodes(items, carriedItems)
}

func restoredItemLoadout(c sav.Character, r *RestoredParty, tables ...*mapload.Table) ([sim.EquipSlots]sim.ItemInstance, []sim.ItemInstance) {
	var worn [sim.EquipSlots]sim.ItemInstance
	var carried []sim.ItemInstance
	for _, p := range c.Worn {
		item := originalItemInstance(p, r, tables...)
		count := int(p.Stack)
		if count == 0 {
			count = 1
		}
		slot, ok := data.EquipSlotFor(data.ItemCode(p.Code))
		if !ok || !worn[slot-1].Empty() {
			// A piece whose code names no slot, or a second piece claiming a
			// slot already filled, goes into the pack. It is on the
			// character either way and dropping it would lose a real item.
			for range count {
				carried = append(carried, item.Clone())
				r.WornSpare++
			}
			continue
		}
		worn[slot-1] = item.Clone()
		r.Worn++
		// One unit occupies the equipment slot. Any remaining units in the
		// saved stack stay in the pack (ITEM-STACK-003).
		for i := 1; i < count; i++ {
			carried = append(carried, item.Clone())
			r.Carried++
		}
	}
	for _, p := range c.Items {
		item := originalItemInstance(p, r, tables...)
		count := int(p.Stack)
		if count == 0 {
			count = 1
		}
		for range count {
			carried = append(carried, item.Clone())
			r.Carried++
		}
	}
	return worn, carried
}

func restoredHireWornEffects(worn [sim.EquipSlots]sim.ItemInstance, defRow byte, t *mapload.Table) [sim.EquipSlots]sim.ItemInstance {
	if t == nil || t.Humans == nil {
		return worn
	}
	i := int(defRow)
	if i <= 0 || i >= t.Humans.Len() {
		return worn
	}
	template, _, _, err := mapload.HumanRowEquipment(t.Humans.EntryStrings(i), t)
	if err != nil {
		return worn
	}
	for slot := range worn {
		if worn[slot].Code == 0 || worn[slot].Code != template[slot].Code {
			continue
		}
		worn[slot].Effects = template[slot].Effects
	}
	return worn
}

func originalItemInstance(p sav.Piece, r *RestoredParty, tables ...*mapload.Table) sim.ItemInstance {
	if p.Class == "" && p.Row == 0 && p.Kind == 0 && p.Price == 0 && p.Weight == 0 &&
		len(p.Effects) == 0 && len(p.UnsupportedEffectStates) == 0 && p.WeaponSpell == nil &&
		p.W52 == ([24]byte{}) && p.W6A == ([22]byte{}) && p.W50 == 0 &&
		p.A52 == ([22]byte{}) && p.A50 == 0 && p.S50 == ([22]byte{}) {
		return sim.PlainItem(p.Code)
	}
	effects := make([]sim.ItemEffect, len(p.Effects))
	for i, effect := range p.Effects {
		effects[i] = sim.ItemEffect{Kind: effect.Kind, Mode: effect.Mode, Operand: effect.Operand}
	}
	if r != nil {
		r.UnsupportedItemEffects += len(p.UnsupportedEffectStates)
	}
	var source sim.SourceEquipment
	switch p.Class {
	case "Weapon":
		source = sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: equipmentDefinitionRow(p.Code, p.Row), OwnKind: p.W50, Attack: p.W52, Defence: p.W6A}
		if s := p.WeaponSpell; s != nil {
			source.Spell = sim.SourceItemSpell{Present: true, ID: s.ID, Range: s.Range, Defensive: s.Defensive, ManaCost: s.ManaCost}
		}
	case "Armor":
		source = sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: equipmentDefinitionRow(p.Code, p.Row), OwnKind: p.A50, Defence: p.A52}
	case "Shield":
		source = sim.SourceEquipment{Class: sim.SourceShield, DefinitionRow: equipmentDefinitionRow(p.Code, p.Row), Defence: p.S50}
	}
	if source.Class != 0 {
		source.EffectsUnsupported = len(p.UnsupportedEffectStates) != 0
	}
	item := sim.ItemInstance{Code: p.Code, Kind: p.Kind, Effects: effects, Price: p.Price, Weight: p.Weight, WeightPresent: true, SourceEquipment: source}
	if len(tables) != 0 {
		item = mapload.BindSourceItemDefinition(item, tables[0])
	}
	return item
}

func itemLoadoutCodes(worn [sim.EquipSlots]sim.ItemInstance, carried []sim.ItemInstance) ([sim.EquipSlots]uint16, []uint16) {
	var codes [sim.EquipSlots]uint16
	for i := range worn {
		codes[i] = worn[i].Code
	}
	items := make([]uint16, len(carried))
	for i := range carried {
		items[i] = carried[i].Code
	}
	return codes, items
}

func cloneOriginalItems(in []sim.ItemInstance) []sim.ItemInstance {
	out := make([]sim.ItemInstance, len(in))
	for i := range in {
		out[i] = in[i].Clone()
	}
	return out
}

func cloneOriginalEquipment(in [sim.EquipSlots]sim.ItemInstance) [sim.EquipSlots]sim.ItemInstance {
	var out [sim.EquipSlots]sim.ItemInstance
	for i := range in {
		out[i] = in[i].Clone()
	}
	return out
}

// wornWeapon is the code in equipment slot 1, and whether that slot is filled.
// Slot 1 is the weapon slot by the code's own class field, which is what
// data.WeaponFromCode itself refuses anything else on.
func wornWeapon(worn [sim.EquipSlots]uint16) (data.ItemCode, bool) {
	code := data.ItemCode(worn[0])
	return code, code != 0
}
