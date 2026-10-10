package graphcopy

import (
	"reflect"
	"testing"
)

func TestCloneRefusesWhatItCannotCopyApart(t *testing.T) {
	type hidden struct {
		Shown []int
		kept  *int
	}
	n := 1
	if _, ok := Clone(&hidden{kept: &n}); ok {
		t.Fatal("a set unexported reference cannot be copied")
	}
	if out, ok := Clone(&hidden{Shown: []int{1}}); !ok || out.Shown[0] != 1 {
		t.Fatal("an unset unexported reference is copyable")
	}
	type held struct{ Any any }
	if _, ok := Clone(&held{Any: 1}); ok {
		t.Fatal("an interface value cannot be copied")
	}
	if _, ok := Clone(&held{}); !ok {
		t.Fatal("a nil interface is copyable")
	}
}

func TestCloneKeepsNilAndEmptyApart(t *testing.T) {
	type graph struct {
		Nil, Empty   []string
		NilM, EmptyM map[string][]int
		P            *[2][]byte
		Name         string
	}
	src := graph{Empty: []string{}, EmptyM: map[string][]int{}, P: &[2][]byte{{1}, nil}, Name: "n"}
	out, ok := Clone(&src)
	if !ok || !reflect.DeepEqual(&src, out) || out.Nil != nil || out.Empty == nil || out.NilM != nil || out.EmptyM == nil {
		t.Fatalf("copy %+v differs from %+v", out, src)
	}
	out.P[0][0] = 9
	if src.P[0][0] != 1 || out.P == src.P {
		t.Fatal("copy shares memory with its source")
	}
}
