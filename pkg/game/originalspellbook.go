package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"fmt"
)

func validateSnapshotBooks(s Snapshot) error {
	check := func(p mapload.PartyMember) error {
		if err := p.Book.Validate(p.KnownSpells); err != nil {
			return fmt.Errorf("party %q: %w", p.ID, err)
		}
		if p.Book.State != sim.BookLegacy && (!p.SpellbookRestored || p.SpellbookPresent != p.Book.HasInstances()) {
			return fmt.Errorf("party %q has inconsistent source book presence", p.ID)
		}
		return nil
	}
	for _, p := range s.Party {
		if err := check(p); err != nil {
			return err
		}
	}
	if s.OriginalCity != nil {
		for _, b := range s.OriginalCity.Bindings {
			if err := check(b.Baseline); err != nil {
				return err
			}
		}
	}
	return nil
}

func importedSpellbook(present bool, spells []sav.SavedSpell) sim.Spellbook {
	b := sim.Spellbook{State: sim.BookAbsent}
	if present {
		b.State = sim.BookPresent
	}
	for _, s := range spells {
		b.Slots[s.ID-1] = sim.BookSpell{Range: s.Range, Defensive: s.Defensive, ManaCost: s.ManaCost}
	}
	return b
}

func learnPartySpell(p *mapload.PartyMember, id uint16, table *mapload.Table) {
	e := sim.Entity{Book: p.Book, KnownSpells: p.KnownSpells}
	sim.LearnBookSpell(&e, id, mapload.SpellRules(table))
	p.Book, p.KnownSpells = e.Book, e.KnownSpells
	refreshDerivedPartyBook(p, table)
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
