// Command tokenidentity proves that the difference between two source trees is
// confined to comments.
//
// Usage:
//
//	tokenidentity <tree>                     print one TSV row per .go file
//	tokenidentity -compare <before> <after>  compare two trees, exit 1 on any
//	                                         code or directive difference
//
// -allow-added permits .go files that exist only in the after tree, for the
// change that adds a file on purpose. -allow-moved names, one by one, the
// files whose tokens are expected to move: a comment cleanup must rewrite
// internal/storyguard/baseline.go's own counters in the same commit, and a
// gate that cannot say so is a gate nobody runs. Every allowance prints the
// file and both hashes, so what was permitted is visible rather than absent.
// Neither flag ever permits a removed file or an unnamed mover.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"againrom/internal/tokenidentity"
)

func main() {
	compare := flag.Bool("compare", false, "compare two trees instead of printing one")
	allowAdded := flag.Bool("allow-added", false, "with -compare, permit .go files added in the after tree")
	allowMoved := flag.String("allow-moved", "", "with -compare, comma-separated paths whose tokens are expected to move")
	flag.Parse()

	if *compare {
		os.Exit(runCompare(flag.Arg(0), flag.Arg(1), *allowAdded, namedPaths(*allowMoved)))
	}
	os.Exit(runPrint(flag.Arg(0)))
}

func namedPaths(list string) map[string]bool {
	out := map[string]bool{}
	for _, p := range strings.Split(list, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out[p] = true
		}
	}
	return out
}

func runPrint(root string) int {
	if root == "" {
		fmt.Fprintln(os.Stderr, "usage: tokenidentity <tree>")
		return 2
	}
	rows, err := tokenidentity.HashTree(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tokenidentity:", err)
		return 2
	}
	for _, r := range rows {
		fmt.Println(tokenidentity.Format(r))
	}
	return 0
}

func runCompare(beforeRoot, afterRoot string, allowAdded bool, allowMoved map[string]bool) int {
	if beforeRoot == "" || afterRoot == "" {
		fmt.Fprintln(os.Stderr, "usage: tokenidentity -compare <before> <after>")
		return 2
	}
	before, err := tokenidentity.HashTree(beforeRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tokenidentity: before:", err)
		return 2
	}
	after, err := tokenidentity.HashTree(afterRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tokenidentity: after:", err)
		return 2
	}

	var moved, removed, added, named int
	for _, d := range tokenidentity.Compare(before, after) {
		switch d.Kind {
		case "added":
			added++
			if allowAdded {
				fmt.Printf("added      %s\n", d.Path)
				continue
			}
		case "removed":
			removed++
		default:
			if allowMoved[d.Path] {
				named++
				fmt.Printf("allowed    %s (%s)\t%s -> %s\n", d.Path, d.Kind, d.Before, d.After)
				continue
			}
			moved++
		}
		fmt.Printf("%-10s %s\t%s -> %s\n", d.Kind, d.Path, d.Before, d.After)
	}

	verdict, code := "OK", 0
	if moved > 0 || removed > 0 || (added > 0 && !allowAdded) {
		verdict, code = "FAIL", 1
	}
	fmt.Printf("TOKEN IDENTITY: %s - %d files in both trees, %d moved, %d allowed by name, %d removed, %d added (%s); %d before, %d after\n",
		verdict, len(before)-removed, moved, named, removed, added, addedPolicy(allowAdded), len(before), len(after))
	return code
}

func addedPolicy(allowAdded bool) string {
	if allowAdded {
		return "allowed"
	}
	return "not allowed"
}
