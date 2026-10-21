package game

import (
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/town"
)

// The hook sets a catalog entry may name. Each resolves to one campaign's
// Go hook, condition and value tables: chapters to the first game's town
// screen, destinations to the second game's.
const (
	townHooksChapters     = "chapters"
	townHooksDestinations = "destinations"
)

// The offer sources a catalog entry may name: chapter-keys are the chapter
// keys of scenario.reg (InnMission, ShopMission, TCMission), scenario-bank is
// the second game's Scenario bank.
const (
	townOffersChapterKeys  = "chapter-keys"
	townOffersScenarioBank = "scenario-bank"
)

// chapterTown is the town a chapter campaign's party starts in: the
// catalog's town 1.
const chapterTown = 1

// townOfferHooks is the hook set each offer source's towns compose under:
// one campaign answers both, so an entry that pairs them otherwise has no
// screen that can draw it.
var townOfferHooks = map[string]string{
	townOffersChapterKeys:  townHooksChapters,
	townOffersScenarioBank: townHooksDestinations,
}

// townHookSetRooms are the rooms a hook set's town screen composes: a
// description under the set without one of them is not usable.
var townHookSetRooms = map[string][]string{
	townHooksChapters: {"tavern", "shop", "school", "gates"},
}

// townDescriptionData are the shipped town descriptions a catalog entry can
// name, by that name.
var townDescriptionData = map[string][]byte{
	"rom1":       rom1TownJSON,
	"rom2-town1": rom2Town1JSON,
	"rom2-town2": rom2Town2JSON,
	"rom2-town3": rom2Town3JSON,
}

// townCatalogData are the shipped catalogs an edition can name, by that name.
var townCatalogData = map[string][]byte{
	"rom1": rom1TownCatalogJSON,
	"rom2": rom2TownCatalogJSON,
}

// townBook is a decoded catalog with every description its entries name,
// each decoded under its entry's hook set.
type townBook struct {
	catalog      *town.Catalog
	descriptions map[string]*town.Description
}

// townBooks are the shipped catalogs by name, and townDescriptions every
// description a shipped catalog names. A catalog or description that does
// not decode, or names a hook set, offer source, hook or condition the
// campaign does not answer, is a build defect, so it stops the process at
// start. They are decoded in init because the hooks they are checked
// against reach the descriptions themselves.
var (
	townBooks        map[string]*townBook
	townDescriptions map[string]*town.Description
)

func init() {
	townBooks = map[string]*townBook{}
	townDescriptions = map[string]*town.Description{}
	hooks := map[string]string{}
	for name, data := range townCatalogData {
		c, err := decodeTownCatalog(data, townDescriptionData, townDescriptions, hooks)
		if err != nil {
			panic(err)
		}
		if c.Catalog != name {
			panic(fmt.Sprintf("town catalog %q is registered as %q", c.Catalog, name))
		}
		townBooks[name] = &townBook{catalog: c, descriptions: townDescriptions}
	}
}

// decodeTownBook decodes a catalog over the descriptions data holds into a
// book of its own.
func decodeTownBook(raw []byte, data map[string][]byte) (*townBook, error) {
	decoded := map[string]*town.Description{}
	c, err := decodeTownCatalog(raw, data, decoded, map[string]string{})
	if err != nil {
		return nil, err
	}
	return &townBook{catalog: c, descriptions: decoded}, nil
}

// townCatalogVocabulary is the hook sets and offer sources the campaigns
// answer.
var townCatalogVocabulary = town.CatalogVocabulary{
	HookSets:     []string{townHooksChapters, townHooksDestinations},
	OfferSources: []string{townOffersChapterKeys, townOffersScenarioBank},
}

// townHookSetVocabulary is the vocabulary hook set name resolves to.
func townHookSetVocabulary(name string) (town.Vocabulary, bool) {
	switch name {
	case townHooksChapters:
		return townVocabulary(), true
	case townHooksDestinations:
		return secondTownVocabulary(), true
	}
	return town.Vocabulary{}, false
}

// decodeTownCatalog decodes a catalog and every description it names from
// data into decoded, each under its entry's hook set, which hooks records by
// description name. A description named under two hook sets is an error:
// one decoded description answers one campaign. So is an entry whose
// description its screen cannot compose: a hook set its offer source's
// campaign does not draw, or no room the hook set's screen needs. Chapter
// towns start the party in town 1, open from the start; a scenario-bank town
// takes its opening and its missions from the bank.
func decodeTownCatalog(raw []byte, data map[string][]byte, decoded map[string]*town.Description, hooks map[string]string) (*town.Catalog, error) {
	c, err := town.DecodeCatalog(raw, townCatalogVocabulary)
	if err != nil {
		return nil, err
	}
	chapters := false
	for _, e := range c.Towns {
		if want := townOfferHooks[e.Offers]; want != e.Hooks {
			return nil, fmt.Errorf("town catalog %s: town %d has no usable description: offers %s compose under hook set %s, not %s",
				c.Catalog, e.ID, e.Offers, want, e.Hooks)
		}
		if e.Offers == townOffersScenarioBank && (e.OpensAfter != 0 || len(e.Missions) != 0) {
			return nil, fmt.Errorf("town catalog %s: town %d takes its opening and missions from the scenario bank", c.Catalog, e.ID)
		}
		chapters = chapters || e.Offers == townOffersChapterKeys
	}
	if start, ok := c.Town(chapterTown); chapters && (!ok || start.Offers != townOffersChapterKeys || start.OpensAfter != 0) {
		return nil, fmt.Errorf("town catalog %s: chapter towns start in town %d, open from the start", c.Catalog, chapterTown)
	}
	for _, e := range c.Towns {
		if prior, ok := hooks[e.Description]; ok && prior != e.Hooks {
			return nil, fmt.Errorf("town catalog %s: description %s named under hook sets %s and %s", c.Catalog, e.Description, prior, e.Hooks)
		}
		hooks[e.Description] = e.Hooks
		if _, ok := decoded[e.Description]; ok {
			continue
		}
		src, ok := data[e.Description]
		if !ok {
			return nil, fmt.Errorf("town catalog %s: town %d names description %q that is not shipped", c.Catalog, e.ID, e.Description)
		}
		vocab, _ := townHookSetVocabulary(e.Hooks)
		d, err := town.Decode(src, vocab)
		if err != nil {
			return nil, fmt.Errorf("town catalog %s: town %d: %w", c.Catalog, e.ID, err)
		}
		for _, room := range townHookSetRooms[e.Hooks] {
			if d.Room(room) == nil {
				return nil, fmt.Errorf("town catalog %s: town %d has no usable description: %s has no room %q", c.Catalog, e.ID, e.Description, room)
			}
		}
		decoded[e.Description] = d
	}
	return c, nil
}

// TownCatalog is the town catalog the profile's edition names; nil when it
// names none. Callers must not change it.
func TownCatalog(p base.Profile) *town.Catalog { return townBookOf(p).entries() }

// townBookOf is the book of the catalog the profile's edition names; nil
// when it names none.
func townBookOf(p base.Profile) *townBook { return townBooks[p.Edition().Towns] }

func (b *townBook) entries() *town.Catalog {
	if b == nil {
		return nil
	}
	return b.catalog
}

// description is the description of town id when its entry names hook set
// hooks, the only set the asking screen answers; nil otherwise.
func (b *townBook) description(id int, hooks string) *town.Description {
	if b == nil {
		return nil
	}
	e, ok := b.catalog.Town(id)
	if !ok || e.Hooks != hooks {
		return nil
	}
	return b.descriptions[e.Description]
}

// townDescriptionOf is the description of shipped catalog c's town id under
// hook set hooks; nil when the entry names another set.
func townDescriptionOf(c *town.Catalog, id int, hooks string) *town.Description {
	return (&townBook{catalog: c, descriptions: townDescriptions}).description(id, hooks)
}

// chapterTowns are the book's towns whose offers are the chapter keys, in
// catalog order: the towns a chapter campaign's party can stand in. A book
// with none, or no book, has town 1 alone.
func (b *townBook) chapterTowns() []town.Entry {
	var out []town.Entry
	if b != nil {
		for _, e := range b.catalog.Towns {
			if e.Offers == townOffersChapterKeys {
				out = append(out, e)
			}
		}
	}
	if len(out) == 0 {
		return []town.Entry{chapterTownEntry}
	}
	return out
}
