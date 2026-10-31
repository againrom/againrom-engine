package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// importedSpellbook is a saved book's presence and instances. A spell past
// the book's slots (the second game's spell 29) keeps its membership in the
// book's native mode and has no instance (DIV-4174).
func importedSpellbook(present bool, spells []sav.SavedSpell) sim.Spellbook {
	b := sim.Spellbook{State: sim.BookAbsent}
	if present {
		b.State = sim.BookPresent
	}
	native := false
	for _, s := range spells {
		if int(s.ID) > len(b.Slots) {
			native = true
			continue
		}
		b.Slots[s.ID-1] = sim.BookSpell{Range: s.Range, Defensive: s.Defensive, ManaCost: s.ManaCost}
	}
	if native {
		b.State = sim.BookNativeAbsent
		if present {
			b.State = sim.BookNativePresent
		}
	}
	return b
}

func learnPartySpell(p *mapload.PartyMember, id uint16, table *mapload.Table) {
	e := sim.Entity{Book: p.Book, KnownSpells: p.KnownSpells}
	sim.LearnBookSpell(&e, id, mapload.SpellRules(table))
	p.Book, p.KnownSpells = e.Book, e.KnownSpells
	refreshDerivedPartyBook(p, table)
}

// partyBookReader projects a town party member of table's game into the
// read-book view. Town members carry no life state, so they are alive.
func partyBookReader(p *mapload.PartyMember, table *mapload.Table) sim.BookReader {
	return sim.BookReader{Alive: true, Mage: p.Mage, Book: p.Book, Known: p.KnownSpells, Game: spellGame(table)}
}

// readPartyBook runs the one read-book transaction on a party member and
// learns the spell into the member once the book is consumed.
func readPartyBook(p *mapload.PartyMember, spell uint16, table *mapload.Table, consume func() bool) bool {
	r := partyBookReader(p, table)
	return sim.ReadBookSpell(&r, spell, mapload.SpellRules(table), func(sim.BookReader) bool {
		if !consume() {
			return false
		}
		// The consume step may rebuild the member's carry, so the spell is
		// learned into the member as it stands afterwards.
		learnPartySpell(p, spell, table)
		return true
	})
}

func refreshPartyBook(r sim.Rules, p *mapload.PartyMember, table []sim.SpellRule) {
	e := sim.Entity{Book: p.Book, KnownSpells: p.KnownSpells, Skill: p.Hero.Skill, Mind: p.Hero.Mind}
	sim.RefreshBook(r, &e, table)
	p.Book = e.Book
}

func refreshDerivedPartyBook(p *mapload.PartyMember, table *mapload.Table) {
	if !p.Book.HasInstances() {
		return
	}
	d, _, _ := mapload.PartySpawnWithTable(*p, table)
	e := sim.Entity{Book: p.Book, KnownSpells: p.KnownSpells, Skill: d.Skill, Mind: d.Mind}
	sim.RefreshBook(mapload.TableRules(table), &e, mapload.SpellRules(table))
	p.Book = e.Book
}

func prepareCityTrainingBook(state *originalCitySaveState, graph *cityObjectTopology, before, next mapload.PartyMember) (*cityObjectTopology, error) {
	if graph != nil {
		return prepareCityBookChanges(graph, before, next, nil)
	}
	if state != nil {
		if err := state.validateBookTraining(before.ID, next); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func cityBookUpdate(c sav.CityCharacter, book sim.Spellbook) []sav.SavedSpell {
	spells := append([]sav.SavedSpell(nil), c.Spells...)
	for i := range spells {
		s := book.Slots[spells[i].ID-1]
		spells[i].Range, spells[i].Defensive, spells[i].ManaCost = s.Range, s.Defensive, s.ManaCost
	}
	return spells
}

func (s *originalCitySaveState) validateBookTraining(id string, candidate mapload.PartyMember) error {
	document, ok := s.document.(*sav.CityProvenance)
	if !ok {
		return nil
	}
	var updates []sav.CityCharacterUpdate
	for _, binding := range s.bindings {
		member, err := binding.expectedMember()
		if err != nil {
			return err
		}
		if binding.partyID == id {
			member = candidate
		}
		u := originalCityBaselineUpdate(binding.character)
		if member.Book.State != sim.BookLegacy {
			spells := cityBookUpdate(binding.character, member.Book)
			u.Spells = &spells
		}
		updates = append(updates, u)
	}
	return document.ValidateSpellUpdates(updates)
}
