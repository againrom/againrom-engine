package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func withCurrentMenuDefinitions(t *testing.T, f *FrontEnd, extraHumanRows ...int) *FrontEnd {
	t.Helper()
	table := mapload.Table{}
	if f.Table != nil {
		table = *f.Table
	}
	if table.Units == nil {
		unit := actorRegistryTable().Units.(dbCollection)[1]
		table.Units = dbCollection{unit, unit}
	}
	humans := append(fourBaseHumans(), dbEntry{name: "actor runtime", params: humansParams(50, 0, 5)})
	if table.Humans != nil {
		for i := range table.Humans.Len() {
			if i == len(humans) {
				humans = append(humans, dbEntry{})
			}
			humans[i] = dbEntry{
				name:    table.Humans.EntryName(i),
				params:  append([]int32(nil), table.Humans.EntryParams(i)...),
				strings: append([]string(nil), table.Humans.EntryStrings(i)...),
			}
		}
	}
	for _, row := range extraHumanRows {
		for len(humans) <= row {
			humans = append(humans, dbEntry{})
		}
		if len(humans[row].params) == 0 {
			humans[row] = dbEntry{name: "turn actor", params: humansParams(50, 0, 5)}
		}
	}
	table.Humans = humans
	f.Table = &table
	if len(f.Campaign.value.Main) == 0 {
		f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	}
	return f
}

func currentMenuWorldDiagnostics(t *testing.T, before, after *sim.World) {
	t.Helper()
	left, _ := before.MarshalBinary()
	right, _ := after.MarshalBinary()
	currentItemWorldDiagnostics(t, left, right)
	count := 0
	var walk func(string, reflect.Value, reflect.Value)
	walk = func(path string, a, b reflect.Value) {
		if count >= 20 {
			return
		}
		if !a.IsValid() || !b.IsValid() {
			t.Log("World field presence differs", path, a.IsValid(), b.IsValid())
			count++
			return
		}
		switch a.Kind() {
		case reflect.Struct:
			for i := range a.NumField() {
				walk(path+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i))
			}
		case reflect.Slice, reflect.Array:
			if a.Len() != b.Len() {
				t.Log("World field length differs", path, a.Len(), b.Len())
				count++
			}
			for i := range min(a.Len(), b.Len()) {
				walk(fmt.Sprintf("%s[%d]", path, i), a.Index(i), b.Index(i))
			}
		case reflect.Pointer, reflect.Interface:
			if a.IsNil() || b.IsNil() {
				if a.IsNil() != b.IsNil() {
					t.Log("World field nil differs", path, a.IsNil(), b.IsNil())
					count++
				}
				return
			}
			walk(path, a.Elem(), b.Elem())
		case reflect.Map:
			if a.Len() != b.Len() {
				t.Log("World field map size differs", path, a.Len(), b.Len())
				count++
			}
			for _, key := range a.MapKeys() {
				walk(fmt.Sprintf("%s[%v]", path, key), a.MapIndex(key), b.MapIndex(key))
			}
		case reflect.Func:
		default:
			if !a.Equal(b) {
				t.Logf("%s: %v -> %v", path, a, b)
				count++
			}
		}
	}
	walk("World", reflect.ValueOf(before).Elem(), reflect.ValueOf(after).Elem())
}
