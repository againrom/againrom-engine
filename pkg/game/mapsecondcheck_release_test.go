//go:build release

package game

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// secondHost opens source and answers its edit host and the checker seam.
func secondHost(t *testing.T, x *MapInspector, source string) (*mapInspectionEdits, ui.InspectionChecker) {
	t.Helper()
	d, err := x.Open(source)
	if err != nil {
		t.Fatal(source, err)
	}
	host := d.Edits.(*mapInspectionEdits)
	c, ok := ui.Seam[ui.InspectionChecker](d.Edits)
	if !ok {
		t.Fatalf("%s: the second-game host offers no checker", source)
	}
	return host, c
}

func TestReleaseSecondMapCheckRunsOnEveryInstalledMap(t *testing.T) {
	x := installedInspector(t)
	sources, _ := installedSecondMaps(t, x)
	errs, warns := 0, 0
	byRule := map[string]int{}
	for _, source := range sources {
		_, c := secondHost(t, x, source)
		rows := c.Report()
		skipped := false
		for _, r := range rows {
			byRule[r.Rule]++
			if r.Rule == ruleSkipped {
				skipped = true
				for _, id := range []string{"script-reference", "owner-slot", "victory", "stock-link"} {
					if !strings.Contains(r.Text, id) {
						t.Errorf("%s: the report does not name %s as not run: %s", source, id, r.Text)
					}
				}
				if strings.Contains(r.Text, "record-outside") {
					t.Errorf("%s: record-outside runs here and is named as skipped", source)
				}
			}
		}
		if !skipped {
			t.Errorf("%s: the report does not say which rules did not run", source)
		}
		e, w := ui.CountFindings(rows)
		errs, warns = errs+e, warns+w
	}
	t.Logf("PASS: %d installed second-game maps checked (%s): %d errors, %d warnings; findings by rule %v", len(sources), installLanguage(x), errs, warns, byRule)
}

func TestReleaseSecondMapPlantedErrorIsReportedAndSaveAsks(t *testing.T) {
	x := installedInspector(t)
	sources, _ := installedSecondMaps(t, x)
	for _, source := range sources {
		host, c := secondHost(t, x, source)
		m, err := host.model.Map()
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Units) == 0 {
			continue
		}
		// A unit copied to a cell far outside the map.
		if _, err := host.model.CloneRecord(mapedit.KindUnit, 0, 0xffffff, 0xffffff); err != nil {
			continue
		}
		rows := c.Report()
		if e, _ := ui.CountFindings(rows); e == 0 || rows[0].Rule != "record-outside" {
			t.Fatalf("%s: the planted unit is not reported: %v", source, rows)
		}
		_, err = host.SaveAs(filepath.Join(t.TempDir(), "planted.alm"))
		var refused *ui.ReportRefusal
		if !errors.As(err, &refused) || !refused.Ask {
			t.Fatalf("%s: save with an error answered %v, not the question", source, err)
		}
		host.AllowErrors()
		if _, err := host.SaveAs(filepath.Join(t.TempDir(), "planted.alm")); err != nil {
			t.Fatalf("%s: save after the answer: %v", source, err)
		}
		t.Logf("PASS: %s: planted unit outside the map reported (%s), save asked, then wrote", source, rows[0].Rule)
		return
	}
	t.Fatal("no installed map took a planted error")
}
