package game

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// checkScenarioKeys reads raw as a scenario and refuses the first key the
// scenario format does not name, at any depth, with the keys that place
// accepts. The strict decoder refuses the same key without that list.
func checkScenarioKeys(raw []byte) error {
	var doc any
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	return checkKeys(doc, reflect.TypeOf(HeadlessScenario{}), "scenario")
}

// jsonKeys are the keys of struct type t in declaration order, an embedded
// struct's keys among them.
func jsonKeys(t reflect.Type) []string {
	var keys []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			keys = append(keys, jsonKeys(f.Type)...)
			continue
		}
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		keys = append(keys, name)
	}
	return keys
}

// fieldType is the type of the field of struct type t that json key names,
// matched as the decoder matches it.
func fieldType(t reflect.Type, key string) (reflect.Type, bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			if ft, ok := fieldType(f.Type, key); ok {
				return ft, true
			}
			continue
		}
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		if strings.EqualFold(name, key) {
			return f.Type, true
		}
	}
	return nil, false
}

func checkKeys(v any, t reflect.Type, path string) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch value := v.(type) {
	case map[string]any:
		switch t.Kind() {
		case reflect.Struct:
			// Keys are visited in sorted order so the refusal is the same on every run.
			for _, key := range slices.Sorted(maps.Keys(value)) {
				ft, ok := fieldType(t, key)
				if !ok {
					return fmt.Errorf("%s: unknown key %q; accepted keys: %s",
						path, key, strings.Join(jsonKeys(t), ", "))
				}
				if err := checkKeys(value[key], ft, path+"."+key); err != nil {
					return err
				}
			}
		case reflect.Map:
			for _, key := range slices.Sorted(maps.Keys(value)) {
				if err := checkKeys(value[key], t.Elem(), path+"."+key); err != nil {
					return err
				}
			}
		}
	case []any:
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			for i, item := range value {
				if err := checkKeys(item, t.Elem(), fmt.Sprintf("%s[%d]", path, i+1)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// headlessPointForms are the keys of a pointer's at that name a surface, in
// declaration order; offset_x and offset_y move the resolved pixel and name
// none.
func headlessPointForms() string {
	var forms []string
	for _, key := range jsonKeys(reflect.TypeOf(HeadlessPoint{})) {
		if !strings.HasPrefix(key, "offset_") {
			forms = append(forms, key)
		}
	}
	return strings.Join(forms, ", ")
}
