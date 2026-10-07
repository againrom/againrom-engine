package storyguard

import (
	"fmt"
	"sort"
	"strings"
)

// Baseline is the one committed record of every ratcheted count this package
// enforces. Test-file identifiers and file names are debt inherited from
// before this story; they may only fall, never rise, and the committed number
// must always equal the tree's actual count. The same shape applies to each
// forbidden comment form and to total comment bytes. Counts holds the remaining ratchets keyed by
// CountKeys: directory names, non-.go file names, struct tags, string literals
// (non-test and test apart) and comment groups over MaxCommentGroupLines. See baseline.go for the
// current committed values and how to regenerate them.
type Baseline struct {
	TestIdentCount int
	TestFileCount  int
	CommentForms   map[string]int
	CommentBytes   int64
	Counts         map[string]int
}

// Violation is one policy failure. Fix always names the exact action to take.
type Violation struct {
	Category string
	Detail   string
	Fix      string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s (%s)", v.Category, v.Detail, v.Fix)
}

// Check compares a measured Report against the committed Baseline and returns
// every violation found. An empty result means the tree matches policy
// exactly: zero story-numbered identifiers and file names in non-test code,
// and every ratcheted count equal to its committed baseline. Scan has already
// applied the one identifier exception (scan.go's notAStoryNumber) before the
// Report reaches here; file names have no exception list at all.
func Check(report Report, baseline Baseline) []Violation {
	var out []Violation

	if len(report.NonTestIdents) > 0 {
		out = append(out, Violation{
			Category: "non-test identifier",
			Detail:   fmt.Sprintf("%d story-numbered identifier(s) in non-test files: %s", len(report.NonTestIdents), identSample(report.NonTestIdents)),
			Fix:      "rename each to a name that says what it holds; the one exception is scan.go's notAStoryNumber, whose exact contents TestNotAStoryNumber pins",
		})
	}
	if len(report.NonTestFiles) > 0 {
		out = append(out, Violation{
			Category: "non-test file name",
			Detail:   fmt.Sprintf("%d story-numbered file name(s): %s", len(report.NonTestFiles), pathSample(report.NonTestFiles)),
			Fix:      "git mv each to a name that says what the file holds; zero tolerance, no allowlist",
		})
	}

	out = append(out, ratchetIdent("test identifier", len(report.TestIdents), baseline.TestIdentCount,
		"internal/storyguard/baseline.go: TestIdentCount", identSample(report.TestIdents))...)
	out = append(out, ratchetPath("test file name", len(report.TestFiles), baseline.TestFileCount,
		"internal/storyguard/baseline.go: TestFileCount", pathSample(report.TestFiles))...)

	for _, form := range commentForms {
		got := report.CommentForms[form.name]
		want := baseline.CommentForms[form.name]
		if got == want {
			continue
		}
		out = append(out, ratchetCount(fmt.Sprintf("comment form %q", form.name), got, want,
			fmt.Sprintf("internal/storyguard/baseline.go: CommentForms[%q]", form.name))...)
	}

	for _, key := range CountKeys {
		out = append(out, ratchetCount(fmt.Sprintf("count %q", key), report.Counts[key], baseline.Counts[key],
			fmt.Sprintf("internal/storyguard/baseline.go: Counts[%q]", key))...)
	}

	if report.CommentBytes != baseline.CommentBytes {
		out = append(out, ratchetCount64("total comment bytes", report.CommentBytes, baseline.CommentBytes,
			"internal/storyguard/baseline.go: CommentBytes")...)
	}

	return out
}

func ratchetCount(category string, got, want int, baselineField string) []Violation {
	if got == want {
		return nil
	}
	if got > want {
		return []Violation{{
			Category: category,
			Detail:   fmt.Sprintf("count rose from %d to %d", want, got),
			Fix:      "regression: fix the new occurrence(s), do not raise the baseline",
		}}
	}
	return []Violation{{
		Category: category,
		Detail:   fmt.Sprintf("count fell from %d to %d", want, got),
		Fix:      fmt.Sprintf("update %s to %d and commit", baselineField, got),
	}}
}

func ratchetCount64(category string, got, want int64, baselineField string) []Violation {
	if got == want {
		return nil
	}
	if got > want {
		return []Violation{{
			Category: category,
			Detail:   fmt.Sprintf("count rose from %d to %d", want, got),
			Fix: fmt.Sprintf("trim the narrative rather than raise the baseline; a rise that belongs to new code is allowed only by saying so and setting %s to %d in the same commit",
				baselineField, got),
		}}
	}
	return []Violation{{
		Category: category,
		Detail:   fmt.Sprintf("count fell from %d to %d", want, got),
		Fix:      fmt.Sprintf("update %s to %d and commit", baselineField, got),
	}}
}

func ratchetIdent(category string, got, want int, baselineField, sample string) []Violation {
	vs := ratchetCount(category, got, want, baselineField)
	if len(vs) == 0 {
		return nil
	}
	vs[0].Detail += "; sample: " + sample
	return vs
}

func ratchetPath(category string, got, want int, baselineField, sample string) []Violation {
	vs := ratchetCount(category, got, want, baselineField)
	if len(vs) == 0 {
		return nil
	}
	vs[0].Detail += "; sample: " + sample
	return vs
}

func identSample(hits []IdentHit) string {
	names := map[string]bool{}
	for _, h := range hits {
		names[fmt.Sprintf("%s (%s:%d)", h.Name, h.File, h.Line)] = true
	}
	var list []string
	for n := range names {
		list = append(list, n)
	}
	sort.Strings(list)
	if len(list) > 5 {
		list = append(list[:5], fmt.Sprintf("... and %d more", len(names)-5))
	}
	return strings.Join(list, ", ")
}

func pathSample(paths []string) string {
	list := append([]string(nil), paths...)
	sort.Strings(list)
	if len(list) > 5 {
		list = append(list[:5], fmt.Sprintf("... and %d more", len(paths)-5))
	}
	return strings.Join(list, ", ")
}
