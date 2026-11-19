package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// Counts, over the installed maps, the hero-split messages the Dialog window
// refuses by cause, and the cells' other tag conditions before and after.
func TestReleaseDialogueRefusalCensus(t *testing.T) {
	x := installedInspector(t)
	var lines, never, ends int
	before, after := map[string]int{}, map[string]int{}
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		s := e.Document().Edits.(*mapInspectionEdits)
		var names []string
		for name := range s.workspace.Pages {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			var number uint32
			if n, _ := fmt.Sscanf(name, "event%d.txt", &number); n != 1 {
				continue
			}
			parts := s.MessageParts(number)
			split := slices.ContainsFunc(parts, func(p ui.InspectionMessagePart) bool { return p.Heroes != 0 })
			if !split {
				continue
			}
			lines++
			seenNever, seenEnds := false, false
			for _, p := range parts {
				seenNever = seenNever || p.Never
				seenEnds = seenEnds || p.Ends
				if p.Source < 0 || p.Never {
					continue
				}
				if was := tagOthers(p.Tag, 0); was != "" {
					before[was]++
				}
				if p.Other != "" {
					after[p.Other]++
					if p.Other == "female" || p.Other == "male" {
						t.Logf("%s %s: tag %q, heroes %04b", key, name, p.Tag, p.Heroes)
					}
				}
			}
			if seenNever {
				never++
			}
			if seenEnds {
				ends++
			}
		}
	}
	sum := func(m map[string]int) (n int) {
		for _, v := range m {
			n += v
		}
		return n
	}
	t.Logf("split lines %d; with a never-shown cell %d; with an ending cell %d", lines, never, ends)
	t.Logf("cells with other tag conditions: before %d, after %d", sum(before), sum(after))
	for _, m := range []map[string]int{before, after} {
		var out []string
		for k, v := range m {
			out = append(out, fmt.Sprintf("%q x%d", k, v))
		}
		slices.Sort(out)
		t.Log(strings.Join(out, "; "))
	}
}
