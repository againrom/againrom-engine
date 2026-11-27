// Package mod finds mod folders and reads their manifests.
//
// A mod is a folder named by its id that holds a mod.toml. manifest.go reads
// its keys by the table ManifestKeys. settings.go reads settings.toml, order.go checks and orders a set of
// mods, digest.go fingerprints a mod folder and a mod set. Running a mod's
// script is the business of another package; this one has no effect on the
// game.
//
// The manifest is read by a deliberately small TOML subset:
//
//   - blank lines and lines whose first non-blank character is '#';
//   - "key = value" lines before the first table header, where a key is made of
//     letters, digits, '_' and '-';
//   - values that are a basic string ("..." with the escapes \" \\ \n \r \t), a
//     literal string ('...'), or a one-line array of such strings, each followed
//     by an optional '#' comment;
//   - a table header ("[name]") ends the top-level keys; the lines after it
//     are read by the same rules and then ignored, so a malformed one is an
//     error with its line.
//
// A top-level key that is not one of ManifestKeys, or one given twice, is an
// error. id, title and version are required.
package mod

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"againrom/pkg/base"
)

// utf8BOM is the byte order mark some editors put at the start of a file.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// maxManifest bounds what the reader will load.
const maxManifest = 64 << 10

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// ValidID reports whether id can name a mod folder.
func ValidID(id string) bool { return idPattern.MatchString(id) }

// Manifest is the part of mod.toml this package reads.
type Manifest struct {
	ID                  string
	Title               string
	Version             string
	Author, Description string
	AppliesTo           []string

	// API is the mod interface version the mod was written for; 1 when absent.
	API int
	// Requires, Conflicts and LoadAfter name other mods by id. A Requires
	// entry may carry a version constraint ("core-rules>=1.0").
	Requires  []string
	Conflicts []string
	LoadAfter []string
	// Entry is the Starlark file run at start, relative to the mod folder; empty
	// for a mod without one. The manifest default is main.star when that file
	// exists, applied by the loader, not here.
	Entry string
}

// ParseManifest reads a manifest; authoring also checks metadata and IDs.
func ParseManifest(data []byte, authoring ...bool) (Manifest, error) {
	strict := len(authoring) != 0 && authoring[0]
	if len(data) > maxManifest {
		return Manifest{}, fmt.Errorf("manifest is larger than %d bytes", maxManifest)
	}
	var m Manifest
	seen := map[string]bool{}
	tables, err := parseTOMLWith(data, tomlOptions{})
	if err != nil {
		return Manifest{}, err
	}
	for _, p := range tables[0].pairs {
		row, ok := ManifestKeyNamed(p.key)
		if !ok {
			return Manifest{}, fmt.Errorf("line %d: %q is not a mod.toml key; the keys are %s", p.line, p.key, strings.Join(manifestKeyNames(), ", "))
		}
		seen[p.key] = true
		value, err := manifestValue(row, p)
		if err != nil {
			if !strict && row.Lenient {
				continue
			}
			return Manifest{}, fmt.Errorf("line %d: %s: %w", p.line, p.key, err)
		}
		row.Set(&m, value)
	}
	for _, f := range []struct{ name, v string }{{"id", m.ID}, {"title", m.Title}, {"version", m.Version}} {
		if strings.TrimSpace(f.v) == "" {
			return Manifest{}, fmt.Errorf("%s is missing or empty", f.name)
		}
	}
	if !ValidID(m.ID) {
		return Manifest{}, fmt.Errorf("id %q is not valid: use 1 to 64 of a-z, 0-9, '.', '_', '-', starting with a letter or digit", m.ID)
	}
	if !seen["applies-to"] {
		m.AppliesTo = []string{base.AllBases}
	}
	if !seen["api"] {
		m.API = 1
	}
	if len(m.AppliesTo) == 0 {
		return Manifest{}, errors.New("applies-to is empty")
	}
	if !strict {
		return m, nil
	}
	for _, value := range m.Requires {
		if _, err := ParseDep(value); err != nil {
			return Manifest{}, fmt.Errorf("requires: %w", err)
		}
	}
	for _, list := range []struct {
		key string
		ids []string
	}{{"conflicts", m.Conflicts}, {"load-after", m.LoadAfter}} {
		for _, id := range list.ids {
			if !ValidID(id) {
				return Manifest{}, fmt.Errorf("%s: %q is not a mod id", list.key, id)
			}
		}
	}
	return m, nil
}

// manifestValue is the value of pair p as the key row wants it: a string, an
// int, or a list of strings, a lone string being a list of one.
func manifestValue(row ManifestKey, p tomlPair) (any, error) {
	v := p.val
	switch row.Value {
	case ManifestString:
		if v.kind != tomlString {
			return nil, errors.New("expected a quoted string")
		}
		return v.str, nil
	case ManifestInt:
		if v.kind != tomlInt || int64(int(v.num)) != v.num {
			return nil, errors.New("expected an integer")
		}
		return int(v.num), nil
	}
	if p.last > p.line && !row.Continued {
		return nil, errors.New("the array must be on one line")
	}
	if v.kind == tomlString {
		return []string{v.str}, nil
	}
	if v.kind != tomlArray {
		return nil, errors.New("expected a quoted string or an array of them")
	}
	var out []string
	for _, item := range v.list {
		if item.kind != tomlString {
			return nil, errors.New("expected a quoted string")
		}
		out = append(out, item.str)
	}
	return out, nil
}

func validKey(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// scanLine reads one line s, whose brackets open at depth, outside strings and
// comments. It is the one bracket, quote and comment scanner of the manifest
// and the data files. It returns the code before the comment, the depth after
// that code, and closeAt: the offset just past the ']' or quote that returns
// the value to depth zero, or -1 when the value does not close on this line.
func scanLine(s string, depth int) (string, int, int) {
	var quote byte
	closeAt := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote == '"' && c == '\\':
			i++
		case quote != 0:
			if c == quote {
				quote = 0
				if depth == 0 && closeAt < 0 {
					closeAt = i + 1
				}
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return s[:i], depth, closeAt
		case c == '[':
			depth++
		case c == ']':
			depth--
			if depth == 0 && closeAt < 0 {
				closeAt = i + 1
			}
		}
	}
	return s, depth, closeAt
}

// continueList is the one continuation step of the readers and the manifest
// writer: first, the text after a key's "=" on lines[n], and the lines after
// it while its brackets stay open. It returns the code of those lines without
// comments, joined by a space; the index of the last line used; tail, the text
// after the value's end on that span, the end being the first depth-zero close
// or else the last code byte; and whether the brackets are closed.
func continueList(first string, lines []string, n int) (string, int, string, bool) {
	code, depth, closeAt := scanLine(first, 0)
	closeLine := -1
	if closeAt >= 0 {
		closeLine = n
	}
	end, endLine := len(strings.TrimRight(code, " \t\r\n")), n
	code = strings.TrimSpace(code)
	last := n
	for depth > 0 && last+1 < len(lines) {
		last++
		var more string
		var at int
		more, depth, at = scanLine(lines[last], depth)
		if closeLine < 0 && at >= 0 {
			closeLine, closeAt = last, at
		}
		code = code + " " + strings.TrimSpace(more)
		if e := len(strings.TrimRight(more, " \t\r\n")); e > 0 {
			end, endLine = e, last
		}
	}
	raw := func(k int) string {
		if k == n {
			return first
		}
		return lines[k]
	}
	var tail string
	if closeLine >= 0 {
		tail = raw(closeLine)[closeAt:] + strings.Join(lines[closeLine+1:last+1], "")
	} else {
		tail = raw(endLine)[end:] + strings.Join(lines[endLine+1:last+1], "")
	}
	return code, last, tail, depth <= 0
}

func onlyComment(rest string) error {
	rest = strings.TrimSpace(rest)
	if rest != "" && rest[0] != '#' {
		return fmt.Errorf("unexpected text %q after the value", rest)
	}
	return nil
}

// stringEscapes are the escapes a basic string holds: the byte after the
// backslash and the byte it stands for.
var stringEscapes = [...][2]byte{{'"', '"'}, {'\\', '\\'}, {'n', '\n'}, {'r', '\r'}, {'t', '\t'}}

// writeString is s as a basic string readString reads back.
func writeString(s string) (string, error) {
	if !utf8.ValidString(s) {
		return "", errors.New("string is not valid UTF-8")
	}
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if k := slices.IndexFunc(stringEscapes[:], func(e [2]byte) bool { return e[1] == c }); k >= 0 {
			b.WriteByte('\\')
			b.WriteByte(stringEscapes[k][0])
			continue
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String(), nil
}

func writeStringList(list []string) (string, error) {
	parts := make([]string, len(list))
	for i, s := range list {
		q, err := writeString(s)
		if err != nil {
			return "", err
		}
		parts[i] = q
	}
	return "[" + strings.Join(parts, ", ") + "]", nil
}

// readString reads a basic or literal string at the start of v.
func readString(v string) (value, rest string, err error) {
	if v == "" {
		return "", "", errors.New("no value")
	}
	switch v[0] {
	case '\'':
		end := strings.IndexByte(v[1:], '\'')
		if end < 0 {
			return "", "", errors.New("unterminated string")
		}
		out := v[1 : 1+end]
		if !utf8.ValidString(out) {
			return "", "", errors.New("string is not valid UTF-8")
		}
		return out, v[2+end:], nil
	case '"':
		var b strings.Builder
		for i := 1; i < len(v); i++ {
			switch c := v[i]; c {
			case '"':
				out := b.String()
				if !utf8.ValidString(out) {
					return "", "", errors.New("string is not valid UTF-8")
				}
				return out, v[i+1:], nil
			case '\\':
				i++
				if i >= len(v) {
					return "", "", errors.New("unterminated string")
				}
				k := slices.IndexFunc(stringEscapes[:], func(e [2]byte) bool { return e[0] == v[i] })
				if k < 0 {
					return "", "", fmt.Errorf("unsupported escape \\%c", v[i])
				}
				b.WriteByte(stringEscapes[k][1])
			default:
				b.WriteByte(c)
			}
		}
		return "", "", errors.New("unterminated string")
	}
	return "", "", errors.New("expected a quoted string")
}

// Entry is one mod, a folder or a packed RES file, and what reading it gave.
type Entry struct {
	// Folder is the mod's name in the mods directory: the folder's name, or
	// the id a packed mod's file name mod-<id>.res names. It is the mod's id
	// when the entry is valid.
	Folder   string
	Dir      string // full path of the folder or of the RES file
	Manifest Manifest
	Err      error // non-nil when the manifest cannot be used
}

// Load reads the mod at dir: a folder, whose name must equal the manifest
// id, or a packed mod's RES file, whose name must be mod-<id>.res.
func Load(dir string) Entry {
	e := Entry{Folder: filepath.Base(dir), Dir: dir}
	t, err := OpenTree(dir)
	what := "folder"
	// OpenTree fails only on a file: a RES it cannot read.
	if err != nil || t.Packed() {
		id, ok := packedID(e.Folder)
		if !ok {
			e.Err = fmt.Errorf("a packed mod's file is named %s<id>%s with a valid id, not %q", PackedPrefix, PackedExt, e.Folder)
			return e
		}
		e.Folder, what = id, "file"
	}
	if err != nil {
		e.Err = err
		return e
	}
	data, err := t.ReadFile(ManifestName)
	if err != nil {
		e.Err = err
		return e
	}
	if e.Manifest, e.Err = ParseManifest(data); e.Err == nil && e.Manifest.ID != e.Folder {
		e.Err = fmt.Errorf("manifest id %q does not match the %s name %q", e.Manifest.ID, what, filepath.Base(dir))
	}
	if e.Err == nil {
		e.Err = checkKindNames(t)
	}
	return e
}

// Scan lists the mods of modsDir, sorted by name: the subfolders that hold a
// mod.toml and the files named mod*.res. A folder and a packed file of one
// mod id are both refused, naming both. A missing modsDir lists nothing.
func Scan(modsDir string) ([]Entry, error) {
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out, packed []Entry
	for _, d := range entries {
		p := filepath.Join(modsDir, d.Name())
		switch {
		case d.IsDir():
			if _, err := os.Stat(filepath.Join(p, ManifestName)); err != nil {
				continue
			}
			out = append(out, Load(p))
		case d.Type().IsRegular() && looksPacked(d.Name()):
			packed = append(packed, Load(p))
		}
	}
	// A folder sorts before the packed file of its id.
	out = append(out, packed...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Folder < out[j].Folder })
	for i := 0; i+1 < len(out); i++ {
		if out[i].Folder != out[i+1].Folder {
			continue
		}
		err := twoForms(out[i].Folder, out[i].Dir, out[i+1].Dir)
		out[i].Err, out[i+1].Err = err, err
	}
	return out, nil
}

// twoForms refuses a mod id the mods directory holds twice: as the folder or
// packed file a and as b.
func twoForms(id, a, b string) error {
	return fmt.Errorf("mod %q is both %s and %s; keep one of them", id, a, b)
}

// Resolve loads the named mods from modsDir in the order given. A name that is
// not a valid id, is repeated, has neither a folder nor a packed file
// mod-<id>.res, has both, or whose manifest cannot be used is an error naming
// the mod.
func Resolve(modsDir string, ids []string) ([]Entry, error) {
	seen := map[string]bool{}
	var out []Entry
	for _, id := range ids {
		switch {
		case !ValidID(id):
			return nil, fmt.Errorf("mod %q: not a valid mod id", id)
		case seen[id]:
			return nil, fmt.Errorf("mod %q: named twice", id)
		}
		seen[id] = true
		dir := filepath.Join(modsDir, id)
		packed := filepath.Join(modsDir, PackedName(id))
		st, err := os.Stat(dir)
		isDir := err == nil && st.IsDir()
		st, err = os.Stat(packed)
		isPacked := err == nil && st.Mode().IsRegular()
		switch {
		case isDir && isPacked:
			return nil, twoForms(id, dir, packed)
		case isPacked:
			dir = packed
		case !isDir:
			return nil, fmt.Errorf("mod %q: no folder %s and no packed file %s", id, dir, PackedName(id))
		}
		e := Load(dir)
		if e.Err != nil {
			return nil, fmt.Errorf("mod %q: %s: %v", id, filepath.Join(dir, ManifestName), e.Err)
		}
		out = append(out, e)
	}
	return out, nil
}
