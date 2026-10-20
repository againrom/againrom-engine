package game

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/sav"
)

// A SAVE writes Player tail bytes 0..30 from the loaded record and the
// formation byte from the World.
func TestPlayerTailBytesComeFromTheLoadedRecord(t *testing.T) {
	player := func(tail []byte) sav.DocumentRecordData {
		return sav.DocumentRecordData{Class: "Player", Raw: []sav.DocumentRawData{{Name: "PRaw32", Bytes: tail}}}
	}
	loadedTail := make([]byte, sav.PlayerTailLen)
	for i := range loadedTail {
		loadedTail[i] = byte(i + 1)
	}
	built := func() sav.DocumentRecordData {
		tail := make([]byte, sav.PlayerTailLen)
		tail[sav.PlayerTailFormationByte] = 2
		return player(tail)
	}
	loaded := player(loadedTail)
	current := built()
	graftUnknownRecord(&current, &loaded, unknownRecordSpans(), nil)
	want := append(bytes.Clone(loadedTail[:sav.PlayerTailFormationByte]), 2)
	if !bytes.Equal(current.Raw[0].Bytes, want) {
		t.Fatalf("written tail %x, want %x", current.Raw[0].Bytes, want)
	}
	// Control: without the fallback span the World-built zeros are written.
	spans := unknownRecordSpans()
	delete(spans, "Player.r.PRaw32")
	control := built()
	graftUnknownRecord(&control, &loaded, spans, nil)
	if bytes.Equal(control.Raw[0].Bytes, want) {
		t.Fatal("the control kept the loaded tail without the fallback span")
	}
}
