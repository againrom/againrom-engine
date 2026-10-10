package game

import (
	"reflect"
	"testing"

	"againrom/pkg/graphcopy"
)

// fillValue gives every exported field reachable from v a non-zero value:
// two elements per slice and map, a target behind every pointer.
func fillCloneSource(v reflect.Value, depth int) {
	if depth > 12 {
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(depth + 1))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(uint64(depth + 1))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(depth) + 0.5)
	case reflect.String:
		v.SetString("x")
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		fillCloneSource(p.Elem(), depth+1)
		v.Set(p)
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := 0; i < 2; i++ {
			fillCloneSource(s.Index(i), depth+1)
		}
		v.Set(s)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fillCloneSource(v.Index(i), depth+1)
		}
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		for i := 0; i < 2; i++ {
			k := reflect.New(v.Type().Key()).Elem()
			fillCloneSource(k, depth+i)
			e := reflect.New(v.Type().Elem()).Elem()
			fillCloneSource(e, depth+1)
			m.SetMapIndex(k, e)
		}
		v.Set(m)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				fillCloneSource(v.Field(i), depth+1)
			}
		}
	}
}

// sharedMemory names the first reference both graphs hold.
func clonedShares(a, b reflect.Value, path string) string {
	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() {
			return ""
		}
		if a.Pointer() == b.Pointer() {
			return path
		}
		return clonedShares(a.Elem(), b.Elem(), path)
	case reflect.Slice:
		if a.Len() == 0 {
			return ""
		}
		if a.Pointer() == b.Pointer() {
			return path
		}
		for i := 0; i < a.Len(); i++ {
			if p := clonedShares(a.Index(i), b.Index(i), path+"[]"); p != "" {
				return p
			}
		}
	case reflect.Array:
		for i := 0; i < a.Len(); i++ {
			if p := clonedShares(a.Index(i), b.Index(i), path+"[]"); p != "" {
				return p
			}
		}
	case reflect.Map:
		if a.IsNil() {
			return ""
		}
		if a.Pointer() == b.Pointer() {
			return path
		}
		for it := a.MapRange(); it.Next(); {
			if p := clonedShares(it.Value(), b.MapIndex(it.Key()), path+"{}"); p != "" {
				return p
			}
		}
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if p := clonedShares(a.Field(i), b.Field(i), path+"."+a.Type().Field(i).Name); p != "" {
				return p
			}
		}
	}
	return ""
}

// The copy a reader receives equals the decoded value and shares no memory
// with it, over every exported field of the supplement's whole type graph.
func TestCurrentActionCloneIsCompleteAndDisjoint(t *testing.T) {
	var src currentActionData
	fillCloneSource(reflect.ValueOf(&src).Elem(), 0)
	out, ok := graphcopy.Clone(&src)
	if !ok {
		t.Fatal("a decoded supplement must be copyable")
	}
	if !reflect.DeepEqual(&src, out) {
		t.Fatal("copy differs from its source")
	}
	if p := clonedShares(reflect.ValueOf(&src).Elem(), reflect.ValueOf(out).Elem(), "actions"); p != "" {
		t.Fatalf("copy shares %s with its source", p)
	}
	var empty currentActionData
	if out, ok := graphcopy.Clone(&empty); !ok || !reflect.DeepEqual(&empty, out) {
		t.Fatal("empty supplement copy differs")
	}
}

// A second read of the same bytes returns the same value, unaffected by what
// the first reader did with its own.
func TestDecodeCurrentActionsReadersDoNotShare(t *testing.T) {
	raw := []byte(`{"Version":1,"Bindings":[{"ID":3,"Structure":false,"Object":7,"Missing":false}],"Objects":[],"Actions":{},"Held":null,"Options":null,"Bolts":null,"Heals":null,"Runs":null,"VisualIdentities":null,"VisualNext":0,"GroupTag":0,"Groups":null,"GroupHighWater":0,"CellCosts":null,"SpellCasters":null}`)
	first, err := decodeCurrentActions(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := decodeCurrentActions(raw)
	if err != nil {
		t.Fatal(err)
	}
	first.Bindings[0].Object = 99
	first.Bindings = append(first.Bindings, currentActionBinding{ID: 4})
	again, err := decodeCurrentActions(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, again) || len(again.Bindings) != 1 || again.Bindings[0].Object != 7 {
		t.Fatalf("a reader's change reached a later reader: %+v", again.Bindings)
	}
	if _, err := decodeCurrentActions(append([]byte(nil), raw[:len(raw)-1]...)); err == nil {
		t.Fatal("truncated supplement decoded")
	}
	if _, err := decodeCurrentActions([]byte(`{"Version":1,"Unknown":1}`)); err == nil {
		t.Fatal("unknown field decoded")
	}
	if _, err := decodeCurrentActions(append(append([]byte(nil), raw...), []byte(` {}`)...)); err == nil {
		t.Fatal("trailing data decoded")
	}
}
