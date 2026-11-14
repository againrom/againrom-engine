package archtest

import (
	"os"
	"strings"
	"testing"
)

// The offer rows of a towns file have one writer: the table header of an offer
// row is written only by the row writer, never by a loop of the towns file.
func TestTownsFileWritesOfferRowsThroughTheRowWriter(t *testing.T) {
	b, err := os.ReadFile("../../pkg/mod/towns.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `[[%s]]\n`) {
		t.Fatal("towns.go writes a table header itself; offer rows go through writeRows")
	}
}
