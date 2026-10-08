package game

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"againrom/pkg/formats/sav"
)

// censusLeaf separates a field pattern from its occurrence.
type censusLeaf struct {
	pattern, instance string
	value             reflect.Value
	topology          bool
}

func (l censusLeaf) bytes() []byte {
	v := l.value
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, v.Uint())
		return b[:int(v.Type().Size())]
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(v.Int()))
		return b[:int(v.Type().Size())]
	case reflect.Bool:
		if v.Bool() {
			return []byte{1}
		}
		return []byte{0}
	case reflect.Float64:
		return binary.LittleEndian.AppendUint64(nil, math.Float64bits(v.Float()))
	case reflect.String:
		return []byte(v.String())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return append([]byte(nil), v.Bytes()...)
		}
		if v.Type().Elem().Kind() == reflect.Uint16 {
			b := make([]byte, 0, 2*v.Len())
			for i := 0; i < v.Len(); i++ {
				b = binary.LittleEndian.AppendUint16(b, uint16(v.Index(i).Uint()))
			}
			return b
		}
	}
	panic(fmt.Sprintf("census leaf kind %s", v.Type()))
}

// censusLeaves enumerates every leaf of a Document. Object records are named by
// class and field category, so every Human "Health" value is one pattern.
func censusLeaves(doc *sav.DocumentData) []censusLeaf {
	var out []censusLeaf
	emit := func(pattern, instance string, v reflect.Value, topo bool) {
		out = append(out, censusLeaf{pattern, instance, v, topo})
	}
	walk := func(v reflect.Value, pattern, instance string) { censusWalk(emit, v, pattern, instance) }
	var record func(r *sav.DocumentRecordData, pattern, instance string)
	record = func(r *sav.DocumentRecordData, pattern, instance string) {
		for i := range r.Values {
			emit(pattern+".v."+r.Values[i].Name, instance, reflect.ValueOf(&r.Values[i].Value).Elem(), false)
		}
		for i := range r.Texts {
			emit(pattern+".t."+r.Texts[i].Name, instance, reflect.ValueOf(&r.Texts[i].Value).Elem(), false)
		}
		for i := range r.Raw {
			emit(pattern+".r."+r.Raw[i].Name, instance, reflect.ValueOf(&r.Raw[i].Bytes).Elem(), false)
		}
		for i := range r.Counts {
			emit(pattern+".n."+r.Counts[i].Name, instance, reflect.ValueOf(&r.Counts[i].Count).Elem(), true)
		}
		for i := range r.RefSlots {
			emit(pattern+".ref."+r.RefSlots[i].Name, instance, reflect.ValueOf(&r.RefSlots[i].Objects).Elem(), true)
		}
		for i := range r.Inline {
			record(&r.Inline[i].Record, pattern+".inline."+r.Inline[i].Name+"/"+r.Inline[i].Record.Class, instance+"/"+r.Inline[i].Name)
		}
		for i := range r.Groups {
			record(&r.Groups[i], pattern+".group/"+r.Groups[i].Class, fmt.Sprintf("%s/g%d", instance, i))
		}
	}
	d := reflect.ValueOf(doc).Elem()
	t := d.Type()
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		switch name {
		case "Objects":
			for j := range doc.Objects {
				r := &doc.Objects[j]
				record(r, r.Class, fmt.Sprintf("obj%d", j+1))
			}
		case "State":
			for j := range doc.State.ValueRecords {
				r := &doc.State.ValueRecords[j]
				p := "State" + r.Path
				emit(p+".Kind", "", reflect.ValueOf(&r.Value.Kind).Elem(), false)
				emit(p+".Int32", "", reflect.ValueOf(&r.Value.Int32).Elem(), false)
				emit(p+".Bytes", "", reflect.ValueOf(&r.Value.Bytes).Elem(), false)
			}
			emit("State.RootKind", "", reflect.ValueOf(&doc.State.RootKind).Elem(), false)
		default:
			walk(d.Field(i), name, "")
		}
	}
	return out
}

// censusPatterns groups leaves by pattern in a stable order.
func censusPatterns(leaves []censusLeaf) ([]string, map[string][]censusLeaf) {
	by := map[string][]censusLeaf{}
	for _, l := range leaves {
		by[l.pattern] = append(by[l.pattern], l)
	}
	names := make([]string, 0, len(by))
	for n := range by {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, by
}

func censusKey(l censusLeaf) string { return l.pattern + "@" + l.instance }

func censusClass(pattern string) string {
	if i := strings.IndexAny(pattern, ".["); i > 0 {
		return pattern[:i]
	}
	return pattern
}

// censusWalk keeps collection indices in the instance; uint16 lists are topology.
func censusWalk(emit func(pattern, instance string, v reflect.Value, topo bool), v reflect.Value, pattern, instance string, writes ...*[]func()) {
	var walk func(v reflect.Value, pattern, instance string)
	walk = func(v reflect.Value, pattern, instance string) {
		switch v.Kind() {
		case reflect.Pointer:
			if !v.IsNil() {
				walk(v.Elem(), pattern, instance)
			}
		case reflect.Map:
			if len(writes) == 0 {
				return
			}
			keys := v.MapKeys()
			sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface()) })
			for _, key := range keys {
				copy := reflect.New(v.Type().Elem()).Elem()
				copy.Set(v.MapIndex(key))
				walk(copy, pattern+"{}", instance+"/"+fmt.Sprint(key.Interface()))
				*writes[0] = append(*writes[0], func() { v.SetMapIndex(key, copy) })
			}
		case reflect.Interface:
			if !v.IsNil() {
				return
			}
		case reflect.Func, reflect.Chan, reflect.Invalid:
			return
		case reflect.Struct:
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				walk(v.Field(i), pattern+"."+t.Field(i).Name, instance)
			}
		case reflect.Array:
			if v.Type().Elem().Kind() == reflect.Uint8 {
				emit(pattern, instance, v.Slice(0, v.Len()), false)
				return
			}
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), pattern+"[]", fmt.Sprintf("%s[%d]", instance, i))
			}
		case reflect.Slice:
			switch v.Type().Elem().Kind() {
			case reflect.Uint8:
				emit(pattern, instance, v, false)
			case reflect.Uint16:
				emit(pattern, instance, v, true)
			default:
				for i := 0; i < v.Len(); i++ {
					walk(v.Index(i), pattern+"[]", fmt.Sprintf("%s[%d]", instance, i))
				}
			}
		default:
			emit(pattern, instance, v, false)
		}
	}
	walk(v, pattern, instance)
}
