// Package ini reads and writes the small settings files the launcher keeps.
//
// The format is the common subset: "[section]" headers, "key = value" lines,
// and comment lines that start with ';' or '#'. Values are the text after the
// first '=' with surrounding blanks trimmed; there is no quoting and no inline
// comment, so a value may hold ';' and '#' (Windows paths do). Section and key
// names compare without regard to case and keep the case they were written in.
//
// A File keeps every line it read. Writing it back returns the original bytes
// except for the lines a Set or Delete touched, so unknown keys, comments,
// blank lines and malformed lines all survive an edit. A line that is neither
// blank, a comment, a header nor a key=value pair is reported by Parse and kept
// verbatim; it is never fatal. Only the first section of a given name is read. When a key appears twice in a section, the first
// occurrence is the one Get returns and Set rewrites.
package ini

import (
	"bytes"
	"fmt"
	"strings"
)

type lineKind int

const (
	kindOther   lineKind = iota // blank, comment or malformed: written back untouched
	kindSection                 // [name]
	kindPair                    // key = value
)

type line struct {
	kind    lineKind
	raw     string // the text of the line without its terminator
	section string // kindSection: the name
	key     string // kindPair: the key as written
	value   string // kindPair: the trimmed value
}

// utf8BOM is the byte order mark some editors put at the start of a file.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Problem is one line Parse could not interpret.
type Problem struct {
	Line int // 1-based
	Text string
}

func (p Problem) String() string {
	return fmt.Sprintf("line %d is not a section, a key=value pair or a comment: %q", p.Line, p.Text)
}

// File is a parsed settings file.
type File struct {
	lines []line
	eol   string
}

// Parse reads data. It never fails; lines it cannot interpret are returned as
// problems and kept in the File.
func Parse(data []byte) (*File, []Problem) {
	data = bytes.TrimPrefix(data, utf8BOM)
	f := &File{eol: "\n"}
	if i := bytes.IndexByte(data, '\n'); i > 0 && data[i-1] == '\r' {
		f.eol = "\r\n"
	}
	var problems []Problem
	text := string(data)
	if text == "" {
		return f, nil
	}
	text = strings.TrimSuffix(text, "\n")
	for n, raw := range strings.Split(text, "\n") {
		raw = strings.TrimSuffix(raw, "\r")
		l := line{raw: raw}
		t := strings.TrimSpace(raw)
		switch {
		case t == "" || t[0] == ';' || t[0] == '#':
		case t[0] == '[':
			if end := strings.IndexByte(t, ']'); end > 1 && strings.TrimSpace(t[end+1:]) == "" {
				l.kind = kindSection
				l.section = strings.TrimSpace(t[1:end])
			} else {
				problems = append(problems, Problem{n + 1, raw})
			}
		default:
			if eq := strings.IndexByte(t, '='); eq > 0 && strings.TrimSpace(t[:eq]) != "" {
				l.kind = kindPair
				l.key = strings.TrimSpace(t[:eq])
				l.value = strings.TrimSpace(t[eq+1:])
			} else {
				problems = append(problems, Problem{n + 1, raw})
			}
		}
		f.lines = append(f.lines, l)
	}
	return f, problems
}

// Bytes returns the file text. Each line ends with the terminator the source
// used first, or "\n" for a new file; a non-empty file ends with a terminator.
func (f *File) Bytes() []byte {
	var b strings.Builder
	for _, l := range f.lines {
		b.WriteString(l.raw)
		b.WriteString(f.eol)
	}
	return []byte(b.String())
}

// span returns the half-open range of line indexes that belong to section, or
// ok false when the section does not exist. Pairs before the first header
// belong to the section named "".
func (f *File) span(section string) (start, end int, ok bool) {
	in := section == ""
	if in {
		ok = true
	}
	for i, l := range f.lines {
		if l.kind != kindSection {
			continue
		}
		if in {
			return start, i, true
		}
		if strings.EqualFold(l.section, section) {
			in, start, ok = true, i+1, true
		}
	}
	if ok {
		return start, len(f.lines), true
	}
	return 0, 0, false
}

func (f *File) find(section, key string) int {
	start, end, ok := f.span(section)
	if !ok {
		return -1
	}
	for i := start; i < end; i++ {
		if f.lines[i].kind == kindPair && strings.EqualFold(f.lines[i].key, key) {
			return i
		}
	}
	return -1
}

// Sections returns the names of the sections in file order, a name that
// appears twice once. Pairs before the first header belong to no listed
// section.
func (f *File) Sections() []string {
	var names []string
	seen := map[string]bool{}
	for _, l := range f.lines {
		if l.kind == kindSection && !seen[strings.ToLower(l.section)] {
			seen[strings.ToLower(l.section)] = true
			names = append(names, l.section)
		}
	}
	return names
}

// Get returns the value of key in section.
func (f *File) Get(section, key string) (string, bool) {
	if i := f.find(section, key); i >= 0 {
		return f.lines[i].value, true
	}
	return "", false
}

// Keys returns the keys of section in file order, repeats included once.
func (f *File) Keys(section string) []string {
	start, end, ok := f.span(section)
	if !ok {
		return nil
	}
	var keys []string
	seen := map[string]bool{}
	for i := start; i < end; i++ {
		l := f.lines[i]
		if l.kind != kindPair || seen[strings.ToLower(l.key)] {
			continue
		}
		seen[strings.ToLower(l.key)] = true
		keys = append(keys, l.key)
	}
	return keys
}

// Set stores value under key in section. An existing line is rewritten in place
// (a value equal to the stored one leaves the line byte-identical); a new key
// goes after the last pair of its section, and a missing section is appended at
// the end of the file. A value holding a line break is cut at the first one.
func (f *File) Set(section, key, value string) {
	if i := strings.IndexAny(value, "\r\n"); i >= 0 {
		value = value[:i]
	}
	value = strings.TrimSpace(value)
	if i := f.find(section, key); i >= 0 {
		if f.lines[i].value != value {
			f.lines[i].value = value
			f.lines[i].raw = key2raw(f.lines[i].key, value)
		}
		return
	}
	nl := line{kind: kindPair, key: key, value: value, raw: key2raw(key, value)}
	start, end, ok := f.span(section)
	if !ok {
		f.lines = append(f.lines, line{kind: kindSection, section: section, raw: "[" + section + "]"}, nl)
		return
	}
	at := start
	for i := start; i < end; i++ {
		if f.lines[i].kind == kindPair {
			at = i + 1
		}
	}
	f.lines = append(f.lines, line{})
	copy(f.lines[at+1:], f.lines[at:])
	f.lines[at] = nl
}

func key2raw(key, value string) string {
	if value == "" {
		return key + " ="
	}
	return key + " = " + value
}

// Delete removes the first occurrence of key in section and reports whether
// there was one.
func (f *File) Delete(section, key string) bool {
	i := f.find(section, key)
	if i < 0 {
		return false
	}
	f.lines = append(f.lines[:i], f.lines[i+1:]...)
	return true
}
