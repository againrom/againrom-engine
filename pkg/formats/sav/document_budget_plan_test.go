package sav

import (
	"fmt"
	"reflect"
	"testing"
)

// referenceBudgetCheck is the budget walk over every field and element, which
// the planned walk must answer as: the same error and the same totals.
func referenceBudgetCheck(b *documentDataBudget, v reflect.Value, depth int) error {
	if depth > maxDocumentDataDepth {
		return fmt.Errorf("sav: document data depth bound exceeded")
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			if err := b.add(uint64(v.Type().Elem().Size()), 1); err != nil {
				return err
			}
			return referenceBudgetCheck(b, v.Elem(), depth+1)
		}
	case reflect.String:
		return b.add(uint64(v.Len()), 0)
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice {
			count := uint64(v.Len())
			elements := count
			if v.Type().Elem().Kind() == reflect.Uint8 {
				elements = 0
			}
			if count > maxDocumentDataBytes/uint64(max(1, v.Type().Elem().Size())) {
				return fmt.Errorf("sav: document data slice byte bound exceeded")
			}
			if err := b.add(count*uint64(v.Type().Elem().Size()), elements); err != nil {
				return err
			}
		}
		switch v.Type().Elem().Kind() {
		case reflect.Struct, reflect.Array, reflect.Slice, reflect.Pointer, reflect.String:
			for i := 0; i < v.Len(); i++ {
				if err := referenceBudgetCheck(b, v.Index(i), depth+1); err != nil {
					return err
				}
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := referenceBudgetCheck(b, v.Field(i), depth+1); err != nil {
				return err
			}
		}
	case reflect.Map, reflect.Interface:
		return fmt.Errorf("sav: document data cannot contain maps or interfaces")
	}
	return nil
}

func TestPlannedBudgetWalkAnswersAsTheFullWalk(t *testing.T) {
	data, err := DecodeDocumentData(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	values := []reflect.Value{reflect.ValueOf(data), reflect.ValueOf(data.Objects), reflect.ValueOf(&data.State), reflect.ValueOf(data.Campaign)}
	for _, v := range values {
		for depth := 0; depth <= maxDocumentDataDepth+1; depth++ {
			var got, want documentDataBudget
			gotErr, wantErr := got.check(v, depth), referenceBudgetCheck(&want, v, depth)
			if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) || got != want {
				t.Fatalf("%s at depth %d: %v %+v, want %v %+v", v.Type(), depth, gotErr, got, wantErr, want)
			}
		}
	}
	full := documentDataBudget{bytes: maxDocumentDataBytes - 64}
	near := full
	ref := full
	if fmt.Sprint(near.check(reflect.ValueOf(data), 0)) != fmt.Sprint(referenceBudgetCheck(&ref, reflect.ValueOf(data), 0)) || near != ref {
		t.Fatalf("near the byte bound: %+v, want %+v", near, ref)
	}
}
