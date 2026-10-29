package mod

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/town"
)

// TownsFile is the data file whose records are a layer over the town
// catalog. A mod loads it from its entry script with game.data.add.
const TownsFile = "data/towns.toml"

// ModArtPrefix starts an art key of a mod's own town description that names
// a file inside the mod folder; every other key names an installed entry.
const ModArtPrefix = "mod:"

// townKey is one key of a [[town]] table besides id and off: how the
// decoder reads it into an entry, how a record copies it into a catalog
// entry, and how a dump writes an entry's value as TOML text. write answers
// false when the entry holds no value the decoder accepts.
type townKey struct {
	name  string
	read  func(file string, p tomlPair, e *town.Entry) error
	copy  func(dst *town.Entry, src town.Entry)
	write func(e town.Entry) (string, bool)
}

// townKeys are the catalog entry's fields a [[town]] table sets, in the
// order a dump writes them. The decoder, Apply and WriteTowns read only this
// list.
var townKeys = []townKey{
	townStringKey("description", func(e *town.Entry) *string { return &e.Description }),
	townStringKey("hooks", func(e *town.Entry) *string { return &e.Hooks }),
	townStringKey("offers", func(e *town.Entry) *string { return &e.Offers }),
	{
		name: "map-point",
		read: func(file string, p tomlPair, e *town.Entry) (err error) {
			e.MapPoint, err = townMapPoint(file, p)
			return err
		},
		copy:  func(dst *town.Entry, src town.Entry) { dst.MapPoint = src.MapPoint },
		write: func(e town.Entry) (string, bool) { return writeTownMapPoint(e.MapPoint) },
	},
	townStringKey("name", func(e *town.Entry) *string { return &e.Name }),
	{
		name: "opens-after",
		read: func(file string, p tomlPair, e *town.Entry) (err error) {
			e.OpensAfter, err = townInt(file, p, 0)
			return err
		},
		copy:  func(dst *town.Entry, src town.Entry) { dst.OpensAfter = src.OpensAfter },
		write: func(e town.Entry) (string, bool) { return strconv.Itoa(e.OpensAfter), e.OpensAfter >= 0 },
	},
	{
		name: "missions",
		read: func(file string, p tomlPair, e *town.Entry) (err error) {
			e.Missions, err = intList(file, p, 1)
			return err
		},
		copy: func(dst *town.Entry, src town.Entry) { dst.Missions = src.Missions },
		write: func(e town.Entry) (string, bool) {
			return writeIntList(e.Missions), !slices.ContainsFunc(e.Missions, func(n int) bool { return n < 1 })
		},
	},
	{
		name: "cite",
		read: func(file string, p tomlPair, e *town.Entry) (err error) {
			e.Cite, err = stringList(file, p)
			return err
		},
		copy:  func(dst *town.Entry, src town.Entry) { dst.Cite = src.Cite },
		write: func(e town.Entry) (string, bool) { return writeStringList(e.Cite) },
	},
}

// townFields are the keys of a [[town]] table: id, townKeys and OffField.
var townFields = func() []string {
	out := []string{"id"}
	for _, k := range townKeys {
		out = append(out, k.name)
	}
	return append(out, OffField)
}()

// townKeyNamed is the key of townKeys named name.
func townKeyNamed(name string) (townKey, bool) {
	i := slices.IndexFunc(townKeys, func(k townKey) bool { return k.name == name })
	if i < 0 {
		return townKey{}, false
	}
	return townKeys[i], true
}

// townStringKey is a key holding a non-empty string field of the entry.
func townStringKey(name string, field func(*town.Entry) *string) townKey {
	return townKey{
		name: name,
		read: func(file string, p tomlPair, e *town.Entry) (err error) {
			*field(e), err = townString(file, p)
			return err
		},
		copy: func(dst *town.Entry, src town.Entry) { *field(dst) = *field(&src) },
		write: func(e town.Entry) (string, bool) {
			s := *field(&e)
			if strings.TrimSpace(s) == "" {
				return "", false
			}
			return writeTOMLString(s)
		},
	}
}

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
	// Dir is the mod's folder or RES file its "mod:" art keys read from.
	Dir string
}

// OfferRecord is one [[tavern]] or [[inn]] table: the offer table it is
// written in, the row key it names and the row fields it sets. Row holds
// the key and the values of the fields named in Fields.
type OfferRecord struct {
	Source
	Table  town.OfferTable
	Row    town.OfferRow
	Fields []PatchField
	Off    bool
}

// Key is the record's row key.
func (r OfferRecord) Key() town.OfferKey { return r.Table.Key(r.Row) }

// offerKey is one key of an offer table: its name, whether it names the
// row, how the decoder reads it into a row, how a record copies it into a
// row and how a dump writes a row's value.
type offerKey struct {
	name  string
	key   bool
	read  func(file string, p tomlPair, r *town.OfferRow) error
	copy  func(dst *town.OfferRow, src town.OfferRow)
	write func(r town.OfferRow) string
}

// offerInt is a key holding an integer field of the row, from min.
func offerInt(name string, key bool, min int64, field func(*town.OfferRow) *int) offerKey {
	return offerKey{name: name, key: key,
		read: func(file string, p tomlPair, r *town.OfferRow) (err error) {
			*field(r), err = townInt(file, p, min)
			return err
		},
		copy:  func(dst *town.OfferRow, src town.OfferRow) { *field(dst) = *field(&src) },
		write: func(r town.OfferRow) string { return strconv.Itoa(*field(&r)) },
	}
}

// offerList is a key holding a list field of the row, each value from min.
func offerList(name string, min int64, field func(*town.OfferRow) *[]int) offerKey {
	return offerKey{name: name,
		read: func(file string, p tomlPair, r *town.OfferRow) (err error) {
			*field(r), err = townInts(file, p, min)
			return err
		},
		copy: func(dst *town.OfferRow, src town.OfferRow) { *field(dst) = slices.Clone(*field(&src)) },
	}
}

// offerTables are the keys of each offer table, in the order a dump writes
// them. A tavern table's chapter is the row's one stage and its mission the
// topic, which also gives the kind; an inn table names every field.
var offerTables = map[town.OfferTable][]offerKey{
	town.TavernTable: {
		offerInt("town", true, 1, func(r *town.OfferRow) *int { return &r.Town }),
		{name: "chapter", key: true,
			read: func(file string, p tomlPair, r *town.OfferRow) error {
				n, err := townInt(file, p, 1)
				r.Stages = []int{n}
				return err
			},
			copy:  func(dst *town.OfferRow, src town.OfferRow) { dst.Stages = slices.Clone(src.Stages) },
			write: func(r town.OfferRow) string { return strconv.Itoa(r.Chapter()) }},
		offerInt("npc", true, 0, func(r *town.OfferRow) *int { return &r.NPC }),
		{name: "mission",
			read: func(file string, p tomlPair, r *town.OfferRow) (err error) {
				r.Topic, err = townInt(file, p, 0)
				r.Kind = town.MissionKind(r.Topic)
				return err
			},
			copy:  func(dst *town.OfferRow, src town.OfferRow) { dst.Topic, dst.Kind = src.Topic, src.Kind },
			write: func(r town.OfferRow) string { return strconv.Itoa(r.Topic) }},
	},
	town.InnTable: {
		offerInt("topic", true, 0, func(r *town.OfferRow) *int { return &r.Topic }),
		offerInt("npc", true, 1, func(r *town.OfferRow) *int { return &r.NPC }),
		offerInt("kind", false, 0, func(r *town.OfferRow) *int { return &r.Kind }),
		offerInt("town", false, 0, func(r *town.OfferRow) *int { return &r.Town }),
		offerList("stages", 1, func(r *town.OfferRow) *[]int { return &r.Stages }),
		offerInt("after", false, 0, func(r *town.OfferRow) *int { return &r.After }),
		offerList("zero", 0, func(r *town.OfferRow) *[]int { return &r.Zero }),
		offerList("nonzero", 0, func(r *town.OfferRow) *[]int { return &r.Nonzero }),
		offerList("one", 0, func(r *town.OfferRow) *[]int { return &r.One }),
		offerList("any", 0, func(r *town.OfferRow) *[]int { return &r.Any }),
	},
}

// offerKeyNames are the names of table's keys that name the row, or of
// every key and OffField.
func offerKeyNames(table town.OfferTable, keysOnly bool) []string {
	var out []string
	for _, k := range offerTables[table] {
		if k.key || !keysOnly {
			out = append(out, k.name)
		}
	}
	if !keysOnly {
		out = append(out, OffField)
	}
	return out
}

// Apply writes the row's key and the fields the record sets into row.
func (r OfferRecord) Apply(row *town.OfferRow) {
	for _, k := range offerTables[r.Table] {
		if k.key || slices.ContainsFunc(r.Fields, func(f PatchField) bool { return f.Name == k.name }) {
			k.copy(row, r.Row)
		}
	}
}

// TownData is what the mods of a set layer over the town catalog, in load
// order, with the load order the mods state.
type TownData struct {
	Records []TownRecord
	Offers  []OfferRecord
	Order   Precedence
}

// Empty reports that no mod layers the town catalog.
func (d TownData) Empty() bool { return len(d.Records) == 0 && len(d.Offers) == 0 }

// First is the source of the first record, a town's or an offer row's.
func (d TownData) First() Source {
	if len(d.Records) > 0 {
		return d.Records[0].Source
	}
	if len(d.Offers) > 0 {
		return d.Offers[0].Source
	}
	return Source{}
}

// OfferRows is the number of the offer records written in table.
func (d TownData) OfferRows(table town.OfferTable) int {
	n := 0
	for _, r := range d.Offers {
		if r.Table == table {
			n++
		}
	}
	return n
}

// ModDescriptionName is the name of a description file of mod id in a
// merged catalog.
func ModDescriptionName(id, path string) string { return id + ":" + path }

// IsTownDescriptionFile reports whether a description value names a file of
// the mod rather than an installed description.
func IsTownDescriptionFile(v string) bool { return strings.HasSuffix(strings.ToLower(v), ".json") }

// ParseTowns reads data/towns.toml of the mod id: its [[town]] tables and
// its offer rows, [[tavern]] and [[inn]] tables. A refusal names the file
// and the line. A description that names a file is not read here.
func ParseTowns(id, file string, data []byte) ([]TownRecord, []OfferRecord, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return nil, nil, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return nil, nil, bad(p.line, "%s is outside a [[town]] table (or a [[tavern]] or [[inn]] table)", p.key)
	}
	var out []TownRecord
	var rows []OfferRecord
	seen := map[int]int{}
	seenRow := map[town.OfferKey]int{}
	for _, t := range tables[1:] {
		if _, ok := offerTables[town.OfferTable(t.name)]; ok && t.array {
			r, err := offerRecordFrom(id, file, town.OfferTable(t.name), t)
			if err != nil {
				return nil, nil, err
			}
			if first, dup := seenRow[r.Key()]; dup {
				return nil, nil, bad(t.line, "%s row %v is named again; its first table is at line %d", r.Table, r.Key(), first)
			}
			seenRow[r.Key()] = t.line
			rows = append(rows, r)
			continue
		}
		if t.name != "town" || !t.array {
			return nil, nil, bad(t.line, "unknown table %q (this file holds [[town]], [[tavern]] and [[inn]])", t.name)
		}
		r, err := townRecordFrom(id, file, t)
		if err != nil {
			return nil, nil, err
		}
		if first, dup := seen[r.Entry.ID]; dup {
			return nil, nil, bad(t.line, "town %d is named again; its first table is at line %d", r.Entry.ID, first)
		}
		seen[r.Entry.ID] = t.line
		out = append(out, r)
	}
	return out, rows, nil
}

// offerRecordFrom reads one table of offer table table: the keys that name
// the row, the fields the table sets and off, which switches the row off
// and stands alone.
func offerRecordFrom(id, file string, table town.OfferTable, t tomlTable) (OfferRecord, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	r := OfferRecord{Source: Source{Mod: id, File: file, Line: t.line}, Table: table}
	keys := offerKeyNames(table, true)
	named := map[string]bool{}
	for _, p := range t.pairs {
		var err error
		i := slices.IndexFunc(offerTables[table], func(k offerKey) bool { return k.name == p.key })
		switch {
		case i >= 0:
			err = offerTables[table][i].read(file, p, &r.Row)
		case p.key == OffField:
			if p.val.kind != tomlBool {
				err = bad(p.line, "off must be true or false, not %s", p.val.kind)
			}
			r.Off = p.val.flag
		default:
			err = bad(p.line, "unknown key %q in [[%s]] (keys: %s)", p.key, table, strings.Join(offerKeyNames(table, false), ", "))
		}
		if err != nil {
			return OfferRecord{}, err
		}
		named[p.key] = true
		if !slices.Contains(keys, p.key) {
			r.Fields = append(r.Fields, PatchField{Name: p.key, Line: p.line})
		}
	}
	for _, k := range keys {
		if !named[k] {
			return OfferRecord{}, bad(t.line, "[[%s]] has no %s; %s name the row", table, k, proseList(keys))
		}
	}
	switch {
	case len(r.Fields) == 0:
		return OfferRecord{}, bad(t.line, "[[%s]] %v changes nothing", table, r.Key())
	case r.Off && len(r.Fields) > 1:
		return OfferRecord{}, bad(t.line, "[[%s]] %v switches the row off and sets other keys; off = true stands alone", table, r.Key())
	}
	return r, nil
}

// proseList is names joined as prose: "a and b", "a, b and c".
func proseList(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// townInts reads an array of integers, each from min to the bound.
func townInts(file string, p tomlPair, min int64) ([]int, error) {
	if p.val.kind != tomlArray {
		return nil, &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf("%s must be an array of integers, not %s", p.key, p.val.kind)}
	}
	out := []int{}
	for _, v := range p.val.list {
		n, err := townInt(file, tomlPair{key: p.key, val: v, line: p.line}, min)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
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
		var err error
		switch k, ok := townKeyNamed(p.key); {
		case p.key == "id":
			hasID = true
			r.Entry.ID, err = townInt(file, p, 1)
		case p.key == OffField:
			if p.val.kind != tomlBool {
				err = bad(p.line, "off must be true or false, not %s", p.val.kind)
			}
			r.Off = p.val.flag
		case ok:
			err = k.read(file, p, &r.Entry)
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

// townString reads a non-empty string value.
func townString(file string, p tomlPair) (string, error) {
	switch {
	case p.val.kind != tomlString:
		return "", &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf("%s must be a string, not %s", p.key, p.val.kind)}
	case strings.TrimSpace(p.val.str) == "":
		return "", &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf("%s is empty", p.key)}
	}
	return p.val.str, nil
}

// townInt reads an integer value from min to 1<<20.
func townInt(file string, p tomlPair, min int64) (int, error) {
	switch {
	case p.val.kind != tomlInt:
		return 0, &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf("%s must be an integer, not %s", p.key, p.val.kind)}
	case p.val.num < min || p.val.num > 1<<20:
		return 0, &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf("%s is %d, outside %d..%d", p.key, p.val.num, min, 1<<20)}
	}
	return int(p.val.num), nil
}

// Apply writes the fields the record sets into e.
func (r TownRecord) Apply(e *town.Entry) {
	e.ID = r.Entry.ID
	for _, f := range r.Fields {
		if k, ok := townKeyNamed(f.Name); ok {
			k.copy(e, r.Entry)
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

// WriteTowns writes entries as a towns file ParseTowns reads: the lines of
// head as comments, then one [[town]] table per entry holding its id, every
// key of townKeys the entry has a value for, and off = false. The file is
// read back through ParseTowns, and an entry it does not give back unchanged
// is an error, so the file never holds a value its entry does not. Each of
// rows, tavern rows, follows as a [[tavern]] table with its key, its mission
// and off = false, read back the same way.
func WriteTowns(head []string, entries []town.Entry, rows ...town.OfferRow) ([]byte, error) {
	var b strings.Builder
	for _, l := range head {
		if strings.ContainsAny(l, "\r\n") {
			return nil, fmt.Errorf("a comment line of %s holds a line break", TownsFile)
		}
		b.WriteString(strings.TrimRight("# "+l, " ") + "\n")
	}
	for i, e := range entries {
		if i > 0 || len(head) > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "[[town]]\n%-11s = %d\n", "id", e.ID)
		for _, k := range townKeys {
			if v, ok := k.write(e); ok {
				fmt.Fprintf(&b, "%-11s = %s\n", k.name, v)
			}
		}
		fmt.Fprintf(&b, "%-11s = false\n", OffField)
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "\n[[%s]]\n", town.TavernTable)
		for _, k := range offerTables[town.TavernTable] {
			fmt.Fprintf(&b, "%-11s = %s\n", k.name, k.write(r))
		}
		fmt.Fprintf(&b, "%-11s = false\n", OffField)
	}
	out := []byte(b.String())
	records, back, err := ParseTowns("", TownsFile, out)
	if err != nil {
		return nil, fmt.Errorf("the written %s does not read back: %w", TownsFile, err)
	}
	if len(records) != len(entries) {
		return nil, fmt.Errorf("the written %s reads back %d towns, not %d", TownsFile, len(records), len(entries))
	}
	for i, r := range records {
		var got town.Entry
		r.Apply(&got)
		if r.Off || !sameTownEntry(got, entries[i]) {
			return nil, fmt.Errorf("town %d of the written %s does not read back as its entry", entries[i].ID, TownsFile)
		}
	}
	if len(back) != len(rows) {
		return nil, fmt.Errorf("the written %s reads back %d tavern rows, not %d", TownsFile, len(back), len(rows))
	}
	for i, r := range back {
		var got town.OfferRow
		r.Apply(&got)
		if r.Off || !got.Equal(rows[i]) {
			return nil, fmt.Errorf("tavern row %v of the written %s does not read back as its row", town.TavernTable.Key(rows[i]), TownsFile)
		}
	}
	return out, nil
}

// sameTownEntry reports whether a and b hold the same values, an empty list
// the same as none.
func sameTownEntry(a, b town.Entry) bool {
	norm := func(e town.Entry) town.Entry {
		for _, l := range []*[]int{&e.Missions, &e.MapPoint.Point, &e.MapPoint.Rect} {
			if len(*l) == 0 {
				*l = nil
			}
		}
		if len(e.Cite) == 0 {
			e.Cite = nil
		}
		return e
	}
	return reflect.DeepEqual(norm(a), norm(b))
}

// writeTOMLString is s as a basic string readString reads back; false for
// text that is not UTF-8 or holds a control character other than tab.
func writeTOMLString(s string) (string, bool) {
	if !utf8.ValidString(s) {
		return "", false
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			return "", false
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String(), true
}

func writeIntList(list []int) string {
	parts := make([]string, len(list))
	for i, n := range list {
		parts[i] = strconv.Itoa(n)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func writeStringList(list []string) (string, bool) {
	parts := make([]string, len(list))
	for i, s := range list {
		q, ok := writeTOMLString(s)
		if !ok {
			return "", false
		}
		parts[i] = q
	}
	return "[" + strings.Join(parts, ", ") + "]", true
}

// writeTownMapPoint is p as the inline table townMapPoint reads; false when
// p names no source.
func writeTownMapPoint(p town.MapPoint) (string, bool) {
	var parts []string
	if p.Object != 0 {
		parts = append(parts, "object = "+strconv.Itoa(p.Object))
	}
	if p.Record {
		parts = append(parts, "record = true")
	}
	if p.Point != nil {
		parts = append(parts, "point = "+writeIntList(p.Point))
	}
	if p.Rect != nil {
		parts = append(parts, "rect = "+writeIntList(p.Rect))
	}
	if len(parts) == 0 {
		return "", false
	}
	return "{ " + strings.Join(parts, ", ") + " }", true
}
