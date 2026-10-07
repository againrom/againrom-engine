package sav

import "testing"

// TestActorBasisSightIsCarriedAndWholeCellIsTheHighByte pins +0xa4
// (ActorBasis.Human.Fields.Sight) through the decoder ActorHoldings already
// exposes it under: a plain 16-bit read, no derive, with the whole-cell scan
// range at the field's own high byte (SAV-795, SAV-796). 0x0600 is one of the
// six values SAV-795's corpus census finds on Unit records (1536, the
// placed-actor mode, 609 of 826) and its high byte is exactly 6 — the
// property e.ScanRange = uint8(basis.Sight >> 8) (pkg/game/
// originalactorregistry.go) already relies on.
func TestActorBasisSightIsCarriedAndWholeCellIsTheHighByte(t *testing.T) {
	c := hero()
	c.sight = 0x0600
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{c}})
	holdings, err := f.ActorHoldings()
	if err != nil {
		t.Fatalf("ActorHoldings: %v", err)
	}
	if len(holdings) == 0 || holdings[0].Basis == nil {
		t.Fatalf("got %d holdings with a basis", len(holdings))
	}
	got := holdings[0].Basis.Human.Fields.Sight
	if got != 0x0600 {
		t.Fatalf("Sight = %#04x, want %#04x", got, 0x0600)
	}
	if cells := uint8(got >> 8); cells != 6 {
		t.Fatalf("whole cells = %d, want 6", cells)
	}
}

// TestActorBasisToken18IsCarried decodes Unit+0x18 (Token+0x18, SAV-636) into
// ActorBasis.Token18, the recipient publication mask SAV-678 and SAV-797
// name and this package carries but wires into no consumer (DIV-969). The
// walk fixture's token() writer always emits the fixed filler 0x1818 for this
// member (matching its own "value spells the offset" convention for every
// other undifferentiated word); this test only pins that the generic Member
// decode a Building's Token18 already exercises also reaches ActorBasis.
func TestActorBasisToken18IsCarried(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	holdings, err := f.ActorHoldings()
	if err != nil {
		t.Fatalf("ActorHoldings: %v", err)
	}
	if len(holdings) == 0 || holdings[0].Basis == nil {
		t.Fatalf("got %d holdings with a basis", len(holdings))
	}
	if got := holdings[0].Basis.Token18; got != 0x1818 {
		t.Fatalf("Token18 = %#04x, want %#04x", got, 0x1818)
	}
}
