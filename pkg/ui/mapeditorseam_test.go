package ui

import (
	"reflect"
	"testing"
)

type seamHost struct{ offered map[reflect.Type]bool }

func (h seamHost) SaveFolder() string { return "" }
func (h seamHost) TextsInside() bool  { return false }
func (h seamHost) SetTextsInside(bool) error {
	return nil
}
func (h seamHost) Offers(seam reflect.Type) bool { return h.offered[seam] }

type plainHost struct{}

func (plainHost) SaveFolder() string { return "" }

// A seam is read when the host implements it and, for a host that names its
// seams, offers it; a host that names none offers all it implements.
func TestSeamReadsWhatTheHostOffers(t *testing.T) {
	host := seamHost{offered: map[reflect.Type]bool{reflect.TypeFor[InspectionSaveFolder](): true}}
	if _, ok := Seam[InspectionSaveFolder](host); !ok {
		t.Fatal("an offered seam is off")
	}
	if _, ok := Seam[InspectionTextsInside](host); ok {
		t.Fatal("a seam the host implements but does not offer is on")
	}
	if _, ok := Seam[InspectionSaveFolder](plainHost{}); !ok {
		t.Fatal("a host that names no seams has a seam off")
	}
	if _, ok := Seam[InspectionTextsInside](plainHost{}); ok {
		t.Fatal("a seam the host does not implement is on")
	}
	if _, ok := Seam[InspectionSaveFolder](nil); ok {
		t.Fatal("a nil host has a seam")
	}
}
