package storyguard

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"againrom/internal/archtest"
)

// TestLiveTreeClean scans the real module tree and asserts it matches the
// committed Baseline exactly: zero story-numbered identifiers and file names
// in non-test code, and every ratcheted count equal to Committed.
func TestLiveTreeClean(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := archtest.FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	report, err := Scan(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if vs := Check(report, Committed); len(vs) != 0 {
		for _, v := range vs {
			t.Errorf("%s", v)
		}
	}
}

// TestDigitRun proves the range bound and the exactly-4-digit rule that keep
// this guard from sweeping up unrelated numeric identifiers.
func TestDigitRun(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		// The first two are the shape this guard exists to catch. Nothing in
		// the tree is named either one any more, which is the point: these
		// cases pin the rule, not a live identifier.
		{"cityholdings1172", "1172"},
		{"savedialog1173", "1173"},
		{"probe3375", ""},        // above the ceiling: 4 digits, not this project's numbering
		{"probe0002", ""},        // below the floor, as TestReleaseGame0002... is in the live tree
		{"x1000", "1000"},        // a real identifier named x1000 would be in range...
		{"decodeCP1251", "1251"}, // ...digitRun does not know about notAStoryNumber; the exception is applied by the caller
		{"foo100", ""},           // only 3 digits
		{"foo10000", ""},         // 5-digit run, not exactly 4
		{"foo999", ""},
		{"foo2001", ""}, // one past the ceiling
	}
	for _, c := range cases {
		if got := digitRun(c.name); got != c.want {
			t.Errorf("digitRun(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestCheckDiscriminates drives the pure evaluator with synthetic reports,
// the same way internal/archtest proves its DAG check with synthetic import
// graphs rather than a transient edit to the live tree: a committed test
// keeps the proof re-runnable and does not risk the real tree.
func TestCheckDiscriminates(t *testing.T) {
	base := Baseline{
		TestIdentCount: 10,
		TestFileCount:  5,
		CommentForms:   map[string]int{"specclause": 3, "storymention": 2, "calendardate": 1, "rom1address": 1, "funaddr": 0, "expmention": 0},
		CommentBytes:   1000,
	}
	cleanReport := func() Report {
		return Report{
			TestIdents:   make([]IdentHit, 10),
			TestFiles:    make([]string, 5),
			CommentForms: map[string]int{"specclause": 3, "storymention": 2, "calendardate": 1, "rom1address": 1, "funaddr": 0, "expmention": 0},
			CommentBytes: 1000,
		}
	}

	t.Run("clean report passes", func(t *testing.T) {
		if vs := Check(cleanReport(), base); len(vs) != 0 {
			t.Fatalf("expected no violations, got %v", vs)
		}
	})

	t.Run("one non-test identifier fails and names its one exception list", func(t *testing.T) {
		r := cleanReport()
		r.NonTestIdents = []IdentHit{{File: "pkg/game/x.go", Line: 1, Name: "worldsave1170"}}
		vs := Check(r, base)
		if len(vs) == 0 {
			t.Fatal("expected a violation")
		}
		if !strings.Contains(vs[0].String(), "worldsave1170") || !strings.Contains(vs[0].String(), "notAStoryNumber") {
			t.Errorf("violation does not name the offender or the rule: %s", vs[0])
		}
	})

	t.Run("one non-test file name fails with no exception list", func(t *testing.T) {
		r := cleanReport()
		r.NonTestFiles = []string{"pkg/game/worldsave1170.go"}
		vs := Check(r, base)
		if len(vs) == 0 || !strings.Contains(vs[0].String(), "worldsave1170.go") {
			t.Fatalf("expected a violation naming the file, got %v", vs)
		}
		if !strings.Contains(vs[0].String(), "no allowlist") {
			t.Errorf("file-name violation should say file names have no exception list: %s", vs[0])
		}
	})

	t.Run("test identifier count rising is a regression", func(t *testing.T) {
		r := cleanReport()
		r.TestIdents = make([]IdentHit, 11)
		vs := Check(r, base)
		if len(vs) == 0 || !strings.Contains(vs[0].Fix, "regression") {
			t.Fatalf("expected a regression violation, got %v", vs)
		}
	})

	t.Run("test identifier count falling demands a baseline update, not silent pass", func(t *testing.T) {
		r := cleanReport()
		r.TestIdents = make([]IdentHit, 9)
		vs := Check(r, base)
		if len(vs) == 0 || !strings.Contains(vs[0].Fix, "TestIdentCount") || !strings.Contains(vs[0].Fix, "9") {
			t.Fatalf("expected a fix naming TestIdentCount and 9, got %v", vs)
		}
	})

	t.Run("test file count rising is a regression", func(t *testing.T) {
		r := cleanReport()
		r.TestFiles = make([]string, 6)
		vs := Check(r, base)
		if len(vs) == 0 || !strings.Contains(vs[0].Fix, "regression") {
			t.Fatalf("expected a regression violation, got %v", vs)
		}
	})

	t.Run("a forbidden comment form rising fails", func(t *testing.T) {
		r := cleanReport()
		r.CommentForms["storymention"] = 3
		vs := Check(r, base)
		if len(vs) == 0 || !strings.Contains(vs[0].Category, "storymention") {
			t.Fatalf("expected a violation naming the form, got %v", vs)
		}
	})

	t.Run("comment bytes rising demands a trim first and names the price of not trimming", func(t *testing.T) {
		r := cleanReport()
		r.CommentBytes = 1001
		vs := Check(r, base)
		if len(vs) == 0 || !strings.Contains(vs[0].Fix, "trim") {
			t.Fatalf("expected a violation telling the reader to trim, got %v", vs)
		}
		if !strings.Contains(vs[0].Fix, "CommentBytes") || !strings.Contains(vs[0].Fix, "1001") {
			t.Errorf("a rise that belongs to new code must be told exactly what to commit: %s", vs[0].Fix)
		}
	})

	t.Run("comment bytes falling demands a baseline update", func(t *testing.T) {
		r := cleanReport()
		r.CommentBytes = 999
		vs := Check(r, base)
		if len(vs) == 0 || !strings.Contains(vs[0].Fix, "CommentBytes") || !strings.Contains(vs[0].Fix, "999") {
			t.Fatalf("expected a fix naming CommentBytes and 999, got %v", vs)
		}
	})
}

// TestNotAStoryNumber pins the exact contents of the identifier exception
// list. The non-test identifier rule is otherwise absolute, so the list is the
// one place a story number can live in production code: without this test a
// lane could hide a real leak by appending one line to scan.go and the guard
// would report a clean tree. Extending the list now fails here until the same
// commit says which name was added and why.
func TestNotAStoryNumber(t *testing.T) {
	want := []string{
		// Windows-1251, the Cyrillic code page ALM/BIN text uses.
		"Windows1251",
		"decodeCP1251",
		"encodeCP1251",
		// gob wire-format field names frozen in saves already on disk
		// (pkg/game/savehistoricalwire.go).
		"Application1170",
		"GameOptions1186",
		"LocalOnly1186",
	}
	sort.Strings(want)
	var got []string
	for name, allowed := range notAStoryNumber {
		if !allowed {
			t.Errorf("%q is present with value false; delete the entry instead", name)
			continue
		}
		got = append(got, name)
	}
	sort.Strings(got)
	if !slices.Equal(got, want) {
		t.Errorf("notAStoryNumber is %v, want %v; state the reason for every change in scan.go and here", got, want)
	}
}

// TestScanAppliesTheExceptionList proves the exception is applied where the
// walker reads identifiers, not only where Check reports them, and that it is
// keyed by the whole identifier name rather than by the digit run.
func TestScanAppliesTheExceptionList(t *testing.T) {
	dir := t.TempDir()
	src := "package probe\n\nvar Application1170 int\nvar somethingElse1170 int\n"
	if err := os.WriteFile(filepath.Join(dir, "probe.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Scan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var names []string
	for _, hit := range report.NonTestIdents {
		names = append(names, hit.Name)
	}
	if !slices.Equal(names, []string{"somethingElse1170"}) {
		t.Errorf("Scan reported %v, want only somethingElse1170", names)
	}
}

// TestScanReadsNamesTagsLiteralsAndGroups proves each new reader on a
// synthetic tree: the filesystem walker reports a story-numbered directory and
// non-.go file but exempts docs/<NNNN>; the go/ast walker reports a struct tag
// and a story-shaped string literal but not a bare digit run in a literal; the
// comment walker counts AC, SC and zero-padded story numbers and one group
// over the line ceiling.
func TestScanReadsNamesTagsLiteralsAndGroups(t *testing.T) {
	dir := t.TempDir()
	mk := func(rel, body string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mk("docs/"+"1170/story.md", "x")
	mk("pkg/probe1170/data1170.txt", "x")
	mk("scenarios/1170-a.json", "{}")
	mk("scenarios/plain.json", "{}")
	long := ""
	for i := 0; i < MaxCommentGroupLines+1; i++ {
		long += "// filler\n"
	}
	mk("probe.go", "package probe\n\n"+
		"// 0151 T12 and AC-3, AC-4 and SC-5.\n"+
		long+"var x int\n\n"+
		"type T struct {\n\tA int `json:\"a1170\"`\n}\n\n"+
		"var s = \"see docs/"+"1170-x\"\nvar t = \"1500 gold\"\nvar u = \"EXP-"+"0375\"\n")
	report, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"dirnames": 1, "nongofilenames": 2, "structtags.nontest": 1,
		"stringliterals.nontest": 2, LongGroupsKey: 1,
	}
	for _, k := range CountKeys {
		if report.Counts[k] != want[k] {
			t.Errorf("Counts[%q] = %d, want %d", k, report.Counts[k], want[k])
		}
	}
	for form, n := range map[string]int{"acclause": 2, "scclause": 1, "barestorynumber": 1} {
		if report.CommentForms[form] != n {
			t.Errorf("CommentForms[%q] = %d, want %d", form, report.CommentForms[form], n)
		}
	}
}

// TestCheckRatchetsCounts proves a rise or fall in a Counts entry is reported
// and names the field to update.
func TestCheckRatchetsCounts(t *testing.T) {
	base := Baseline{Counts: map[string]int{LongGroupsKey: 10}}
	if vs := Check(Report{Counts: map[string]int{LongGroupsKey: 10}}, base); len(vs) != 0 {
		t.Fatalf("equal count reported %v", vs)
	}
	if vs := Check(Report{Counts: map[string]int{LongGroupsKey: 11}}, base); len(vs) != 1 || !strings.Contains(vs[0].Fix, "regression") {
		t.Fatalf("rise not reported as a regression: %v", vs)
	}
	if vs := Check(Report{Counts: map[string]int{LongGroupsKey: 9}}, base); len(vs) != 1 || !strings.Contains(vs[0].Fix, "Counts[") {
		t.Fatalf("fall not reported with its baseline field: %v", vs)
	}
}

// TestScanSkipsGitIgnoredNames proves a numbered file or directory that
// .gitignore excludes is not counted, so local logs and saves cannot move the
// baseline, while an unignored numbered file in the same tree still is.
func TestScanSkipsGitIgnoredNames(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init: %v %s", err, out)
	}
	mk := func(rel, body string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mk(".gitignore", "saves/\nlogs/\n*.log\n")
	mk("saves/game1017.sav", "x")
	mk("logs/run1500/a.txt", "x")
	mk("note1500.log", "x")
	mk("kept1500.json", "{}")
	report, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Counts["nongofilenames"]; got != 1 {
		t.Errorf("nongofilenames = %d (%v), want only kept1500.json", got, report.NonGoFiles)
	}
	if got := report.Counts["dirnames"]; got != 0 {
		t.Errorf("dirnames = %d (%v), want 0", got, report.DirNames)
	}
}
