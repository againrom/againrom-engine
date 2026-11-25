package mod

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// CloneManifest is m with lists of its own.
func CloneManifest(m Manifest) Manifest {
	for _, k := range ManifestKeys {
		if k.Value == ManifestList {
			k.Set(&m, slices.Clone(k.Get(m).([]string)))
		}
	}
	return m
}

// WriteManifest sets top-level keys, preserving the source outside their values.
// Parsing the result remains the caller's admission boundary.
func WriteManifest(source []byte, values map[string]any) ([]byte, error) {
	encoded := map[string]string{}
	for key, value := range values {
		if !slices.Contains(manifestKeyNames(), key) {
			return nil, fmt.Errorf("%q is not a mod.toml key", key)
		}
		switch v := value.(type) {
		case string:
			encoded[key] = writeString(v)
		case []string:
			var list []string
			for _, s := range v {
				list = append(list, writeString(s))
			}
			encoded[key] = "[" + strings.Join(list, ", ") + "]"
		case int:
			encoded[key] = strconv.Itoa(v)
		default:
			return nil, fmt.Errorf("%s: unsupported manifest value %T", key, value)
		}
	}
	tables, err := parseTOMLWith(source, tomlOptions{})
	if err != nil {
		return nil, err
	}
	lines := strings.SplitAfter(string(bytes.TrimPrefix(source, utf8BOM)), "\n")
	for _, p := range tables[0].pairs {
		value, change := encoded[p.key]
		if !change {
			continue
		}
		if _, err := rewritePair(lines, p.line-1, 0, value); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.line, err)
		}
		delete(encoded, p.key)
	}
	split := len(lines)
	if len(tables) > 1 {
		split = tables[1].line - 1
	}
	newline := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		newline = "\r\n"
	}
	var out strings.Builder
	var bom []byte
	if bytes.HasPrefix(source, utf8BOM) {
		bom = utf8BOM
		out.Write(bom)
	}
	out.WriteString(strings.Join(lines[:split], ""))
	for _, key := range manifestKeyNames() {
		if value, ok := encoded[key]; ok {
			if out.Len() > len(bom) && !strings.HasSuffix(out.String(), "\n") {
				out.WriteString(newline)
			}
			out.WriteString(key + " = " + value + newline)
		}
	}
	out.WriteString(strings.Join(lines[split:], ""))
	return []byte(out.String()), nil
}
