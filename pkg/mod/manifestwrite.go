package mod

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

func CloneManifest(m Manifest) Manifest {
	m.AppliesTo = slices.Clone(m.AppliesTo)
	m.Requires = slices.Clone(m.Requires)
	m.Conflicts = slices.Clone(m.Conflicts)
	m.LoadAfter = slices.Clone(m.LoadAfter)
	return m
}

// WriteManifest sets top-level keys, preserving the source outside their values.
// Parsing the result remains the caller's admission boundary.
func WriteManifest(source []byte, values map[string]any) ([]byte, error) {
	encoded := map[string]string{}
	for key, value := range values {
		if !slices.Contains(manifestKeys, key) {
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
		for _, key := range manifestKeys {
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
		start := eq + 1
		for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
			start++
		}
		for depth := bracketDepth(line[start:]); depth > 0 && i+1 < len(lines); {
			i++
			depth += bracketDepth(lines[i])
			line += lines[i]
		}
		old := strings.TrimSpace(line[start:])
		var prior any
		var err error
		switch values[key].(type) {
		case string:
			prior, _, err = readString(old)
		case int:
			prior, err = parseInt(old)
		default:
			var flat []string
			for _, l := range strings.Split(old, "\n") {
				flat = append(flat, stripComment(l))
			}
			prior, err = parseStrings(strings.Join(flat, " "))
		}
		if err == nil && reflect.DeepEqual(prior, values[key]) {
			out.WriteString(line)
		} else {
			end := start + manifestValueEnd(line[start:])
			out.WriteString(line[:start] + value + line[end:])
		}
		delete(encoded, key)
	}
	appendMissing()
	return []byte(out.String()), nil
}

func manifestValueEnd(s string) int {
	var quote byte
	depth, end := 0, 0
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
			if depth == 0 {
				return end
			}
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		case c == '[':
			depth++
		case c == ']':
			depth--
		}
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			end = i + 1
		}
		if quote == 0 && depth == 0 && (c == ']' || c == '"' || c == '\'') {
			return end
		}
	}
	return end
}
