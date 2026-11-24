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
	lines := strings.SplitAfter(string(bytes.TrimPrefix(source, utf8BOM)), "\n")
	var out strings.Builder
	if bytes.HasPrefix(source, utf8BOM) {
		out.Write(utf8BOM)
	}
	newline := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		newline = "\r\n"
	}
	appendMissing := func() {
		for _, key := range manifestKeyNames() {
			if value, ok := encoded[key]; ok {
				if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
					out.WriteString(newline)
				}
				out.WriteString(key + " = " + value + newline)
			}
		}
		clear(encoded)
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		text := strings.TrimSpace(line)
		if strings.HasPrefix(text, "[") {
			appendMissing()
			out.WriteString(strings.Join(lines[i:], ""))
			return []byte(out.String()), nil
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 || strings.HasPrefix(text, "#") {
			out.WriteString(line)
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value, change := encoded[key]
		if !change {
			out.WriteString(line)
			continue
		}
		last, err := rewritePair(lines, i, 0, value)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		out.WriteString(lines[i])
		delete(encoded, key)
		i = last
	}
	appendMissing()
	return []byte(out.String()), nil
}
