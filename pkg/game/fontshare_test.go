package game

import (
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

func TestDocumentFontSharesTheNameFont(t *testing.T) {
	name := &text.Font{Spacing: FontSpacing, Selector: 1}
	in := &InstallResources{Archives: &Archives{}, ChargenAssets: &ChargenAssets{Presentation: &ui.ChargenPresentation{NameFont: name}}}
	p := &Presentation{}
	if got := p.documentFont(in); got != name {
		t.Fatalf("document font %p, want the name font %p", got, name)
	}
	if !p.docFontCache.Tried() || p.documentFont(in) != name {
		t.Fatal("the shared font is not cached")
	}
}
