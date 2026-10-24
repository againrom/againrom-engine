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

// TavernRecord is one [[tavern]] table: the row key it names and the row
// fields it sets. Row holds the key and the values of the fields named in
// Fields.
type TavernRecord struct {
	Source
	Row    town.TavernRow
	Fields []PatchField
	Off    bool
}

// tavernKeys are the keys naming a row; tavernFields are every key of a
// [[tavern]] table. mission is the one field a record sets.
var (
	tavernKeys   = []string{"town", "chapter", "npc"}
	tavernFields = append(slices.Clone(tavernKeys), "mission", OffField)
)

// Apply writes the row's key and the fields the record sets into r.
func (r TavernRecord) Apply(row *town.TavernRow) {
	row.Town, row.Chapter, row.NPC = r.Row.Town, r.Row.Chapter, r.Row.NPC
	if slices.ContainsFunc(r.Fields, func(f PatchField) bool { return f.Name == "mission" }) {
		row.Mission = r.Row.Mission
	}
}

// TownData is what the mods of a set layer over the town catalog, in load
// order, with the load order the mods state.
type TownData struct {
	Records []TownRecord
	Tavern  []TavernRecord
	Order   Precedence
}

// Empty reports that no mod layers the town catalog.
func (d TownData) Empty() bool { return len(d.Records) == 0 && len(d.Tavern) == 0 }

// First is the source of the first record, a town's or a tavern row's.
func (d TownData) First() Source {
	if len(d.Records) > 0 {
		return d.Records[0].Source
	}
	if len(d.Tavern) > 0 {
		return d.Tavern[0].Source
	}
	return Source{}
}

// ModDescriptionName is the name of a description file of mod id in a
// merged catalog.
func ModDescriptionName(id, path string) string { return id + ":" + path }

// IsTownDescriptionFile reports whether a description value names a file of
// the mod rather than an installed description.
func IsTownDescriptionFile(v string) bool { return strings.HasSuffix(strings.ToLower(v), ".json") }

// ParseTowns reads data/towns.toml of the mod id: its [[town]] and its
// [[tavern]] tables. A refusal names the file and the line. A description
// that names a file is not read here.
func ParseTowns(id, file string, data []byte) ([]TownRecord, []TavernRecord, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return nil, nil, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return nil, nil, bad(p.line, "%s is outside a [[town]] table (or a [[tavern]] table)", p.key)
	}
	var out []TownRecord
	var rows []TavernRecord
	seen := map[int]int{}
	seenRow := map[town.TavernKey]int{}
	for _, t := range tables[1:] {
		switch {
		case t.name == "tavern" && t.array:
			r, err := tavernRecordFrom(id, file, t)
			if err != nil {
				return nil, nil, err
			}
			if first, dup := seenRow[r.Row.Key()]; dup {
				return nil, nil, bad(t.line, "tavern row %v is named again; its first table is at line %d", r.Row.Key(), first)
			}
			seenRow[r.Row.Key()] = t.line
			rows = append(rows, r)
			continue
		case t.name != "town" || !t.array:
			return nil, nil, bad(t.line, "unknown table %q (this file holds [[town]] and [[tavern]])", t.name)
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

// tavernRecordFrom reads one [[tavern]] table: town, chapter and npc name
// the row, mission sets its mission and off switches it off.
func tavernRecordFrom(id, file string, t tomlTable) (TavernRecord, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	r := TavernRecord{Source: Source{Mod: id, File: file, Line: t.line}}
	named := map[string]bool{}
	for _, p := range t.pairs {
		var err error
		switch p.key {
		case "town":
			r.Row.Town, err = townInt(file, p, 1)
		case "chapter":
			r.Row.Chapter, err = townInt(file, p, 1)
		case "npc":
			r.Row.NPC, err = townInt(file, p, 0)
		case "mission":
			r.Row.Mission, err = townInt(file, p, 0)
		case OffField:
			if p.val.kind != tomlBool {
				err = bad(p.line, "off must be true or false, not %s", p.val.kind)
			}
			r.Off = p.val.flag
		default:
			err = bad(p.line, "unknown key %q in [[tavern]] (keys: %s; a row sets mission and off)", p.key, strings.Join(tavernFields, ", "))
		}
		if err != nil {
			return TavernRecord{}, err
		}
		named[p.key] = true
		if !slices.Contains(tavernKeys, p.key) {
			r.Fields = append(r.Fields, PatchField{Name: p.key, Line: p.line})
		}
	}
	for _, k := range tavernKeys {
		if !named[k] {
			return TavernRecord{}, bad(t.line, "[[tavern]] has no %s; town, chapter and npc name the row", k)
		}
	}
	switch {
	case len(r.Fields) == 0:
		return TavernRecord{}, bad(t.line, "[[tavern]] %v changes nothing", r.Row.Key())
	case r.Off && len(r.Fields) > 1:
		return TavernRecord{}, bad(t.line, "[[tavern]] %v switches the row off and sets other keys; off = true stands alone", r.Row.Key())
	}
	return r, nil
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
// rows follows as a [[tavern]] table with its key, its mission and off =
// false, read back the same way.
func WriteTowns(head []string, entries []town.Entry, rows ...town.TavernRow) ([]byte, error) {
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
		fmt.Fprintf(&b, "\n[[tavern]]\n%-11s = %d\n%-11s = %d\n%-11s = %d\n%-11s = %d\n%-11s = false\n",
			"town", r.Town, "chapter", r.Chapter, "npc", r.NPC, "mission", r.Mission, OffField)
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
		var got town.TavernRow
		r.Apply(&got)
		if r.Off || got != rows[i] {
			return nil, fmt.Errorf("tavern row %v of the written %s does not read back as its row", rows[i].Key(), TownsFile)
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
