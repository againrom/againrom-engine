package game

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

func logCurrentCarrierDiff(t *testing.T, a, b *sim.World) {
	t.Helper()
	left, right := a.Entities(), b.Entities()
	for i := range left {
		left[i].NativeBasis = sim.NativeBasisNow(left[i])
	}
	for i := range right {
		right[i].NativeBasis = sim.NativeBasisNow(right[i])
	}
	remaining := 48
	var compare func(string, reflect.Value, reflect.Value)
	compare = func(path string, x, y reflect.Value) {
		if !x.CanInterface() || !y.CanInterface() {
			return
		}
		if remaining == 0 || reflect.DeepEqual(x.Interface(), y.Interface()) {
			return
		}
		if x.Type() != y.Type() {
			t.Logf("current carrier %s type differs", path)
			remaining--
			return
		}
		switch x.Kind() {
		case reflect.Struct:
			for i := range x.NumField() {
				compare(path+"."+x.Type().Field(i).Name, x.Field(i), y.Field(i))
			}
			return
		case reflect.Array, reflect.Slice:
			if x.Len() == y.Len() {
				for i := range x.Len() {
					name := fmt.Sprintf("[%d]", i)
					if strings.HasSuffix(path, ".Scalars") && i < len(nativeScalarFields) {
						name = "[" + nativeScalarFields[i] + "]"
					}
					compare(path+name, x.Index(i), y.Index(i))
				}
				return
			}
		case reflect.Pointer:
			if !x.IsNil() && !y.IsNil() {
				compare(path, x.Elem(), y.Elem())
				return
			}
		}
		t.Logf("current carrier %s: uninterrupted=%v cold=%v", path, x.Interface(), y.Interface())
		remaining--
	}
	compare("Entities", reflect.ValueOf(left), reflect.ValueOf(right))
	compare("RemovedNativeBases", reflect.ValueOf(a.RemovedNativeActorBases()), reflect.ValueOf(b.RemovedNativeActorBases()))
	compare("Actions", reflect.ValueOf(a.Actions()), reflect.ValueOf(b.Actions()))
}
