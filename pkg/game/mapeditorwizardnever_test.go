package game

import (
	"os"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// The shipped pair of 131.alm trigger 0 compares a check with itself by the
// code 0xFFFFFFFF. Editing it in the wizard writes, and the edit does not
// show the host's English refusal.
func TestReleaseTriggerWizardEditsTheInstalledNeverComparisonPair(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	x := installedInspector(t)
	e := x.Editor()
	e.SetWords(words.For("en"))
	e.Layout(1920, 1080)
	if err := e.Open("scenario/131.alm"); err != nil {
		t.Skip("no scenario/131.alm:", err)
	}
	sc := wizScript(t, e.Document().SourceBytes)
	if len(sc.Triggers) == 0 || sc.Triggers[0].Cmp[0] != 0xFFFFFFFF {
		t.Skip("trigger 0 of this install has no never-comparison pair")
	}
	before := append([]byte(nil), e.Document().SourceBytes...)
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	trigChoose(t, e, wizField(t, e, "c0.p0"), "New variable")
	wizApply(t, e)
	if strings.Contains(trigLines(e), "outside 0..5") {
		t.Fatalf("the edit was refused:\n%s", trigLines(e))
	}
	after := wizScript(t, e.Document().SourceBytes)
	if string(before) == string(e.Document().SourceBytes) || after.Triggers[0].Cmp[0] != 0xFFFFFFFF {
		t.Fatalf("the edit wrote nothing or lost the comparison code %#x", after.Triggers[0].Cmp[0])
	}
}
