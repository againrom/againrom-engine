package mod

import (
	"fmt"
	"strings"

	"againrom/pkg/town"
)

// TownsFile is the data file whose records are a layer over the town
// catalog. A mod loads it from its entry script with game.data.add.
const TownsFile = "data/towns.toml"

// ModArtPrefix starts an art key of a mod's own town description that names
// a file inside the mod folder; every other key names an installed entry.
const ModArtPrefix = "mod:"

// townFields are the keys of a [[town]] table: the catalog entry's fields,
// and OffField.
var townFields = []string{"id", "description", "hooks", "offers", "map-point", "name", "opens-after", "missions", "cite", OffField}

// TownRecord is one [[town]] table: the town ID it names and the catalog
// entry fields it sets, in the catalog's entry format. Entry holds the
// values of the fields named in Fields; the others are zero.
type TownRecord struct {
	Source
	Dir    string
	Entry  town.Entry
	Fields []PatchField
	Off    bool
	// Description is set when the record's description is a file of the
	// mod: its path inside the mod folder and its bytes. Entry.Description
	// is then the mod's ID and that path, the description's name in the
	// merged catalog.
	Description *TownDescriptionFile
}

// TownDescriptionFile is a town description a mod carries.
type TownDescriptionFile struct {
	Path string
	Data []byte
	// Dir is the mod folder its "mod:" art keys read from.
	Dir string
}

// TownData is what the mods of a set layer over the town catalog, in load
// order, with the load order the mods state.
type TownData struct {
	Records []TownRecord
	Order   Precedence
}

// Empty reports that no mod layers the town catalog.
func (d TownData) Empty() bool { return len(d.Records) == 0 }

// ModDescriptionName is the name of a description file of mod id in a
// merged catalog.
func ModDescriptionName(id, path string) string { return id + ":" + path }

// IsTownDescriptionFile reports whether a description value names a file of
// the mod rather than an installed description.
func IsTownDescriptionFile(v string) bool { return strings.HasSuffix(strings.ToLower(v), ".json") }

// ParseTowns reads data/towns.toml of the mod id. A refusal names the file
// and the line. A description that names a file is not read here.
func ParseTowns(id, file string, data []byte) ([]TownRecord, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return nil, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return nil, bad(p.line, "%s is outside a [[town]] table", p.key)
	}
	var out []TownRecord
	seen := map[int]int{}
	for _, t := range tables[1:] {
		if t.name != "town" || !t.array {
			return nil, bad(t.line, "unknown table %q (this file holds [[town]])", t.name)
		}
		r, err := townRecordFrom(id, file, t)
		if err != nil {
			return nil, err
		}
		if first, dup := seen[r.Entry.ID]; dup {
			return nil, bad(t.line, "town %d is named again; its first table is at line %d", r.Entry.ID, first)
		}
		seen[r.Entry.ID] = t.line
		out = append(out, r)
	}
	return out, nil
}

func townRecordFrom(id, file string, t tomlTable) (TownRecord, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	r := TownRecord{Source: Source{Mod: id, File: file, Line: t.line}}
	hasID := false
	for _, p := range t.pairs {
		str := func() (string, error) {
			if p.val.kind != tomlString {
				return "", bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
			}
			if strings.TrimSpace(p.val.str) == "" {
				return "", bad(p.line, "%s is empty", p.key)
			}
			return p.val.str, nil
		}
		integer := func(min int64) (int, error) {
			if p.val.kind != tomlInt {
				return 0, bad(p.line, "%s must be an integer, not %s", p.key, p.val.kind)
			}
			if p.val.num < min || p.val.num > 1<<20 {
				return 0, bad(p.line, "%s is %d, outside %d..%d", p.key, p.val.num, min, 1<<20)
			}
			return int(p.val.num), nil
		}
		var err error
		e := &r.Entry
		switch p.key {
		case "id":
			hasID = true
			e.ID, err = integer(1)
		case "description":
			e.Description, err = str()
		case "hooks":
			e.Hooks, err = str()
		case "offers":
			e.Offers, err = str()
		case "name":
			e.Name, err = str()
		case "opens-after":
			e.OpensAfter, err = integer(0)
		case "missions":
			e.Missions, err = intList(file, p, 1)
		case "cite":
			e.Cite, err = stringList(file, p)
		case "map-point":
			e.MapPoint, err = townMapPoint(file, p)
		case OffField:
			if p.val.kind != tomlBool {
				err = bad(p.line, "off must be true or false, not %s", p.val.kind)
			}
			r.Off = p.val.flag
		default:
			err = bad(p.line, "unknown key %q in [[town]] (keys: %s)", p.key, strings.Join(townFields, ", "))
		}
		if err != nil {
			return TownRecord{}, err
		}
		if p.key != "id" {
			r.Fields = append(r.Fields, PatchField{Name: p.key, Line: p.line})
		}
	}
	switch {
	case !hasID:
		return TownRecord{}, bad(t.line, "[[town]] has no id")
	case len(r.Fields) == 0:
		return TownRecord{}, bad(t.line, "[[town]] %d changes nothing", r.Entry.ID)
	case r.Off && len(r.Fields) > 1:
		return TownRecord{}, bad(t.line, "[[town]] %d switches the town off and sets other keys; off = true stands alone", r.Entry.ID)
	}
	return r, nil
}

// Apply writes the fields the record sets into e.
func (r TownRecord) Apply(e *town.Entry) {
	e.ID = r.Entry.ID
	for _, f := range r.Fields {
		switch f.Name {
		case "description":
			e.Description = r.Entry.Description
		case "hooks":
			e.Hooks = r.Entry.Hooks
		case "offers":
			e.Offers = r.Entry.Offers
		case "map-point":
			e.MapPoint = r.Entry.MapPoint
		case "name":
			e.Name = r.Entry.Name
		case "opens-after":
			e.OpensAfter = r.Entry.OpensAfter
		case "missions":
			e.Missions = r.Entry.Missions
		case "cite":
			e.Cite = r.Entry.Cite
		}
	}
}

// LineOf is the line that sets field name, or the table's line.
func (r TownRecord) LineOf(name string) int {
	for _, f := range r.Fields {
		if f.Name == name {
			return f.Line
		}
	}
	return r.Line
}

func intList(file string, p tomlPair, min int64) ([]int, error) {
	bad := func(format string, args ...any) error {
		return &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf(format, args...)}
	}
	if p.val.kind != tomlArray {
		return nil, bad("%s must be an array of integers, not %s", p.key, p.val.kind)
	}
	out := []int{}
	for _, v := range p.val.list {
		if v.kind != tomlInt {
			return nil, bad("%s holds %s; use integers", p.key, v.kind)
		}
		if v.num < min || v.num > 1<<20 {
			return nil, bad("%s holds %d, outside %d..%d", p.key, v.num, min, 1<<20)
		}
		out = append(out, int(v.num))
	}
	return out, nil
}

func stringList(file string, p tomlPair) ([]string, error) {
	if p.val.kind != tomlArray {
		return nil, &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf("%s must be an array of strings, not %s", p.key, p.val.kind)}
	}
	out := []string{}
	for _, v := range p.val.list {
		if v.kind != tomlString {
			return nil, &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf("%s holds %s; use strings", p.key, v.kind)}
		}
		out = append(out, v.str)
	}
	return out, nil
}

// townMapPoint reads map-point: an inline table of object, record, or
// point with rect.
func townMapPoint(file string, p tomlPair) (town.MapPoint, error) {
	bad := func(format string, args ...any) error {
		return &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf(format, args...)}
	}
	var mp town.MapPoint
	if p.val.kind != tomlInline {
		return mp, bad("map-point must be an inline table { object = n }, { record = true } or { point = [x, y], rect = [x, y, w, h] }, not %s", p.val.kind)
	}
	for _, q := range p.val.tbl {
		q.line = p.line
		var err error
		switch q.key {
		case "object":
			if q.val.kind != tomlInt || q.val.num < 1 || q.val.num > 1<<16 {
				err = bad("map-point object must be an object number from 1")
			}
			mp.Object = int(q.val.num)
		case "record":
			if q.val.kind != tomlBool || !q.val.flag {
				err = bad("map-point record must be true")
			}
			mp.Record = true
		case "point":
			mp.Point, err = intList(file, q, 0)
		case "rect":
			mp.Rect, err = intList(file, q, 0)
		default:
			err = bad("unknown key %q in map-point (keys: object, record, point, rect)", q.key)
		}
		if err != nil {
			return town.MapPoint{}, err
		}
	}
	return mp, nil
}
