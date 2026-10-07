// measure prints a fresh Baseline literal for internal/storyguard/baseline.go
// by scanning the live tree. It is a one-off regeneration tool, not part of
// the guard itself.
package main

import (
	"fmt"
	"os"

	"againrom/internal/archtest"
	"againrom/internal/storyguard"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, err := archtest.FindModuleRoot(wd)
	if err != nil {
		panic(err)
	}
	report, err := storyguard.Scan(root)
	if err != nil {
		panic(err)
	}
	fmt.Printf("NonTestIdents: %d\n", len(report.NonTestIdents))
	for _, h := range report.NonTestIdents {
		fmt.Printf("  %s %s:%d\n", h.Name, h.File, h.Line)
	}
	fmt.Printf("NonTestFiles: %d\n", len(report.NonTestFiles))
	for _, p := range report.NonTestFiles {
		fmt.Printf("  %s\n", p)
	}
	fmt.Printf("TestIdentCount: %d\n", len(report.TestIdents))
	fmt.Printf("TestFileCount: %d\n", len(report.TestFiles))
	fmt.Println("CommentForms:")
	for _, k := range storyguard.CommentFormNames() {
		fmt.Printf("  %s: %d\n", k, report.CommentForms[k])
	}
	fmt.Printf("CommentBytes: %d\n", report.CommentBytes)

	fmt.Println("Against the committed baseline (committed -> measured; a rise is marked):")
	diff("TestIdentCount", int64(len(report.TestIdents)), int64(storyguard.Committed.TestIdentCount))
	diff("TestFileCount", int64(len(report.TestFiles)), int64(storyguard.Committed.TestFileCount))
	for _, k := range storyguard.CommentFormNames() {
		diff("CommentForms."+k, int64(report.CommentForms[k]), int64(storyguard.Committed.CommentForms[k]))
	}
	for _, k := range storyguard.CountKeys {
		diff("Counts."+k, int64(report.Counts[k]), int64(storyguard.Committed.Counts[k]))
	}
	diff("CommentBytes", report.CommentBytes, storyguard.Committed.CommentBytes)
}

func diff(name string, measured, committed int64) {
	mark := ""
	if measured > committed {
		mark = "  RISE"
	}
	fmt.Printf("  %s: %d -> %d (%+d)%s\n", name, committed, measured, measured-committed, mark)
}
