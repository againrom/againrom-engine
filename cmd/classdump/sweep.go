package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/res"
)

// The two extensions the walk takes, matched case-folded like regtool's sweep:
// a loose map, and a container that may hold maps.
const (
	almExt = ".alm"
	resExt = ".res"
)

// type4KeyMask selects a type-4 record's class key out of its Kind word. The
// low 16 bits are the structures.reg ID; the high half is not decoded, so it is
// masked off rather than read into the key.
const type4KeyMask = 0xffff

// census is one source's placement tally — and, summed over the sweep, the
// totals block's. Every field is a count of references, never of classes: the
// same class placed twice is two references.
type census struct {
	// type-3: a nonzero overlay cell, whose code-1 is an objects.reg ID. A code
	// that resolves past the registry is counted here and nowhere else. Such
	// cells exist on shipped maps and whether the engine reads them is
	// undecided, so this figure is a measurement and not a fault.
	t3Nonzero, t3Resolved, t3Past int

	// type-4: every record is a reference, keyed by the low half of Kind against
	// structures.reg.
	t4Refs, t4Resolved, t4Unresolved int

	// type-6: every record is counted, but only a direct one is resolved
	// against units.reg — the other three buckets name a table this story does
	// not hold.
	//
	// The four bucket counters are filled by the bucket's whole value and never
	// by a chain of conditions. t6Records is counted independently of them, so
	// a bucket with no counter of its own would show up as records exceeding
	// their sum rather than silently joining another bucket's total.
	t6Records                      int
	t6Direct, t6NPC, t6Def, t6Both int
	t6Resolved, t6Unresolved       int

	// t6WouldResolve is how many diverted records name a units.reg ID that
	// exists anyway — the figure that says whether a divert is load-bearing. A
	// diverted record whose key would also have resolved tells nothing on its
	// own; one whose key would NOT have resolved is a record only the divert
	// explains.
	t6WouldResolve int
}

// add accumulates one source's census into the running totals.
func (c *census) add(o census) {
	c.t3Nonzero += o.t3Nonzero
	c.t3Resolved += o.t3Resolved
	c.t3Past += o.t3Past

	c.t4Refs += o.t4Refs
	c.t4Resolved += o.t4Resolved
	c.t4Unresolved += o.t4Unresolved

	c.t6Records += o.t6Records
	c.t6Direct += o.t6Direct
	c.t6NPC += o.t6NPC
	c.t6Def += o.t6Def
	c.t6Both += o.t6Both
	c.t6Resolved += o.t6Resolved
	c.t6Unresolved += o.t6Unresolved
	c.t6WouldResolve += o.t6WouldResolve
}

// countBucket tallies one type-6 record under its bucket.
func (c *census) countBucket(b bucket) {
	switch b {
	case bucketDirect:
		c.t6Direct++
	case bucketNPC:
		c.t6NPC++
	case bucketDef:
		c.t6Def++
	case bucketBoth:
		c.t6Both++
	}
}

// failure is one placed reference that did not resolve: the source it was placed
// by, which kind of placement it is, its index within that kind, and the key it
// named. The key is signed because a type-6 key is (ALM-CLS-038): a negative one
// is reported at the value the file gave, not folded to ~65000.
type failure struct {
	source string
	kind   string
	index  int
	value  int32
}

// writeCensus writes one census line: a prefix field — a quoted source path, or
// the bare "totals" — then the fixed tokens, in the fixed order and the fixed
// spelling, so evidence quotes this line rather than narrating it.
//
// The four bucket names come from the buckets themselves rather than from
// literals here: a bucket's token and its count are then unable to drift apart,
// which is what bucket.String exists for.
func writeCensus(w io.Writer, prefix string, c census) error {
	_, err := fmt.Fprintf(w,
		"%s type3 nonzero=%d resolved=%d past-registry=%d"+
			" type4 refs=%d resolved=%d unresolved=%d"+
			" type6 records=%d %v=%d resolved=%d unresolved=%d %v=%d %v=%d %v=%d diverted-would-resolve=%d\n",
		prefix,
		c.t3Nonzero, c.t3Resolved, c.t3Past,
		c.t4Refs, c.t4Resolved, c.t4Unresolved,
		c.t6Records, bucketDirect, c.t6Direct, c.t6Resolved, c.t6Unresolved,
		bucketNPC, c.t6NPC, bucketDef, c.t6Def, bucketBoth, c.t6Both,
		c.t6WouldResolve)
	return err
}

// sweepDir resolves every class reference placed by every .alm under dir —
// loose files and .res entries alike — against the three loaded collections, and
// prints one census line per source, then the totals block, then one line per
// reference that did not resolve.
//
// Sources are not de-duplicated: a loose map and the same map inside a .res are
// two sources under two paths. Telling them apart needs a content comparison,
// and a comparison that answered "same map" would hide exactly the divergence a
// sweep exists to find.
//
// The returned error — hence the process exit status — is a statement about
// class resolution and about nothing else. It is non-nil iff a type-4 or a
// direct type-6 reference did not resolve. In particular:
//
//   - a diverted type-6 record is a reference into a table this story does not
//     hold: counted apart, never failed;
//   - a type-3 code resolving past the registry is counted and printed, because
//     whether the engine reads such a cell is undecided;
//   - a source that would not decode is reported and counted as skipped, since a
//     source that never decoded was never asked this story's question — the
//     skipped figure is what keeps that visible in the recorded output;
//   - an absent sprite file is never even looked for: this layer resolves
//     classes, not art.
//
// A sweep that failed on a residual we already know about would say nothing
// about whether the loader is right, which is what that rule exists to prevent.
// Being unable to walk dir at all is not a sweep result and is returned as the
// operational failure it is.
func sweepDir(dir string, c *classes, w io.Writer) error {
	var (
		total   census
		fails   []failure
		sources int
		skipped int
	)

	// skip records a source (or a container) that could not be read or decoded.
	skip := func(source string, cause error) error {
		skipped++
		_, err := fmt.Fprintf(w, "skipped %s: %v\n", quote(source), cause)
		return err
	}

	sweepOne := func(source string, b []byte) error {
		m, err := alm.Open(b)
		if err != nil {
			return skip(source, err)
		}
		sc, sf := sweepMap(m, c, source)
		if err := writeCensus(w, quote(source), sc); err != nil {
			return err
		}
		total.add(sc)
		fails = append(fails, sf...)
		sources++
		return nil
	}

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch {
		case strings.EqualFold(filepath.Ext(path), almExt):
			b, err := os.ReadFile(path)
			if err != nil {
				return skip(path, err)
			}
			return sweepOne(path, b)

		case strings.EqualFold(filepath.Ext(path), resExt):
			a, err := res.Open(path)
			if err != nil {
				return skip(path, err)
			}
			for _, e := range a.Entries() {
				if !strings.EqualFold(filepath.Ext(e.Path), almExt) {
					continue
				}
				// The source identity of an archived map is the pair that
				// located it, so two entries of two archives can never collide
				// under one name.
				source := path + "::" + e.Path
				b, err := a.ReadFile(e.Path)
				if err != nil {
					if werr := skip(source, err); werr != nil {
						return werr
					}
					continue
				}
				if err := sweepOne(source, b); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return walkErr
	}

	if _, err := fmt.Fprintf(w, "totals sources=%d skipped=%d\n", sources, skipped); err != nil {
		return err
	}
	if err := writeCensus(w, "totals", total); err != nil {
		return err
	}
	for _, f := range fails {
		if _, err := fmt.Fprintf(w, "unresolved %s %s index=%d value=%d\n",
			quote(f.source), f.kind, f.index, f.value); err != nil {
			return err
		}
	}

	if n := total.t4Unresolved + total.t6Unresolved; n > 0 {
		return fmt.Errorf("%d placed reference(s) did not resolve", n)
	}
	return nil
}

// sweepMap resolves one decoded map's three placement kinds. It opens nothing
// and derives every count from the map and the collections alone.
//
// The three kinds are three different lookups against three different
// collections and share nothing but the shape of the loop: a type-3 cell's
// placement byte names an objects.reg class, a type-4 record's key is a
// structures.reg ID, and a direct type-6 record's ClassID is a units.reg ID.
//
// The type-3 lookup goes through ObjectClasses.ByCode, which is the ONE
// place the byte-to-identity offset is written. The zero guard stays because
// the census counts nonzero cells; it no longer also protects the
// arithmetic.
func sweepMap(m *alm.Map, c *classes, source string) (census, []failure) {
	var (
		out   census
		fails []failure
	)

	for _, code := range m.Overlay {
		if code == 0 {
			continue
		}
		out.t3Nonzero++
		if _, ok := c.objects.ByCode(code); ok {
			out.t3Resolved++
		} else {
			out.t3Past++
		}
	}

	for i, o := range m.Objects {
		out.t4Refs++
		key := int32(o.Kind & type4KeyMask)
		if _, ok := c.structures.ByID(key); ok {
			out.t4Resolved++
			continue
		}
		out.t4Unresolved++
		fails = append(fails, failure{source: source, kind: "type4", index: i, value: key})
	}

	for i, u := range m.Units {
		ref := classifyUnit(u)
		out.t6Records++
		out.countBucket(ref.Bucket)

		_, ok := c.units.ByID(ref.ClassID)
		if ref.Bucket != bucketDirect {
			// A diverted record names a table this tool does not hold, so its
			// key is measured rather than judged: whether it would have resolved
			// is the datum, not a verdict.
			if ok {
				out.t6WouldResolve++
			}
			continue
		}
		if ok {
			out.t6Resolved++
			continue
		}
		out.t6Unresolved++
		fails = append(fails, failure{source: source, kind: "type6", index: i, value: ref.ClassID})
	}

	return out, fails
}
