package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
)

func TestModTextFallbackWritesThePlainFormsBytes(t *testing.T) {
	cases := map[string]string{
		"«»“”„":   `"""""`,
		"‘’‚‹›":   `'''''`,
		"a–b—c−d": `a-b-c-d`,
		"wait…":   `wait...`,
		"a b c":   `a b c`,
	}
	for _, selector := range []int{0, 1} {
		for typed, plain := range cases {
			got := EncodeInstallTextFile([]byte(typed+"\r\n"), selector)
			want := EncodeInstallTextFile([]byte(plain+"\r\n"), selector)
			if !bytes.Equal(got, want) || !bytes.Equal(want, []byte(plain+"\r\n")) {
				t.Errorf("selector %d %q: % x, want % x", selector, typed, got, want)
			}
			if got := EncodeModText(typed, selector); got != plain {
				t.Errorf("selector %d EncodeModText %q: %q", selector, typed, got)
			}
		}
	}
	// Loss control: the installed-text encoder keeps its behaviour. At the
	// Russian selector the guillemet has no byte and is dropped.
	if got := EncodeInstallText("«x", 1); got != "x" {
		t.Errorf("EncodeInstallText changed: %q", got)
	}
	// A character with no byte and no plain form is still dropped.
	if got := EncodeInstallTextFile([]byte("aéb中c"), 1); string(got) != "abc" {
		t.Errorf("dropped: %q", got)
	}
}

func TestModTextBoundaryOfTheUTF8Rule(t *testing.T) {
	// The CP866 bytes of a short Russian word can be valid UTF-8: e2 a0 ac is
	// U+282C, which has no install byte, so the file is served empty.
	legacy := []byte{0xe2, 0xa0, 0xac}
	if got := EncodeInstallTextFile(legacy, 1); len(got) != 0 {
		t.Errorf("e2 a0 ac served as % x", got)
	}
	// A sentence of CP866 letters is not valid UTF-8 and stays unchanged.
	sentence := []byte("\x8f\xe0\xa8\xa2\xa5\xe2 \xe2\xa0\xac\r\n")
	if got := EncodeInstallTextFile(sentence, 1); !bytes.Equal(got, sentence) {
		t.Errorf("sentence % x", got)
	}
}

func TestModTextDroppedNamesTheCharacter(t *testing.T) {
	if r, ok := ModTextDropped([]byte("“quoted” …\r\n"), 1); ok {
		t.Errorf("table characters reported: %q", r)
	}
	if r, ok := ModTextDropped([]byte("ok 中"), 1); !ok || r != '中' {
		t.Errorf("got %q %v", r, ok)
	}
	if _, ok := ModTextDropped([]byte("\xef\xbb\xbfplain\r\n"), 0); ok {
		t.Error("BOM reported")
	}
	if _, ok := ModTextDropped([]byte("\x8f\xe0\xa8"), 1); ok {
		t.Error("legacy file reported")
	}
}

func modTextRuleFixture(t *testing.T) (*MapInspector, string, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "t"), 0o755); err != nil {
		t.Fatal(err)
	}
	host := filepath.Join(dir, "map.alm")
	if err := os.WriteFile(host, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	x := checkInspector()
	x.archives = &Archives{Containers: worldFS(t, synth.File{Path: LanguagePath, Data: []byte("1")})}
	x.mod = &editorMod{dir: dir, id: "m", missions: mod.MissionData{Missions: []mod.Mission{
		{Mod: "m", Key: "k", Number: 25, Map: "map.alm", Texts: "t"}}}}
	return x, dir, host
}

func TestMapRuleModMessageTextNamesADroppedCharacterOncePerFile(t *testing.T) {
	x, dir, host := modTextRuleFixture(t)
	m := built(t, cleanOptions(), nil, nil)
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, "t", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("event01.txt", "«Hi» — ok…\r\n")
	write("event02.txt", "\x8f\xe0\xa8\r\n")
	if rows := ofRule(findings(t, x, m, host), "mod-message-text"); len(rows) != 0 {
		t.Fatalf("table characters and a legacy file warned: %+v", rows)
	}
	write("event03.txt", "中文 and 中\r\n")
	rows := ofRule(findings(t, x, m, host), "mod-message-text")
	if len(rows) != 1 || rows[0].Error || !strings.Contains(rows[0].Text, "t/event03.txt") || !strings.Contains(rows[0].Text, "U+4E2D") {
		t.Fatalf("one warning for event03.txt expected: %+v", rows)
	}
	// Outside a mod save the rule is silent.
	x.mod = nil
	if rows := ofRule(findings(t, x, m, host), "mod-message-text"); len(rows) != 0 {
		t.Fatalf("no mod: %+v", rows)
	}
}

func TestMapReportFollowsThePagesAddedToTheTextsFolder(t *testing.T) {
	x, dir, host := modTextRuleFixture(t)
	m := built(t, withScript(cleanOptions(), messageScript(3, 7)), nil, nil)
	s := &mapInspectionEdits{inspector: x, model: m, source: host, saved: m.Bytes()}
	rule := func() []string {
		var out []string
		for _, r := range ofRule(s.Report(), "mod-message-text") {
			out = append(out, r.Text)
		}
		return out
	}
	if got := rule(); len(got) != 2 {
		t.Fatalf("two missing pages expected: %v", got)
	}
	for _, n := range []string{"event03.txt", "event07.txt"} {
		if err := os.WriteFile(filepath.Join(dir, "t", n), []byte("<part=1>\r\nHi\r\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if n == "event03.txt" {
			if got := rule(); len(got) != 1 {
				t.Fatalf("after one page: %v", got)
			}
		}
	}
	if got := rule(); len(got) != 0 {
		t.Fatalf("after both pages with the map unchanged: %v", got)
	}
}
