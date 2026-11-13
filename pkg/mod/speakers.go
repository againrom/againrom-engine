package mod

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/namefold"
)

// FirstModSpeaker and LastModSpeaker bound the record numbers a mod's speaker
// record takes. A message tag names a record by this number, the game's own
// records stand below the band and a row's server id never reaches it. The
// band ends at the largest five-digit number: the second game's message tag
// reads five bytes after `npc=`, so a longer number would name another record.
const (
	FirstModSpeaker = 20000
	LastModSpeaker  = 99999
)

// Speaker is one speaker record a mod holds: a number a message tag names, a
// name, a picture file of the mod and, by choice, one placement of one map.
type Speaker struct {
	Mod, File string
	Line      int
	// Dir is the mod's folder or RES file.
	Dir string

	ID   int32
	Name string
	// Picture is the record's PNG, relative to the mod; PictureBytes its
	// content when the mod loaded.
	Picture      string
	PictureBytes []byte
	// Map and Unit bind the record to one unit of one map, when Bound: the
	// map's path in the mod and the unit's script number on that map.
	Map   string
	Unit  int32
	Bound bool
	Off   bool

	Fields []PatchField
}

// SpeakerData is the speaker records the mods declare, in load order.
type SpeakerData struct {
	Speakers []Speaker
	Order    Precedence
}

// Empty reports whether no mod declares a speaker record.
func (d SpeakerData) Empty() bool { return len(d.Speakers) == 0 }

// ParseSpeakers reads data/speakers.toml of the mod id: [[speaker]] tables
// with the keys id, name, picture, map, unit and off. A refusal names the file
// and the line.
func ParseSpeakers(id, file string, data []byte) (SpeakerData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return SpeakerData{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return SpeakerData{}, bad(p.line, "%s is outside a [[speaker]] table", p.key)
	}
	var out SpeakerData
	seen := map[int32]bool{}
	for _, t := range tables[1:] {
		if t.name != "speaker" || !t.array {
			return SpeakerData{}, bad(t.line, "unknown table %q (this file holds [[speaker]])", t.name)
		}
		s := Speaker{Mod: id, File: file, Line: t.line}
		hasID, hasMap, hasUnit := false, false, false
		for _, p := range t.pairs {
			var err error
			text := func(dst *string) {
				if p.val.kind != tomlString || p.val.str == "" {
					err = bad(p.line, "%s must be a string that is not empty", p.key)
					return
				}
				*dst = p.val.str
			}
			switch p.key {
			case "id":
				if p.val.kind != tomlInt {
					err = bad(p.line, "id must be an integer, not %s", p.val.kind)
				} else if p.val.num < FirstModSpeaker || p.val.num > LastModSpeaker {
					err = bad(p.line, "id is %d; a mod's speaker record takes a number from %d to %d, below that the numbers are the game's records", p.val.num, FirstModSpeaker, LastModSpeaker)
				}
				s.ID, hasID = int32(p.val.num), true
			case "name":
				text(&s.Name)
			case "picture":
				text(&s.Picture)
			case "map":
				text(&s.Map)
				hasMap = true
			case "unit":
				if p.val.kind != tomlInt || p.val.num < 1 || p.val.num > 65535 {
					err = bad(p.line, "unit must be a unit's script number, from 1 to 65535")
				}
				s.Unit, hasUnit = int32(p.val.num), true
			case OffField:
				if p.val.kind != tomlBool {
					err = bad(p.line, "off must be true or false, not %s", p.val.kind)
				}
				s.Off = p.val.flag
			default:
				err = bad(p.line, "unknown key %q (keys: id, name, picture, map, unit, off)", p.key)
			}
			if err != nil {
				return SpeakerData{}, err
			}
			if p.key != "id" {
				s.Fields = append(s.Fields, PatchField{Name: p.key, Line: p.line})
			}
		}
		if !hasID {
			return SpeakerData{}, bad(t.line, "[[speaker]] has no id")
		}
		if seen[s.ID] {
			return SpeakerData{}, bad(t.line, "speaker record %d stands twice in this file", s.ID)
		}
		seen[s.ID] = true
		if hasMap != hasUnit {
			return SpeakerData{}, bad(t.line, "a speaker record names map and unit together or neither of them")
		}
		s.Bound = hasMap
		out.Speakers = append(out.Speakers, s)
	}
	return out, nil
}

// fieldLine is the line of the field a record sets; the record's own line
// when it sets none.
func (s Speaker) fieldLine(name string) int {
	if i := slices.IndexFunc(s.Fields, func(f PatchField) bool { return f.Name == name }); i >= 0 {
		return s.Fields[i].Line
	}
	return s.Line
}

// Errorf is a refusal of the record's field, naming the mod and its file.
func (s Speaker) Errorf(field, format string, args ...any) error {
	return Source{Mod: s.Mod, File: s.File, Line: s.fieldLine(field)}.Errorf(format, args...)
}

// apply writes the fields s sets into r.
func (s Speaker) apply(r *Speaker) {
	r.Mod, r.File, r.Line, r.Dir, r.ID = s.Mod, s.File, s.Line, s.Dir, s.ID
	for _, f := range s.Fields {
		switch f.Name {
		case "name":
			r.Name = s.Name
		case "picture":
			r.Picture, r.PictureBytes = s.Picture, s.PictureBytes
		case "map", "unit":
			r.Map, r.Unit, r.Bound = s.Map, s.Unit, s.Bound
		}
	}
}

// MergeSpeakers is the speaker records of the set after the layer step. Layer
// 0 holds none in the mods' number band. The result is sorted by number; a
// record the layers leave with no name or picture is refused at its line.
func MergeSpeakers(d SpeakerData) ([]Speaker, []Override, error) {
	patches := make([]Patch[int32, Speaker], 0, len(d.Speakers))
	for _, s := range d.Speakers {
		patches = append(patches, Patch[int32, Speaker]{Source: Source{Mod: s.Mod, File: s.File, Line: s.Line},
			Key: s.ID, Fields: s.Fields, Off: s.Off, Apply: s.apply})
	}
	l, err := Merge("speaker", nil, func(s Speaker) int32 { return s.ID }, patches, d.Order, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range l.Records {
		if s.Name == "" {
			return nil, nil, s.Errorf("name", "speaker record %d has no name", s.ID)
		}
		if len(s.PictureBytes) == 0 {
			return nil, nil, s.Errorf("picture", "speaker record %d has no picture", s.ID)
		}
	}
	slices.SortFunc(l.Records, func(a, b Speaker) int { return int(a.ID - b.ID) })
	return l.Records, l.Overrides, nil
}

// MoveSpeakerMaps is the speakers file src with each record's map that a
// move takes From, compared folded, written as its To; the file's other text
// stands. moved is false when no record names a moved map.
func MoveSpeakerMaps(src []byte, moves []FileMove) (out []byte, moved bool, err error) {
	d, err := ParseSpeakers("", SpeakersFile, src)
	if err != nil {
		return nil, false, err
	}
	lines := strings.SplitAfter(string(src), "\n")
	for _, s := range d.Speakers {
		i := slices.IndexFunc(moves, func(m FileMove) bool { return s.Bound && namefold.ModTree(m.From) == namefold.ModTree(s.Map) })
		if i < 0 {
			continue
		}
		line := s.fieldLine("map") - 1
		if lines[line], err = rewritePair(lines[line], writeString(moves[i].To)); err != nil {
			return nil, false, err
		}
		moved = true
	}
	return []byte(strings.Join(lines, "")), moved, nil
}

// AppendSpeaker is the speakers file src with the record s added at its end.
// It leaves the file's other text as it stands.
func AppendSpeaker(src []byte, s Speaker) []byte {
	var b strings.Builder
	b.Write(src)
	if len(src) != 0 && src[len(src)-1] != '\n' {
		b.WriteByte('\n')
	}
	b.WriteString("[[speaker]]\n")
	b.WriteString("id = " + strconv.Itoa(int(s.ID)) + "\n")
	b.WriteString("name = " + writeString(s.Name) + "\npicture = " + writeString(s.Picture) + "\n")
	if s.Bound {
		b.WriteString("map = " + writeString(s.Map) + "\nunit = " + strconv.Itoa(int(s.Unit)) + "\n")
	}
	return []byte(b.String())
}
