package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckCampaignRegistryReaders(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a parse", "pkg/game/campaignchain_release_test.go",
			"package game\nfunc f(b []byte) { reg.Parse(b) }\n", "reg.Parse in a campaign release test"},
		{"a parse in another campaign file", "pkg/game/campaigntail_release_test.go",
			"package game\nfunc f(b []byte) { r, _ := reg.Parse(b); _ = r }\n", "reg.Parse in a campaign release test"},
		{"the accessor", "pkg/game/campaignchain_release_test.go",
			"package game\nfunc f(t *testing.T, fe *FrontEnd) { _ = installedCampaign(t, fe) }\n", ""},
		{"another test file", "pkg/game/campaignregistry_test.go",
			"package game\nfunc f(b []byte) { reg.Parse(b) }\n", ""},
		{"another receiver", "pkg/game/campaignchain_release_test.go",
			"package game\nfunc f(b []byte) { json.Parse(b) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckCampaignRegistryReaders(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckCampaignRegistryReaders = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckCampaignRegistryReaders = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestCampaignReleaseTestsReadTheRegistryThroughProduction(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadCampaignReleaseTests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 2 {
		t.Fatalf("scanned %d campaign release tests; the scan guards nothing", len(files))
	}
	for _, v := range CheckCampaignRegistryReaders(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
