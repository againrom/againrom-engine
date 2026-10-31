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

// secondTownForm is a second-game town SAVE's engine document in the second
// game's records, with the campaign tail.
func secondTownForm(table *mapload.Table, s Snapshot, doc sav.DocumentData) (sav.DocumentData, error) {
	if err := writeSecondTownHolders(&doc, s, table); err != nil {
		return sav.DocumentData{}, err
	}
	keeps, err := secondFormKeeps(s)
	if err != nil {
		return sav.DocumentData{}, err
	}
	out, ok, err := sav.DocumentInProfile(doc, sav.SecondGame, keeps)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if !ok {
		return sav.DocumentData{}, fmt.Errorf("the second game's profile has no record form")
	}
	if out.SecondTail, err = secondTownTail(s.second); err != nil {
		return sav.DocumentData{}, err
	}
	return out, nil
}

// secondFormKeeps is what the second game's records hold that a mission's
// World keeps beside the engine's records: each Pointer's word, by its
// Token key. A town SAVE has no World.
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

// writeSecondTownHolders writes what the second game's Humans hold that the
// engine's records keep elsewhere: the hero's main skill, every bag's insert
// index, and the campaign's bag in the hero's Inventory (DIV-4153).
func writeSecondTownHolders(doc *sav.DocumentData, s Snapshot, table *mapload.Table) error {
	hero, err := secondTownHeroRecord(*doc)
	if err != nil {
		return err
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		// R2-ENGINE-319: a Human built from a `_Hero` or `Start_` template,
		// such as a companion's identity row, holds his largest school; one
		// built from another keeps the Humanoid initializer's 0 (DIV-4305).
		main := uint32(0)
		if uint16(i+1) == hero {
			main = secondMainSkill(nativeCityHeroOf(s.Party).Hero)
		} else if h, ok := secondTemplateHero(r, table); ok {
			main = secondMainSkill(h)
		}
		at := slices.IndexFunc(r.Raw, func(v sav.DocumentRawData) bool { return v.Name == "H1CC" })
		if at < 0 || len(r.Raw[at].Bytes) < 4 {
			return fmt.Errorf("Human record %d has no experience block", i+1)
		}
		block := slices.Clone(r.Raw[at].Bytes)
		binary.LittleEndian.PutUint32(block, main)
		mustSetRaw(r, "H1CC", block)
		// R2-ENGINE-410: the bag constructor's insert index, which appends.
		if _, err := savedStructureValue(r, "Inventory1C"); err == nil {
			mustSetValue(r, "Inventory1C", 0x7fff)
		}
	}
	var bag mapload.PartyMember
	bag.ID = "bag"
	for _, m := range s.second.Party {
		if m.Member != nil {
			bag.CarriedItems = append(bag.CarriedItems, m.Member.CarriedItems...)
		}
	}
	if len(bag.CarriedItems) != 0 {
		if err := writeSecondTownBag(doc, hero, bag, table); err != nil {
			return err
		}
	}
	// R2-ENGINE-403: an Item holds its row at +0xc, from which LOAD
	// rebuilds its descriptor; a magic item's ID names that row. A Player's
	// obfuscated F48 holds 0, the originals' value (DIV-4305), and an
	// Effect the constructor's +0xe = 0 (R2-ENGINE-403).
	for i := range doc.Objects {
		r := &doc.Objects[i]
		for class, name := range map[string]string{"Player": "F48", "Effect": "T0E"} {
			if r.Class == class {
				if err := savedStructureSetValue(r, name, 0); err != nil {
					return err
				}
			}
		}
		if code, err := savedStructureValue(r, "F40"); r.Class == "Item" && err == nil {
			if row, ok := secondshop.MagicRow(uint16(code)); ok {
				if err := savedStructureSetValue(r, "T0C", uint32(row)); err != nil {
					return err
				}
			}
		}
	}
	// R2-ENGINE-402: a bag's weight is the sum of its Items' unit weight
	// times quantity.
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if _, err := savedStructureValue(r, "Inventory20"); r.Class != "Human" || err != nil {
			continue
		}
		var weight uint32
		for _, slot := range r.RefSlots {
			if slot.Name != "Inventory" {
				continue
			}
			for _, ref := range slot.Objects {
				if ref == 0 || int(ref) > len(doc.Objects) {
					continue
				}
				unit, errW := savedStructureValue(&doc.Objects[ref-1], "F4A")
				count, errC := savedStructureValue(&doc.Objects[ref-1], "F42")
				if errW == nil && errC == nil {
					weight += uint32(int16(unit)) * count
				}
			}
		}
		mustSetValue(r, "Inventory20", weight)
	}
	return nil
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

// secondTownTail is a town SAVE's campaign tail: the scenario bank, the shop
// records, the availability list and the current location's ordinal
// (R2-SESSION-021, R2-SESSION-155). No current state and no claim gives the
// pair list, the two words after it or a location's 16 raw bytes: they are
// the loaded tail's, else the [[1,1]] every original town holds and zero.
// The nine client buffers are empty (DIV-4306).
func secondTownTail(c *currentSecondCampaign) (*sav.SecondTail, error) {
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
