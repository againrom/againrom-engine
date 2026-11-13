package archtest

import (
	"os"
	"strings"
	"testing"
)

// entryReturnTree is the smallest tree the entry return scan accepts.
func entryReturnTree() map[string]string {
	return map[string]string{
		"pkg/game/campaignreturn.go": "package game\nfunc (s *CampaignSession) leaveLive(n int) string { _ = s.returnToEntry(nil, n, false); return \"\" }\n" +
			"func (s *CampaignSession) returnToEntry(t *mapload.Table, n int, restart bool) error { return s.partyItems(t).returnEntry() }\n",
		"pkg/game/missionleave.go": "package game\nfunc (s *CampaignSession) canLeaveMission(n int) bool { return s.partyItems(nil).departedTown() }\n",
	}
}

func TestCheckEntryReturn(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"the tree as it stands", "", "", ""},
		{"a second-campaign branch in the return", "pkg/game/campaignreturn.go",
			"package game\nfunc (s *CampaignSession) returnToEntry(n int) error { if c := secondOf(s); c != nil { c.current = partyPlace{} }; return nil }\n",
			"returnToEntry names secondOf"},
		{"a record field in the admission", "pkg/game/missionleave.go",
			"package game\nfunc (s *CampaignSession) canLeaveMission(n int) bool { return s.Town.second != nil }\n", "canLeaveMission names second"},
		{"a row picked in leaveLive", "pkg/game/campaignreturn.go",
			"package game\nfunc (s *CampaignSession) leaveLive(n int) string { _ = secondPartyItems.over(s, nil); return \"\" }\n" +
				"func (s *CampaignSession) returnToEntry() error { return nil }\n", "leaveLive names secondPartyItems"},
		{"the campaign service in the return", "pkg/game/campaignreturn.go",
			"package game\nfunc (s *CampaignSession) returnToEntry(f *FrontEnd) error { _ = f.campaign(); return nil }\n", "returnToEntry names campaign"},
		{"a second return", "pkg/game/campaignsecond.go",
			"package game\nfunc (c *secondCampaign) returnToEntry() error { return nil }\n", "returnToEntry declared 2 times"},
		{"no return", "pkg/game/campaignreturn.go",
			"package game\n", "returnToEntry declared 0 times"},
		{"a record read elsewhere", "pkg/game/partyitemhome.go",
			"package game\nfunc (h partyItemHome) departure() *secondCampaign { return secondOf(h.sess) }\n", ""},
		{"a test file", "pkg/game/missionabandon_test.go",
			"package game\nfunc returnToEntry() { _ = secondOf(nil) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := entryReturnTree()
			if tc.file != "" {
				files[tc.file] = tc.src
			}
			var got []string
			for _, v := range CheckEntryReturn(files) {
				got = append(got, v.From+": "+v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckEntryReturn = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckEntryReturn = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestCheckEntryReturnRefusesAnEmptyScan(t *testing.T) {
	if vs := CheckEntryReturn(map[string]string{"pkg/game/x_test.go": "package game\n"}); len(vs) != 1 || !strings.Contains(vs[0].Reason, "no sources scanned") {
		t.Fatalf("CheckEntryReturn over no production file = %v, want the empty-scan refusal", vs)
	}
}

func TestEntryReturnHoldsInTheTree(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadTreeSources(root, "pkg/game")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckEntryReturn(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
