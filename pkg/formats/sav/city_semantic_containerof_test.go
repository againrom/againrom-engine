package sav

import (
	"encoding/binary"
	"testing"
)

func cityIdentityToken(identity, owner uint32) []byte {
	b := make([]byte, 37)
	binary.LittleEndian.PutUint32(b[29:33], identity)
	binary.LittleEndian.PutUint32(b[33:37], owner)
	return b
}

// A base document not this package's own writing can bake an item's owner
// token to a Human who already left the roster before it was written, with
// nothing left afterwards to rewrite the item's own frozen owner byte:
// remintCityIdentities cannot translate an owner identity no surviving object
// carries. Parsing that base document (tolerateStaleOwners true) recovers the
// current owner from the retained Human's own container edge instead of
// refusing the whole document.
func TestRemintCityIdentitiesRecoversOwnerFromRetainedContainer(t *testing.T) {
	const retainedIdentity, itemIdentity, staleOwnerIdentity = 0x1000, 0x2000, 0x9999
	item := &cityObject{sourceIndex: 2, class: "Item", item: &cityItem{token: cityIdentityToken(itemIdentity, staleOwnerIdentity)}}
	human := &cityObject{sourceIndex: 1, class: "Human", unit: &cityUnit{
		token:     cityIdentityToken(retainedIdentity, 0),
		container: []*cityObject{item},
	}}
	d := &cityDocument{objects: map[uint16]*cityObject{1: human, 2: item}}

	if err := remintCityIdentities(d, true); err != nil {
		t.Fatalf("remintCityIdentities: %v, want the item's stale owner %#08x recovered from its retaining Human's own container edge", err, staleOwnerIdentity)
	}
	mintedOwner := binary.LittleEndian.Uint32(human.unit.token[29:33])
	itemOwner := binary.LittleEndian.Uint32(item.item.token[33:37])
	if itemOwner != mintedOwner {
		t.Fatalf("item owner after remint = %#08x, want the retained Human's own reminted identity %#08x", itemOwner, mintedOwner)
	}
}

// Without a retaining container edge, an item whose owner names an identity
// no surviving object carries is a genuine unsupported document, not a gap
// containerOf can or should paper over, no matter which caller asks.
func TestRemintCityIdentitiesStillRefusesAnUnrecoverableStaleOwner(t *testing.T) {
	const itemIdentity, staleOwnerIdentity = 0x2000, 0x9999
	item := &cityObject{sourceIndex: 1, class: "Item", item: &cityItem{token: cityIdentityToken(itemIdentity, staleOwnerIdentity)}}
	d := &cityDocument{objects: map[uint16]*cityObject{1: item}}

	if err := remintCityIdentities(d, true); err == nil {
		t.Fatal("remintCityIdentities: want an error for an owner identity no object or container edge recovers")
	}
}

// This package's own export never guesses at an owner reference it cannot
// mint, even when a container edge would happily supply one: the same shape
// that a departed companion's frozen leftover produces in an imported base
// document also matches a corrupted current owner byte, and only the base
// document is a source this package did not just validate on the way in.
func TestRemintCityIdentitiesRefusesAStaleOwnerOnExportEvenWithARetainedContainer(t *testing.T) {
	const retainedIdentity, itemIdentity, staleOwnerIdentity = 0x1000, 0x2000, 0x9999
	item := &cityObject{sourceIndex: 2, class: "Item", item: &cityItem{token: cityIdentityToken(itemIdentity, staleOwnerIdentity)}}
	human := &cityObject{sourceIndex: 1, class: "Human", unit: &cityUnit{
		token:     cityIdentityToken(retainedIdentity, 0),
		container: []*cityObject{item},
	}}
	d := &cityDocument{objects: map[uint16]*cityObject{1: human, 2: item}}

	if err := remintCityIdentities(d, false); err == nil {
		t.Fatal("remintCityIdentities: want export (tolerateStaleOwners false) to refuse rather than recover a stale owner from a container edge")
	}
}
