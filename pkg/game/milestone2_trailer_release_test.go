package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestReleaseMilestone2Trailer(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := trailerAcceptanceRaw(source.Body, source.TrailerOff)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("natural", func(t *testing.T) { trailerAcceptanceApp(t, raw, want, func() *FrontEnd { return releaseFront(t) }) })
	// All known source trailers were zero at the prior census. Distinct
	// values expose omission/swap bugs on the real load/save composition.
	// This is a private derived input, never an original-runtime witness.
	want = trailerAcceptancePattern()
	for i, v := range want {
		binary.LittleEndian.PutUint32(source.Body[source.TrailerOff+4*i:], v)
	}
	t.Run("derived-nonzero", func(t *testing.T) {
		trailerAcceptanceApp(t, source.Marshal(), want, func() *FrontEnd { return releaseFront(t) })
	})
}
