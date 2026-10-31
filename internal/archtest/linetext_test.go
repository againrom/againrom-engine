package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckLineTextNamesEachShape plants each shape of a second install text
// reader the census names and shows the scan names it.
func TestCheckLineTextNamesEachShape(t *testing.T) {
	const consts = "package game\nconst mainPrefix = \"main/\"\nconst HeroPictureAddress = mainPrefix + \"text/heropicture.txt\"\n"
	fn := func(body string) string {
		return "package game\nfunc reader(src entrySource, b []byte) {\n" + body + "\n}\n"
	}
	cases := []struct {
		name, file, src string
		want            string // a substring of the one violation; "" for none
	}{
		{"a read at a text literal", "pkg/game/a.go", fn(`src.ReadFile(mainPrefix + "text/credits.txt")`), "reads an install text address through ReadFile"},
		{"a read at a text pattern", "pkg/game/a.go", fn(`src.ReadFile(fmt.Sprintf("%stext/mission%d.txt", mainPrefix, 1))`), "reads an install text address through ReadFile"},
		{"a read at a Text name", "pkg/game/a.go", fn(`src.ReadFile(MainTextPath)`), "reads an install text address through ReadFile"},
		{"a read at a constant holding text/", "pkg/game/a.go", fn(`src.ReadFile(HeroPictureAddress)`), "reads an install text address through ReadFile"},
		{"a read at a name assigned from a text path", "pkg/game/a.go", fn("addr, ok := EventTextPath(1, 1)\n_ = ok\nsrc.ReadFile(addr)"), "reads an install text address through ReadFile"},
		{"a read at a name assigned from a text literal", "pkg/game/a.go", fn("dir := mainPrefix + \"text/battle/m1/\"\nsrc.ReadFile(dir + name)"), "reads an install text address through ReadFile"},
		{"an Entry read at a text literal", "pkg/game/a.go", fn(`x.Entry("main/text/a.txt")`), "through Entry"},
		{"a ReadEntry read at a text path call", "pkg/game/a.go", fn(`ReadEntry(EventTextPath(1, 1))`), "through ReadEntry"},
		{"a read in a package-level function literal", "pkg/game/a.go",
			"package game\nvar files = gameFiles{read: func(src entrySource) { src.ReadFile(\"main/text/town.txt\") }}\n", "a package-level function literal reads"},
		{"TextCode.Bytes on a parameter", "pkg/game/a.go",
			"package game\nfunc decode(code TextCode, b []byte) []byte { return code.Bytes(b) }\n", "decodes install text with TextCode.Bytes"},
		{"TextCode.Bytes on the install's code", "pkg/game/a.go", fn(`_ = in.textCode().Bytes(b)`), "decodes install text with TextCode.Bytes"},
		{"TextCode.Bytes on a local code", "pkg/game/a.go", fn("c := InstallTextCode(src, e)\n_ = c.Bytes(b)"), "decodes install text with TextCode.Bytes"},
		{"TextCode.FileBytes", "pkg/game/a.go", fn(`_ = TextCode{}.FileBytes(b)`), "decodes install text with TextCode.FileBytes"},
		{"DecodeInstallTextFile", "pkg/game/a.go", fn(`DecodeInstallTextFile(b, 1)`), "calls DecodeInstallTextFile"},
		{"SplitTextTable", "pkg/game/a.go", fn(`SplitTextTable(b)`), "calls SplitTextTable"},
		{"secondGameTextSection", "pkg/game/a.go", fn(`secondGameTextSection(b, "briefing")`), "calls secondGameTextSection"},
		{"a code page converter", "pkg/game/a.go", fn(`vfs.WindowsCyrillicToDOS(b)`), "calls WindowsCyrillicToDOS"},
		{"a #key section scan", "pkg/game/a.go", fn(`strings.Index(string(b), "#"+key)`), "scans a #key section"},
		{"a trim of a reader's answer", "pkg/game/a.go", fn(`strings.TrimSpace(MissionObjectives(src, 1))`), "trims or splits a reader's install text"},
		{"a split of a name a reader filled", "pkg/game/a.go", fn("s := installText(src, textTableSource.named(\"x\"), TextCode{}).text()\nstrings.Split(s, \"\\n\")"), "trims or splits a reader's install text"},
		{"a line text literal", "pkg/game/a.go", "package game\nvar t = lineText{ok: true}\n", "a lineText literal"},
		{"a text source row literal", "pkg/game/a.go", "package game\nvar row = &textSource{codec: fontBytes}\n", "a textSource literal"},
		{"a read in the builder file", "pkg/game/installtext.go", fn(`src.ReadFile(mainPrefix + "text/credits.txt")`), ""},
		{"an os read", "pkg/game/a.go", fn(`os.ReadFile("main/text/x.txt")`), ""},
		{"a read at a map address", "pkg/game/a.go", fn(`src.ReadFile(address)`), ""},
		{"Bytes on a buffer", "pkg/game/a.go", fn(`_ = s.model.Bytes()`), ""},
		{"a trim of other text", "pkg/game/a.go", fn(`strings.TrimSpace(e.Name)`), ""},
		{"a comparison of a reader's answer", "pkg/game/a.go", fn(`strings.EqualFold(MissionObjectives(src, 1), "x")`), ""},
		{"a test file", "pkg/game/a_test.go", fn(`src.ReadFile("main/text/x.txt")`), ""},
		{"a sub-package", "pkg/game/sub/a.go", fn(`src.ReadFile("main/text/x.txt")`), ""},
		{"another package", "pkg/ui/a.go", fn(`src.ReadFile("main/text/x.txt")`), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{"pkg/game/consts.go": consts, tc.file: tc.src}
			vs := CheckLineText(files)
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckLineText = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) {
				t.Fatalf("CheckLineText = %v, want one naming %q", vs, tc.want)
			}
		})
	}
	if vs := CheckLineText(nil); len(vs) != 1 {
		t.Fatalf("CheckLineText(nil) = %v, want one violation", vs)
	}
}

// TestInstallTextHasOneReader is the live-tree half.
func TestInstallTextHasOneReader(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckLineText(files) {
		t.Errorf("%s", v)
	}
}
