package mod

import "testing"

func TestManifestAuthoringChecksDoNotTightenOrdinaryReading(t *testing.T) {
	for _, extra := range []string{
		"author = ['A', 'B']", "author = 5", "description = 42", "description = ['notes']", "description = \"\"\"multi\"\"\"",
		"conflicts = ['Old_Mod!']", "load-after = ['Other Mod']", "requires = ['core => 1']",
	} {
		source := []byte("id='compatible'\ntitle='Compatible'\nversion='1'\n" + extra + "\n")
		if _, err := ParseManifest(source); err != nil {
			t.Fatalf("ordinary reader: %s: %v", extra, err)
		}
		if _, err := ParseManifest(source, true); err == nil {
			t.Fatalf("authoring reader accepted %s", extra)
		}
	}
}
