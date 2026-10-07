package ui

import (
	"image"
	"testing"
)

const soundFogGrid = 16

// soundFogViewer is numeralViewer with a fog plane in which exactly one cell
// is visible: the cell numeralUnit stands on. Everything else is unseen, so a
// unit moved one cell over is a unit in the dark.
func soundFogViewer(t *testing.T, visible image.Point) *Viewer {
	t.Helper()
	v := numeralViewer(t)
	plane := make([]byte, soundFogGrid*soundFogGrid) // the zero value is FogUnseen
	plane[visible.Y*soundFogGrid+visible.X] = FogVisible
	v.SetFog(plane, soundFogGrid, soundFogGrid)
	return v
}

// TestAGruntInTheFogIsSilent is stepSound's entity gate: an enemy struck in a
// cell this participant cannot see makes no sound, exactly as it draws no
// numeral. A local owner's own unit in the same cell still sounds, because
// fogGateEntity excepts him — the same exception every other gated draw takes.
func TestAGruntInTheFogIsSilent(t *testing.T) {
	dark := image.Pt(2, 2)

	t.Run("an enemy in the dark is silent", func(t *testing.T) {
		v := soundFogViewer(t, image.Pt(6, 6))
		rec := &recordingPlayer{}
		v.SetAudio(rec, soundBankFor())
		v.SetSpeechAudio(rec)

		e := soundUnit(1, 2, 100) // owner 2: not the local participant
		e.Cell = dark
		push(v, at0, e)
		push(v, at0, blow(v, &e, 60))

		if len(rec.plays) != 0 {
			t.Fatalf("plays = %d, want 0 — an enemy struck in the fog is silent", len(rec.plays))
		}
	})

	t.Run("the local owner's own unit in the same cell still sounds", func(t *testing.T) {
		v := soundFogViewer(t, image.Pt(6, 6))
		rec := &recordingPlayer{}
		v.SetAudio(rec, soundBankFor())
		v.SetSpeechAudio(rec)

		e := soundUnit(1, 0, 100) // owner 0 is the viewer's own default local owner
		e.Cell = dark
		push(v, at0, e)
		push(v, at0, blow(v, &e, 60))

		if len(rec.plays) != 1 {
			t.Fatalf("plays = %d, want 1 — the local owner is excepted by the gate", len(rec.plays))
		}
	})
}

// TestASwingInTheFogIsSilent is playSlotAt's CELL gate, reached through the
// exported entry the tier that owns the attack-run counter calls. It carries
// no entity and so no owner, which is the whole reason a second gate exists.
func TestASwingInTheFogIsSilent(t *testing.T) {
	seen := image.Pt(6, 6)
	v := soundFogViewer(t, seen)
	rec := &recordingPlayer{}
	v.SetAudio(rec, soundBankFor())
	v.SetSpeechAudio(rec)

	v.PlaySlotAt(soundSwingSlot, 2, image.Pt(2, 2))
	if len(rec.plays) != 0 {
		t.Fatalf("plays = %d, want 0 — a swing in an unseen cell is silent", len(rec.plays))
	}

	v.PlaySlotAt(soundSwingSlot, 2, seen)
	if len(rec.plays) != 1 {
		t.Fatalf("plays = %d, want 1 — a swing in a visible cell sounds", len(rec.plays))
	}
}
