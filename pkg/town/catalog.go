package town

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"slices"
)

// Catalog is the list of towns a game offers, as data. Each game fills one
// from its own file; a later source may supply entries in the same format.
// The composer reads no entry itself: the campaign resolves each name an
// entry carries against its own tables.
type Catalog struct {
	Catalog string   `json:"catalog"`
	Cite    []string `json:"cite"`
	Towns   []Entry  `json:"towns"`
	// OfferTable names the shape and key of Offers.
	OfferTable OfferTable `json:"offer-table,omitempty"`
	// Offers are the option rows of the towns' taverns or inns, in offer
	// order. Nil states none: each tavern then reads its chapter's own
	// offer data.
	Offers []OfferRow `json:"offers,omitempty"`
	// Mercenaries is the rule the towns' taverns build their mercenary
	// roster and price their squads by; nil for towns with none.
	Mercenaries *MercenaryRule `json:"mercenaries,omitempty"`
}

// maxTavernValue bounds an NPC record and a mission: a save holds each in
// 16 bits.
const maxTavernValue = 1<<16 - 1

// Entry is one town of a catalog. Every field is plain data: the hook set,
// the offer source and the map point are names the campaign resolves, never
// code.
type Entry struct {
	// ID is the town's identifier, unique within the catalog and positive;
	// it is the key of the town's state and of the place "town ID".
	ID int `json:"id"`
	// Description is the name of the town description the town screen is
	// composed from.
	Description string `json:"description"`
	// Hooks names the campaign hook set the description's hooks, conditions
	// and values resolve in.
	Hooks string `json:"hooks"`
	// Offers names where the town's tavern, shop and school offers come from.
	Offers string `json:"offers"`
	// MapPoint is where the town stands on the world map.
	MapPoint MapPoint `json:"map-point"`
	// Name is the town's title on the world map's travel card; empty draws
	// the map's own town word.
	Name string `json:"name,omitempty"`
	// OpensAfter is the mission whose win opens the town to travel; zero
	// opens it from the start.
	OpensAfter int `json:"opens-after,omitempty"`
	// Missions are the missions the town's buildings offer. An empty list
	// offers every mission no other entry lists.
	Missions []int    `json:"missions,omitempty"`
	Cite     []string `json:"cite"`
}

// MapPoint says where a town's world-map point is read from: exactly one of
// Object, Record and Point is set.
type MapPoint struct {
	// Object is the number of the world-map registry object whose point the
	// town stands at, counted from 1.
	Object int `json:"object,omitempty"`
	// Record is true when the point is the world-map description's catalog
	// record of this town: the record of the town kind and the town's ID.
	Record bool `json:"record,omitempty"`
	// Point and Rect state the point as a registry object states it: the
	// route point [x, y] and the clickable rectangle [x, y, width, height],
	// in map pixels. Rect is given with Point and only with it.
	Point []int `json:"point,omitempty"`
	Rect  []int `json:"rect,omitempty"`
}

// sources counts the point sources the map point names.
func (p MapPoint) sources() int {
	n := 0
	for _, set := range []bool{p.Object != 0, p.Record, p.Point != nil} {
		if set {
			n++
		}
	}
	return n
}

// check is the map point's own rule: exactly one source, a positive
// object, and a stated point of two coordinates with a rectangle of four,
// none negative and the rectangle not empty.
func (p MapPoint) check() error {
	switch {
	case p.sources() != 1, p.Object < 0:
		return fmt.Errorf("must name exactly one map point source")
	case p.Point == nil && p.Rect != nil:
		return fmt.Errorf("states a map rect without a point")
	case p.Point == nil:
		return nil
	case len(p.Point) != 2 || p.Point[0] < 0 || p.Point[1] < 0:
		return fmt.Errorf("map point %v is not [x, y] of two coordinates from 0", p.Point)
	case len(p.Rect) != 4 || p.Rect[0] < 0 || p.Rect[1] < 0 || p.Rect[2] <= 0 || p.Rect[3] <= 0:
		return fmt.Errorf("map rect %v is not [x, y, width, height] with coordinates from 0 and a positive size", p.Rect)
	}
	return nil
}

// Stated answers a stated point and its rectangle; false for a point read
// from the install.
func (p MapPoint) Stated() (point image.Point, rect image.Rectangle, ok bool) {
	if len(p.Point) != 2 || len(p.Rect) != 4 {
		return image.Point{}, image.Rectangle{}, false
	}
	return image.Pt(p.Point[0], p.Point[1]), image.Rect(p.Rect[0], p.Rect[1], p.Rect[0]+p.Rect[2], p.Rect[1]+p.Rect[3]), true
}

// EntryError is a refusal of one town's entry: Town names the town.
type EntryError struct {
	Town int
	Err  error
}

func (e *EntryError) Error() string { return e.Err.Error() }
func (e *EntryError) Unwrap() error { return e.Err }

// EntryErrorf is a refusal of town id's entry.
func EntryErrorf(id int, format string, args ...any) error {
	return &EntryError{Town: id, Err: fmt.Errorf(format, args...)}
}

// CatalogVocabulary is the hook set and offer source names a campaign
// answers.
type CatalogVocabulary struct {
	HookSets     []string
	OfferSources []string
}

// DecodeCatalog reads a catalog strictly: an unknown field, data after the
// catalog, a hook set or offer source outside vocab, or an entry that fails
// Validate is a named error.
func DecodeCatalog(data []byte, vocab CatalogVocabulary) (*Catalog, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var c Catalog
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("town catalog: %w", err)
	}
	if rest := bytes.TrimSpace(data[dec.InputOffset():]); len(rest) > 0 {
		return nil, fmt.Errorf("town catalog %s: trailing data %q", c.Catalog, firstBytes(rest))
	}
	if err := c.Check(vocab); err != nil {
		return nil, err
	}
	return &c, nil
}

// Check is the decode's rules over a catalog built in memory: Validate, and
// every hook set and offer source inside vocab.
func (c *Catalog) Check(vocab CatalogVocabulary) error {
	if err := c.Validate(); err != nil {
		return err
	}
	for _, e := range c.Towns {
		if !slices.Contains(vocab.HookSets, e.Hooks) {
			return EntryErrorf(e.ID, "town catalog %s: town %d names hook set %q the campaign does not answer", c.Catalog, e.ID, e.Hooks)
		}
		if !slices.Contains(vocab.OfferSources, e.Offers) {
			return EntryErrorf(e.ID, "town catalog %s: town %d names offer source %q the campaign does not answer", c.Catalog, e.ID, e.Offers)
		}
	}
	return nil
}

// Validate checks the catalog's own rules: a name, at least one town, and
// for each town a positive unique ID, a description, a hook set, an offer
// source, exactly one well-formed map point source, no negative opening
// mission and positive missions no other town lists. A refusal of one entry
// is an EntryError.
func (c *Catalog) Validate() error {
	if c.Catalog == "" {
		return fmt.Errorf("town catalog: no name")
	}
	if len(c.Towns) == 0 {
		return fmt.Errorf("town catalog %s: no town", c.Catalog)
	}
	seen := map[int]bool{}
	listed := map[int]int{}
	for _, e := range c.Towns {
		bad := func(format string, args ...any) error { return EntryErrorf(e.ID, format, args...) }
		switch {
		case e.ID <= 0:
			return bad("town catalog %s: town ID %d is not positive", c.Catalog, e.ID)
		case seen[e.ID]:
			return bad("town catalog %s: town ID %d repeats", c.Catalog, e.ID)
		case e.Description == "":
			return bad("town catalog %s: town %d names no description", c.Catalog, e.ID)
		case e.Hooks == "":
			return bad("town catalog %s: town %d names no hook set", c.Catalog, e.ID)
		case e.Offers == "":
			return bad("town catalog %s: town %d names no offer source", c.Catalog, e.ID)
		case e.OpensAfter < 0:
			return bad("town catalog %s: town %d opens after mission %d", c.Catalog, e.ID, e.OpensAfter)
		}
		if err := e.MapPoint.check(); err != nil {
			return bad("town catalog %s: town %d %v", c.Catalog, e.ID, err)
		}
		for _, m := range e.Missions {
			if m <= 0 {
				return bad("town catalog %s: town %d offers mission %d", c.Catalog, e.ID, m)
			}
			if other, ok := listed[m]; ok {
				return bad("town catalog %s: mission %d is listed by towns %d and %d", c.Catalog, m, other, e.ID)
			}
			listed[m] = e.ID
		}
		seen[e.ID] = true
	}
	if err := c.validateOffers(seen); err != nil {
		return err
	}
	return c.Mercenaries.validate(c.Catalog)
}

// Town answers the entry of town id.
func (c *Catalog) Town(id int) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	for _, e := range c.Towns {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}

// IDs answers the catalog's town IDs in catalog order.
func (c *Catalog) IDs() []int {
	if c == nil {
		return nil
	}
	ids := make([]int, 0, len(c.Towns))
	for _, e := range c.Towns {
		ids = append(ids, e.ID)
	}
	return ids
}
