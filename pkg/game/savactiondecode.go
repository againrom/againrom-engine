package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync"
)

// One load and save path reads a document's current-action supplement many
// times over. The JSON decode is the same each time, so the last few decoded
// supplements are kept by their exact bytes and each reader receives its own
// deep copy; validation against the document still runs on every read.
const currentActionDecodeEntries = 4

type currentActionDecoded struct {
	raw   string
	value *currentActionData
}

var currentActionDecodes struct {
	mu      sync.Mutex
	entries []currentActionDecoded
}

// decodeCurrentActions decodes b strictly into a fresh value that shares no
// memory with any other caller's value.
func decodeCurrentActions(b []byte) (*currentActionData, error) {
	currentActionDecodes.mu.Lock()
	for i, e := range currentActionDecodes.entries {
		if e.raw == string(b) {
			if out, ok := cloneDecoded(e.value); ok {
				copy(currentActionDecodes.entries[1:i+1], currentActionDecodes.entries[:i])
				currentActionDecodes.entries[0] = e
				currentActionDecodes.mu.Unlock()
				return out, nil
			}
			break
		}
	}
	currentActionDecodes.mu.Unlock()
	var a currentActionData
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return nil, fmt.Errorf("current actions: %w", err)
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return nil, fmt.Errorf("current actions contain trailing data")
	}
	if kept, ok := cloneDecoded(&a); ok {
		currentActionDecodes.mu.Lock()
		entries := append([]currentActionDecoded{{raw: string(b), value: kept}}, currentActionDecodes.entries...)
		currentActionDecodes.entries = entries[:min(len(entries), currentActionDecodeEntries)]
		currentActionDecodes.mu.Unlock()
	}
	return &a, nil
}

// cloneDecoded deep-copies a value produced by encoding/json. It refuses
// (false) a graph it cannot copy without sharing memory: a set unexported
// reference field, an interface, function or channel.
func cloneDecoded[T any](src *T) (*T, bool) {
	out := new(T)
	if !deepCopyValue(reflect.ValueOf(out).Elem(), reflect.ValueOf(src).Elem()) {
		return nil, false
	}
	return out, true
}

var referenceKinds sync.Map // reflect.Type -> bool

// holdsReferences reports whether copying a value of t by assignment would
// share mutable memory. Strings are immutable and count as plain.
func holdsReferences(t reflect.Type) bool {
	if v, ok := referenceKinds.Load(t); ok {
		return v.(bool)
	}
	var r bool
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		r = true
	case reflect.Array:
		r = holdsReferences(t.Elem())
	case reflect.Struct:
		// Store a provisional answer so a recursive type terminates; a
		// recursive struct necessarily reaches itself through a reference.
		referenceKinds.Store(t, true)
		for i := 0; i < t.NumField(); i++ {
			if holdsReferences(t.Field(i).Type) {
				r = true
				break
			}
		}
	}
	referenceKinds.Store(t, r)
	return r
}

func deepCopyValue(dst, src reflect.Value) bool {
	t := src.Type()
	if !holdsReferences(t) {
		dst.Set(src)
		return true
	}
	switch src.Kind() {
	case reflect.Pointer:
		if src.IsNil() {
			return true
		}
		p := reflect.New(t.Elem())
		if !deepCopyValue(p.Elem(), src.Elem()) {
			return false
		}
		dst.Set(p)
	case reflect.Slice:
		if src.IsNil() {
			return true
		}
		n := src.Len()
		s := reflect.MakeSlice(t, n, n)
		if !holdsReferences(t.Elem()) {
			reflect.Copy(s, src)
		} else {
			for i := 0; i < n; i++ {
				if !deepCopyValue(s.Index(i), src.Index(i)) {
					return false
				}
			}
		}
		dst.Set(s)
	case reflect.Map:
		if src.IsNil() {
			return true
		}
		if holdsReferences(t.Key()) {
			return false
		}
		m := reflect.MakeMapWithSize(t, src.Len())
		for it := src.MapRange(); it.Next(); {
			v := reflect.New(t.Elem()).Elem()
			if !deepCopyValue(v, it.Value()) {
				return false
			}
			m.SetMapIndex(it.Key(), v)
		}
		dst.Set(m)
	case reflect.Array:
		for i := 0; i < src.Len(); i++ {
			if !deepCopyValue(dst.Index(i), src.Index(i)) {
				return false
			}
		}
	case reflect.Struct:
		dst.Set(src)
		for i := 0; i < src.NumField(); i++ {
			f := t.Field(i)
			if !holdsReferences(f.Type) {
				continue
			}
			if !f.IsExported() {
				if !src.Field(i).IsZero() {
					return false
				}
				continue
			}
			field := dst.Field(i)
			field.SetZero()
			if !deepCopyValue(field, src.Field(i)) {
				return false
			}
		}
	default:
		// Interface, function, channel and unsafe pointer values.
		return src.IsNil()
	}
	return true
}
