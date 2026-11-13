package storydoc

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func firstBound(t *testing.T, root string) int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "docs", "STORY-FORMAT.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`from story (\d+) on`).FindSubmatch(b)
	if m == nil {
		t.Fatal("docs/STORY-FORMAT.md names no first bound story")
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

func TestStoryDocumentsFollowTheFormat(t *testing.T) {
	root := repoRoot(t)
	first := firstBound(t, root)
	dirs, err := filepath.Glob(filepath.Join(root, "docs", "[0-9][0-9][0-9][0-9]"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		id, _ := strconv.Atoi(filepath.Base(d))
		if id < first {
			continue
		}
		b, err := os.ReadFile(filepath.Join(d, "story.md"))
		if err != nil {
			t.Errorf("docs/%s: no story.md (%v)", filepath.Base(d), err)
			continue
		}
		for _, p := range Check(string(b)) {
			t.Errorf("docs/%s/story.md: %s", filepath.Base(d), p)
		}
	}
}

func TestTheFormatExampleHolds(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "good.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bad := Check(string(b)); len(bad) != 0 {
		t.Fatalf("good.md: %q", bad)
	}
}

func TestEachDepartureIsNamed(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "good.md"))
	if err != nil {
		t.Fatal(err)
	}
	good := string(b)
	cases := []struct {
		name, old, new, want string
	}{
		{"a code word in Result", "can now set", "can now set `x`", "Result holds a backtick"},
		{"a section missing", "## Open debt\n", "", "sections"},
		{"a rule without proof", "\n\n## Code\n", "\n4. Unproved.\n\n## Code\n", "rule 4 has no Proof row"},
		{"a rule not numbered", "1. Every declared", "Every declared", "not a numbered one-line rule"},
		{"a long line", "## Authority\n", "## Authority\n\n" + strings.Repeat("x", 121) + "\n", "over 120"},
		{"a long paragraph", "## Result\n\n", "## Result\n\n" + strings.Repeat("a line.\n", 6) + "\n", "a paragraph of 6 lines"},
		{"too many proof rows", "| 3 |\n", "| 3 |\n" + strings.Repeat("| `TestX` | 1 |\n", 12), "Proof has 14 rows"},
		{"too large", "## Open debt\n", "## Open debt\n\n" + strings.Repeat("- debt item.\n", 600), "bytes, over"},
	}
	for _, c := range cases {
		text := strings.Replace(good, c.old, c.new, 1)
		if text == good {
			t.Fatalf("%s: the edit did not apply", c.name)
		}
		found := false
		for _, p := range Check(text) {
			if strings.Contains(p, c.want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: want %q in %q", c.name, c.want, Check(text))
		}
	}
}
