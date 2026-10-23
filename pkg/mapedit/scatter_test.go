package mapedit_test

import (
	"bytes"
	"math/rand"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

func scatterMap(t *testing.T) *mapedit.Editor {
	t.Helper()
	m, err := mapedit.NewMap(alm.NewMapSpec{Width: 32, Height: 32, Tile: 0x14, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func scatterBrush(seed int64, sc mapedit.Scatter) mapedit.Brush {
	sc.Draw = rand.New(rand.NewSource(seed)).Intn
	return mapedit.Brush{Plane: mapedit.PlaneObject, Radius: 6, Shape: mapedit.BrushSquare, Scatter: sc}
}

// One source gives one forest; another gives another; a stroke over a full
// solid forest changes nothing and records nothing.
func TestScatterStrokeFollowsItsSource(t *testing.T) {
	at := []mapedit.Cell{{X: 16, Y: 16}}
	paint := func(seed int64, share int) []byte {
		m := scatterMap(t)
		if _, err := m.Paint(scatterBrush(seed, mapedit.Scatter{Classes: []int{3, 7}, Share: share}), at, testClass); err != nil {
			t.Fatal(err)
		}
		return m.Bytes()
	}
	if !bytes.Equal(paint(1, 40), paint(1, 40)) {
		t.Fatal("one seed gave two forests")
	}
	if bytes.Equal(paint(1, 40), paint(2, 40)) {
		t.Fatal("two seeds gave one forest")
	}
	m := scatterMap(t)
	if _, err := m.Paint(scatterBrush(1, mapedit.Scatter{Classes: []int{3}, Share: 100}), at, testClass); err != nil {
		t.Fatal(err)
	}
	full := m.Bytes()
	changed, err := m.Paint(scatterBrush(2, mapedit.Scatter{Classes: []int{7}, Share: 100}), at, testClass)
	if err != nil || changed || !bytes.Equal(m.Bytes(), full) {
		t.Fatalf("a second solid stroke over a full forest changed %v (%v)", changed, err)
	}
}

// A forest stroke with a share outside 1..100, no source, a class a byte
// cannot hold, a structure plane of the wrong size or no classifier is
// refused and writes nothing.
func TestScatterStrokeRefusesBadSettings(t *testing.T) {
	at := []mapedit.Cell{{X: 16, Y: 16}}
	for name, b := range map[string]mapedit.Brush{
		"share 0":     scatterBrush(1, mapedit.Scatter{Classes: []int{3}, Share: 0}),
		"share 101":   scatterBrush(1, mapedit.Scatter{Classes: []int{3}, Share: 101}),
		"class 256":   scatterBrush(1, mapedit.Scatter{Classes: []int{256}, Share: 50}),
		"class 0":     scatterBrush(1, mapedit.Scatter{Classes: []int{0}, Share: 50}),
		"fenced size": scatterBrush(1, mapedit.Scatter{Classes: []int{3}, Share: 50, Fenced: make([]bool, 5)}),
		"no source":   {Plane: mapedit.PlaneObject, Radius: 2, Scatter: mapedit.Scatter{Classes: []int{3}, Share: 50}},
	} {
		m := scatterMap(t)
		before := m.Bytes()
		if _, err := m.Paint(b, at, testClass); err == nil {
			t.Errorf("%s: the stroke was taken", name)
		}
		if !bytes.Equal(m.Bytes(), before) || m.CanUndo() {
			t.Errorf("%s: a refused stroke wrote bytes or history", name)
		}
	}
	m := scatterMap(t)
	if _, err := m.Paint(scatterBrush(1, mapedit.Scatter{Classes: []int{3}, Share: 50}), at, nil); err == nil {
		t.Error("a forest stroke without a classifier was taken")
	}
}
