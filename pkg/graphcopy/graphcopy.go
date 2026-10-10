// Package graphcopy deep-copies decoded value graphs, so a decoder can keep
// one decoded value and hand each reader its own copy.
package graphcopy

import (
	"reflect"
	"sync"
)

// Clone deep-copies a decoded value graph: no pointer, slice or map of the
// copy is shared with src, and nil and empty slices and maps keep their
// distinction. It refuses (false) a graph it cannot copy without sharing
// memory: a set unexported reference field, or an interface, function or
// channel value.
func Clone[T any](src *T) (*T, bool) {
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
