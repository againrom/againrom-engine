package mod

import (
	"fmt"
	"strconv"
)

// SetRecord is the stored form of a Set: what a save keeps of the mod set it
// was written under. Values are held as their canonical text with the kind that
// reads them back.
type SetRecord struct {
	Base string        `json:"base"`
	Mods []EntryRecord `json:"mods"`
}

// EntryRecord is the stored form of one mod of a set.
type EntryRecord struct {
	ID       string          `json:"id"`
	Version  string          `json:"version"`
	Digest   string          `json:"digest"`
	Settings []SettingRecord `json:"settings,omitempty"`
}

// SettingRecord is the stored form of one setting value.
type SettingRecord struct {
	Key   string `json:"key"`
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Record returns the stored form of the set.
func (s Set) Record() SetRecord {
	r := SetRecord{Base: s.Base, Mods: []EntryRecord{}}
	for _, m := range s.Mods {
		e := EntryRecord{ID: m.ID, Version: m.Version, Digest: m.Digest}
		for _, v := range m.Settings {
			e.Settings = append(e.Settings, SettingRecord{Key: v.Key, Kind: v.Value.Kind.String(), Value: v.Value.String()})
		}
		r.Mods = append(r.Mods, e)
	}
	return r
}

// Set reads the stored form back. It refuses a kind or value it cannot read.
func (r SetRecord) Set() (Set, error) {
	s := Set{Base: r.Base}
	for _, m := range r.Mods {
		e := SetEntry{ID: m.ID, Version: m.Version, Digest: m.Digest}
		for _, v := range m.Settings {
			val, err := parseRecordedValue(v)
			if err != nil {
				return Set{}, fmt.Errorf("mod %q setting %s: %v", m.ID, v.Key, err)
			}
			e.Settings = append(e.Settings, SettingValue{Key: v.Key, Value: val})
		}
		s.Mods = append(s.Mods, e)
	}
	return s, nil
}

func parseRecordedValue(v SettingRecord) (Value, error) {
	switch v.Kind {
	case "int":
		n, err := strconv.ParseInt(v.Value, 10, 64)
		if err != nil {
			return Value{}, fmt.Errorf("%q is not an integer", v.Value)
		}
		return Value{Kind: KindInt, Int: n}, nil
	case "bool":
		b, err := strconv.ParseBool(v.Value)
		if err != nil {
			return Value{}, fmt.Errorf("%q is not a boolean", v.Value)
		}
		return Value{Kind: KindBool, Bool: b}, nil
	case "choice":
		return Value{Kind: KindChoice, Str: v.Value}, nil
	}
	return Value{}, fmt.Errorf("unknown kind %q", v.Kind)
}
