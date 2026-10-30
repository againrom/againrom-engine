package sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"reflect"
	"testing"
)

func forgetProvenForms() {
	provenForms.mu.Lock()
	defer provenForms.mu.Unlock()
	provenForms.n = 0
}

// provenScalarForms returns a wrapper form and its noncanonical twin: the same
// length and the same decoded scalars, with one maintained bit written the way
// the encoder never writes it.
func provenScalarForms(t *testing.T) (*World, []byte, []byte) {
	t.Helper()
	w := nativeBasisWorld(t)
	e := &w.entities[0]
	basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1<<ScalarUA4 | 1<<ScalarU4C | 1<<ScalarT1C | 1<<ScalarU6C}
	basis.Scalars[ScalarUA4], basis.Scalars[ScalarU4C] = 0x09ab, 0xa1
	basis.Scalars[ScalarT1C], basis.Scalars[ScalarU6C] = 17, 63
	e.NativeBasis = basis
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	twin := bytes.Clone(raw)
	start := len(raw) - 9 - (4 + 105)
	binary.LittleEndian.PutUint32(twin[start+13+4*ScalarUA4:], 0x08ab)
	if _, ok := formWrapperOf(raw[0]); !ok || bytes.Equal(raw, twin) {
		t.Fatal("fixture is not a wrapper form with a distinct twin")
	}
	return w, raw, twin
}

// A second decode of the same bytes into a zero World skips the canonical
// comparisons and must build the World the checked decode built.
func TestAProvenFormDecodesToTheCheckedWorld(t *testing.T) {
	w, raw, _ := provenScalarForms(t)
	forgetProvenForms()
	var checked World
	if err := checked.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if !provenForms.has(sha256.Sum256(raw)) {
		t.Fatal("a checked zero-receiver decode was not remembered")
	}
	var proven World
	if err := proven.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&checked, &proven) || proven.Hash() != w.Hash() {
		t.Fatal("the proven decode differs from the checked decode")
	}
	again, err := proven.MarshalBinary()
	if err != nil || !bytes.Equal(again, raw) {
		t.Fatal("the proven decode does not re-encode to its input", err)
	}
}

// Loss control: a memo keyed on anything weaker than the whole byte string,
// or one that remembered refusals, admits the twin of a proven form.
func TestAProvenFormDoesNotAdmitItsNoncanonicalTwin(t *testing.T) {
	_, raw, twin := provenScalarForms(t)
	forgetProvenForms()
	for range 2 {
		var w World
		if err := w.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		var w World
		if w.UnmarshalBinary(twin) == nil {
			t.Fatal("the noncanonical twin of a proven form was admitted")
		}
		if !reflect.ValueOf(&w).Elem().IsZero() {
			t.Fatal("a refused decode changed its receiver")
		}
	}
	if provenForms.has(sha256.Sum256(twin)) {
		t.Fatal("a refused form was remembered")
	}
}

func TestOnlyAZeroWorldUsesTheProvenFormMemo(t *testing.T) {
	_, raw, _ := provenScalarForms(t)
	var zero World
	if d, ok := zero.formDigest(raw); !ok || d != sha256.Sum256(raw) {
		t.Fatal("a wrapper form into a zero World is outside the memo")
	}
	base := mustWorld(t, 7, Bounds{Width: 2, Height: 2}, nil)
	plain, err := base.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := zero.formDigest(plain); ok || plain[0] != formatVersion {
		t.Fatal("a base form entered the memo")
	}
	if _, ok := zero.formDigest(nil); ok {
		t.Fatal("an empty form entered the memo")
	}
	for name, set := range map[string]func(*World){
		"tick":  func(w *World) { w.tick = 1 },
		"ghost": func(w *World) { w.ghost.MaxHP = 1 },
		"source derive": func(w *World) {
			w.sourceDerive = func(a SourceActor, _ int32, _ Rules) (SourceActor, error) { return a, nil }
		},
		"merge operands": func(w *World) { w.mergeOperands = []ItemWeight{} },
		"burst phases":   func(w *World) { w.burst.phases = 1 },
		"rules": func(w *World) {
			r, err := NewRules(RulesParams{SkillCap: 150})
			if err != nil {
				t.Fatal(err)
			}
			w.rules = r
		},
		"diary":         func(w *World) { w.diary.rows = 1 },
		"loaded record": func(w *World) { w.loadedRecord = LoadedDocument },
	} {
		var w World
		set(&w)
		if _, ok := w.formDigest(raw); ok {
			t.Fatalf("a receiver with %s entered the memo", name)
		}
	}
}

func TestTheProvenFormMemoKeepsTheNewestDigests(t *testing.T) {
	var m provenFormMemo
	digest := func(i int) [sha256.Size]byte { return sha256.Sum256([]byte{byte(i)}) }
	for i := range provenFormCount {
		m.add(digest(i))
	}
	m.add(digest(3))
	if m.n != provenFormCount || m.digests[0] != digest(3) {
		t.Fatal("re-adding a remembered digest did not move it to the front")
	}
	if !m.has(digest(0)) || m.digests[0] != digest(0) {
		t.Fatal("a hit did not move its digest to the front")
	}
	m.add(digest(provenFormCount))
	if m.n != provenFormCount || m.has(digest(1)) {
		t.Fatal("the least recently used digest was kept past the bound")
	}
	for _, i := range []int{0, 2, 3, provenFormCount - 1, provenFormCount} {
		if !m.has(digest(i)) {
			t.Fatalf("digest %d was dropped", i)
		}
	}
}
