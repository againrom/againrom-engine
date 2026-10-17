package ui

import (
	"strings"
	"testing"
)

func experienceWithheldSubjects() map[string]PanelSubject {
	plain := sheetSubject()
	piercing := sheetSubject()
	piercing.OriginalPanel = OriginalPanelActor{Known: true, Byte14A: 2}
	return map[string]PanelSubject{"xp alone": plain, "xp beside armor piercing": piercing}
}

func experienceWithheldLayouts() map[string]PanelLayout {
	return map[string]PanelLayout{"authored": AuthoredPanelLayout(), "compact": CompactPanelLayout(nil)}
}

// A withheld character loses exactly the XP cell (DIV-3613): a right cell
// sharing its row takes the row's place, a row left empty closes up, and no
// other row changes its text.
func TestAWithheldCharacterOmitsOnlyTheExperienceCell(t *testing.T) {
	for ln, l := range experienceWithheldLayouts() {
		for sn, shown := range experienceWithheldSubjects() {
			t.Run(ln+" "+sn, func(t *testing.T) {
				withheld := shown
				withheld.Char.ExperienceWithheld = true
				control := PanelStatement(l, shown)
				var want []string
				found := false
				for _, row := range control {
					if !strings.HasPrefix(row, "XP 1593") {
						want = append(want, row)
						continue
					}
					found = true
					if _, right, ok := strings.Cut(row, "  "); ok {
						want = append(want, right)
					}
				}
				if !found {
					t.Fatalf("loss control: the shown card states no XP row: %q", control)
				}
				got := PanelStatement(l, withheld)
				if strings.Join(got, "\n") != strings.Join(want, "\n") {
					t.Fatalf("withheld card\n%q\nwant\n%q", got, want)
				}
			})
		}
	}
}

// The rows keep their places: every row above the XP row stands where it
// stood, and the rows below it move up by one whole row, keeping their
// spacing, so the card leaves no hole.
func TestAWithheldExperienceRowLeavesNoHole(t *testing.T) {
	f := panelFont()
	for ln, l := range experienceWithheldLayouts() {
		for sn, shown := range experienceWithheldSubjects() {
			t.Run(ln+" "+sn, func(t *testing.T) {
				withheld := shown
				withheld.Char.ExperienceWithheld = true
				control := CharacterPanelReport(l, f, shown)
				got := CharacterPanelReport(l, f, withheld)
				xp := -1
				for i, r := range control {
					if r.Label == "XP" {
						xp = i
					}
				}
				if xp < 0 {
					t.Fatal("loss control: the shown card has no XP row")
				}
				if len(got) != len(control) && len(got) != len(control)-1 {
					t.Fatalf("withheld card has %d rows against %d", len(got), len(control))
				}
				for i := 0; i < xp; i++ {
					if got[i].At != control[i].At {
						t.Fatalf("row %d above XP moved: %v, was %v", i, got[i].At, control[i].At)
					}
				}
				if len(got) == len(control) {
					if got[xp].Label != "ARMOR PIERCING" || got[xp].At != control[xp].At {
						t.Fatalf("the shared row is %+v, want ARMOR PIERCING at %v", got[xp], control[xp].At)
					}
					for i := xp + 1; i < len(got); i++ {
						if got[i].At != control[i].At {
							t.Fatalf("row %d below moved: %v, was %v", i, got[i].At, control[i].At)
						}
					}
					return
				}
				if xp < len(got) && got[xp].At.Y != control[xp].At.Y {
					t.Fatalf("the row after XP stands at y=%d, not on the vacated y=%d", got[xp].At.Y, control[xp].At.Y)
				}
				for i := xp; i < len(got); i++ {
					if got[i].Label != control[i+1].Label {
						t.Fatalf("row %d is %q, want %q", i, got[i].Label, control[i+1].Label)
					}
					if dy, was := got[i].At.Y-got[xp].At.Y, control[i+1].At.Y-control[xp+1].At.Y; dy != was {
						t.Fatalf("row %q spacing %d, was %d", got[i].Label, dy, was)
					}
				}
			})
		}
	}
}
