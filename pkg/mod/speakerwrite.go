package mod

import (
	"fmt"
	"slices"
	"strconv"
)

// NewSpeaker is a record of the number id that sets a name and a picture, and
// one placement of one map when bound.
func NewSpeaker(id int32, name, picture string) Speaker {
	s := Speaker{ID: id, Name: name, Picture: picture}
	s.SetField("name")
	s.SetField("picture")
	return s
}

// Bind places the record on unit of the map.
func (s *Speaker) Bind(mapPath string, unit int32) {
	s.Map, s.Unit, s.Bound = mapPath, unit, true
	s.SetField("map")
	s.SetField("unit")
}

// SetField marks name as a key the record sets, so the writer writes it.
func (s *Speaker) SetField(name string) {
	if !slices.ContainsFunc(s.Fields, func(f PatchField) bool { return f.Name == name }) {
		s.Fields = append(slices.Clone(s.Fields), PatchField{Name: name, Line: s.Line})
	}
}

// writeSpeakers is the speakers kind's writer: source made to hold
// d.Speakers.Speakers, each [[speaker]] table kept where it stands with its
// lines but for the keys it changes. A record the source does not hold is
// appended; one d lacks leaves with its table.
func writeSpeakers(rel string, source []byte, d Data) ([]byte, error) {
	before, err := ParseSpeakers("", rel, source)
	if err != nil {
		return nil, err
	}
	was, err := speakerSourceRows(before.Speakers)
	if err != nil {
		return nil, err
	}
	after, err := speakerSourceRows(d.Speakers.Speakers)
	if err != nil {
		return nil, err
	}
	out, err := writeRows(source, was, after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseSpeakers("", rel, out)
	if err != nil {
		return nil, err
	}
	again, err := speakerSourceRows(back.Speakers)
	if err != nil {
		return nil, err
	}
	if !sameRows(again, after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", rel)
	}
	return out, nil
}

// speakerSourceRows is the records as the row writer sees them: a record is
// named by its number and writes the keys it sets.
func speakerSourceRows(list []Speaker) ([]sourceRow, error) {
	rows := make([]sourceRow, len(list))
	for i, s := range list {
		sets := func(name string) bool {
			return slices.ContainsFunc(s.Fields, func(f PatchField) bool { return f.Name == name })
		}
		pairs := []missionPair{{"id", strconv.Itoa(int(s.ID))}}
		add := func(key, value string) error {
			q, err := writeString(value)
			if err != nil {
				return fmt.Errorf("speaker %d: %s: %w", s.ID, key, err)
			}
			pairs = append(pairs, missionPair{key, q})
			return nil
		}
		if sets("name") {
			if err := add("name", s.Name); err != nil {
				return nil, err
			}
		}
		if sets("picture") && s.Picture != "" {
			if err := add("picture", s.Picture); err != nil {
				return nil, err
			}
		}
		if sets("map") {
			if err := add("map", s.Map); err != nil {
				return nil, err
			}
		}
		if sets("unit") {
			pairs = append(pairs, missionPair{"unit", strconv.Itoa(int(s.Unit))})
		}
		if sets(OffField) {
			pairs = append(pairs, missionPair{OffField, strconv.FormatBool(s.Off)})
		}
		rows[i] = sourceRow{table: "speaker", identity: strconv.Itoa(int(s.ID)), pairs: pairs}
	}
	return rows, nil
}
