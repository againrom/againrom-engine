package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestEditorMissionEmitterUsesTheSharedWriter(t *testing.T) {
	b, err := os.ReadFile("../../pkg/game/mapedittestplay.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "mod.WriteMissions(") || strings.Contains(s, `"[[mission]]`) {
		t.Fatal("editor missions must use the shared declaration writer")
	}
}

func TestModWindowUsesTheSharedEditorRows(t *testing.T) {
	b, err := os.ReadFile("../../pkg/ui/mapeditormod.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "e.objectFieldRows(") || strings.Contains(s, "drawPushButton(") || strings.Contains(s, "func draw") || strings.Contains(s, "NewPicker(") {
		t.Fatal("mod window must use the shared popup rows and choices")
	}
}
