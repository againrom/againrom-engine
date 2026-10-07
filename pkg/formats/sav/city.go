package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
)

// CityCharacter is an immutable descriptor for one member of a lawful
// original no-world save's current group. Identity is a provenance-local
// handle: callers use it to map live characters to updates, but must not carry
// it across independently loaded save files.
type CityCharacter struct {
	Identity    uint32
	Name        string
	Hero        bool
	DefRow      uint8
	Stats       [UnitStatWords]uint16
	SkillLevels [CharacterSkillSlots]uint16
	SkillXP     [CharacterSkillSlots]uint32
	Experience  uint32
	// Source membership guards export of native records written before the
	// importer carried saved books. It is not reconstructed from a template.
	HasSpellbook bool
	KnownSpells  uint32
	Spells       []SavedSpell
}

// CityCharacterUpdate is the structurally mapped state for one member. Updates
// are keyed by Identity, never by roster position: original actor-list order
// is runtime state and RestoreParty is allowed to reorder it. This type names
// wire fields; it does not establish that an arbitrary combination is safe in
// ROM1. A production caller must enforce its evidence-backed update policy
// before calling Marshal (SAV-ORIGVALUE-399).
type CityCharacterUpdate struct {
	Identity    uint32
	Name        string
	Stats       [UnitStatWords]uint16
	SkillLevels [CharacterSkillSlots]uint16
	SkillXP     [CharacterSkillSlots]uint32
	Experience  uint32
	Human       *CityHumanFields
	// Returned applies the established kept-actor stage/reference reset. It
	// does not copy any completed World's placement, mover or effect graph.
	Returned bool
	Sales    []CityItemSale
	// A present graph atomically replaces this character's complete pack.
	Holdings *CityItemGraph
	// A present graph atomically replaces this character's weapon, shield and
	// armour references. nil preserves the base document's frozen equipment.
	Worn *CityEquipmentGraph
	// nil preserves the existing book. Only instance parameters may change.
	Spells *[]SavedSpell
}

// CityUpdate is the mapped state that the no-world writer can author. A nil
// Campaign preserves the detached semantic campaign imported with the city.
// Sales may only decrement explicitly identified source container objects.
// Holdings replaces a complete pack and Worn a complete equipped set; diary
// graphs remain an unsupported live update.
type CityUpdate struct {
	Label      []byte
	Money      uint32
	Characters []CityCharacterUpdate
	Campaign   *CampaignProjection
	// nil preserves the participant name independently of the hero's name.
	ParticipantName *string
	// nil preserves the source store. A present value writes all four cells,
	// including -1 for each unbound key (AI-QUICKSAVE-281).
	Shortcuts *CityShortcuts
	// nil preserves Player+58; a present raw percentage follows SAV-726.
	ManaReservePercent *uint32
	// nil preserves the document head's difficulty word; a present value is
	// the level 1..3 the native city writer stores in the same word.
	Difficulty *uint32
}

// CityShortcuts carries signed controller cell indices in F5-F8 order, not
// real spell IDs. AI-SPELLIDENT-286 defines the 24-cell identity mapping.
type CityShortcuts [4]int32

func (slots CityShortcuts) validate() error {
	for i, cell := range slots {
		if cell < -1 || cell > 23 {
			return fmt.Errorf("sav: city shortcut F%d index %d outside -1 or 0..23", i+5, cell)
		}
		if cell == -1 {
			continue
		}
		for j := 0; j < i; j++ {
			if slots[j] == cell {
				return fmt.Errorf("sav: city shortcut F%d duplicates F%d", i+5, j+5)
			}
		}
	}
	return nil
}

// CityProvenance is a detached semantic model of a lawful original no-world
// save. It contains neither a File nor a decoded/encoded whole-save byte slice.
// Every Marshal clones it, applies live fields, remints identities and rebuilds
// every physical layer independently.
type CityProvenance struct {
	version                uint32
	document               *cityDocument
	state                  *cityState
	campaign               *cityCampaign
	participantSourceIndex uint16
	roster                 []CityCharacter
	characterSourceIndex   map[uint32]uint16
	characterHasXP         map[uint32]bool
}

// CityProvenance decodes the semantic values required to author a no-world
// original-compatible save. World saves and incomplete/unsupported object,
// state or campaign graphs are rejected rather than partially carried.
func (f *File) CityProvenance() (*CityProvenance, error) {
	if f == nil {
		return nil, fmt.Errorf("sav: cannot derive city provenance from a nil File")
	}
	document, err := parseCityDocument(f.Body)
	if err != nil {
		return nil, err
	}
	if len(f.Store) == 0 {
		return nil, fmt.Errorf("sav: city save has no framed state store")
	}
	state, err := parseCityState(f.Store)
	if err != nil {
		return nil, err
	}
	if len(f.TailRest) == 0 {
		return nil, fmt.Errorf("sav: city save has no campaign record")
	}
	campaign, err := parseCityCampaign(f.TailRest)
	if err != nil {
		return nil, err
	}
	return newCityProvenance(f.Version, document, state, campaign)
}

func newCityProvenance(version uint32, document *cityDocument, state *cityState, campaign *cityCampaign) (*CityProvenance, error) {

	var participant *cityObject
	seenPlayers := make(map[*cityObject]bool)
	for _, candidate := range document.players {
		if candidate == nil || seenPlayers[candidate] {
			continue
		}
		seenPlayers[candidate] = true
		if candidate.player == nil || len(candidate.player.fixed) != 51 {
			return nil, fmt.Errorf("sav: city has an incomplete Player object")
		}
		if binary.LittleEndian.Uint32(candidate.player.fixed[15:19]) != 0 {
			continue
		}
		if participant != nil {
			return nil, fmt.Errorf("sav: city has more than one human participant")
		}
		participant = candidate
	}
	if participant == nil {
		return nil, fmt.Errorf("sav: city has no human participant")
	}
	playerIdentity := cityObjectIdentity(participant)
	if playerIdentity == 0 {
		return nil, fmt.Errorf("sav: city human participant has zero identity")
	}
	heroIdentity := binary.LittleEndian.Uint32(participant.player.fixed[43:47])
	if heroIdentity == 0 {
		return nil, fmt.Errorf("sav: city human participant has no primary-character identity")
	}

	p := &CityProvenance{
		version: version, document: document, state: state, campaign: campaign,
		participantSourceIndex: participant.sourceIndex,
		characterSourceIndex:   map[uint32]uint16{},
		characterHasXP:         map[uint32]bool{},
	}
	heroes := 0
	seenObjects := map[*cityObject]bool{}
	for _, group := range participant.player.groups {
		for _, actor := range group.actors {
			if actor == nil || actor.unit == nil || (actor.class != "Human" && actor.class != "Unit") {
				class := "<nil>"
				if actor != nil {
					class = actor.class
				}
				return nil, fmt.Errorf("sav: city participant group contains unsupported actor class %s", class)
			}
			if seenObjects[actor] {
				return nil, fmt.Errorf("sav: city participant group repeats archive object %d", actor.sourceIndex)
			}
			seenObjects[actor] = true
			identity := cityObjectIdentity(actor)
			if identity == 0 {
				return nil, fmt.Errorf("sav: city character %q has zero identity", actor.unit.name)
			}
			if _, duplicate := p.characterSourceIndex[identity]; duplicate {
				return nil, fmt.Errorf("sav: city character identity %#08x is duplicated", identity)
			}
			descriptor, err := cityCharacterDescriptor(actor, identity, identity == heroIdentity)
			if err != nil {
				return nil, err
			}
			if descriptor.Hero {
				heroes++
			}
			p.roster = append(p.roster, descriptor)
			p.characterSourceIndex[identity] = actor.sourceIndex
			p.characterHasXP[identity] = len(actor.unit.xp) == 24
		}
	}
	if len(p.roster) == 0 {
		return nil, fmt.Errorf("sav: city human participant has an empty roster")
	}
	if heroes != 1 {
		return nil, fmt.Errorf("sav: city primary-character identity matches %d roster members, want 1", heroes)
	}
	// This validates every known identity relation while mutating only a deep
	// clone. A dangling source pointer is not deferred until the user saves.
	// This is the base document as city.ags itself carries it: tolerate a
	// stale owner token an earlier writer left behind.
	if err := remintCityIdentities(cloneCityDocument(document), true); err != nil {
		return nil, err
	}
	return p, nil
}

func cityCharacterDescriptor(actor *cityObject, identity uint32, hero bool) (CityCharacter, error) {
	u := actor.unit
	out := CityCharacter{Identity: identity, Name: u.name, Hero: hero, DefRow: u.token[16], HasSpellbook: u.spellbookFlag == 1}
	for i := range out.Stats {
		out.Stats[i] = binary.LittleEndian.Uint16(u.scalar2[2*i:])
	}
	for i := range out.SkillLevels {
		out.SkillLevels[i] = binary.LittleEndian.Uint16(u.rawA6[2+2*i:])
		if len(u.xp) == 24 {
			out.SkillXP[i] = binary.LittleEndian.Uint32(u.xp[4*i:])
		}
	}
	out.Experience = binary.LittleEndian.Uint32(u.scalar2[35:39])
	for slot, spell := range u.spells {
		if spell == nil {
			continue
		}
		if !out.HasSpellbook || spell.class != "Spell" || spell.spell == nil || len(spell.spell.fields) != 9 {
			return CityCharacter{}, fmt.Errorf("%w: city character %#x slot %d has no valid Spell", ErrSpellbook, identity, slot+1)
		}
		id := spell.spell.fields[0]
		if id == 0 || id > 28 || int(id) != slot+1 {
			return CityCharacter{}, fmt.Errorf("%w: city character %#x slot %d has spell ID %d", ErrSpellbook, identity, slot+1, id)
		}
		out.KnownSpells |= uint32(1) << id
		out.Spells = append(out.Spells, SavedSpell{Slot: slot + 1, ID: id,
			Range: spell.spell.fields[1], Defensive: spell.spell.fields[2],
			ManaCost:     binary.LittleEndian.Uint16(spell.spell.fields[3:5]),
			ArchiveIndex: spell.sourceIndex, Key: cityObjectIdentity(spell)})
	}
	return out, nil
}

// Roster returns a copy in the original actor-list order. The caller may sort
// or mutate it without changing the provenance.
func (p *CityProvenance) Roster() []CityCharacter {
	if p == nil {
		return nil
	}
	out := append([]CityCharacter(nil), p.roster...)
	for i := range out {
		out[i].Spells = append([]SavedSpell(nil), out[i].Spells...)
	}
	return out
}

// ParticipantName is the human Player's saved join name.
func (p *CityProvenance) ParticipantName() string {
	if p == nil || p.document == nil {
		return ""
	}
	if player := p.document.objects[p.participantSourceIndex]; player != nil && player.player != nil {
		return player.player.name
	}
	return ""
}

func cityObjectLocalOrdinals(document *cityDocument) map[uint16]uint16 {
	indices := make([]int, 0, len(document.objects))
	for index := range document.objects {
		indices = append(indices, int(index))
	}
	sort.Ints(indices)
	out := make(map[uint16]uint16, len(indices))
	for i, index := range indices {
		out[uint16(index)] = uint16(i + 1)
	}
	return out
}

func remapCityCurrentState(source CityData, sourceObjectOrdinals map[uint16]uint16, document *cityDocument, state *cityState, campaign *cityCampaign, version uint32) error {
	if _, present, err := NativeActions(documentStateData(source.State)); err != nil || !present {
		return err
	}
	working := (&CityProvenance{version: version, document: document, state: state, campaign: campaign}).Data()
	permutation := make([]uint16, len(source.Objects)+1)
	newObjectOrdinals := cityObjectLocalOrdinals(document)
	for newArchiveIndex, object := range document.objects {
		oldIndex := sourceObjectOrdinals[object.sourceIndex]
		newIndex := newObjectOrdinals[newArchiveIndex]
		if oldIndex == 0 || int(oldIndex) > len(source.Objects) || newIndex == 0 {
			continue
		}
		if permutation[oldIndex] != 0 {
			return fmt.Errorf("sav: city current-state source object %d has duplicate replacement", oldIndex)
		}
		permutation[oldIndex] = newIndex
	}
	docState := documentStateData(working.State)
	if err := remapNativeActionObjectsForCity(&docState, permutation); err != nil {
		return err
	}
	working.State.RootKind, working.State.DirectoryRecords, working.State.ValueRecords = docState.RootKind, docState.DirectoryRecords, docState.ValueRecords
	next, err := cityStateFromData(working.State, CityDataVersion)
	if err != nil {
		return err
	}
	*state = *next
	return nil
}

func documentStateData(state CityStateData) DocumentStateData {
	return DocumentStateData{RootKind: state.RootKind, DirectoryRecords: state.DirectoryRecords, ValueRecords: state.ValueRecords}
}

// Marshal independently authors a structurally exact Asg& city save from
// semantic provenance and mapped values. It requires the complete original
// roster by identity; no character may be silently retained, dropped or
// replaced. Structural closure is not evidence that invented Human values are
// safe in ROM1; the game route therefore supplies the source-faithful baseline.
func (p *CityProvenance) Marshal(update CityUpdate) ([]byte, error) {
	if p == nil || p.document == nil || p.campaign == nil || p.state == nil {
		return nil, fmt.Errorf("sav: city provenance is incomplete")
	}
	if len(update.Label)+1 > labelLen {
		return nil, fmt.Errorf("sav: city label of %d bytes does not fit the %d-byte region", len(update.Label), labelLen)
	}
	if bytes.IndexByte(update.Label, 0) >= 0 {
		return nil, fmt.Errorf("sav: city label contains an embedded NUL")
	}
	if update.ParticipantName != nil && (*update.ParticipantName == "" || bytes.IndexByte([]byte(*update.ParticipantName), 0) >= 0) {
		return nil, fmt.Errorf("sav: city participant name is empty or contains NUL")
	}
	if update.Shortcuts != nil {
		if err := update.Shortcuts.validate(); err != nil {
			return nil, err
		}
	}
	if update.Difficulty != nil && (*update.Difficulty < 1 || *update.Difficulty > 3) {
		return nil, fmt.Errorf("sav: city difficulty %d outside 1..3", *update.Difficulty)
	}
	if len(update.Characters) != len(p.roster) {
		return nil, fmt.Errorf("sav: city update has %d characters, provenance requires %d", len(update.Characters), len(p.roster))
	}
	// Native current-state rows address the detached document's local object
	// table. A source edit can retire one physical node and shift every later
	// archive index, so retain the supplement only after joining those rows by
	// their preserved source archive ordinals.
	sourceData := p.Data()
	sourceObjectOrdinals := cityObjectLocalOrdinals(p.document)
	updates := make(map[uint32]CityCharacterUpdate, len(update.Characters))
	for _, character := range update.Characters {
		if character.Identity == 0 {
			return nil, fmt.Errorf("sav: city update contains zero character identity")
		}
		if bytes.IndexByte([]byte(character.Name), 0) >= 0 {
			return nil, fmt.Errorf("sav: city character %#08x name contains NUL", character.Identity)
		}
		if _, known := p.characterSourceIndex[character.Identity]; !known {
			return nil, fmt.Errorf("sav: city update contains unknown character identity %#08x", character.Identity)
		}
		if _, duplicate := updates[character.Identity]; duplicate {
			return nil, fmt.Errorf("sav: city update repeats character identity %#08x", character.Identity)
		}
		updates[character.Identity] = character
	}

	document := cloneCityDocument(p.document)
	if update.Difficulty != nil {
		document.head[12] = *update.Difficulty
	}
	spellUpdates, err := p.spellUpdates(update.Characters)
	if err != nil {
		return nil, err
	}
	for index, fields := range spellUpdates {
		copy(document.objects[index].spell.fields[:5], fields[:])
	}
	state := cloneCityState(p.state)
	if update.Shortcuts != nil {
		value := state.values["/SpellBook/Shortcuts"]
		value.bytes = make([]byte, 16)
		for i, cell := range update.Shortcuts {
			binary.LittleEndian.PutUint32(value.bytes[4*i:], uint32(cell))
		}
		state.values["/SpellBook/Shortcuts"] = value
	}
	campaign := cloneCityCampaign(p.campaign)
	participant := document.objects[p.participantSourceIndex]
	if participant == nil || participant.player == nil {
		return nil, fmt.Errorf("sav: city participant provenance no longer resolves")
	}
	if update.ParticipantName != nil {
		participant.player.name = *update.ParticipantName
	}
	binary.LittleEndian.PutUint32(participant.player.fixed[21:25], update.Money^obfuscator)
	if update.ManaReservePercent != nil {
		binary.LittleEndian.PutUint32(participant.player.fixed[39:43], *update.ManaReservePercent)
	}
	heroName := ""
	var added []*cityObject
	for _, descriptor := range p.roster {
		live, ok := updates[descriptor.Identity]
		if !ok {
			return nil, fmt.Errorf("sav: city update omits character identity %#08x", descriptor.Identity)
		}
		actor := document.objects[p.characterSourceIndex[descriptor.Identity]]
		if actor == nil || actor.unit == nil {
			return nil, fmt.Errorf("sav: city character provenance %#08x no longer resolves", descriptor.Identity)
		}
		if err := applyCityItemSales(document, actor, live.Sales); err != nil {
			return nil, err
		}
		if len(live.Sales) != 0 && live.Holdings != nil {
			return nil, fmt.Errorf("sav: current pack cannot combine with source sales")
		}
		objects, err := applyCityCharacterGraphs(document, actor, live)
		if err != nil {
			return nil, err
		}
		added = append(added, objects...)
		u := actor.unit
		if live.Returned {
			if len(u.effects) != 0 {
				return nil, fmt.Errorf("sav: source city Human has unsupported attached effects")
			}
			u.scalar2[46] = 0        // Stage (+0x13c), PARTY-ENDCULL-026.
			clear(u.scalarTail[:16]) // +0x5c, +0x64, +0x44, +0x40.
			u.reference68 = nil
		}
		if live.Human != nil {
			if actor.class != "Human" {
				return nil, fmt.Errorf("sav: city Unit %#08x cannot receive Human fields", descriptor.Identity)
			}
			applyCityHumanFields(u, *live.Human)
		}
		u.name = live.Name
		for i, value := range live.Stats {
			binary.LittleEndian.PutUint16(u.scalar2[2*i:], value)
		}
		for i, value := range live.SkillLevels {
			binary.LittleEndian.PutUint16(u.rawA6[2+2*i:], value)
		}
		if p.characterHasXP[descriptor.Identity] {
			for i, value := range live.SkillXP {
				binary.LittleEndian.PutUint32(u.xp[4*i:], value)
			}
		} else if live.SkillXP != ([CharacterSkillSlots]uint32{}) {
			return nil, fmt.Errorf("sav: city Unit %#08x cannot serialize skill XP", descriptor.Identity)
		}
		binary.LittleEndian.PutUint32(u.scalar2[35:39], live.Experience)
		if descriptor.Hero {
			heroName = live.Name
		}
	}
	if heroName == "" {
		return nil, fmt.Errorf("sav: city update gives the primary character an empty name")
	}
	nameState := state.values["/Character/Name"]
	nameState.bytes = append(append([]byte(nil), heroName...), 0)
	state.values["/Character/Name"] = nameState
	if update.Campaign != nil {
		if err := applyCityCampaignProjection(campaign, *update.Campaign); err != nil {
			return nil, err
		}
	}
	// Whole sales can remove an Item and its otherwise unreferenced children.
	// They must not consume reminted keys: the next reader sees only the live
	// archive, so minting through removed objects changes surviving identities
	// again on SAV -> AGS -> SAV.
	for _, live := range update.Characters {
		if len(live.Sales) != 0 || live.Holdings != nil || live.Worn != nil {
			if err := pruneCitySoldObjects(document); err != nil {
				return nil, err
			}
			break
		}
	}
	clearUnresolvedCityOwners(document, added)
	// This document was just built from this package's own base document plus
	// live Holdings we validated on the way in: an owner reference that still
	// fails to mint here is our own defect, not the base document's, and is
	// refused rather than guessed at.
	if err := remintCityIdentities(document, false); err != nil {
		return nil, err
	}
	if err := remapCityCurrentState(sourceData, sourceObjectOrdinals, document, state, campaign, p.version); err != nil {
		return nil, err
	}
	body, err := serializeCityDocument(document)
	if err != nil {
		return nil, err
	}
	store, err := serializeCityState(state)
	if err != nil {
		return nil, err
	}
	campaignBytes, err := serializeCityCampaign(campaign)
	if err != nil {
		return nil, err
	}
	blob := Compress(body)
	out := make([]byte, headerLen, headerLen+len(blob)+labelLen+len(store)+len(campaignBytes))
	copy(out, Magic)
	binary.LittleEndian.PutUint32(out[4:8], uint32(headerLen+len(blob)))
	binary.LittleEndian.PutUint32(out[8:12], p.version)
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(blob)))
	out = append(out, blob...)
	label := make([]byte, labelLen)
	copy(label, update.Label)
	out = append(out, label...)
	out = append(out, store...)
	out = append(out, campaignBytes...)

	// The ordinary production reader is the final structural gate. It checks
	// the container, codec, head, Players, state extent and campaign framing.
	opened, err := Open(out)
	if err != nil {
		return nil, fmt.Errorf("sav: authored city failed full reader: %w", err)
	}
	current, err := opened.CityProvenance()
	if err != nil {
		return nil, fmt.Errorf("sav: authored city failed semantic reader: %w", err)
	}
	for _, live := range update.Characters {
		if live.Holdings == nil {
			continue
		}
		// A valid individual pack can exceed the combined city's object or
		// expanded loadout budgets. Admit the same semantic DTO and source
		// import that cold LOAD needs before publishing any output bytes.
		if _, err := CityFromData(current.Data()); err != nil {
			return nil, fmt.Errorf("sav: current city exceeds persistence bounds: %w", err)
		}
		if _, err := current.SourceParty(); err != nil {
			return nil, fmt.Errorf("sav: current city exceeds source import bounds: %w", err)
		}
		break
	}
	return out, nil
}
