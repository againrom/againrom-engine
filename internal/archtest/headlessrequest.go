package archtest

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// namedControlAccessorsRemaining is the falling-only count of exported MapEditor
// rectangle accessors outside FindControl. A fall updates it in the same commit.
const namedControlAccessorsRemaining = 5

var (
	headlessPictureMethod = regexp.MustCompile(`(?m)^func \(a \*App\) (Headless[A-Za-z]*)\(.*\*image\.RGBA`)
	windowSizeSscanf      = regexp.MustCompile(`Sscanf\([^\n]*"%dx%d"`)
	windowSizeParser      = regexp.MustCompile(`(?m)^func ParseWindowSize\(`)
	namedControlAccessor  = regexp.MustCompile(`(?m)^func \(e \*MapEditor\) [A-Z][A-Za-z]*Rects?\([^\n]*image\.Rectangle`)
)

// CheckHeadlessRequest states the three single builders of the headless
// instruments: one picture entry point on App, one window size parser, and a
// falling count of named editor control accessors beside the control finder.
func CheckHeadlessRequest(files map[string]string) []Violation {
	var out []Violation
	var pictures []string
	parsers := 0
	named := 0
	for path, src := range files {
		if strings.HasPrefix(path, "pkg/ui/") {
			for _, m := range headlessPictureMethod.FindAllStringSubmatch(src, -1) {
				pictures = append(pictures, m[1])
			}
			named += len(namedControlAccessor.FindAllString(src, -1))
		}
		if windowSizeSscanf.MatchString(src) || strings.Contains(src, "routeWindowSize") {
			out = append(out, Violation{From: path, Reason: "a window size parser beside ui.ParseWindowSize"})
		}
		parsers += len(windowSizeParser.FindAllString(src, -1))
	}
	if len(pictures) != 1 || pictures[0] != "HeadlessPicture" {
		out = append(out, Violation{From: "pkg/ui", Reason: fmt.Sprintf("headless picture methods on App: %v, want HeadlessPicture only", pictures)})
	}
	if parsers != 1 {
		out = append(out, Violation{From: "pkg/ui", Reason: fmt.Sprintf("%d definitions of ParseWindowSize, want 1", parsers)})
	}
	switch {
	case named > namedControlAccessorsRemaining:
		out = append(out, Violation{From: "pkg/ui", Reason: fmt.Sprintf("named editor control accessors rose to %d, baseline %d: add a FindControl row instead", named, namedControlAccessorsRemaining)})
	case named < namedControlAccessorsRemaining:
		out = append(out, Violation{From: "pkg/ui", Reason: fmt.Sprintf("named editor control accessors fell to %d: lower the baseline to match", named)})
	}
	return out
}

// LoadHeadlessRequestSources reads the non-test Go sources under pkg and cmd.
func LoadHeadlessRequestSources(root string) (map[string]string, error) {
	files := map[string]string{}
	for _, top := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = string(b)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}
