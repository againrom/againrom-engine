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
	keeps func(s Snapshot) (sav.ProfileKeeps, error)
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
		if keeps, err = form.keeps(s); err != nil {
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
// World keeps beside the engine's records: each Pointer's word, by its
// Token key.
func secondFormKeeps(s Snapshot) (sav.ProfileKeeps, error) {
	if len(s.World) == 0 {
		return sav.ProfileKeeps{}, nil
	}
	var w sim.World
	if err := w.UnmarshalBinary(s.World); err != nil {
		return sav.ProfileKeeps{}, err
	}
	var keeps sav.ProfileKeeps
	source, _, _ := w.SavedStructures()
	for _, st := range source {
		if st.Class == sim.SavedPointer {
			if keeps.Pointers == nil {
				keeps.Pointers = map[uint32]uint32{}
			}
			keeps.Pointers[st.SourceKey] = st.Pointer6C
		}
	}
	return keeps, nil
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
	// holderRecords is a reference slot the source fills with records.
	holderRecords
)

// formHolding is what the holders' sources read: the document being
// formed, the snapshot, the installed table and the hero's record.
type formHolding struct {
	doc   *sav.DocumentData
	s     Snapshot
	table *mapload.Table
	hero  uint16
}

// secondFormHolders are the fields the second game's records hold that the
// engine's records keep elsewhere.
var secondFormHolders = []savedFormHolder{
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
	// R2-ENGINE-402: a bag's weight is the sum of its Items' unit weight
	// times quantity.
	{class: "Human", field: "Inventory20", site: holderPresentValue, value: secondBagWeight},
}

// writeSecondTownHolders writes each holder's field in doc's records from
// its source, in the holders' order.
func writeSecondTownHolders(doc *sav.DocumentData, holders []savedFormHolder, s Snapshot, table *mapload.Table) error {
	hero, err := secondTownHeroRecord(*doc)
	if err != nil {
		return err
	}
	h := &formHolding{doc: doc, s: s, table: table, hero: hero}
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
	var bag mapload.PartyMember
	bag.ID = "bag"
	for _, m := range h.s.second.Party {
		if m.Member != nil {
			bag.CarriedItems = append(bag.CarriedItems, m.Member.CarriedItems...)
		}
	}
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
// location's ordinal (R2-SESSION-021, R2-SESSION-155). No current state and
// no claim gives the pair list, the two words after it or a location's 16
// raw bytes: they are the loaded tail's, else the [[1,1]] every original
// town holds and zero. The nine client buffers are empty (DIV-4306).
func secondCampaignTail(s Snapshot) (*sav.SecondTail, error) {
	c := s.second
	t := &sav.SecondTail{Bank: c.Bank, Pairs: [][2]uint32{{1, 1}}}
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
