package mod

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// The settings file is read by a TOML subset that adds to the manifest's:
//
//   - "[name]" tables, whose names are plain keys;
//   - values that are a string, a decimal integer, true or false, a one-line
//     array of values, or a one-line inline table "{ key = value, ... }".
//
// Dotted keys, multi-line values, dates and floats are errors.

type tomlKind int

const (
	tomlString tomlKind = iota
	tomlInt
	tomlBool
	tomlArray
	tomlInline
)

type tomlValue struct {
	kind tomlKind
	str  string
	num  int64
	flag bool
	list []tomlValue
	tbl  []tomlPair
}

type tomlPair struct {
	key  string
	val  tomlValue
	line int
}

type tomlTable struct {
	name  string
	line  int
	pairs []tomlPair
	// array marks a "[[name]]" table; every occurrence is its own table.
	array bool
}

// tomlOptions widens the subset for the data files.
type tomlOptions struct {
	// arrays admits "[[name]]" tables, repeated under one name.
	arrays bool
	// quotedKeys admits keys written as a basic or literal string, which may
	// hold any character, in place of a plain key.
	quotedKeys bool
}

func (k tomlKind) String() string {
	return [...]string{"a string", "an integer", "a boolean", "an array", "an inline table"}[k]
}

// parseTOML reads the subset. Top-level pairs come first as the table with the
// empty name; every other table follows in file order.
func parseTOML(data []byte) ([]tomlTable, error) {
	return parseTOMLWith(data, tomlOptions{})
}

// maxDataFile bounds a data or text file of a mod.
const maxDataFile = 1 << 20

// parseTOMLWith reads the subset with the options o adds.
func parseTOMLWith(data []byte, o tomlOptions) ([]tomlTable, error) {
	limit := maxManifest
	if o.arrays || o.quotedKeys {
		limit = maxDataFile
	}
	if len(data) > limit {
		return nil, fmt.Errorf("file is larger than %d bytes", limit)
	}
	tables := []tomlTable{{}}
	seenTable, seenArray := map[string]bool{}, map[string]bool{}
	lines := strings.Split(string(bytes.TrimPrefix(data, utf8BOM)), "\n")
	for n, raw := range lines {
		t := strings.TrimSpace(raw)
		if t == "" || t[0] == '#' {
			continue
		}
		if t[0] == '[' {
			if strings.HasPrefix(t, "[[") {
				if !o.arrays {
					return nil, fmt.Errorf("line %d: arrays of tables are not supported", n+1)
				}
				end := strings.Index(t, "]]")
				if end < 0 {
					return nil, fmt.Errorf("line %d: unterminated table header", n+1)
				}
				name := strings.TrimSpace(t[2:end])
				if !validKey(name) {
					return nil, fmt.Errorf("line %d: %q is not a plain table name", n+1, name)
				}
				if err := onlyComment(t[end+2:]); err != nil {
					return nil, fmt.Errorf("line %d: %w", n+1, err)
				}
				if seenTable[name] && !seenArray[name] {
					return nil, fmt.Errorf("line %d: %s is a table and an array of tables", n+1, name)
				}
				seenTable[name], seenArray[name] = true, true
				tables = append(tables, tomlTable{name: name, line: n + 1, array: true})
				continue
			}
			end := strings.IndexByte(t, ']')
			if end < 0 {
				return nil, fmt.Errorf("line %d: unterminated table header", n+1)
			}
			name := strings.TrimSpace(t[1:end])
			if !validKey(name) {
				return nil, fmt.Errorf("line %d: %q is not a plain table name", n+1, name)
			}
			if err := onlyComment(t[end+1:]); err != nil {
				return nil, fmt.Errorf("line %d: %w", n+1, err)
			}
			if seenTable[name] {
				return nil, fmt.Errorf("line %d: table %s is given twice", n+1, name)
			}
			seenTable[name] = true
			tables = append(tables, tomlTable{name: name, line: n + 1})
			continue
		}
		var key, after string
		if o.quotedKeys && (t[0] == '"' || t[0] == '\'') {
			k, rest, err := readString(t)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", n+1, err)
			}
			rest = strings.TrimSpace(rest)
			if !strings.HasPrefix(rest, "=") {
				return nil, fmt.Errorf("line %d: expected key = value", n+1)
			}
			if k == "" {
				return nil, fmt.Errorf("line %d: the key is empty", n+1)
			}
			key, after = k, rest[1:]
		} else {
			eq := strings.IndexByte(t, '=')
			if eq <= 0 {
				return nil, fmt.Errorf("line %d: expected key = value", n+1)
			}
			key = strings.TrimSpace(t[:eq])
			if !validKey(key) {
				return nil, fmt.Errorf("line %d: %q is not a plain key", n+1, key)
			}
			after = t[eq+1:]
		}
		val, rest, err := parseTOMLValue(strings.TrimSpace(after))
		if err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", n+1, key, err)
		}
		if err := onlyComment(rest); err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", n+1, key, err)
		}
		cur := &tables[len(tables)-1]
		for _, p := range cur.pairs {
			if p.key == key {
				return nil, fmt.Errorf("line %d: %s is given twice", n+1, key)
			}
		}
		cur.pairs = append(cur.pairs, tomlPair{key: key, val: val, line: n + 1})
	}
	return tables, nil
}

func parseTOMLValue(v string) (tomlValue, string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return tomlValue{}, "", errors.New("no value")
	}
	switch c := v[0]; {
	case c == '"' || c == '\'':
		s, rest, err := readString(v)
		return tomlValue{kind: tomlString, str: s}, rest, err
	case c == '[':
		rest := strings.TrimSpace(v[1:])
		var list []tomlValue
		for {
			if strings.HasPrefix(rest, "]") {
				return tomlValue{kind: tomlArray, list: list}, rest[1:], nil
			}
			item, after, err := parseTOMLValue(rest)
			if err != nil {
				return tomlValue{}, "", err
			}
			list = append(list, item)
			rest = strings.TrimSpace(after)
			if strings.HasPrefix(rest, ",") {
				rest = strings.TrimSpace(rest[1:])
			} else if !strings.HasPrefix(rest, "]") {
				return tomlValue{}, "", errors.New("expected ',' or ']' in the array")
			}
		}
	case c == '{':
		rest := strings.TrimSpace(v[1:])
		var pairs []tomlPair
		for {
			if strings.HasPrefix(rest, "}") {
				return tomlValue{kind: tomlInline, tbl: pairs}, rest[1:], nil
			}
			eq := strings.IndexByte(rest, '=')
			if eq <= 0 {
				return tomlValue{}, "", errors.New("expected key = value in the inline table")
			}
			key := strings.TrimSpace(rest[:eq])
			if !validKey(key) {
				return tomlValue{}, "", fmt.Errorf("%q is not a plain key", key)
			}
			for _, p := range pairs {
				if p.key == key {
					return tomlValue{}, "", fmt.Errorf("%s is given twice in the inline table", key)
				}
			}
			item, after, err := parseTOMLValue(rest[eq+1:])
			if err != nil {
				return tomlValue{}, "", err
			}
			pairs = append(pairs, tomlPair{key: key, val: item})
			rest = strings.TrimSpace(after)
			if strings.HasPrefix(rest, ",") {
				rest = strings.TrimSpace(rest[1:])
			} else if !strings.HasPrefix(rest, "}") {
				return tomlValue{}, "", errors.New("expected ',' or '}' in the inline table")
			}
		}
	case strings.HasPrefix(v, "true") && wordEnd(v, 4):
		return tomlValue{kind: tomlBool, flag: true}, v[4:], nil
	case strings.HasPrefix(v, "false") && wordEnd(v, 5):
		return tomlValue{kind: tomlBool, flag: false}, v[5:], nil
	case c == '-' || c == '+' || c >= '0' && c <= '9':
		end := 1
		for end < len(v) && v[end] >= '0' && v[end] <= '9' {
			end++
		}
		if end < len(v) && (v[end] == '.' || v[end] == 'e' || v[end] == 'E' || v[end] == '_' || v[end] == ':' || v[end] == '-') {
			return tomlValue{}, "", errors.New("only decimal integers are supported")
		}
		n, err := strconv.ParseInt(v[:end], 10, 64)
		if err != nil {
			return tomlValue{}, "", fmt.Errorf("%q is not an integer", v[:end])
		}
		return tomlValue{kind: tomlInt, num: n}, v[end:], nil
	}
	return tomlValue{}, "", fmt.Errorf("cannot read the value %q", v)
}

// wordEnd reports whether v[i:] starts a new token rather than continuing a word.
func wordEnd(v string, i int) bool {
	if i >= len(v) {
		return true
	}
	c := v[i]
	return !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-')
}
