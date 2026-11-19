package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A note is in the font's bytes; an error is UTF-8. A note from an error is
// encoded when set, never when drawn.
var editorTextBytesRules = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`\.note = err\.Error\(\)`), "a note is set from an error without the encoder"},
	{regexp.MustCompile(`editorHostText\(\w+(\.\w+)*\.note\b`), "a note is encoded as it is drawn"},
	{regexp.MustCompile(`\.note = .*plainWord\(`), "a note is set from UTF-8 words"},
	{regexp.MustCompile(`(Errorf|errors\.New)\(.*\be\.word\(`), "an error is built from a word in the font's bytes"},
}

func checkEditorTextBytes(files map[string]string) []string {
	var out []string
	for name, src := range files {
		for i, line := range strings.Split(src, "\n") {
			if strings.Contains(line, "mcpDisplay(") {
				continue
			}
			for _, r := range editorTextBytesRules {
				if r.pattern.MatchString(line) {
					out = append(out, name+":"+itoa(i+1)+": "+r.reason)
				}
			}
		}
	}
	return out
}

func itoa(n int) string {
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

func TestEditorTextKeepsBytesAndUTF8Apart(t *testing.T) {
	if got := checkEditorTextBytes(map[string]string{"p.go": "o.note = err.Error()\nx := fmt.Errorf(e.word(\"a\"))\no.note = e.editorHostText(err.Error())\nreturn fmt.Errorf(\"%s\", e.plainWord(\"a\"))"}); len(got) != 2 {
		t.Fatalf("planted violations: %v", got)
	}
	files := map[string]string{}
	paths, err := filepath.Glob("../../pkg/ui/*.go")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no editor sources: %v", err)
	}
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		files[filepath.Base(p)] = string(b)
	}
	for _, v := range checkEditorTextBytes(files) {
		t.Error(v)
	}
}
