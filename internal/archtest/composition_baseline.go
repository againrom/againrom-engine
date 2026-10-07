package archtest

// CommittedComposition is the one committed record of pkg/game's composition
// root. Regenerate both numbers with:
//
//	go run ./internal/archtest/cmd/composition
//
// then read them before pasting: a fall is the point of the guard and must be
// written down in the commit that caused it, and a rise is a regression this
// file must not absorb.
//
// Fields is every field across FrontEnd's five owned components. Coordinators
// is how many non-test function declarations in pkg/game reach three or more
// of those components at once through a FrontEnd-typed value, resolved by
// go/types rather than by how the reach is spelled.
//
// 20 to 18 is a MEASUREMENT CORRECTION, not "two files fixed". This ratchet's
// prior walk counted the FILE, unioning every method's reach together, so two or
// more methods that each touched under three components could still union
// past the threshold, and a reach through an untracked local alias did not
// count at all - real in this tree, not just a probe: RestoreOriginal
// (pkg/game/originalsave.go) copies the receiver to a local value, draft :=
// *f, and every draft.field reach after that was invisible to the old walk.
// Measured per function on the same tree: 14 files hold 18 coordinating
// functions - frontend.go holds four and gameoptions.go two - seven of the
// twenty prior file entries drop out (docsart.go, sound_channels.go,
// speech.go, speech_witness.go, townexterior.go, townscreen.go, worldmap.go:
// each was a union of methods no one of which alone reaches three
// components), and originalsave.go is newly visible.
// 81 to 82 is DIV-1385's own textSmoothingOff, one new PersistenceContext
// field mirroring tipsOff's shape exactly (the presentation-layer text
// overlay's own on/off switch). Coordinators is unchanged at 17.
// 17 to 16: arriveInTown and enterMission stop reaching three components, and
// missionPorts is the one new coordination point that wires the entry's ports.
// 16 to 15: restoreOriginal is a decode, a detached-session operation and two
// short coordinators, and none of them reaches three components alone.
var CommittedComposition = CompositionBaseline{
	Fields:       80,
	Coordinators: 15,
}
