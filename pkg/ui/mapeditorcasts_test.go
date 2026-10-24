package ui

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestEditorNativeCellSpellUsesAddBuilder(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	updateEditorKeys(t, e, ebiten.KeyF3)
	if key, _, ok := e.AddStep(); !ok || key != "template" || e.prompt.add.kind != "Enchantment" {
		t.Fatal("F3 did not open the shared Add builder", key, ok)
	}
	updateEditorKeys(t, e, ebiten.KeyEscape)
	if e.prompt.open() {
		t.Fatal("native Cancel retained popup")
	}
	updateEditorKeys(t, e, ebiten.KeyF3)
	for _, v := range []string{"0", "7", "8,8", "13", "20"} {
		_ = e.Key("clear")
		e.TextInput(v)
		updateEditorKeys(t, e, ebiten.KeyEnter)
	}
	if len(calls.lines) != 1 || !strings.HasPrefix(calls.lines[0], "AddRecord Enchantment") {
		t.Fatal("native Enter did not reach AddRecord", calls.lines)
	}
}
