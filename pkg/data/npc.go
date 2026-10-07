package data

import (
	"strconv"

	"againrom/pkg/formats/reg"
)

// npcSectionPrefix is the stem every per-NPC section of the scenario NPC
// registry carries; the whole section name is this plus the subscript in
// decimal. The registry's own lookup folds case, so the spelling here decides
// nothing but which characters are compared.
const npcSectionPrefix = "npc"

// npcDefinitionKey is the key inside such a section that names a definition-table
// entry. It is the ONE key this type reads: the other nine a section may carry —
// the flag list, the face, the portrait rectangle, the two price terms — belong
// to screens and to the mercenary economy, and a table that carried them would be
// a table three unrelated consumers keep in agreement.
const npcDefinitionKey = "DataBinID"

const (
	npcPriceAKey = "PriceA"
	npcPriceBKey = "PriceB"
)

// npcComposed is the definition id that is not one. A section carrying this value
// does not name an entry: the original composes a template from that section's
// own flag tokens instead. The value ships four times, one of them on the
// player's own record.
//
// It is applied HERE and not at the placement that reads the table, because it is
// a fact about what the registry's own column means. A consumer handed the raw
// number back would have to know the sentinel to be correct, and every consumer
// would have to know it separately.
const npcComposed int32 = 26

// NPCDefs is the scenario NPC registry as a placement resolves it: for each NPC
// subscript, the definition-table entry that subscript names.
//
// It is a LOOKUP and not the parsed registry. What a placement needs of that file
// is one integer per section, so carrying the registry would let a later caller
// reach for a key whose meaning nothing in this tree has established — and would
// put the section-naming rule at every such call site instead of at this one.
//
// The zero value and a nil pointer are both "no registry": every subscript
// answers no entry. That is what lets a caller hold "the campaign container was
// not opened" with no branch of its own, exactly as a nil Collection does one file
// over.
type NPCDefs struct {
	byID      map[int32]int32
	mercenary map[int32]MercenaryTerms
	composed  map[int32]NPCTokens
	flags     map[int32]NPCTokens
	hero      map[int32]heroTemplate
	heroFaces [4]int32
}

// MercenaryTerms is the pair used by the tavern's whole-squad price.
type MercenaryTerms struct {
	PriceA int32
	PriceB int32
}

// LoadNPCDefs reads every `npc<n>` section of r and records the definition id it
// names.
//
// A section whose name is not that stem followed by a decimal number is skipped,
// which is how the four character-archetype blocks and the multiplayer face lists
// stay out of the lookup without being named here — a registry that grew a sixth
// such block would be skipped by the same rule rather than by a list needing an
// edit. A section carrying no definition id is skipped too, and 82 of the shipped
// 105 are exactly that.
//
// The sentinel is kept in the separate composed map rather than the direct-row
// lookup, so ServerID still holds only ids that name a concrete row and
// "absent" keeps one meaning at that reader.
//
// A nil registry is no sections. It cannot fail: every value it takes is already
// parsed, and a key of the wrong kind is simply not an integer and is skipped.
func LoadNPCDefs(r *reg.Reg) *NPCDefs {
	n := &NPCDefs{byID: make(map[int32]int32), mercenary: make(map[int32]MercenaryTerms),
		composed: make(map[int32]NPCTokens), flags: make(map[int32]NPCTokens),
		hero: make(map[int32]heroTemplate), heroFaces: [4]int32{1, 1, 1, 1}}
	if r == nil || r.Root == nil {
		return n
	}
	n.loadHeroDefaults(r)
	for _, sec := range r.Root.Children {
		id, ok := npcSubscript(sec.Name)
		if !ok {
			continue
		}
		def, ok := r.GetInt(sec.Name, npcDefinitionKey)
		flags, hasFlags := r.GetString(sec.Name, npcFlagsKey)
		if hasFlags {
			n.flags[id] = npcTokens(flags)
			n.loadHeroTemplate(r, sec.Name, id, flags)
		}
		if ok {
			if def != npcComposed {
				n.byID[id] = def
			} else if hasFlags {
				n.composed[id] = npcTokens(flags)
			}
		}
		a, aok := r.GetInt(sec.Name, npcPriceAKey)
		b, bok := r.GetInt(sec.Name, npcPriceBKey)
		if aok && bok {
			n.mercenary[id] = MercenaryTerms{PriceA: a, PriceB: b}
		}
	}
	return n
}

// Hero reports whether the NPC section carries the exact Hero flag. The
// placement constructor uses this one token as its player-character mode:
// ordinary NPC and explicit Humans placements preserve their table TypeID,
// while a Hero NPC is rewritten into the persistent player-character band.
func (n *NPCDefs) Hero(id int32) bool {
	if n == nil {
		return false
	}
	return n.flags[id].Has(NPCTokenHero)
}

// ComposedArchetype resolves one of the registry's composed Start records
// against the existing player's class and sex. The record's own tokens state
// each axis: Me inherits both, MyClass/MySex inherit one, their negations
// invert it, and an explicit Mage/Female (or negation) supplies it directly.
// A record whose DataBinID is not the composition sentinel answers false.
func (n *NPCDefs) ComposedArchetype(id int32, playerMage, playerFemale bool) (mage, female bool, ok bool) {
	if n == nil {
		return false, false, false
	}
	t, ok := n.composed[id]
	if !ok {
		return false, false, false
	}
	mage, female = playerMage, playerFemale
	switch {
	case t.Has(NPCTokenMage):
		mage = true
	case t.Has(NPCTokenNotMage):
		mage = false
	case t.Has(NPCTokenNotMyClass):
		mage = !playerMage
	case t.Has(NPCTokenMyClass), t.Has(NPCTokenMe):
		mage = playerMage
	}
	switch {
	case t.Has(NPCTokenFemale):
		female = true
	case t.Has(NPCTokenNotFemale):
		female = false
	case t.Has(NPCTokenNotMySex):
		female = !playerFemale
	case t.Has(NPCTokenMySex), t.Has(NPCTokenMe):
		female = playerFemale
	}
	return mage, female, true
}

// CampaignServerID resolves the definition row a campaign NPC uses in one
// mission. Direct DataBinID records keep their own server id. A composed Start
// record selects one of the four sex/class rows and advances that selector by
// four rows for every completed forty-mission tier.
//
// The selector order is the collection's own: male fighter, female fighter,
// male mage, female mage. Mission 30 therefore uses server ids 26..29, mission
// 70 uses 30..33, mission 100 uses 34..37 and mission 140 uses 38..41. Integer
// division is intentional; these are the mission-number bands the original
// uses, not chapter ordinals inferred by a caller.
func (n *NPCDefs) CampaignServerID(id int32, mission int, playerMage, playerFemale bool) (int32, bool) {
	if serverID, ok := n.ServerID(id); ok {
		return serverID, true
	}
	mage, female, ok := n.ComposedArchetype(id, playerMage, playerFemale)
	if !ok || mission < 0 {
		return 0, false
	}
	selector := int32(0)
	if female {
		selector++
	}
	if mage {
		selector += 2
	}
	return npcComposed + selector + 4*int32(mission/40), true
}

// CampaignCompanionForRow reverses the registry's companion construction for
// a saved Humans row. The player's own record and ordinary NPCs are excluded.
// Composed companions occupy the four shipped selector bands, server ids
// 26..41; direct records such as Brian retain their exact row. Ambiguity is
// absence, never a map-iteration-dependent identity.
func (n *NPCDefs) CampaignCompanionForRow(humans Collection, row int, playerMage, playerFemale bool) (int32, bool) {
	if n == nil || humans == nil || row <= 0 || row >= humans.Len() {
		return 0, false
	}
	var found int32
	for id, flags := range n.flags {
		if !flags.Has(NPCTokenHero) || flags.Has(NPCTokenMe) {
			continue
		}
		bands := 1
		if _, composed := n.composed[id]; composed {
			bands = 4
		}
		for band := 0; band < bands; band++ {
			server, ok := n.CampaignServerID(id, band*40, playerMage, playerFemale)
			if !ok || FindHumanByServerID(humans, server) != row {
				continue
			}
			if found != 0 && found != id {
				return 0, false
			}
			found = id
		}
	}
	return found, found != 0
}

// Mercenary reports the two price terms for a one-based mercenary type.
func (n *NPCDefs) Mercenary(id int32) (MercenaryTerms, bool) {
	if n == nil {
		return MercenaryTerms{}, false
	}
	v, ok := n.mercenary[id]
	return v, ok
}

// npcSubscript is the number a per-NPC section name carries, or not-a-section.
//
// The stem is matched case-insensitively because every other name comparison in
// this file's registry is, and the remainder must be a plain non-negative decimal
// number: `npc21x` and `npc+21` name no section this reads. The subscript is
// bounded by the int32 the placement record supplies it from, so a section
// numbered past that range is skipped rather than folded onto another.
func npcSubscript(name string) (int32, bool) {
	if len(name) <= len(npcSectionPrefix) || !nameEqualFold(name[:len(npcSectionPrefix)], npcSectionPrefix) {
		return 0, false
	}
	digits := name[len(npcSectionPrefix):]
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return 0, false
		}
	}
	v, err := strconv.ParseInt(digits, 10, 32)
	if err != nil {
		return 0, false
	}
	return int32(v), true
}

// ServerID is the definition id NPC subscript id names, and whether it names one.
//
// Three distinct absences answer the same way, and deliberately: a subscript with
// no section, a section with no definition id, and a section whose definition id
// is the composition sentinel. All three mean "this table cannot say which entry
// this placement is", which is the one thing a caller can act on; telling them
// apart would offer a caller a choice it has no rule to make.
func (n *NPCDefs) ServerID(id int32) (int32, bool) {
	if n == nil {
		return 0, false
	}
	def, ok := n.byID[id]
	return def, ok
}

// Len is how many subscripts the table answers, for a census that wants to say
// what it read without reading it again.
func (n *NPCDefs) Len() int {
	if n == nil {
		return 0
	}
	return len(n.byID)
}
