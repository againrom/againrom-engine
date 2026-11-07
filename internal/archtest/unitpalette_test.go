package archtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnitPaletteAndStoneShareTheIndexedBlit(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ file, function, builder string }{
		{"pkg/render/terrain/blit.go", "BlitStaticLit", "blitPalette"},
		{"pkg/render/terrain/blit.go", "spriteRampRow", "SpriteRGBA"},
		{"pkg/ui/statics.go", "stoneImage", "spritePixels"},
		{"pkg/ui/spritetint.go", "tintedPixels", "spritePixels"},
		{"pkg/ui/headless_art.go", "headlessSpritePixels", "RGBALit"},
		{"pkg/game/world.go", "entityDraws", "ownerFrame"},
		{"pkg/game/world.go", "ownerFrame", "repalette"},
	} {
		raw, err := os.ReadFile(filepath.Join(root, row.file))
		if err != nil {
			t.Fatal(err)
		}
		calls := editorIdentityCalls(string(raw))
		if calls[row.function][row.builder] != 1 {
			t.Errorf("%s: %s must call %s once", row.file, row.function, row.builder)
		}
		if calls[row.function]["grayscaleRGBA"] != 0 {
			t.Errorf("%s: %s adds a second grayscale transform", row.file, row.function)
		}
	}
}
