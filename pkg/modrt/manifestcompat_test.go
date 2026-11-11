package modrt

import (
	"testing"

	"againrom/pkg/mod"
)

func TestOrdinaryLoadKeepsDocumentaryMetadataAndIgnoredOrderingEntries(t *testing.T) {
	for _, extra := range []string{
		"author = ['A', 'B']", "author = 5", "description = 42", "description = ['notes']", "description = \"\"\"multi\"\"\"",
		"conflicts = ['Old_Mod!']", "load-after = ['Other Mod']",
	} {
		t.Run(extra, func(t *testing.T) {
			entries := makeMods(t, map[string]map[string]string{"compatible": {
				mod.ManifestName: "id='compatible'\ntitle='Compatible'\nversion='1'\n" + extra + "\n",
			}})
			if _, err := Load(entries, "rom1-en", nil, Options{}); err != nil {
				t.Fatalf("ordinary load refused an accepted manifest: %v", err)
			}
			if _, err := Load(entries, "rom1-en", nil, Options{Authoring: true}); err == nil {
				t.Fatal("authoring load did not check the edited manifest")
			}
		})
	}
}
