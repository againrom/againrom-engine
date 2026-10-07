package mod

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func textOf(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

const goodItems = `# armour
[[item]]
key      = "gambeson"
name     = "item.gambeson"
slot     = "body"
defence  = 2
weight   = 3
price    = 120
sprite   = "assets/sprites/gambeson.png"
stand-in = "Soft Mail"
shop     = true

[[change]]
item   = "Chain Mail"
weight = 5
`

func TestParseItemsReadsRowsAndChanges(t *testing.T) {
	d, err := ParseItems("heavy-armor", ItemsFile, []byte(goodItems), textOf(map[string]string{"item.gambeson": "Gambeson"}), "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Rows) != 1 || len(d.Changes) != 1 {
		t.Fatalf("rows %d changes %d", len(d.Rows), len(d.Changes))
	}
	r := d.Rows[0]
	if r.Key != "gambeson" || r.Name != "Gambeson" || r.SlotNo != 7 || r.Defence != 2 || r.Weight != 3 || r.Price != 120 ||
		r.Sprite != "assets/sprites/gambeson.png" || r.StandIn != "Soft Mail" || !r.Stock || r.Suit != SuitAny || r.Line != 2 || r.Mod != "heavy-armor" {
		t.Fatalf("row %+v", r)
	}
	c := d.Changes[0]
	if c.Item != "Chain Mail" || c.Weight == nil || *c.Weight != 5 || c.Price != nil || c.Line != 13 {
		t.Fatalf("change %+v", c)
	}
}

func TestParseItemsRefusalsNameFileAndLine(t *testing.T) {
	row := func(extra string) string {
		return "[[item]]\nkey = \"a\"\nname = \"n\"\nslot = \"body\"\nstand-in = \"Soft Mail\"\n" + extra
	}
	cases := []struct {
		name, src string
		line      int
		want      string
	}{
		{"unknown slot", strings.Replace(row(""), `"body"`, `"belt"`, 1), 4, `unknown slot "belt"`},
		{"weapon slot", strings.Replace(row(""), `"body"`, `"weapon"`, 1), 4, "cannot add a weapon"},
		{"missing key", "[[item]]\nname = \"n\"\nslot = \"body\"\nstand-in = \"x\"\n", 1, "has no key"},
		{"missing stand-in", "[[item]]\nkey = \"a\"\nname = \"n\"\nslot = \"body\"\n", 1, "has no stand-in"},
		{"missing text", strings.Replace(row(""), `"n"`, `"item.none"`, 1), 3, `text key "item.none" is not in text/en/strings.toml`},
		{"unknown key", row("colour = 3\n"), 6, `unknown key "colour"`},
		{"wrong type", row("defence = \"high\"\n"), 6, "defence must be an integer"},
		{"out of range", row("defence = 900\n"), 6, "outside 0..250"},
		{"negative", row("price = -1\n"), 6, "outside 0.."},
		{"bad sprite", row("sprite = \"../x.png\"\n"), 6, "not a clean path"},
		{"sprite extension", row("sprite = \"x.bmp\"\n"), 6, "not a .png file"},
		{"bad for", row("for = \"all\"\n"), 6, "use any, fighter or mage"},
		{"duplicate key", row("") + row(""), 6, `item key "a" is already used at line 1`},
		{"stray top key", "key = \"a\"\n", 1, "outside an [[item]] or [[change]] table"},
		{"unknown table", "[item]\nkey = \"a\"\n", 0, "line 1: arrays of tables"},
		{"empty change", "[[change]]\nitem = \"Chain Mail\"\n", 1, "changes nothing"},
		{"change without item", "[[change]]\nprice = 3\n", 1, "has no item"},
		{"syntax", "[[item]]\nkey\n", 2, "expected key = value"},
	}
	for _, c := range cases {
		_, err := ParseItems("m", ItemsFile, []byte(c.src), textOf(map[string]string{"n": "Name"}), "en")
		var fe *ItemFileError
		if err == nil {
			t.Errorf("%s: accepted", c.name)
			continue
		}
		if c.name == "unknown table" {
			// The plain-table form is refused by the table name, not the parser.
			if !strings.Contains(err.Error(), "unknown table") {
				t.Errorf("%s: %v", c.name, err)
			}
			continue
		}
		if !errors.As(err, &fe) || fe.File != ItemsFile || fe.Line != c.line || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v (file %v)", c.name, err, fe)
		}
	}
}

func TestParseItemsRefusesNamesOutsideRange(t *testing.T) {
	src := "[[item]]\nkey = \"a\"\nname = \"n\"\nslot = \"body\"\nstand-in = \"Soft Mail\"\n"
	long := strings.Repeat("x", 41)
	if _, err := ParseItems("m", ItemsFile, []byte(src), textOf(map[string]string{"n": long}), "en"); err == nil || !strings.Contains(err.Error(), "1 to 40") {
		t.Fatalf("long name: %v", err)
	}
	if _, err := ParseItems("m", ItemsFile, []byte(src), textOf(map[string]string{"n": ""}), "en"); err == nil {
		t.Fatal("empty name accepted")
	}
}

func TestParseStringsReadsQuotedKeysAndNamesTheLine(t *testing.T) {
	s, err := ParseStrings("text/ru/strings.toml", []byte("# c\n\"item.gambeson\" = \"Поддоспешник\"\nplain = 'x'\n"))
	if err != nil || s["item.gambeson"] != "Поддоспешник" || s["plain"] != "x" {
		t.Fatalf("%v %v", s, err)
	}
	for _, src := range []string{"\"a\" = 3\n", "[t]\n", "\"a\" 3\n", "\"\" = \"x\"\n"} {
		if _, err := ParseStrings("text/en/strings.toml", []byte(src)); err == nil || !strings.HasPrefix(err.Error(), "text/en/strings.toml:") {
			t.Errorf("%q: %v", src, err)
		}
	}
}

func TestTextLookupFallsBackToEnglish(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "text/en/strings.toml", "\"a\" = \"Alpha\"\n\"b\" = \"Beta\"\n")
	write(t, dir, "text/ru/strings.toml", "\"a\" = \"Альфа\"\n")
	look, err := TextLookup(dir, "ru")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := look("a"); v != "Альфа" {
		t.Fatalf("a = %q", v)
	}
	if v, _ := look("b"); v != "Beta" {
		t.Fatalf("b = %q", v)
	}
	if _, ok := look("c"); ok {
		t.Fatal("c found")
	}
	en, _ := TextLookup(dir, "en")
	if v, _ := en("a"); v != "Alpha" {
		t.Fatalf("en a = %q", v)
	}
}

func TestItemSlotNamesAreInSlotOrder(t *testing.T) {
	names := ItemSlotNames()
	if names[0] != "shield" || names[len(names)-1] != "boots" || len(names) != len(ItemSlots) {
		t.Fatal(names)
	}
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseItemsReadsLayers(t *testing.T) {
	src := "[[item]]\nkey = \"a\"\nname = \"n\"\nslot = \"body\"\nstand-in = \"Soft Mail\"\n"
	text := textOf(map[string]string{"n": "Name"})
	cases := []struct {
		name, extra string
		layer       string
		anchor      string
		anchorNo    int
		figure      string
	}{
		{"no layer", "", "", "", 0, ""},
		{"main is no layer", "layer = \"main\"\n", "", "", 0, ""},
		{"under defaults to the slot", "layer = \"under\"\n", LayerUnder, "body", 7, ""},
		{"over with an anchor", "layer = \"over\"\nanchor = \"legs\"\nfigure = \"Chain Boots\"\n", LayerOver, "legs", 11, "Chain Boots"},
		{"anchor before layer", "anchor = \"head\"\nlayer = \"under\"\n", LayerUnder, "head", 6, ""},
	}
	for _, c := range cases {
		d, err := ParseItems("m", ItemsFile, []byte(src+c.extra), text, "en")
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		r := d.Rows[0]
		if r.Layer != c.layer || r.Anchor != c.anchor || r.AnchorNo != c.anchorNo || r.Figure != c.figure {
			t.Errorf("%s: layer %q anchor %q/%d figure %q", c.name, r.Layer, r.Anchor, r.AnchorNo, r.Figure)
		}
	}
	// A layer on the shield table anchors to a slot of its own choosing.
	shield := strings.Replace(src, `"body"`, `"shield"`, 1) + "layer = \"over\"\nanchor = \"body\"\n"
	d, err := ParseItems("m", ItemsFile, []byte(shield), text, "en")
	if err != nil || d.Rows[0].SlotNo != 2 || d.Rows[0].AnchorNo != 7 {
		t.Fatalf("shield-table layer: %+v %v", d.Rows, err)
	}
}

func TestParseItemsRefusesBadLayers(t *testing.T) {
	src := "[[item]]\nkey = \"a\"\nname = \"n\"\nslot = \"body\"\nstand-in = \"Soft Mail\"\n"
	cases := []struct {
		name, extra string
		line        int
		want        string
	}{
		{"unknown placement", "layer = \"beside\"\n", 6, `layer is "beside"; use under, main or over`},
		{"layer type", "layer = 3\n", 6, "layer must be a string"},
		{"unknown anchor", "layer = \"over\"\nanchor = \"belt\"\n", 7, `unknown anchor "belt"`},
		{"anchor without layer", "anchor = \"body\"\n", 6, `anchor needs layer = "under" or "over"`},
		{"anchor with main", "layer = \"main\"\nanchor = \"body\"\n", 7, `anchor needs layer = "under" or "over"`},
		{"empty figure", "figure = \" \"\n", 6, "figure is empty"},
	}
	for _, c := range cases {
		_, err := ParseItems("m", ItemsFile, []byte(src+c.extra), textOf(map[string]string{"n": "Name"}), "en")
		var fe *ItemFileError
		if err == nil || !errors.As(err, &fe) || fe.File != ItemsFile || fe.Line != c.line || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestLoadStringsReadsInsideTheModFolderOnly(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	write(t, outside, "strings.toml", "\"a\" = \"Outside\"\n")
	if err := os.MkdirAll(filepath.Join(dir, "text"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "..", filepath.Base(outside)), filepath.Join(dir, "text", "en")); err != nil {
		t.Skipf("no symbolic links here: %v", err)
	}
	_, err := LoadStrings(dir, "en")
	var fe *ItemFileError
	if err == nil || !errors.As(err, &fe) || !strings.Contains(err.Error(), "outside the mod folder") {
		t.Fatalf("a strings file reached through a link out of the folder: %v", err)
	}
	if s, err := LoadStrings(dir, "ru"); err != nil || s != nil {
		t.Fatalf("an absent language: %v %v", s, err)
	}
}
