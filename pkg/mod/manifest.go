// Package mod finds mod folders and reads their manifests.
//
// A mod is a folder named by its id that holds a mod.toml. manifest.go reads
// the manifest fields the launcher and the engine need to name and order a mod:
// id, title, version, applies-to, api, requires, conflicts, load-after and
// entry. settings.go reads settings.toml, order.go checks and orders a set of
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
//   - a table header ("[name]" or "[[name]]") ends the top-level keys; every
//     line after it is ignored.
//
// Top-level keys other than those named are skipped without interpreting their
// values, except that a multi-line array is skipped to its closing bracket.
// id, title and version are required strings; applies-to is a string or an
// array of strings and defaults to ["common"]. A key given twice is an error.
package mod

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ManifestName is the file that makes a folder a mod.
const ManifestName = "mod.toml"

// utf8BOM is the byte order mark some editors put at the start of a file.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// maxManifest bounds what the reader will load.
const maxManifest = 64 << 10

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// ValidID reports whether id can name a mod folder.
func ValidID(id string) bool { return idPattern.MatchString(id) }

// Manifest is the part of mod.toml this package reads.
type Manifest struct {
	ID        string
	Title     string
	Version   string
	AppliesTo []string

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

// ParseManifest reads a manifest. The error names the offending line.
func ParseManifest(data []byte) (Manifest, error) {
	if len(data) > maxManifest {
		return Manifest{}, fmt.Errorf("manifest is larger than %d bytes", maxManifest)
	}
	var m Manifest
	seen := map[string]bool{}
	lines := strings.Split(string(bytes.TrimPrefix(data, utf8BOM)), "\n")
	for n := 0; n < len(lines); n++ {
		t := strings.TrimSpace(lines[n])
		if t == "" || t[0] == '#' {
			continue
		}
		if t[0] == '[' {
			break
		}
		eq := strings.IndexByte(t, '=')
		if eq <= 0 {
			return Manifest{}, fmt.Errorf("line %d: expected key = value", n+1)
		}
		key := strings.TrimSpace(t[:eq])
		if !validKey(key) {
			return Manifest{}, fmt.Errorf("line %d: %q is not a plain key", n+1, key)
		}
		if seen[key] {
			return Manifest{}, fmt.Errorf("line %d: %s is given twice", n+1, key)
		}
		seen[key] = true
		val := strings.TrimSpace(t[eq+1:])
		switch key {
		case "id", "title", "version":
			s, err := parseString(val)
			if err != nil {
				return Manifest{}, fmt.Errorf("line %d: %s: %w", n+1, key, err)
			}
			switch key {
			case "id":
				m.ID = s
			case "title":
				m.Title = s
			default:
				m.Version = s
			}
		case "applies-to":
			list, err := parseStrings(val)
			if err != nil {
				return Manifest{}, fmt.Errorf("line %d: applies-to: %w", n+1, err)
			}
			m.AppliesTo = list
		case "requires", "conflicts", "load-after":
			for depth := bracketDepth(val); depth > 0 && n+1 < len(lines); {
				n++
				val += " " + stripComment(lines[n])
				depth += bracketDepth(lines[n])
			}
			list, err := parseStrings(strings.TrimSpace(stripComment(val)))
			if err != nil {
				return Manifest{}, fmt.Errorf("line %d: %s: %w", n+1, key, err)
			}
			switch key {
			case "requires":
				m.Requires = list
			case "conflicts":
				m.Conflicts = list
			default:
				m.LoadAfter = list
			}
		case "entry":
			e, err := parseString(val)
			if err != nil {
				return Manifest{}, fmt.Errorf("line %d: entry: %w", n+1, err)
			}
			m.Entry = e
		case "api":
			v, err := parseInt(val)
			if err != nil {
				return Manifest{}, fmt.Errorf("line %d: api: %w", n+1, err)
			}
			m.API = v
		default:
			depth := bracketDepth(val)
			for depth > 0 && n+1 < len(lines) {
				n++
				depth += bracketDepth(lines[n])
			}
		}
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
		m.AppliesTo = []string{"common"}
	}
	if !seen["api"] {
		m.API = 1
	}
	if len(m.AppliesTo) == 0 {
		return Manifest{}, errors.New("applies-to is empty")
	}
	return m, nil
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

// bracketDepth is the net number of '[' over ']' in s outside strings and
// comments.
func bracketDepth(s string) int {
	depth := 0
	var quote rune
	for i := 0; i < len(s); i++ {
		c := rune(s[i])
		switch {
		case quote == '"' && c == '\\':
			i++
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return depth
		case c == '[':
			depth++
		case c == ']':
			depth--
		}
	}
	return depth
}

// parseString reads one string value and the optional comment after it.
func parseString(v string) (string, error) {
	s, rest, err := readString(v)
	if err != nil {
		return "", err
	}
	if err := onlyComment(rest); err != nil {
		return "", err
	}
	return s, nil
}

func onlyComment(rest string) error {
	rest = strings.TrimSpace(rest)
	if rest != "" && rest[0] != '#' {
		return fmt.Errorf("unexpected text %q after the value", rest)
	}
	return nil
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
		return v[1 : 1+end], v[2+end:], nil
	case '"':
		var b strings.Builder
		for i := 1; i < len(v); i++ {
			switch c := v[i]; c {
			case '"':
				return b.String(), v[i+1:], nil
			case '\\':
				i++
				if i >= len(v) {
					return "", "", errors.New("unterminated string")
				}
				switch v[i] {
				case '"', '\\':
					b.WriteByte(v[i])
				case 'n':
					b.WriteByte('\n')
				case 'r':
					b.WriteByte('\r')
				case 't':
					b.WriteByte('\t')
				default:
					return "", "", fmt.Errorf("unsupported escape \\%c", v[i])
				}
			default:
				b.WriteByte(c)
			}
		}
		return "", "", errors.New("unterminated string")
	}
	return "", "", errors.New("expected a quoted string")
}

// stripComment removes a trailing '#' comment that lies outside strings.
func stripComment(s string) string {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote == '"' && c == '\\':
			i++
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

// parseInt reads a decimal integer and the optional comment after it.
func parseInt(v string) (int, error) {
	end := 0
	for end < len(v) && (v[end] >= '0' && v[end] <= '9' || end == 0 && (v[end] == '-' || v[end] == '+')) {
		end++
	}
	if end == 0 {
		return 0, errors.New("expected an integer")
	}
	n, err := strconv.Atoi(v[:end])
	if err != nil {
		return 0, fmt.Errorf("%q is not an integer", v[:end])
	}
	if err := onlyComment(v[end:]); err != nil {
		return 0, err
	}
	return n, nil
}

// parseStrings reads a string or a one-line array of strings.
func parseStrings(v string) ([]string, error) {
	if !strings.HasPrefix(v, "[") {
		s, err := parseString(v)
		if err != nil {
			return nil, err
		}
		return []string{s}, nil
	}
	rest := strings.TrimSpace(v[1:])
	var out []string
	for {
		if strings.HasPrefix(rest, "]") {
			return out, onlyComment(rest[1:])
		}
		s, after, err := readString(rest)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
		rest = strings.TrimSpace(after)
		if strings.HasPrefix(rest, ",") {
			rest = strings.TrimSpace(rest[1:])
		} else if !strings.HasPrefix(rest, "]") {
			return nil, errors.New("expected ',' or ']' in the array")
		}
	}
}

// DefaultDir is the mods directory beside a program: the "mods" folder of the
// directory that holds its executable.
func DefaultDir(programDir string) string {
	return filepath.Join(programDir, "mods")
}

// Entry is one mod folder and what reading it gave.
type Entry struct {
	Folder   string // folder name, which is the mod's id when the entry is valid
	Dir      string // full path of the folder
	Manifest Manifest
	Err      error // non-nil when the manifest cannot be used
}

// Load reads the mod in dir. The folder name must equal the manifest id.
func Load(dir string) Entry {
	e := Entry{Folder: filepath.Base(dir), Dir: dir}
	data, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		e.Err = err
		return e
	}
	if e.Manifest, e.Err = ParseManifest(data); e.Err == nil && e.Manifest.ID != e.Folder {
		e.Err = fmt.Errorf("manifest id %q does not match the folder name %q", e.Manifest.ID, e.Folder)
	}
	return e
}

// Scan lists the subfolders of modsDir that hold a mod.toml, sorted by folder
// name. A missing modsDir lists nothing.
func Scan(modsDir string) ([]Entry, error) {
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Entry
	for _, d := range entries {
		if !d.IsDir() {
			continue
		}
		dir := filepath.Join(modsDir, d.Name())
		if _, err := os.Stat(filepath.Join(dir, ManifestName)); err != nil {
			continue
		}
		out = append(out, Load(dir))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Folder < out[j].Folder })
	return out, nil
}

// Resolve loads the named mods from modsDir in the order given. A name that is
// not a valid id, is repeated, has no folder, or whose manifest cannot be used
// is an error naming the mod.
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
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			return nil, fmt.Errorf("mod %q: no folder %s", id, dir)
		}
		e := Load(dir)
		if e.Err != nil {
			return nil, fmt.Errorf("mod %q: %s: %v", id, filepath.Join(dir, ManifestName), e.Err)
		}
		out = append(out, e)
	}
	return out, nil
}
