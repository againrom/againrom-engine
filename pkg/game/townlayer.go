package game

import (
	"errors"
	"fmt"
	"image"
	"maps"
	"slices"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
	"againrom/pkg/town"
)

// SetModTowns applies the mods' layers over the install's town catalog: the
// merged catalog passes the catalog decoder's rules, the campaign's opening
// check and, for a stated map point, the world map's bounds, after each mod.
// Every refusal names the mod, the file and the line. The campaign and the
// session's town take the merged catalog. The returned lines name every
// field a later mod sets over an earlier one. It runs once, after SetMods
// and before any game is opened.
func (f *FrontEnd) SetModTowns(d mod.TownData) (overrides []string, err error) {
	if d.Empty() {
		return nil, nil
	}
	if f == nil || f.Archives == nil || f.Archives.Containers == nil {
		return nil, errors.New("the front end has no archives to carry the mods' towns")
	}
	c := f.Campaign.Value()
	if err := f.Campaign.Err(); err != nil {
		return nil, d.Records[0].Errorf("the install's campaign is unavailable for its towns: %v", err)
	}
	// A stated point stands inside the picture of every mode of the map.
	bounds := func() (b image.Rectangle, ok bool) {
		for _, m := range loadWorldMapAssets(f.Archives).modes {
			switch {
			case m.background == nil:
			case !ok:
				b, ok = m.background.Bounds(), true
			default:
				b = b.Intersect(m.background.Bounds())
			}
		}
		return b, ok
	}
	// A panel draws the install's own byte alphabet, not UTF-8.
	encode := func(name string) (string, error) { return encodeSaveLabel(name, f.textSelector()) }
	book, list, err := layerTownBook(townBookOf(f.Archives.Base.Profile), d, c, bounds, encode)
	if err != nil {
		return nil, err
	}
	c.Towns = book
	f.Campaign = resolved(c, nil)
	f.Town = NewTown(c)
	for _, o := range list {
		overrides = append(overrides, o.String())
	}
	return overrides, nil
}

// layerTownBook is base, the install's book, with the mods' town layers
// applied through the one layer step and judged after each mod. bounds
// answers the world map's picture bounds for a stated point; encode turns a
// record's UTF-8 name into the install's alphabet, refusing a character
// the alphabet lacks.
func layerTownBook(base *townBook, d mod.TownData, c Campaign, bounds func() (image.Rectangle, bool), encode func(string) (string, error)) (*townBook, []mod.Override, error) {
	first := d.Records[0]
	if base == nil {
		return nil, nil, first.Errorf("this base has no town catalog to layer")
	}
	if slices.ContainsFunc(base.catalog.Towns, func(e town.Entry) bool { return e.Offers != townOffersChapterKeys }) {
		return nil, nil, first.Errorf("town layers apply to the first game's catalog; catalog %s is not one", base.catalog.Catalog)
	}
	data := maps.Clone(townDescriptionData)
	files := map[string]*mod.TownDescriptionFile{}
	described := map[string]mod.Source{}
	firstOf := map[string]mod.Source{}
	patches := make([]mod.Patch[int, town.Entry], 0, len(d.Records))
	for _, r := range d.Records {
		// A first-game town is a chapter town; the catalog's other hook
		// set and offer source would add a town that never shows.
		for _, field := range r.Fields {
			at := mod.Source{Mod: r.Mod, File: r.File, Line: field.Line}
			switch {
			case field.Name == "hooks" && r.Entry.Hooks != townHooksChapters:
				return nil, nil, at.Errorf("town %d takes hooks %q; a first-game town takes hooks = %q", r.Entry.ID, r.Entry.Hooks, townHooksChapters)
			case field.Name == "offers" && r.Entry.Offers != townOffersChapterKeys:
				return nil, nil, at.Errorf("town %d takes offers %q; a first-game town takes offers = %q", r.Entry.ID, r.Entry.Offers, townOffersChapterKeys)
			case field.Name == "name":
				shown, err := encode(r.Entry.Name)
				if err != nil {
					return nil, nil, at.Errorf("the name %q cannot be drawn by this install: %v", r.Entry.Name, err)
				}
				r.Entry.Name = shown
			}
		}
		if r.Description != nil {
			data[r.Entry.Description] = r.Description.Data
			files[r.Entry.Description] = r.Description
		}
		if slices.ContainsFunc(r.Fields, func(f mod.PatchField) bool { return f.Name == "description" }) {
			if _, ok := described[r.Entry.Description]; !ok {
				described[r.Entry.Description] = mod.Source{Mod: r.Mod, File: r.File, Line: r.LineOf("description")}
			}
		}
		if _, ok := firstOf[r.Mod]; !ok {
			firstOf[r.Mod] = r.Source
		}
		patches = append(patches, mod.Patch[int, town.Entry]{Source: r.Source, Key: r.Entry.ID, Fields: r.Fields, Off: r.Off, Apply: r.Apply})
	}
	decoded := maps.Clone(base.descriptions)
	hooks := shippedTownHooks()
	catalogOf := func(entries []town.Entry) *town.Catalog {
		return &town.Catalog{Catalog: base.catalog.Catalog, Cite: base.catalog.Cite, Towns: slices.Clone(entries)}
	}
	check := func(modID string, l *mod.Layered[int, town.Entry]) error {
		if src, off := l.Off[chapterTown]; off {
			return src.Errorf("town %d is the campaign's starting town; it cannot be switched off", chapterTown)
		}
		blame := func(err error) error {
			src := firstOf[modID]
			var ce *townCheckError
			var ee *town.EntryError
			if errors.As(err, &ce) {
				if i := slices.IndexFunc(ce.towns, func(id int) bool { return l.Named[id].Mod == modID }); i >= 0 {
					src = l.Named[ce.towns[i]]
				}
			} else if errors.As(err, &ee) {
				if named, ok := l.Named[ee.Town]; ok && named.Mod == modID {
					src = named
				}
			}
			return src.Errorf("%v", err)
		}
		cat := catalogOf(l.Records)
		if err := cat.Check(townCatalogVocabulary); err != nil {
			return blame(err)
		}
		if err := checkTownCatalog(cat, data, decoded, hooks); err != nil {
			return blame(err)
		}
		probe := c
		probe.Towns = &townBook{catalog: cat, descriptions: decoded}
		if err := probe.checkTowns(); err != nil {
			return blame(err)
		}
		for _, e := range cat.Towns {
			p, r, stated := e.MapPoint.Stated()
			if !stated {
				continue
			}
			b, ok := bounds()
			switch {
			case !ok:
				return blame(town.EntryErrorf(e.ID, "town %d states a map point, but the world map has no picture to bound it", e.ID))
			case !p.In(b) || !r.In(b):
				return blame(town.EntryErrorf(e.ID, "town %d's map point %v or rect %v leaves the world map %v", e.ID, p, r, b))
			}
		}
		return nil
	}
	l, err := mod.Merge("town", base.catalog.Towns, func(e town.Entry) int { return e.ID }, patches, d.Order, check)
	if err != nil {
		return nil, nil, err
	}
	book := &townBook{catalog: catalogOf(l.Records), descriptions: map[string]*town.Description{}}
	for _, e := range book.catalog.Towns {
		desc := decoded[e.Description]
		book.descriptions[e.Description] = desc
		file := files[e.Description]
		if file == nil || book.art[desc] != nil {
			continue
		}
		art, err := readModTownArt(desc, file)
		if err != nil {
			return nil, nil, described[e.Description].Errorf("description %s: %v", file.Path, err)
		}
		if book.art == nil {
			book.art = map[*town.Description]map[string][]byte{}
		}
		book.art[desc] = art
	}
	return book, l.Overrides, nil
}

// shippedTownHooks is the hook set every shipped catalog names each of its
// descriptions under, so a layer cannot name a shipped description under
// another set.
func shippedTownHooks() map[string]string {
	hooks := map[string]string{}
	for _, b := range townBooks {
		for _, e := range b.catalog.Towns {
			hooks[e.Description] = e.Hooks
		}
	}
	return hooks
}

// townArtEntries are every art entry of description d: the square's, and
// each room's scene and page.
func townArtEntries(d *town.Description) []town.ArtSpec {
	out := slices.Clone(d.Art)
	for _, r := range d.Rooms {
		if r.Scene != nil {
			out = append(out, r.Scene.Art...)
		}
		out = append(out, r.Art...)
	}
	return out
}

// readModTownArt reads every file a mod's description names by a "mod:"
// key, by that key. A missing file, or a bound entry naming one, is an
// error.
func readModTownArt(d *town.Description, file *mod.TownDescriptionFile) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, e := range townArtEntries(d) {
		if e.Bound && strings.HasPrefix(e.Key, mod.ModArtPrefix) {
			return nil, fmt.Errorf("art %q is bound at run time and cannot name a file of the mod", e.Name)
		}
		for _, key := range e.Keys() {
			if !strings.HasPrefix(key, mod.ModArtPrefix) || out[key] != nil {
				continue
			}
			b, _, err := mod.ReadInside(file.Dir, strings.TrimPrefix(key, mod.ModArtPrefix))
			if err != nil {
				return nil, fmt.Errorf("art %q: %v", key, err)
			}
			out[key] = b
		}
	}
	return out, nil
}

// modTownArtSource reads a mod's town description's art: a "mod:" key from
// the files the mod carries, every other key from the install.
type modTownArtSource struct {
	install terrain.EntrySource
	files   map[string][]byte
}

func (s modTownArtSource) ReadFile(key string) ([]byte, error) {
	if strings.HasPrefix(key, mod.ModArtPrefix) {
		if b, ok := s.files[key]; ok {
			return b, nil
		}
		return nil, fmt.Errorf("%s: the mod carries no such file", key)
	}
	if s.install == nil {
		return nil, fmt.Errorf("%s: no archive", key)
	}
	return s.install.ReadFile(key)
}
