// Package storyguard enforces the owner's story-number ban as an executable
// check rather than a review convention: no Go identifier, file name,
// directory name, struct tag, string literal or Go comment under version
// control may carry the numbering this project uses for its own story and
// experiment tracking. Two walkers read the tree: Scan's go/ast walker
// (identifiers, comments, struct tags, string literals) and walkNames' filesystem
// walker (directory and non-.go file names, which skips paths git ignores). Directories directly under docs/
// are exempt because docs/<NNNN> is where a story document lives. Identifiers
// have one exception list, notAStoryNumber below, pinned by its own test;
// .go file names have none. Directory names, non-.go file names, struct tags
// and string literals carry inherited debt that is ratcheted in the same
// Baseline as the comment counts. String literals are read narrowly: only the
// story, experiment, address and docs-path shapes in literalForms, because a
// bare four-digit run in a literal is mostly a game value or a fixture. Comment
// counts also cover the AC-n and SC-n clauses, the bare zero-padded story number, and
// the number of comment groups longer than MaxCommentGroupLines. The guard
// counts its own comment bytes, so new code here adds to CommentBytes.
// It is a non-tier build/test helper under internal/ and is not itself policed.
//
// The check is split the way internal/archtest splits its DAG check: a pure
// evaluator (Check) exercised by synthetic Report values, and a walker (Scan)
// that reads the live tree. A bare provenance id such as DIV-1359 or SAV-1011
// is not a violation; only the narrative forms this file's regexes name are.
package storyguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// MinStoryNumber and MaxStoryNumber bound the digit run a 4-digit identifier
// or file-name substring must fall inside to count as story-number-shaped.
// The bound says the rule is about this project's own numbering rather than
// about digits: docs/<NNNN> runs 1058-1205 at this writing, and 2000 leaves
// headroom past a doubling of the story count.
//
// Measured over every .go file in the module, the only 4-digit identifier run
// outside [1000, 2000] is 0002, in TestReleaseGame0002PotionRetainsItsSavedEffect
// (pkg/game/enchantment_release_test.go) - a game content id the floor keeps
// out. The exactly-4 rule carries the rest of the separation: a 3- or 5-digit
// run is never a story number. TestDigitRun fixes both boundaries.
const (
	MinStoryNumber = 1000
	MaxStoryNumber = 2000
)

// notAStoryNumber lists identifiers whose 4-digit run is a real, verified
// value that is not a story number a rename can remove, so renaming would
// replace a correct name with a worse one. Each entry needs a reason; this is
// not a general allowlist for story numbers, only for names already checked.
//
// It is the one door in an otherwise absolute rule, so its exact contents are
// asserted by TestNotAStoryNumber in this package: adding an entry fails that
// test until the same commit states the new name, which makes growing the list
// a reviewed act rather than a silent one.
var notAStoryNumber = map[string]bool{
	// Windows-1251 is the Cyrillic code page ALM/BIN text actually uses
	// (pkg/formats/alm/alm.go, encode.go; pkg/formats/textinput/input.go).
	// 1251 is the code page number, not a story number.
	"decodeCP1251": true,
	"encodeCP1251": true,
	"Windows1251":  true,

	// These three are gob wire-format field names, written into every .ags
	// save already on disk (pkg/game/savehistoricalwire.go). encoding/gob
	// matches fields by name, so the number is a format constant that lives
	// in the bytes; renaming the Go field would drop the saved value rather
	// than remove the number. Nothing outside that one file and its test
	// reads these names.
	"Application1170": true,
	"GameOptions1186": true,
	"LocalOnly1186":   true,
}

// digitRun returns the first maximal run of ASCII digits in s whose length is
// exactly 4 and whose value falls in [MinStoryNumber, MaxStoryNumber], or ""
// if none exists.
func digitRun(s string) string {
	start := -1
	for i := 0; i <= len(s); i++ {
		isDigit := i < len(s) && s[i] >= '0' && s[i] <= '9'
		if isDigit {
			if start == -1 {
				start = i
			}
			continue
		}
		if start != -1 {
			run := s[start:i]
			start = -1
			if len(run) == 4 {
				if v, err := strconv.Atoi(run); err == nil && v >= MinStoryNumber && v <= MaxStoryNumber {
					return run
				}
			}
		}
	}
	return ""
}

// commentForms names the six forbidden comment shapes the owner asked to be
// swept from the tree; each is a research-repo or Git concern, not a code
// comment's job. Order is fixed so Report/Baseline field iteration is stable.
var commentForms = []struct {
	name string
	re   *regexp.Regexp
}{
	{"specclause", regexp.MustCompile(`\b(?:FR|DD|P|S)-[0-9]+\b`)},
	{"storymention", regexp.MustCompile(`(?i)\bstory[ ]?[0-9]{3,4}\b`)},
	{"calendardate", regexp.MustCompile(`\b20[0-9]{2}-[0-9]{2}-[0-9]{2}\b`)},
	{"rom1address", regexp.MustCompile(`\b0x00[0-9A-Fa-f]{6}\b`)},
	{"funaddr", regexp.MustCompile(`\bFUN_[0-9A-Fa-f]{8}\b`)},
	{"expmention", regexp.MustCompile(`\bEXP-[0-9]{3,4}\b`)},
	{"acclause", regexp.MustCompile(`\bAC-[0-9]+\b`)},
	{"scclause", regexp.MustCompile(`\bSC-[0-9]+\b`)},
	// A zero-padded four-digit run is a story number such as "0151 T12" or
	// "(0140)". The leading class keeps 0x hex, decimals and longer digit
	// runs out.
	{"barestorynumber", regexp.MustCompile(`(?:^|[^0-9A-Za-z_.])0[0-9]{3}\b`)},
}

// MaxCommentGroupLines is the guidance ceiling for one comment group; the
// count of groups above it is ratcheted, not forbidden.
const MaxCommentGroupLines = 12

// LongGroupsKey names the long-comment-group count in Report.Counts and
// Baseline.Counts.
const LongGroupsKey = "longcommentgroups"

// literalForms is the narrow set a string literal or struct tag is read for.
var literalForms = regexp.MustCompile(`(?i)\bstory[ _-]?[0-9]{3,4}\b|\bEXP-[0-9]{3,4}\b|\bFUN_[0-9A-Fa-f]{8}\b|\bdocs/[0-9]{4}\b|\bwt-story-[0-9]{4}`)

// CommentFormNames lists every comment-form counter in its fixed order.
func CommentFormNames() []string {
	names := make([]string, len(commentForms))
	for i, f := range commentForms {
		names[i] = f.name
	}
	return names
}

// CountKeys names the ratcheted name, tag, literal and long-group counters in
// Report.Counts and Baseline.Counts.
var CountKeys = []string{
	"dirnames", "nongofilenames", "structtags.nontest", "structtags.test",
	"stringliterals.nontest", "stringliterals.test", LongGroupsKey,
}

func commentLines(group *ast.CommentGroup) int {
	n := 0
	for _, c := range group.List {
		n += strings.Count(c.Text, "\n") + 1
	}
	return n
}

// IdentHit is one identifier occurrence whose name carries a story-shaped
// digit run.
type IdentHit struct {
	File string
	Line int
	Name string
}

// Report is what Scan measures on one tree. Check compares it to a Baseline.
type Report struct {
	NonTestIdents []IdentHit
	TestIdents    []IdentHit
	NonTestFiles  []string
	TestFiles     []string
	DirNames      []string
	NonGoFiles    []string
	CommentForms  map[string]int
	CommentBytes  int64
	Counts        map[string]int
}

// Scan walks every .go file under moduleRoot (skipping .git, other hidden
// directories, and any nested module) and measures it into a Report.
func Scan(moduleRoot string) (Report, error) {
	report := Report{CommentForms: map[string]int{}, Counts: map[string]int{}}
	err := filepath.Walk(moduleRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if info.IsDir() {
			name := info.Name()
			if path != moduleRoot && (name == ".git" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			if path != moduleRoot {
				if _, statErr := os.Stat(filepath.Join(path, "go.mod")); statErr == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		isTest := strings.HasSuffix(path, "_test.go")

		base := filepath.Base(path)
		base = strings.TrimSuffix(base, ".go")
		if run := digitRun(base); run != "" {
			if isTest {
				report.TestFiles = append(report.TestFiles, path)
			} else {
				report.NonTestFiles = append(report.NonTestFiles, path)
			}
		}

		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if perr != nil {
			return perr
		}

		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if notAStoryNumber[id.Name] {
				return true
			}
			if run := digitRun(id.Name); run != "" {
				hit := IdentHit{File: path, Line: fset.Position(id.Pos()).Line, Name: id.Name}
				if isTest {
					report.TestIdents = append(report.TestIdents, hit)
				} else {
					report.NonTestIdents = append(report.NonTestIdents, hit)
				}
			}
			return true
		})

		ast.Inspect(f, func(n ast.Node) bool {
			var text, kind string
			switch v := n.(type) {
			case *ast.Field:
				if v.Tag == nil {
					return true
				}
				text, kind = v.Tag.Value, "structtags"
				if digitRun(text) == "" && !literalForms.MatchString(text) {
					return true
				}
			case *ast.BasicLit:
				if v.Kind != token.STRING {
					return true
				}
				text, kind = v.Value, "stringliterals"
				if !literalForms.MatchString(text) {
					return true
				}
			default:
				return true
			}
			if isTest {
				report.Counts[kind+".test"]++
			} else {
				report.Counts[kind+".nontest"]++
			}
			return true
		})

		for _, group := range f.Comments {
			if commentLines(group) > MaxCommentGroupLines {
				report.Counts[LongGroupsKey]++
			}
			for _, c := range group.List {
				report.CommentBytes += int64(len(c.Text))
				for _, form := range commentForms {
					report.CommentForms[form.name] += len(form.re.FindAllString(c.Text, -1))
				}
			}
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	dirs, files, err := walkNames(moduleRoot)
	report.Counts["dirnames"] = len(dirs)
	report.Counts["nongofilenames"] = len(files)
	report.DirNames, report.NonGoFiles = dirs, files
	return report, err
}

// walkNames is the filesystem walker: it lists directories and non-.go files
// whose base name carries a story-shaped digit run. It skips the same
// directories Scan does. A directory directly under docs/ is exempt because
// docs/<NNNN> holds a story document.
func walkNames(moduleRoot string) (dirs, files []string, err error) {
	ignored := gitIgnored(moduleRoot)
	err = filepath.Walk(moduleRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if info.IsDir() {
			name := info.Name()
			if path == moduleRoot {
				return nil
			}
			if name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if ignored[relSlash(moduleRoot, path)+"/"] {
				return filepath.SkipDir
			}
			if _, statErr := os.Stat(filepath.Join(path, "go.mod")); statErr == nil {
				return filepath.SkipDir
			}
			if digitRun(name) != "" && filepath.Base(filepath.Dir(path)) != "docs" {
				dirs = append(dirs, path)
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") || ignored[relSlash(moduleRoot, path)] {
			return nil
		}
		if digitRun(info.Name()) != "" && !under(moduleRoot, path, "docs") {
			files = append(files, path)
		}
		return nil
	})
	return dirs, files, err
}

// under reports whether path lies inside moduleRoot/dir.
func under(moduleRoot, path, dir string) bool {
	rel, err := filepath.Rel(filepath.Join(moduleRoot, dir), path)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// gitIgnored returns the slash-separated paths git ignores under moduleRoot;
// a directory is keyed with a trailing slash. The name check must not depend on
// local files such as logs and saves that .gitignore excludes. Outside a git
// work tree it returns an empty set, so a synthetic tree is read in full.
func gitIgnored(moduleRoot string) map[string]bool {
	out, err := exec.Command("git", "-C", moduleRoot, "ls-files", "-z",
		"--others", "--ignored", "--exclude-standard", "--directory").Output()
	set := map[string]bool{}
	if err != nil {
		return set
	}
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			set[p] = true
		}
	}
	return set
}

func relSlash(moduleRoot, path string) string {
	rel, err := filepath.Rel(moduleRoot, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
