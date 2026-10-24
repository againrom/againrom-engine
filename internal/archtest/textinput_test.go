package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestTextFieldsHaveOneModel(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range CheckTextInputRoutes(files) {
		t.Error(violation)
	}
}

func TestTextInputGuardRefusesAnInlineEditor(t *testing.T) {
	for _, source := range []string{
		"package ui; func(c *Chargen) EditName(s string) { c.name += s }",
		"package ui; func(c *Chargen) EditName(s string) { EditText(); c.name = s[:1] }",
	} {
		found := false
		for _, violation := range CheckTextInputRoutes(map[string]string{"pkg/ui/chargen.go": source}) {
			found = found || strings.Contains(violation.From, "Chargen.EditName") && strings.Contains(violation.Reason, "without inline editing")
		}
		if !found {
			t.Fatal("inline editor accepted", source)
		}
	}
	for _, violation := range CheckTextInputRoutes(map[string]string{"pkg/ui/other.go": "package ui; func(m *TextInputState) replace(s string) {}"}) {
		if strings.Contains(violation.Reason, "implementation outside") {
			return
		}
	}
	t.Fatal("second model accepted")
}
