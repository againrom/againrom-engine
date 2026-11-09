package game

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
)

// savedForm is how one game writes one document kind: the records its
// original reads, built from the engine's own document. The producer
// applies it (ExportCurrentSave); a form whose records are the engine's own
// is the identity form.
type savedForm struct {
	// records is the SAV profile whose record programmes and form rows the
	// records take (sav.DocumentInProfile).
	records *sav.Profile
	// holders are the fields the profile's records hold that the engine's
	// records keep elsewhere, each with its source, written in this order
	// before the form rows.
	holders []savedFormHolder
	// keeps is what the profile's records hold that the World keeps; nil
	// for a document with no World.
	keeps func(s Snapshot, doc *sav.DocumentData) (sav.ProfileKeeps, error)
	// tail writes the campaign tail after the state store; nil writes none.
	tail func(s Snapshot) (*sav.SecondTail, error)
	// leaf is that the engine's own document travels in the AgainromForm
	// leaf, which a LOAD reads (DIV-4304).
	leaf bool
}

// savedForms is a game's saved form of each document kind: missionRecord
// and townRecord.
type savedForms map[campaignRecordKind]savedForm

// firstSavedForms: the first game's original reads the engine's own records.
var firstSavedForms = savedForms{
	missionRecord: {records: sav.FirstGame},
	townRecord:    {records: sav.FirstGame},
}

// secondSavedForms: the second game's original reads its own records, and
// the engine's own document travels beside them (DIV-4304, DIV-4350). A town
// SAVE has no World (exportable refuses one), so its records keep nothing
// from it.
var secondSavedForms = savedForms{
	missionRecord: {records: sav.SecondGame, holders: secondFormHolders, keeps: secondFormKeeps, tail: secondCampaignTail, leaf: true},
	townRecord:    {records: sav.SecondGame, holders: secondFormHolders, tail: secondCampaignTail, leaf: true},
}

// savedFormKind is the document kind a snapshot writes.
func savedFormKind(s Snapshot) campaignRecordKind {
	if s.Mission == 0 {
		return townRecord
	}
	return missionRecord
}

// write is doc in the form's records, and whether the engine's own document
// travels in the form's leaf: false, and doc, for the identity form.
func (form savedForm) write(f *FrontEnd, s Snapshot, doc sav.DocumentData) (sav.DocumentData, bool, error) {
	if form.records == nil {
		return sav.DocumentData{}, false, fmt.Errorf("no saved form for the document kind")
	}
	if len(form.holders) != 0 {
		if err := writeSecondTownHolders(&doc, form.holders, s, f.Table); err != nil {
			return sav.DocumentData{}, false, err
		}
	}
	var keeps sav.ProfileKeeps
	if form.keeps != nil {
		var err error
		if keeps, err = form.keeps(s, &doc); err != nil {
			return sav.DocumentData{}, false, err
		}
	}
	out, formed, err := sav.DocumentInProfile(doc, form.records, keeps)
	if err != nil {
		return sav.DocumentData{}, false, err
	}
	if formed != form.leaf {
		return sav.DocumentData{}, false, fmt.Errorf("the saved form's records and its engine leaf disagree")
	}
	if form.tail != nil {
		if out.SecondTail, err = form.tail(s); err != nil {
			return sav.DocumentData{}, false, err
		}
	}
	return out, form.leaf, nil
}

// secondFormKeeps is what the second game's records hold that a mission's
// World keeps beside the engine's records: the word of each structure of a
// Pointer kind, its action slot, by its Token key, and the fog's owner gate
// for the session matrix's sight, and each member order's blow record and
// stamp by its record's key (secondCommandStrikes).
func secondFormKeeps(s Snapshot, doc *sav.DocumentData) (sav.ProfileKeeps, error) {
	if len(s.World) == 0 {
		return sav.ProfileKeeps{}, nil
	}
	var w sim.World
	if err := w.UnmarshalBinary(s.World); err != nil {
		return sav.ProfileKeeps{}, err
	}
	keeps := sav.ProfileKeeps{Sees: w.RevealsOwner}
	casters, err := virtualCasterRecords(&w, doc)
	if err != nil {
		return sav.ProfileKeeps{}, err
	}
	keeps.Extra = casters
	source, _, _ := w.SavedStructures()
	live := w.Structures()
	for _, st := range source {
		at := slices.IndexFunc(live, func(l sim.Structure) bool { return l.ID == st.ID })
		if at < 0 {
			continue
		}
		if word, ok := live[at].PointerWord(); ok {
			if keeps.Pointers == nil {
				keeps.Pointers = map[uint32]uint32{}
			}
			keeps.Pointers[st.SourceKey] = word
		}
	}
	keeps.Commands, err = secondCommandStrikes(&w, doc)
	return keeps, err
}

// secondCommandStrikes is each member order's blow record and stamp in w,
// by the Token Identity of its actor's record in doc (R2-ENGINE-845,
// R2-ENGINE-846): +0x16 as the order holds it; +0x94 the
// striker's record key, 0 when doc holds no record of it.
func secondCommandStrikes(w *sim.World, doc *sav.DocumentData) (map[uint32]sav.CommandStrike, error) {
	_, orders, _ := w.SavedGroups()
	if !slices.ContainsFunc(orders, func(o sim.SavedActorOrder) bool { return o.Strike != (sim.SavedStrike{}) }) {
		return nil, nil
	}
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		return nil, err
	}
	key := func(id sim.EntityID) uint32 {
		for _, b := range a.Bindings {
			if b.ID == id && !b.Structure && !b.Missing && b.Object != 0 && int(b.Object) <= len(doc.Objects) {
				v, _ := savedStructureValue(&doc.Objects[b.Object-1], "Identity")
				return v
			}
		}
		return 0
	}
	out := map[uint32]sav.CommandStrike{}
	for _, o := range orders {
		k := o.Strike
		if k == (sim.SavedStrike{}) {
			continue
		}
		victim := key(o.Entity)
		if victim == 0 {
			continue
		}
		c := sav.CommandStrike{Stamp: k.CommandStamp(), Clock: k.Clock, Away: k.Away}
		if k.Attacker != 0 {
			c.Striker = key(k.Attacker)
		}
		out[victim] = c
	}
	return out, nil
}

// savedFormHolder is a field a form's records hold that the engine's
// records keep elsewhere, and the source the form writes it from.
type savedFormHolder struct {
	// class and field name the field of each record of the class.
	class, field string
	site         holderSite
	// value is the field's value in the record at index i of the holding's
	// document; false leaves the field as it is.
	value func(h *formHolding, i int) (uint32, bool)
	// records writes a holderRecords field's records.
	records func(h *formHolding) error
}

type holderSite uint8

const (
	// holderFirstWord is a raw block's first DWORD; a record without the
	// block refuses.
	holderFirstWord holderSite = iota
	// holderValue is a value every record of the class holds.
	holderValue
	// holderPresentValue is a value written where the record holds it.
	holderPresentValue
	// holderRecords is a reference slot the source fills with records, or
	// the state leaves it writes.
	holderRecords
)

// formHolding is what the holders' sources read: the document being
// formed, the snapshot, the installed table and the hero's record. keys is
// each +4 a holder rewrote, the form's key by the engine's.
type formHolding struct {
	doc     *sav.DocumentData
	s       Snapshot
	table   *mapload.Table
	hero    uint16
	rewards map[int]bool
	keys    map[uint32]uint32
}

// secondFormHolders are the fields the second game's records hold that the
// engine's records keep elsewhere.
var secondFormHolders = []savedFormHolder{
	{class: "Unit", field: "U148", site: holderValue, value: secondActorTemplateKey},
	{class: "Human", field: "U148", site: holderValue, value: secondActorTemplateKey},
	// R2-ENGINE-652, R2-ENGINE-900: a creature whose Units row holds a
	// knownSpells mask is a mage; the engine keeps that in the row (DIV-4560).
	{class: "Unit", field: "U4C", site: holderValue, value: secondUnitMageFlags},
	// R2-ENGINE-319, R2-ENGINE-320: the experience block opens with the main
	// skill.
	{class: "Human", field: "H1CC", site: holderFirstWord, value: secondMainSkillHeld},
	// R2-ENGINE-410: the bag constructor's insert index, which appends.
	{class: "Human", field: "Inventory1C", site: holderPresentValue, value: holderConstant(0x7fff)},
	// DIV-4153: the campaign's bag in the hero's Inventory.
	{class: "Human", field: "Inventory", site: holderRecords, records: secondCampaignBag},
	// A Player's obfuscated F48 holds 0, the originals' value (DIV-4305);
	// an Effect the constructor's +0xe = 0 (R2-ENGINE-403).
	{class: "Player", field: "F48", site: holderValue, value: holderConstant(0)},
	{class: "Effect", field: "T0E", site: holderValue, value: holderConstant(0)},
	// R2-ENGINE-403: an Item holds its row at +0xc, from which LOAD
	// rebuilds its descriptor; a magic item's ID names that row.
	{class: "Item", field: "T0C", site: holderValue, value: secondMagicItemRow},
	// R2-ENGINE-857: a mission's structure holds its map id plus 0x6000 at
	// +4, the key the original finds a clicked structure by. A town holds
	// no map structure.
	{class: "Building", field: "RuntimeID", site: holderValue, value: secondStructureKey},
	{class: "Outpost", field: "RuntimeID", site: holderValue, value: secondStructureKey},
	{class: "Tavern", field: "RuntimeID", site: holderValue, value: secondStructureKey},
	{class: "Shop", field: "RuntimeID", site: holderValue, value: secondStructureKey},
	// R2-ENGINE-156, R2-ENGINE-950: a mission's map-built creature or person
	// holds its map id plus 0x6000 at +4, whoever owns it, the key the
	// original's client builds its unit object under from the map
	// (R2-ENGINE-615). The hero and the members carried into the mission
	// keep their key (R2-SESSION-360).
	{class: "Unit", field: "RuntimeID", site: holderValue, value: secondMapActorKey},
	{class: "Human", field: "RuntimeID", site: holderValue, value: secondMapActorKey},
	// R2-ENGINE-402: a bag's weight is the sum of its Items' unit weight
	// times quantity.
	{class: "Human", field: "Inventory20", site: holderPresentValue, value: secondBagWeight},
	// R2-ENGINE-614, R2-SESSION-245: the selection and a projectile's
	// target name an object by its +4.
	{field: "RuntimeID", site: holderRecords, records: secondKeyEdges},
}

// secondStructureKey is a mission structure's +4: 0x6000 plus its map id, the
// record's +8 (R2-ENGINE-857).
func secondStructureKey(h *formHolding, i int) (uint32, bool) {
	if h.s.Mission == 0 {
		return 0, false
	}
	id, err := savedStructureValue(&h.doc.Objects[i], "T08")
	if err != nil {
		return 0, false
	}
	return h.rekeyed(i, 0x6000+id)
}

// secondMapActorKey is a mission actor's +4: 0x6000 plus its map id, the
// record's +8, for an actor the map placed (+8 from 1 to 0x9fff, so that
// the key fits the client's word), whoever owns it. The hero and a member
// carried into the mission, whose map id a file's record gave him, were
// not built from this map and keep their key.
func secondMapActorKey(h *formHolding, i int) (uint32, bool) {
	id, err := savedStructureValue(&h.doc.Objects[i], "T08")
	if h.s.Mission == 0 || err != nil || id == 0 || id > 0x9fff || uint16(i+1) == h.hero {
		return 0, false
	}
	for n := range h.s.CurrentPartyIDs {
		if n < len(h.s.Party) && h.s.Party[n].Saved != nil && uint32(h.s.Party[n].Saved.MapUnitID) == id {
			return 0, false
		}
	}
	return h.rekeyed(i, 0x6000+id)
}

// rekeyed is key, record i's +4 in the form, noted by the record's engine
// key for the leaves that name the record (secondKeyEdges).
func (h *formHolding) rekeyed(i int, key uint32) (uint32, bool) {
	if was, err := savedStructureValue(&h.doc.Objects[i], "RuntimeID"); err == nil && was != 0 && was != key {
		if h.keys == nil {
			h.keys = map[uint32]uint32{}
		}
		h.keys[was] = key
	}
	return key, true
}

// secondKeyEdges writes the form's key in the state leaves that name a
// rekeyed record by its engine key: each entry of the selection and each
// projectile's actiontarget.
func secondKeyEdges(h *formHolding) error {
	if len(h.keys) == 0 {
		return nil
	}
	st := &h.doc.State
	rekey := func(key uint32) uint32 {
		if to, found := h.keys[key]; found {
			return to
		}
		return key
	}
	if _, err := patchStateLeaf(st.RootKind, st.DirectoryRecords, st.ValueRecords, "/Objects/Selection", func(leaf *sav.CityStateValueData) error {
		for n := 0; n+4 <= len(leaf.Bytes); n += 4 {
			binary.LittleEndian.PutUint32(leaf.Bytes[n:], rekey(binary.LittleEndian.Uint32(leaf.Bytes[n:])))
		}
		return nil
	}); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, record := range st.ValueRecords {
		root, _ := splitStatePath(record.Path)
		if len(root) < 3 || seen[root] || !stateRootMatches(st.RootKind, record.Path, "Prj"+root[3:]) {
			continue
		}
		seen[root] = true
		if _, err := patchStateLeaf(st.RootKind, st.DirectoryRecords, st.ValueRecords, "/Prj"+root[3:]+"/actiontarget", func(leaf *sav.CityStateValueData) error {
			leaf.Int32 = int32(rekey(uint32(leaf.Int32)))
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// secondUnitMageFlags is a creature's +0x4c with the mage bit, 4, set when
// its Units row's knownSpells mask makes it a mage (R2-ENGINE-652); without
// the bit the original's LOAD keeps it from casting its book and casts its
// weapon Spell only from a strike (R2-ENGINE-901, R2-ENGINE-902). Every other
// bit, and the byte of any other creature, stays as the record holds it.
func secondUnitMageFlags(h *formHolding, i int) (uint32, bool) {
	r := &h.doc.Objects[i]
	flags, err := savedStructureValue(r, "U4C")
	if err != nil || h.table == nil {
		return flags, false
	}
	row, err := savedStructureValue(r, "T0C")
	if err != nil || !h.table.UnitMaskMage(int(row)) {
		return flags, false
	}
	return flags | 4, true
}

func secondActorTemplateKey(h *formHolding, i int) (uint32, bool) {
	r := &h.doc.Objects[i]
	packed, err := savedStructureValue(r, "U148")
	if err != nil {
		return packed, false
	}
	if marked, present := h.rewards[i]; present {
		packed &= 0x00ffffff
		if marked {
			packed |= 1 << 24
		}
	}
	if uint16(packed) != 0 || h.table == nil {
		return packed, true
	}
	if r.Class == "Human" && uint16(i+1) == h.hero {
		if id, ok := mapload.PartyIdentity(nativeCityHeroOf(h.s.Party), h.table); ok {
			return packed | uint32(uint16(id)), true
		}
	}
	rows, serverID := h.table.Units, data.UnitServerID
	if r.Class == "Human" {
		rows, serverID = h.table.Humans, data.HumanServerID
	}
	row, err := savedStructureValue(r, "T0C")
	if err != nil || rows == nil || row == 0 || uint64(row) >= uint64(rows.Len()) {
		return packed, true
	}
	key, ok := serverID(rows.EntryParams(int(row)))
	if !ok || key <= 0 {
		return packed, true
	}
	word := uint16(key)
	if r.Class == "Human" {
		word = mapload.PersonNPCWord(key)
	}
	return packed | uint32(word), true
}

// writeSecondTownHolders writes each holder's field in doc's records from
// its source, in the holders' order.
func writeSecondTownHolders(doc *sav.DocumentData, holders []savedFormHolder, s Snapshot, table *mapload.Table) error {
	hero, err := secondTownHeroRecord(*doc)
	if err != nil {
		return err
	}
	h := &formHolding{doc: doc, s: s, table: table, hero: hero}
	actions, err := readCurrentActions(doc)
	if err != nil {
		return err
	}
	if actions != nil {
		h.rewards = make(map[int]bool)
		for _, b := range actions.Bindings {
			if v, present := actions.Values[b.ID]; present && !b.Structure && !b.Missing && b.Object != 0 {
				h.rewards[int(b.Object)-1] = v.SuppressRewards
			}
		}
	}
	for _, holder := range holders {
		if holder.site == holderRecords {
			if err := holder.records(h); err != nil {
				return err
			}
			continue
		}
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class != holder.class {
				continue
			}
			if _, err := savedStructureValue(r, holder.field); holder.site == holderPresentValue && err != nil {
				continue
			}
			v, ok := holder.value(h, i)
			if !ok {
				continue
			}
			switch holder.site {
			case holderFirstWord:
				at := slices.IndexFunc(r.Raw, func(b sav.DocumentRawData) bool { return b.Name == holder.field })
				if at < 0 || len(r.Raw[at].Bytes) < 4 {
					return fmt.Errorf("%s record %d has no %s block", r.Class, i+1, holder.field)
				}
				block := slices.Clone(r.Raw[at].Bytes)
				binary.LittleEndian.PutUint32(block, v)
				mustSetRaw(r, holder.field, block)
			case holderValue, holderPresentValue:
				if err := savedStructureSetValue(r, holder.field, v); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func holderConstant(v uint32) func(*formHolding, int) (uint32, bool) {
	return func(*formHolding, int) (uint32, bool) { return v, true }
}

// secondMainSkillHeld is a Human's main skill: the hero's from his party
// member; one built from a `_Hero` or `Start_` template, such as a
// companion's identity row, his largest school (R2-ENGINE-319); one built
// from another the Humanoid initializer's 0 (DIV-4305).
func secondMainSkillHeld(h *formHolding, i int) (uint32, bool) {
	if uint16(i+1) == h.hero {
		return secondMainSkill(nativeCityHeroOf(h.s.Party).Hero), true
	}
	if hero, ok := secondTemplateHero(&h.doc.Objects[i], h.table); ok {
		return secondMainSkill(hero), true
	}
	return 0, true
}

// secondCampaignBag writes the items the campaign's party carries as the
// hero's bag.
func secondCampaignBag(h *formHolding) error {
	bag := savedBagView(h.s.second.Party)
	if len(bag.CarriedItems) == 0 {
		return nil
	}
	return writeSecondTownBag(h.doc, h.hero, bag, h.table)
}

// secondMagicItemRow is the row a magic Item's ID names; false for an Item
// that is not magic.
func secondMagicItemRow(h *formHolding, i int) (uint32, bool) {
	code, err := savedStructureValue(&h.doc.Objects[i], "F40")
	if err != nil {
		return 0, false
	}
	row, ok := secondshop.MagicRow(uint16(code))
	return uint32(row), ok
}

// secondBagWeight is the weight of a Human's bag: its Inventory's Items.
func secondBagWeight(h *formHolding, i int) (uint32, bool) {
	var weight uint32
	for _, slot := range h.doc.Objects[i].RefSlots {
		if slot.Name != "Inventory" {
			continue
		}
		for _, ref := range slot.Objects {
			if ref == 0 || int(ref) > len(h.doc.Objects) {
				continue
			}
			unit, errW := savedStructureValue(&h.doc.Objects[ref-1], "F4A")
			count, errC := savedStructureValue(&h.doc.Objects[ref-1], "F42")
			if errW == nil && errC == nil {
				weight += uint32(int16(unit)) * count
			}
		}
	}
	return weight, true
}

// writeSecondTownBag writes the campaign's bag as Item records after the
// hero record's Inventory, each owned by the hero's Player (R2-ENGINE-403).
func writeSecondTownBag(doc *sav.DocumentData, hero uint16, bag mapload.PartyMember, table *mapload.Table) error {
	projection, err := newCityObjectProjection(nil, []mapload.PartyMember{bag}, table)
	if err != nil {
		return err
	}
	keys, err := reserveCurrentCityDocumentKeys(*doc, projection.graph, 65536)
	if err != nil {
		return err
	}
	b := generatedDocumentBuilder{doc: *doc, table: table, reservedKeys: keys}
	// The actions the emit records describe the bag's records; the engine's
	// own document carries the bag, so they are not written.
	var actions currentActionData
	if err := projection.emit(&b, &actions); err != nil {
		return err
	}
	pack, err := projection.appendPack(&b, hero, bag.ID)
	if err != nil {
		return err
	}
	owner, err := savedStructureValue(&b.doc.Objects[hero-1], "Reference")
	if err != nil {
		return err
	}
	for _, ref := range pack {
		if ref != 0 {
			mustSetValue(&b.doc.Objects[ref-1], "Reference", owner)
		}
	}
	*doc, _, err = sav.ReindexDocumentData(b.doc)
	return err
}

// secondTownHeroRecord is the record of the Human the first Player names as
// its hero.
func secondTownHeroRecord(doc sav.DocumentData) (uint16, error) {
	if len(doc.Players) == 0 || doc.Players[0] == 0 || int(doc.Players[0]) > len(doc.Objects) {
		return 0, fmt.Errorf("second-game town records name no Player")
	}
	hero, err := savedStructureValue(&doc.Objects[doc.Players[0]-1], "Hero")
	if err != nil {
		return 0, err
	}
	for i := range doc.Objects {
		if doc.Objects[i].Class != "Human" {
			continue
		}
		if id, err := savedStructureValue(&doc.Objects[i], "Identity"); err == nil && id == hero {
			return uint16(i + 1), nil
		}
	}
	return 0, fmt.Errorf("second-game town records hold no hero")
}

// secondMainSkill is the main skill of a hero built from a `_Hero` or
// `Start_` template: the school whose skill is largest, the first of equal
// ones (R2-ENGINE-319, R2-ENGINE-320; DIV-4305).
func secondMainSkill(h data.Hero) uint32 {
	best := data.SkillBlade
	for i := data.SkillBlade + 1; i < data.SkillSlots; i++ {
		if h.Skill[i] > h.Skill[best] {
			best = i
		}
	}
	return uint32(best)
}

// secondTemplateHero is the skill levels of a Human record whose Humans row
// is a `_Hero` or `Start_` template (R2-ENGINE-319), read from its own skill
// block.
func secondTemplateHero(r *sav.DocumentRecordData, table *mapload.Table) (data.Hero, bool) {
	row, err := savedStructureValue(r, "T0C")
	if err != nil || table == nil || table.Humans == nil || row == 0 || int(row) >= table.Humans.Len() {
		return data.Hero{}, false
	}
	if name := table.Humans.EntryName(int(row)); !strings.Contains(name, "_Hero") && !strings.Contains(name, "Start_") {
		return data.Hero{}, false
	}
	at := slices.IndexFunc(r.Raw, func(v sav.DocumentRawData) bool { return v.Name == "UA6" })
	if at < 0 || len(r.Raw[at].Bytes) != 24 {
		return data.Hero{}, false
	}
	var h data.Hero
	for i := range data.SkillSlots {
		h.Skill[i] = int32(binary.LittleEndian.Uint16(r.Raw[at].Bytes[2+2*i:]))
	}
	return h, true
}

// secondCampaignTail is the campaign tail of a town or mission SAVE: the
// scenario bank, the shop records, the availability list and the current
// location's ordinal (R2-SESSION-021, R2-SESSION-155); a mission's bank is
// its World's live one (R2-SESSION-031). No current state and
// no claim gives the pair list, the two words after it or a location's 16
// raw bytes: they are the loaded tail's, else the [[1,1]] every original
// town holds and zero. The nine client buffers are empty (DIV-4306).
func secondCampaignTail(s Snapshot) (*sav.SecondTail, error) {
	c := s.second
	t := &sav.SecondTail{Bank: c.Bank, Pairs: [][2]uint32{{1, 1}}}
	if s.liveBank != nil {
		t.Bank = *s.liveBank
	}
	if c.Kept != nil {
		t.Pairs, t.PairTail = slices.Clone(c.Kept.Pairs), c.Kept.PairTail
	}
	records := secondshop.NewGameRecords
	if c.Shop != nil {
		records = c.Shop.Records
	}
	for l, record := range records {
		for k, cell := range record {
			t.Shops[l][k] = sav.SecondShopCell(cell)
		}
	}
	current := slices.Index(c.Available, c.Current)
	if current < 0 {
		return nil, fmt.Errorf("second-game town save: the current location is not available")
	}
	for _, l := range c.Available {
		t.Locations = append(t.Locations, sav.SecondLocation{ID: uint32(l.ID), Kind: uint32(l.Kind), Raw: c.Kept.raw(l)})
	}
	t.Current = uint32(current)
	return t, nil
}
