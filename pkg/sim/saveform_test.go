package sim

import (
	"strings"
	"testing"
)

func TestCheckSaveFormAdmitsCurrentAndPredecessorOnly(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, nil)
	raw := mustMarshal(t, w)
	if err := CheckSaveForm(raw); err != nil {
		t.Fatalf("current form refused: %v", err)
	}
	if err := CheckSaveForm([]byte{oldestReadableVersion}); err != nil {
		t.Fatalf("predecessor form refused: %v", err)
	}
	for v := 0; v < int(oldestReadableVersion); v++ {
		err := CheckSaveForm([]byte{byte(v)})
		if err == nil || !strings.HasPrefix(err.Error(), "this save is too old to open") {
			t.Fatalf("version %d: %v", v, err)
		}
	}
	if err := CheckSaveForm(nil); err == nil {
		t.Fatal("empty form admitted")
	}
	if err := CheckSaveForm([]byte{200}); err == nil || !strings.Contains(err.Error(), "newer than this build") {
		t.Fatalf("version 200: %v", err)
	}
}
