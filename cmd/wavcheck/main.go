// Command wavcheck counts archive WAV encodings and decode failures without
// playing audio or writing to an install. wav.Parse owns all format inspection.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"againrom/pkg/formats/res"
	"againrom/pkg/formats/wav"
)

func main() {
	root := flag.String("root", "", "read-only install root")
	prefix := flag.String("frames-prefix", "", "print decoded frame counts under this archive path")
	flag.Parse()
	if *root == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: wavcheck -root <install> [-frames-prefix <archive path>]")
		os.Exit(2)
	}
	for _, archive := range []string{"sfx.res", "speech.res"} {
		if err := census(filepath.Join(*root, archive), *prefix); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

type counts struct {
	entries, decoded, refused int
	first                     string
}

func census(file, prefix string) error {
	a, err := res.OpenFileIndex(file)
	if err != nil {
		return err
	}
	entries := a.Entries()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	tags := map[uint16]*counts{}
	other := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Path) != ".wav" {
			other++
			continue
		}
		raw, err := a.ReadFile(entry.Path)
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Path, err)
		}
		p, err := wav.Parse(raw)
		c := tags[p.FormatTag]
		if c == nil {
			c = &counts{}
			tags[p.FormatTag] = c
		}
		c.entries++
		if err != nil {
			c.refused++
			if c.first == "" {
				c.first = fmt.Sprintf("%s: %v", entry.Path, err)
			}
		} else {
			c.decoded++
			if prefix != "" && strings.HasPrefix(entry.Path, prefix) {
				fmt.Printf("%s %s tag=%d frames=%d\n", filepath.Base(file), entry.Path, p.FormatTag, p.Frames())
			}
		}
	}
	fmt.Printf("%s entries=%d wav=%d other=%d\n", filepath.Base(file), len(entries), len(entries)-other, other)
	keys := make([]int, 0, len(tags))
	for tag := range tags {
		keys = append(keys, int(tag))
	}
	sort.Ints(keys)
	for _, tag := range keys {
		c := tags[uint16(tag)]
		fmt.Printf("tag=%d entries=%d decoded=%d refused=%d\n", tag, c.entries, c.decoded, c.refused)
		if c.first != "" {
			fmt.Printf("first error: %s\n", c.first)
		}
	}
	return nil
}
