// Command regtool inspects .reg registries (developer tool).
//
// Usage:
//
//	regtool dump <archive.res> <entry.reg>   print the entry's parsed tree
//	regtool sweep <dir>                      parse every .reg entry of every .res archive under <dir>
//
// regtool is a developer-run tool for verifying the parser against a lawful game
// install; it is never part of the test suite. dump's output is converted game
// data and must never be committed — send it to your own screen or a
// git-ignored path, never into this repository (golden rule 1). The archive
// root is whatever path you pass on the command line; regtool embeds none
// (golden rule 3).
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/res"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage(os.Stdout)
		return
	}
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "regtool:", err)
		os.Exit(1)
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: regtool <dump|sweep> ...")
	fmt.Fprintln(w, "  regtool dump <archive.res> <entry.reg>   print the entry's parsed tree")
	fmt.Fprintln(w, "  regtool sweep <dir>                      parse every .reg entry of every .res archive under <dir>")
}

func run(args []string) error {
	switch args[0] {
	case "dump":
		if len(args) != 3 {
			return fmt.Errorf("usage: regtool dump <archive.res> <entry.reg>")
		}
		return doDump(args[1], args[2])
	case "sweep":
		if len(args) != 2 {
			return fmt.Errorf("usage: regtool sweep <dir>")
		}
		return doSweep(args[1])
	default:
		return fmt.Errorf("unknown command %q (want dump or sweep)", args[0])
	}
}

// doDump is the archive-opening half of dump: it reads entryPath's bytes out
// of archivePath, parses them and renders the tree to stdout. render is the
// pure half SC-9 tests without a file or an archive (DD14).
func doDump(archivePath, entryPath string) error {
	a, err := res.Open(archivePath)
	if err != nil {
		return err
	}
	data, err := a.ReadFile(entryPath)
	if err != nil {
		return err
	}
	r, err := reg.Parse(data)
	if err != nil {
		return err
	}
	if err := render(os.Stdout, r); err != nil {
		return err
	}
	// The summary line is the only thing dump sends to stderr, so the
	// rendered tree on stdout is exactly what a test asserts on.
	fmt.Fprintf(os.Stderr, "regtool: %d nodes\n", r.NodeCount)
	return nil
}

// doSweep walks dir recursively (DD15), taking every regular file whose name
// ends ".res" case-folded. For each archive it parses every entry whose path
// ends ".reg" case-folded. An archive that fails to open is reported on
// stderr and the walk continues — one unreadable file must not hide the
// other registries. A .reg entry that fails to parse is reported on stderr
// and counted as failed. It prints the spec's summary line on stdout and
// reports a non-nil error (hence a non-zero exit via main) iff any registry
// failed.
func doSweep(dir string) error {
	var total, failed int
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".res") {
			return nil
		}
		a, err := res.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "regtool: %s: %v\n", path, err)
			return nil
		}
		for _, e := range a.Entries() {
			if !strings.EqualFold(filepath.Ext(e.Path), ".reg") {
				continue
			}
			total++
			data, err := a.ReadFile(e.Path)
			if err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "regtool: %s %s: %v\n", path, e.Path, err)
				continue
			}
			r, err := reg.Parse(data)
			if err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "regtool: %s %s: %v\n", path, e.Path, err)
				continue
			}
			fmt.Printf("%s  %s  %d nodes\n", path, e.Path, r.NodeCount)
		}
		return nil
	})
	if walkErr != nil {
		return walkErr
	}
	fmt.Printf("%d regs: %d parsed, %d failed\n", total, total-failed, failed)
	if failed > 0 {
		return fmt.Errorf("%d of %d registries failed to parse", failed, total)
	}
	return nil
}

// render writes r's tree to w following the display convention DD14 fixes:
// two spaces per indentation level (the root's children at level 0), a
// directory line "<name>:" with its children indented one level deeper, and
// a value line "<name> = <value>".
func render(w io.Writer, r *reg.Reg) error {
	return renderChildren(w, r.Root.Children, 0)
}

func renderChildren(w io.Writer, children []*reg.Node, depth int) error {
	indent := strings.Repeat("  ", depth)
	for _, n := range children {
		name := escapeBytes(n.Name)
		if n.Dir {
			if _, err := fmt.Fprintf(w, "%s%s:\n", indent, name); err != nil {
				return err
			}
			if err := renderChildren(w, n.Children, depth+1); err != nil {
				return err
			}
			continue
		}
		val, err := renderValue(n)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s%s = %s\n", indent, name, val); err != nil {
			return err
		}
	}
	return nil
}

// renderValue renders one value node's Value column. Parse never produces a
// non-directory node of any type but these four (every other type is
// rejected during parsing), so the default branch is unreachable in
// practice; it returns an error rather than guessing.
func renderValue(n *reg.Node) (string, error) {
	switch n.Type {
	case reg.TypeString:
		return `"` + escapeBytes(n.Str) + `"`, nil
	case reg.TypeInt:
		return strconv.FormatInt(int64(n.Int), 10), nil
	case reg.TypeFloat:
		return formatFloat(n.Float), nil
	case reg.TypeIntArray:
		return renderIntArray(n.Ints), nil
	default:
		return "", fmt.Errorf("regtool: node %q: unrendered value type %d", n.Name, n.Type)
	}
}

// formatFloat renders v as strconv.FormatFloat(v, 'g', -1, 64), with ".0"
// appended when the result contains none of ".", "e", "E", "N" or "I" — so 0
// renders "0.0" and 1 renders "1.0", non-finite values are left alone, and a
// double stays distinguishable from an int32 in a dump with no type column
// (DD14; AC-8's startfade/endfade pair).
func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eEIN") {
		s += ".0"
	}
	return s
}

// renderIntArray renders "[" + space-separated elements + "]" for 8 or fewer
// elements; for more than 8 it abbreviates to the first eight, then "...",
// then "(N total)". An empty array renders "[]". Eight is the last
// unabbreviated length and nine the first abbreviated one.
func renderIntArray(ints []int32) string {
	var b strings.Builder
	b.WriteByte('[')
	n := len(ints)
	shown := n
	if shown > 8 {
		shown = 8
	}
	for i := 0; i < shown; i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.FormatInt(int64(ints[i]), 10))
	}
	if n > 8 {
		fmt.Fprintf(&b, " ... (%d total)", n)
	}
	b.WriteByte(']')
	return b.String()
}

// escapeBytes renders s's bytes under the convention AC-10 fixes: a byte in
// 0x20-0x7E prints as itself and every other byte prints as \xNN, two
// lower-case hex digits, always two. The result is pure ASCII, hence valid
// UTF-8 by construction, and it maps to no code page. This is the only
// function in the tool that maps a registry byte to a character.
func escapeBytes(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x20 && c <= 0x7e {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, `\x%02x`, c)
		}
	}
	return b.String()
}
