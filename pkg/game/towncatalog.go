package game

import (
	"errors"
	"fmt"
	"slices"

	"againrom/pkg/base"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The hook sets a catalog entry may name. Both bind the common host's
// vocabulary over the selected campaign state.
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
// each decoded under its entry's hook set. art holds, for a description a
// mod carries, the files its "mod:" art keys name.
type townBook struct {
	catalog      *town.Catalog
	descriptions map[string]*town.Description
	art          map[*town.Description]map[string][]byte
	// loads are the deferred art reads of each description in art.
	loads map[*town.Description]*townArtLoads
	// addedOffers are the catalog's offer rows a mod added over layer 0.
	addedOffers map[town.OfferKey]bool
}

// modArt is the files a mod carries for description d's art; nil for a
// shipped description.
func (b *townBook) modArt(d *town.Description) map[string][]byte {
	if b == nil {
		return nil
	}
	return b.art[d]
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
		if err := checkOfferBank(c.Offers, c.OfferTable); err != nil {
			panic(err)
		}
		if c.Mercenaries != nil {
			if tavernRule != nil {
				panic(fmt.Sprintf("town catalog %q states a second mercenary rule", name))
			}
			if c.Mercenaries.StockTypes >= len(Town{}.mercHired) {
				panic(fmt.Sprintf("town catalog %q: %d stock types pass the town's %d mercenary types", name, c.Mercenaries.StockTypes, len(Town{}.mercHired)-1))
			}
			tavernRule = c.Mercenaries
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
	case townHooksChapters, townHooksDestinations:
		return townVocabulary(), true
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
	if err := checkTownCatalog(c, data, decoded, hooks); err != nil {
		return nil, err
	}
	return c, nil
}

// checkTownCatalog is decodeTownCatalog's rules past the catalog's own, for
// a catalog read from data or built by a layer: every description it names
// decoded from data into decoded under its entry's hook set. A refusal of
// one entry is a town.EntryError.
func checkTownCatalog(c *town.Catalog, data map[string][]byte, decoded map[string]*town.Description, hooks map[string]string) error {
	chapters := false
	for _, e := range c.Towns {
		if want := townOfferHooks[e.Offers]; want != e.Hooks {
			return town.EntryErrorf(e.ID, "town catalog %s: town %d has no usable description: offers %s compose under hook set %s, not %s",
				c.Catalog, e.ID, e.Offers, want, e.Hooks)
		}
		if e.Offers == townOffersScenarioBank && (e.OpensAfter != 0 || len(e.Missions) != 0) {
			return town.EntryErrorf(e.ID, "town catalog %s: town %d takes its opening and missions from the scenario bank", c.Catalog, e.ID)
		}
		chapters = chapters || e.Offers == townOffersChapterKeys
	}
	if start, ok := c.Town(chapterTown); chapters && (!ok || start.Offers != townOffersChapterKeys || start.OpensAfter != 0) {
		return town.EntryErrorf(chapterTown, "town catalog %s: chapter towns start in town %d, open from the start", c.Catalog, chapterTown)
	}
	for _, e := range c.Towns {
		if prior, ok := hooks[e.Description]; ok && prior != e.Hooks {
			return town.EntryErrorf(e.ID, "town catalog %s: description %s named under hook sets %s and %s", c.Catalog, e.Description, prior, e.Hooks)
		}
		hooks[e.Description] = e.Hooks
		if _, ok := decoded[e.Description]; ok {
			continue
		}
		src, ok := data[e.Description]
		if !ok {
			return town.EntryErrorf(e.ID, "town catalog %s: town %d names description %q that is not shipped", c.Catalog, e.ID, e.Description)
		}
		vocab, _ := townHookSetVocabulary(e.Hooks)
		d, err := town.Decode(src, vocab)
		if err != nil {
			return town.EntryErrorf(e.ID, "town catalog %s: town %d: %w", c.Catalog, e.ID, err)
		}
		for _, room := range townHookSetRooms[e.Hooks] {
			if d.Room(room) == nil {
				return town.EntryErrorf(e.ID, "town catalog %s: town %d has no usable description: %s has no room %q", c.Catalog, e.ID, e.Description, room)
			}
		}
		if e.Hooks == townHooksChapters {
			if err := checkChapterDescription(d); err != nil {
				return town.EntryErrorf(e.ID, "town catalog %s: town %d has no usable description: %s %v", c.Catalog, e.ID, e.Description, err)
			}
		}
		if err := checkTownStateBinding(d, e.Offers); err != nil {
			return town.EntryErrorf(e.ID, "town catalog %s: town %d: %v", c.Catalog, e.ID, err)
		}
		decoded[e.Description] = d
	}
	return nil
}

func checkTownStateBinding(d *town.Description, offers string) error {
	if h := d.Host; h != nil {
		shop, talk, leave, cached, starts := "chapter", "building", "room-map", true, false
		if offers == townOffersScenarioBank {
			shop, talk, leave, cached, starts = "bank", "section", "destinations", false, true
		}
		if h.Shop != shop || h.Talk != talk || h.Leave != leave || h.CacheScreen != cached || h.StartsInTown != starts {
			return fmt.Errorf("host state binding does not match offer source %s", offers)
		}
	}
	var hooks []string
	steps := func(list []town.Step) {
		for _, st := range list {
			if st.Hook != "" {
				hooks = append(hooks, st.Hook)
			}
		}
	}
	var ops func([]town.Op)
	ops = func(list []town.Op) {
		for _, op := range list {
			if op.Hook != "" {
				hooks = append(hooks, op.Hook)
			}
			ops(op.Then)
			ops(op.Else)
		}
	}
	scene := func(s *town.SceneSpec) {
		if s == nil {
			return
		}
		for _, h := range s.Hotspots {
			ops(h.Click)
			ops(h.Hover)
		}
		ops(s.Pointer.Every)
		ops(s.Pointer.Off)
	}
	steps(d.Square.Enter)
	scene(d.SquareScene())
	for _, r := range d.Rooms {
		required := ""
		switch r.Page {
		case "tavern", "school":
			required = townOffersChapterKeys
		case "inn":
			required = townOffersScenarioBank
		}
		if required != "" && required != offers {
			return fmt.Errorf("room %q page %s requires offer source %s", r.Name, r.Page, required)
		}
		steps(r.Enter)
		steps(r.Exit)
		scene(r.Scene)
	}
	for _, name := range hooks {
		if required := townHooks[name].offers; required != "" && required != offers {
			return fmt.Errorf("hook %q requires offer source %s", name, required)
		}
	}
	return nil
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

// chapterSchoolActors are the actors, by name, that the first game's school
// page drives, each with the program it must run.
var chapterSchoolActors = []struct{ name, program string }{
	{"column", "target"}, {"training", "training"}, {"diamond", "bounce"}, {"shine", "cycle"},
}

// checkChapterDescription is what the first game's town screen needs of a
// description beyond its rooms, each of which, missing, leaves the party
// stuck or stops the game on entry: the steps that move the screen between
// rooms, the scenes of the tavern, shop and school pages, and the school
// page's four actors. The refusal names the first thing missing.
func checkChapterDescription(d *town.Description) error {
	if !runsHook(d.Square.Enter, "set-room") {
		return errors.New(`square enter steps do not run hook "set-room", so the party never stands on the square`)
	}
	for _, name := range townHookSetRooms[townHooksChapters] {
		r := d.Room(name)
		if !runsHook(r.Enter, "set-room") {
			return fmt.Errorf(`room %q enter steps do not run hook "set-room", so the party never enters it`, name)
		}
	}
	if !runsHook(d.Room("gates").Enter, "navigate") {
		return errors.New(`room "gates" enter steps do not run hook "navigate", so the gates never open the world map`)
	}
	for _, name := range []string{"tavern", "shop", "school"} {
		if d.Room(name).Scene == nil {
			return fmt.Errorf("room %q has no scene; the %s page is drawn from it", name, name)
		}
		// The page's own controls fill the screen beside the character
		// region, and only the shop has a spellbook page.
		if p := d.Room(name).Pane; p != nil && p.Rect.Rectangle() != ui.TownCharacterRegion {
			return fmt.Errorf("room %q pane rectangle %v is not the character region %v", name, p.Rect.Rectangle(), ui.TownCharacterRegion)
		} else if p != nil && p.Book && name != "shop" {
			return fmt.Errorf("room %q pane has the spellbook corner; the %s page has no spellbook", name, name)
		} else if p != nil && p.BookOffset != (town.Point{}) {
			return fmt.Errorf("room %q pane book_offset %v is not zero; the %s page does not move its spellbook", name, p.BookOffset.Pt(), name)
		}
	}
	scene := d.Room("school").Scene
	for _, want := range chapterSchoolActors {
		if !slices.ContainsFunc(scene.Actors, func(a town.ActorSpec) bool { return a.Name == want.name && a.Program == want.program }) {
			return fmt.Errorf("room %q scene has no actor %q running program %q; the school page drives it", "school", want.name, want.program)
		}
	}
	return nil
}

// runsHook reports whether steps run the named campaign hook.
func runsHook(steps []town.Step, hook string) bool {
	return slices.ContainsFunc(steps, func(s town.Step) bool { return s.Hook == hook })
}
