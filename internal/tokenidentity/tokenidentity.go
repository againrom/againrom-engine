// Package tokenidentity proves that a change touched only comments.
//
// It reduces every .go file to two independent hashes:
//
//   - Tokens is the go/scanner token stream with every COMMENT token dropped,
//     hashing each remaining token's kind and literal. Two trees whose code is
//     identical and whose comments differ produce the same value; moving,
//     adding or removing one non-comment token changes it.
//   - Directives is the ordered list of comment lines the toolchain itself
//     reads: //go:build, //go:embed, //go:generate, //line and every other
//     //go: or "+build" line. The token hash is blind to those by
//     construction, because to the scanner they are comments. A cleanup that
//     deletes a build constraint or an embed passes the token half and fails
//     this one.
//
// Neither half alone is sufficient and the pair is: the first covers code, the
// second covers the comments that are code in disguise.
package tokenidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go/scanner"
	"go/token"
)

// Row is one file's identity. Path is slash-separated and relative to the
// scanned root. A file the scanner could not read produces an empty Tokens and
// Directives and a non-empty ScanError; a scan error is reported rather than
// hashed, so a broken file can never compare equal to a working one.
type Row struct {
	Path       string
	Tokens     string
	Directives string
	ScanError  string
}

// hashLen is the number of leading sha256 bytes kept in each hex value. 16
// bytes is 128 bits, far past any accidental collision over a tree of a few
// thousand files, and short enough that a row stays readable.
const hashLen = 16

// HashFile reduces one file's source to its token and directive hashes. path
// is used only for scanner error positions.
func HashFile(path string, src []byte) Row {
	row := Row{Path: filepath.ToSlash(path)}

	fset := token.NewFileSet()
	file := fset.AddFile(path, fset.Base(), len(src))

	var errs scanner.ErrorList
	var s scanner.Scanner
	// ScanComments is on because the directive half cannot see a directive any
	// other way; the token half drops every COMMENT explicitly below.
	s.Init(file, src, func(pos token.Position, msg string) {
		errs.Add(pos, msg)
	}, scanner.ScanComments)

	tokens := sha256.New()
	var directives []string
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT {
			directives = append(directives, DirectiveLines(lit)...)
			continue
		}
		// Kind, literal length and literal text. The length delimiter stops
		// two different streams colliding by concatenation, which an
		// unseparated kind+text encoding allows.
		fmt.Fprintf(tokens, "%d:%d:%s;", int(tok), len(lit), lit)
	}
	if len(errs) > 0 {
		row.ScanError = errs[0].Error()
		return row
	}

	sum := tokens.Sum(nil)
	row.Tokens = hex.EncodeToString(sum[:hashLen])
	dir := sha256.Sum256([]byte(strings.Join(directives, "\n")))
	row.Directives = hex.EncodeToString(dir[:hashLen])
	return row
}

// DirectiveLines returns the toolchain-readable lines inside one comment
// literal, in source order. A //go: line counts only with no space after the
// slashes, which is the rule the toolchain applies; "// go:build linux" is
// prose and is deliberately not returned.
//
// Order is preserved rather than sorted: swapping two build constraint lines
// is a change to the file, not a reordering of an unordered set.
func DirectiveLines(lit string) []string {
	body := lit
	block := strings.HasPrefix(lit, "/*")
	if block {
		body = strings.TrimSuffix(strings.TrimPrefix(lit, "/*"), "*/")
	}
	var out []string
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimRight(line, " \t\r")
		if block {
			t = strings.TrimSpace(t)
		}
		switch {
		case strings.HasPrefix(t, "//go:"):
			out = append(out, t)
		case strings.HasPrefix(t, "// +build"), strings.HasPrefix(t, "//+build"):
			out = append(out, t)
		case strings.HasPrefix(t, "//line "):
			out = append(out, t)
		}
	}
	return out
}

// HashTree walks root and returns one Row per .go file, sorted by path. Hidden
// directories, including .git, are skipped; nothing else is filtered, so a
// generated or vendored .go file is compared like any other.
func HashTree(root string) ([]Row, error) {
	var rows []Row
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		row := HashFile(path, src)
		row.Path = filepath.ToSlash(rel)
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
	return rows, nil
}

// Format renders one Row as the tab-separated line HashTree callers compare.
func Format(r Row) string {
	if r.ScanError != "" {
		return fmt.Sprintf("%s\tSCANERROR\t%s", r.Path, r.ScanError)
	}
	return fmt.Sprintf("%s\t%s\t%s", r.Path, r.Tokens, r.Directives)
}

// Difference is one file whose identity is not the same in two trees.
type Difference struct {
	Path   string
	Kind   string // "tokens", "directives", "scan", "added" or "removed"
	Before string
	After  string
}

// Compare reports how two trees' rows differ. Kinds "tokens", "directives" and
// "scan" mean a file present in both changed; "added" and "removed" mean the
// file set itself changed. A caller deciding whether a change was
// comment-only treats every kind as disqualifying except a deliberate,
// separately justified file addition.
func Compare(before, after []Row) []Difference {
	byPath := func(rows []Row) map[string]Row {
		m := make(map[string]Row, len(rows))
		for _, r := range rows {
			m[r.Path] = r
		}
		return m
	}
	b, a := byPath(before), byPath(after)

	var diffs []Difference
	for _, r := range before {
		other, ok := a[r.Path]
		if !ok {
			diffs = append(diffs, Difference{Path: r.Path, Kind: "removed", Before: Format(r)})
			continue
		}
		switch {
		case r.ScanError != "" || other.ScanError != "":
			if r.ScanError != other.ScanError {
				diffs = append(diffs, Difference{Path: r.Path, Kind: "scan", Before: r.ScanError, After: other.ScanError})
			}
		case r.Tokens != other.Tokens:
			diffs = append(diffs, Difference{Path: r.Path, Kind: "tokens", Before: r.Tokens, After: other.Tokens})
		case r.Directives != other.Directives:
			diffs = append(diffs, Difference{Path: r.Path, Kind: "directives", Before: r.Directives, After: other.Directives})
		}
	}
	for _, r := range after {
		if _, ok := b[r.Path]; !ok {
			diffs = append(diffs, Difference{Path: r.Path, Kind: "added", After: Format(r)})
		}
	}
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Path != diffs[j].Path {
			return diffs[i].Path < diffs[j].Path
		}
		return diffs[i].Kind < diffs[j].Kind
	})
	return diffs
}
