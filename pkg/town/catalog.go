package town

import (
	"bytes"
	"encoding/json"
	"fmt"
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
}

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
	Cite     []string `json:"cite"`
}

// MapPoint says where a town's world-map point is read from: exactly one of
// Object and Record is set.
type MapPoint struct {
	// Object is the number of the world-map registry object whose point the
	// town stands at, counted from 1.
	Object int `json:"object,omitempty"`
	// Record is true when the point is the world-map description's catalog
	// record of this town: the record of the town kind and the town's ID.
	Record bool `json:"record,omitempty"`
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
	if err := c.Validate(); err != nil {
		return nil, err
	}
	for _, e := range c.Towns {
		if !slices.Contains(vocab.HookSets, e.Hooks) {
			return nil, fmt.Errorf("town catalog %s: town %d names hook set %q the campaign does not answer", c.Catalog, e.ID, e.Hooks)
		}
		if !slices.Contains(vocab.OfferSources, e.Offers) {
			return nil, fmt.Errorf("town catalog %s: town %d names offer source %q the campaign does not answer", c.Catalog, e.ID, e.Offers)
		}
	}
	return &c, nil
}

// Validate checks the catalog's own rules: a name, at least one town, and
// for each town a positive unique ID, a description, a hook set, an offer
// source and exactly one map point source.
func (c *Catalog) Validate() error {
	if c.Catalog == "" {
		return fmt.Errorf("town catalog: no name")
	}
	if len(c.Towns) == 0 {
		return fmt.Errorf("town catalog %s: no town", c.Catalog)
	}
	seen := map[int]bool{}
	for _, e := range c.Towns {
		switch {
		case e.ID <= 0:
			return fmt.Errorf("town catalog %s: town ID %d is not positive", c.Catalog, e.ID)
		case seen[e.ID]:
			return fmt.Errorf("town catalog %s: town ID %d repeats", c.Catalog, e.ID)
		case e.Description == "":
			return fmt.Errorf("town catalog %s: town %d names no description", c.Catalog, e.ID)
		case e.Hooks == "":
			return fmt.Errorf("town catalog %s: town %d names no hook set", c.Catalog, e.ID)
		case e.Offers == "":
			return fmt.Errorf("town catalog %s: town %d names no offer source", c.Catalog, e.ID)
		case e.MapPoint.Object < 0, (e.MapPoint.Object > 0) == e.MapPoint.Record:
			return fmt.Errorf("town catalog %s: town %d must name exactly one map point source", c.Catalog, e.ID)
		}
		seen[e.ID] = true
	}
	return nil
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
