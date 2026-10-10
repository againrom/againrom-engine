package game

import (
	"os"
	"sort"
	"testing"
)

func TestReleaseShippedCampaignDocumentsAllResolve(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the shipped document sweep needs a lawful install")
	}
	f := releaseFront(t)

	missions := make([]int, 0, len(f.Campaign.Value().Chapters))
	for n := range f.Campaign.Value().Chapters {
		missions = append(missions, n)
	}
	sort.Ints(missions)

	texts, pictures := 0, 0
	for _, n := range missions {
		ch := f.Campaign.Value().Chapters[n]
		elements := make([]Document, 0, len(ch.TextDocuments)+len(ch.PictureDocuments))
		for _, v := range ch.TextDocuments {
			elements = append(elements, Document{Value: v, Kind: DocumentText})
		}
		for _, v := range ch.PictureDocuments {
			elements = append(elements, Document{Value: v, Kind: DocumentPicture})
		}
		for _, d := range elements {
			page, ok := LoadDocumentPage(f.Archives.Containers, d, f.textCode())
			if !ok {
				t.Errorf("[Mission%d] grants element %+v and it does not resolve", n, d)
				continue
			}
			switch d.Kind {
			case DocumentText:
				texts++
				if page.Text == "" {
					t.Errorf("[Mission%d] text element %d resolved to an empty string (%s)",
						n, d.Value, DocumentTextPath(d.Value))
				}
				if page.Picture != nil {
					t.Errorf("[Mission%d] text element %d also carried a picture", n, d.Value)
				}
			case DocumentPicture:
				pictures++
				if page.Picture == nil || page.Picture.Bounds().Empty() {
					t.Errorf("[Mission%d] picture element %d resolved to no image (%s)",
						n, d.Value, DocumentPicturePath(d.Value))
				}
				if page.Text != "" {
					t.Errorf("[Mission%d] picture element %d also carried text", n, d.Value)
				}
			}
		}
	}

	if texts == 0 || pictures == 0 {
		t.Fatalf("the campaign registry grants %d text and %d picture elements over %d [Mission] sections; the shipped corpus carries both kinds",
			texts, pictures, len(missions))
	}
	t.Logf("%d text and %d picture elements resolve over %d [Mission] sections", texts, pictures, len(missions))
}

// TestReleaseDocumentPanelArtAndFontLoad requires the panel's own eleven
// bitmaps and font4 to load from the named root. Both loaders size-check what
// they read, so this fails on a mis-sized or missing node rather than on a
// panel drawn with a control's picture off its hit test.
func TestReleaseDocumentPanelArtAndFontLoad(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the document panel art witness needs a lawful install")
	}
	f := releaseFront(t)
	if art := f.documentArt(); art == nil {
		t.Fatal("the document panel art did not load from this root")
	}
	font := f.documentFont()
	if font == nil {
		t.Fatal("font4 did not load from this root")
	}
	if len(font.Glyphs) == 0 {
		t.Fatal("font4 loaded with no glyph records")
	}
	t.Logf("font4 carries %d glyph records", len(font.Glyphs))
}
