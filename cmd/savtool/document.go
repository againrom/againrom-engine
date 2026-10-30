package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
)

// document decodes each file as a document, encodes it again and compares
// the two part by part.
func document(out io.Writer, paths []string) error {
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		doc, err := sav.DecodeDocumentData(b)
		if err != nil {
			fmt.Fprintf(out, "%s DECODE FAILED: %v\n", path, err)
			continue
		}
		again, err := sav.EncodeDocumentData(doc)
		if err != nil {
			fmt.Fprintf(out, "%s ENCODE FAILED: %v\n", path, err)
			continue
		}
		if err := compareFiles(out, path, "re-encoded", b, again); err != nil {
			return err
		}
	}
	return nil
}

// compareFiles reports, for the file, the decoded body, the label region,
// the state store and the tail after it, whether two SAVs are the same
// bytes and where they differ; for the store, also whether both hold the
// same leaves in another order.
func compareFiles(out io.Writer, a, b string, x, y []byte) error {
	fx, err := sav.Open(x)
	if err != nil {
		return fmt.Errorf("%s: %w", a, err)
	}
	fy, err := sav.Open(y)
	if err != nil {
		return fmt.Errorf("%s: %w", b, err)
	}
	fmt.Fprintf(out, "%s vs %s: magic %q %q, file %s\n", a, b, x[:4], y[:4], equalText(x, y))
	for _, part := range []struct {
		name string
		x, y []byte
	}{
		{"body", fx.Body, fy.Body},
		{"label", fx.LabelRegion, fy.LabelRegion},
		{"store", fx.Store, fy.Store},
		{"tail", fx.TailRest, fy.TailRest},
	} {
		fmt.Fprintf(out, "  %-5s %s\n", part.name, equalText(part.x, part.y))
	}
	if !bytes.Equal(fx.Store, fy.Store) {
		fmt.Fprintf(out, "  store leaves %s\n", sameLeaves(x, y))
	}
	if !bytes.Equal(fx.Body, fy.Body) {
		dx, errx := sav.DecodeDocumentData(x)
		dy, erry := sav.DecodeDocumentData(y)
		if errx == nil && erry == nil {
			for _, d := range recordDiffs(dx, dy) {
				fmt.Fprintf(out, "  field %s\n", d)
			}
		}
	}
	return nil
}

// recordDiffs names each head word and record field two documents hold
// differently, by record ordinal, class and field name.
func recordDiffs(x, y sav.DocumentData) []string {
	var out []string
	add := func(at, name string, a, b any) {
		if fmt.Sprint(a) != fmt.Sprint(b) {
			out = append(out, fmt.Sprintf("%s %s: %v > %v", at, name, a, b))
		}
	}
	add("head", "", fmt.Sprintf("%+v", x.Head), fmt.Sprintf("%+v", y.Head))
	add("trailer", "marker/global", [2]uint32{x.Marker, x.GlobalDWord}, [2]uint32{y.Marker, y.GlobalDWord})
	add("document", "players/dead", [2][]uint16{x.Players, x.DeadActors}, [2][]uint16{y.Players, y.DeadActors})
	add("document", "records", len(x.Objects), len(y.Objects))
	for i := range min(len(x.Objects), len(y.Objects)) {
		a, b := x.Objects[i], y.Objects[i]
		at := fmt.Sprintf("#%d %s", i+1, a.Class)
		if a.Class != b.Class {
			add(at, "class", a.Class, b.Class)
			continue
		}
		fields := func(r sav.DocumentRecordData) map[string]string {
			m := map[string]string{}
			for _, v := range r.Values {
				m["v "+v.Name] = fmt.Sprint(v.Value)
			}
			for _, v := range r.Texts {
				m["t "+v.Name] = fmt.Sprintf("%q", v.Value)
			}
			for _, v := range r.Raw {
				m["r "+v.Name] = fmt.Sprintf("%x", v.Bytes)
			}
			for _, v := range r.Counts {
				m["c "+v.Name] = fmt.Sprint(v.Count)
			}
			for _, v := range r.RefSlots {
				m["s "+v.Name] = fmt.Sprint(v.Objects)
			}
			m["inline"], m["groups"] = fmt.Sprint(len(r.Inline)), fmt.Sprintf("%+v", r.Groups)
			return m
		}
		fa, fb := fields(a), fields(b)
		var names []string
		for k := range fa {
			names = append(names, k)
		}
		for k := range fb {
			if _, ok := fa[k]; !ok {
				names = append(names, k)
			}
		}
		slices.Sort(names)
		for _, k := range names {
			va, vb := fa[k], fb[k]
			if va == vb {
				continue
			}
			if strings.HasPrefix(k, "r ") && len(va) == len(vb) && len(va) > 16 {
				ba, _ := hex.DecodeString(va)
				bb, _ := hex.DecodeString(vb)
				out = append(out, fmt.Sprintf("%s %s: %s", at, k, equalText(ba, bb)))
				continue
			}
			add(at, k, shorten(va), shorten(vb))
		}
	}
	return out
}

func shorten(s string) string {
	if len(s) > 96 {
		return s[:96] + "..."
	}
	return s
}

// equalText is "equal (N bytes)" or the lengths and the first differing
// ranges.
func equalText(x, y []byte) string {
	if bytes.Equal(x, y) {
		return fmt.Sprintf("equal (%d bytes)", len(x))
	}
	var ranges []string
	n := min(len(x), len(y))
	for i := 0; i < n; i++ {
		if x[i] == y[i] {
			continue
		}
		j := i
		for j < n && x[j] != y[j] {
			j++
		}
		r := fmt.Sprintf("[%#x,%#x)", i, j)
		if j-i <= 4 {
			r += fmt.Sprintf("%x>%x", x[i:j], y[i:j])
		}
		ranges = append(ranges, r)
		i = j
	}
	if len(x) != len(y) {
		ranges = append(ranges, fmt.Sprintf("[%#x,end)", n))
	}
	shown := ranges[:min(len(ranges), 8)]
	return fmt.Sprintf("differs (%d and %d bytes; %d ranges: %s)", len(x), len(y), len(ranges), strings.Join(shown, " "))
}

// sameLeaves compares the decoded stores' directories and leaves as sets.
func sameLeaves(x, y []byte) string {
	dx, errx := sav.DecodeDocumentData(x)
	dy, erry := sav.DecodeDocumentData(y)
	if errx != nil || erry != nil {
		return "not decoded"
	}
	key := func(s sav.DocumentStateData) []string {
		var out []string
		for _, r := range s.DirectoryRecords {
			out = append(out, fmt.Sprintf("dir %s %d", strings.ToLower(r.Path), r.Kind))
		}
		for _, r := range s.ValueRecords {
			sum := sha256.Sum256(r.Value.Bytes)
			out = append(out, fmt.Sprintf("leaf %s %d %d %d:%x", strings.ToLower(r.Path), r.Value.Kind, r.Value.Int32, len(r.Value.Bytes), sum[:6]))
		}
		slices.Sort(out)
		return out
	}
	kx, ky := key(dx.State), key(dy.State)
	if slices.Equal(kx, ky) {
		return fmt.Sprintf("the same %d entries in another order or framing", len(kx))
	}
	var only []string
	for _, k := range kx {
		if !slices.Contains(ky, k) {
			only = append(only, "-"+k)
		}
	}
	for _, k := range ky {
		if !slices.Contains(kx, k) {
			only = append(only, "+"+k)
		}
	}
	return fmt.Sprintf("differ: %s", strings.Join(only[:min(len(only), 12)], "; "))
}

// compare compares two files part by part.
func compare(out io.Writer, a, b string) error {
	x, err := os.ReadFile(a)
	if err != nil {
		return err
	}
	y, err := os.ReadFile(b)
	if err != nil {
		return err
	}
	return compareFiles(out, a, b, x, y)
}
