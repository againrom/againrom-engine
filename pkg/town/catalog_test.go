package town

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

var catalogVocab = CatalogVocabulary{HookSets: []string{"square-hooks"}, OfferSources: []string{"keys", "bank"}}

func TestCatalogDecodesTheEntryFormat(t *testing.T) {
	raw := `{"catalog": "fixture", "cite": ["C-1"], "towns": [
  {"id": 2, "description": "two", "hooks": "square-hooks", "offers": "bank", "map-point": {"record": true}, "cite": ["C-2"]},
  {"id": 1, "description": "one", "hooks": "square-hooks", "offers": "keys", "map-point": {"object": 3},
   "name": "One", "opens-after": 30, "missions": [31, 32], "cite": []}]}`
	c, err := DecodeCatalog([]byte(raw), catalogVocab)
	if err != nil {
		t.Fatal(err)
	}
	want := &Catalog{Catalog: "fixture", Cite: []string{"C-1"}, Towns: []Entry{
		{ID: 2, Description: "two", Hooks: "square-hooks", Offers: "bank", MapPoint: MapPoint{Record: true}, Cite: []string{"C-2"}},
		{ID: 1, Description: "one", Hooks: "square-hooks", Offers: "keys", MapPoint: MapPoint{Object: 3},
			Name: "One", OpensAfter: 30, Missions: []int{31, 32}, Cite: []string{}},
	}}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("decoded %+v, want %+v", c, want)
	}
	if got := c.IDs(); !reflect.DeepEqual(got, []int{2, 1}) {
		t.Errorf("IDs %v, want catalog order [2 1]", got)
	}
	if e, ok := c.Town(1); !ok || e.Description != "one" {
		t.Errorf("Town(1) = %+v %v", e, ok)
	}
	if _, ok := c.Town(4); ok {
		t.Error("Town(4) found a town the catalog does not list")
	}
	var none *Catalog
	if _, ok := none.Town(1); ok || none.IDs() != nil {
		t.Error("a nil catalog answers a town")
	}
}

func TestCatalogRejectsBrokenEntries(t *testing.T) {
	entry := `{"id": 1, "description": "one", "hooks": "square-hooks", "offers": "keys", "map-point": {"object": 1}, "cite": []}`
	for _, c := range []struct{ name, raw, want string }{
		{"unknown field", `{"catalog": "f", "towns": [` + entry + `], "extra": 1}`, "unknown field"},
		{"trailing data", `{"catalog": "f", "towns": [` + entry + `]} {}`, "trailing data"},
		{"no name", `{"towns": [` + entry + `]}`, "no name"},
		{"no town", `{"catalog": "f", "towns": []}`, "no town"},
		{"zero ID", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"id": 1`, `"id": 0`, 1) + `]}`, "not positive"},
		{"repeated ID", `{"catalog": "f", "towns": [` + entry + `,` + entry + `]}`, "repeats"},
		{"no description", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"one"`, `""`, 1) + `]}`, "no description"},
		{"no hook set", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"square-hooks"`, `""`, 1) + `]}`, "no hook set"},
		{"no offer source", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"keys"`, `""`, 1) + `]}`, "no offer source"},
		{"no map point", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{}`, 1) + `]}`, "exactly one map point"},
		{"two map points", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{"object": 1, "record": true}`, 1) + `]}`, "exactly one map point"},
		{"negative object", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{"object": -1, "record": true}`, 1) + `]}`, "exactly one map point"},
		{"negative opening", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"cite"`, `"opens-after": -1, "cite"`, 1) + `]}`, "opens after"},
		{"zero mission", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"cite"`, `"missions": [0], "cite"`, 1) + `]}`, "offers mission 0"},
		{"mission listed twice", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"cite"`, `"missions": [31], "cite"`, 1) + `,` +
			strings.Replace(strings.Replace(entry, `"id": 1`, `"id": 2`, 1), `"cite"`, `"missions": [31], "cite"`, 1) + `]}`, "listed by towns 1 and 2"},
		{"unknown hook set", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"square-hooks"`, `"other"`, 1) + `]}`, "hook set"},
		{"unknown offer source", `{"catalog": "f", "towns": [` + strings.Replace(entry, `"keys"`, `"other"`, 1) + `]}`, "offer source"},
		{"point and object", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{"object": 1, "point": [1, 2], "rect": [0, 0, 4, 4]}`, 1) + `]}`, "exactly one map point"},
		{"rect without point", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{"object": 1, "rect": [0, 0, 4, 4]}`, 1) + `]}`, "map rect without a point"},
		{"point without rect", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{"point": [1, 2]}`, 1) + `]}`, "map rect [] is not"},
		{"short point", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{"point": [1], "rect": [0, 0, 4, 4]}`, 1) + `]}`, "map point [1] is not"},
		{"negative point", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{"point": [-1, 2], "rect": [0, 0, 4, 4]}`, 1) + `]}`, "map point [-1 2] is not"},
		{"empty rect", `{"catalog": "f", "towns": [` + strings.Replace(entry, `{"object": 1}`, `{"point": [1, 2], "rect": [0, 0, 0, 4]}`, 1) + `]}`, "map rect [0 0 0 4] is not"},
	} {
		_, err := DecodeCatalog([]byte(c.raw), catalogVocab)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want one naming %q", c.name, err, c.want)
		}
	}
}

// A stated map point decodes as the point and rectangle a registry object
// states, and every refusal of one entry names its town.
func TestCatalogStatesAMapPoint(t *testing.T) {
	raw := `{"catalog": "f", "towns": [{"id": 4, "description": "one", "hooks": "square-hooks", "offers": "keys",
  "map-point": {"point": [430, 230], "rect": [410, 205, 40, 30]}, "cite": []}]}`
	c, err := DecodeCatalog([]byte(raw), catalogVocab)
	if err != nil {
		t.Fatal(err)
	}
	p, r, ok := c.Towns[0].MapPoint.Stated()
	if !ok || p.X != 430 || p.Y != 230 || r.Min.X != 410 || r.Min.Y != 205 || r.Dx() != 40 || r.Dy() != 30 {
		t.Fatalf("stated point %v rect %v %v", p, r, ok)
	}
	if _, _, ok := (MapPoint{Object: 1}).Stated(); ok {
		t.Fatal("an object point reads as stated")
	}
	_, err = DecodeCatalog([]byte(strings.Replace(raw, `[410, 205, 40, 30]`, `[410, 205, 40]`, 1)), catalogVocab)
	var ee *EntryError
	if !errors.As(err, &ee) || ee.Town != 4 {
		t.Fatalf("a broken entry's error %v names no town 4", err)
	}
}
