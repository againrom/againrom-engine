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

// chapterTown is the town a chapter campaign's party stands in: the first
// game travels between no towns, so its town is always the catalog's town 1.
const chapterTown = 1

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

// townCatalogs are the decoded catalogs by name, and townDescriptions every
// description a catalog names, decoded under its entry's hook set. A catalog
// or description that does not decode, or names a hook set, offer source,
// hook or condition the campaign does not answer, is a build defect, so it
// stops the process at start. They are decoded in init because the hooks
// they are checked against reach the descriptions themselves.
var (
	townCatalogs     map[string]*town.Catalog
	townDescriptions map[string]*town.Description
)

func init() {
	townCatalogs = map[string]*town.Catalog{}
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
		townCatalogs[name] = c
	}
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
// one decoded description answers one campaign.
func decodeTownCatalog(raw []byte, data map[string][]byte, decoded map[string]*town.Description, hooks map[string]string) (*town.Catalog, error) {
	c, err := town.DecodeCatalog(raw, townCatalogVocabulary)
	if err != nil {
		return nil, err
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
		decoded[e.Description] = d
	}
	return c, nil
}

// TownCatalog is the town catalog the profile's edition names; nil when it
// names none. Callers must not change it.
func TownCatalog(p base.Profile) *town.Catalog { return townCatalogs[p.Edition().Towns] }

// townDescriptionOf is the description of catalog town id when its entry
// names hook set hooks, the only set the asking screen answers; nil
// otherwise.
func townDescriptionOf(c *town.Catalog, id int, hooks string) *town.Description {
	e, ok := c.Town(id)
	if !ok || e.Hooks != hooks {
		return nil
	}
	return townDescriptions[e.Description]
}
