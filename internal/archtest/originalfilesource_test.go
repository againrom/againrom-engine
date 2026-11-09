package archtest

import (
	"strings"
	"testing"
)

// An original file's sources are chosen and read in one table.
func TestOriginalFileSourcesAreOneBoundary(t *testing.T) {
	packages, defined := ownedProductionFunctions(t)
	checkOwnersDefined(t, originalFileSourceOwners, defined)
	for _, finding := range CheckOriginalFileSources(packages) {
		t.Error(finding)
	}
}

func TestOriginalFileSourcesRejectCaptureAndCasterBypasses(t *testing.T) {
	const head = `package game
type FrontEnd struct{}
type originalSource struct{}
func readOriginalApplicationState() {}
func applyOriginalBuildingCasters() {}
func originalFileSources() { readOriginalApplicationState(); applyOriginalBuildingCasters() }
`
	for _, tc := range []struct {
		name, source, want string
	}{
		{"table", "func (f *FrontEnd) captureApplicationState() { originalFileSources() }", ""},
		{"capture bypass", "func (f *FrontEnd) captureApplicationState() { readOriginalApplicationState() }", "FrontEnd.captureApplicationState references"},
		{"capture function value", "func (f *FrontEnd) captureApplicationState() { read := readOriginalApplicationState; read() }", "FrontEnd.captureApplicationState references"},
		{"caster bypass", "func restore() { applyOriginalBuildingCasters() }", "restore references"},
		{"caster function value", "var caster = applyOriginalBuildingCasters", "package scope references"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module := checkedFixture(t, fixturePackage{path: "againrom/pkg/game", files: map[string]string{"pkg/game/source.go": head + tc.source}})
			got := strings.Join(CheckOriginalFileSources([]*CheckedPackage{module.Packages["againrom/pkg/game"]}), "\n")
			if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
				t.Fatalf("findings %q, want %q", got, tc.want)
			}
		})
	}
}
