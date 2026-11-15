package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

// A draft keyed in another letter case is read as the same file by every
// reader of the drafts over the mod's run: the speakers file and its
// pictures, the terrain file, the strings, the items' sprites, the install
// files and a mission's map.
func TestDraftKeyedInAnotherCaseReadsTheSameThroughEveryOverlay(t *testing.T) {
	speakers := []byte("[[speaker]]\nid = 20000\nname = \"Karl\"\npicture = \"speakers/new.png\"\n")
	drafts := map[string][]byte{
		mod.SpeakersFile:                           speakers,
		"speakers/new.png":                         {1, 2},
		mod.TerrainFile:                            []byte("# rows\n"),
		mod.StringsFile("en"):                      []byte("\"item.star\" = \"Star\"\n"),
		"assets/items/star.png":                    {3},
		"files/common/graphics/interface/a.bmp":    []byte("BMnew"),
		"maps/1.alm":                               {4},
		"files/common/graphics/interface/gone.bmp": nil,
	}
	read := func(drafts map[string][]byte) string {
		view := projectView([]string{"files/common/graphics/interface/gone.bmp"}, drafts)
		var b strings.Builder
		src, ok := view.Draft(mod.SpeakersFile)
		d, err := mod.ReadSpeakers("m", t.TempDir(), mod.SpeakersFile, src, view.Draft)
		fmt.Fprintf(&b, "speakers %t %v %v;", ok, err, d.Speakers[0].PictureBytes)
		terrain, ok := view.Draft(mod.TerrainFile)
		fmt.Fprintf(&b, "terrain %t %q;", ok, terrain)
		look, err := mod.DraftText("en", view.Draft)
		text, _ := look("item.star")
		fmt.Fprintf(&b, "text %v %q;", err, text)
		var result modrt.Result
		err = mod.OverlayDrafts(mod.DataReader{Mod: "m", Dir: t.TempDir(), Lang: "en"}, &result.Data, view)
		fmt.Fprintf(&b, "sprites %v %d;", err, len(result.Items.Pictures))
		files, err := mod.OverlayInstallFiles("m", "en", nil, view)
		fmt.Fprintf(&b, "files %v", err)
		for _, f := range files {
			fmt.Fprintf(&b, " %s:%s=%q", f.Lang, mod.FoldAddress(f.Address), f.Bytes)
		}
		m, ok := view.Draft("maps/1.alm")
		fmt.Fprintf(&b, "; map %t %v; names %d", ok, m, len(view.names))
		return b.String()
	}
	want := read(drafts)
	if strings.Contains(want, "false") || strings.Contains(want, "error") || !strings.Contains(want, "Star") {
		t.Fatalf("the drafts as keyed read %s", want)
	}
	upper := map[string][]byte{}
	for name, b := range drafts {
		upper[strings.ToUpper(name)] = b
	}
	if got := read(upper); got != want {
		t.Fatalf("drafts keyed in capitals read\n%s\nwant\n%s", got, want)
	}
}
