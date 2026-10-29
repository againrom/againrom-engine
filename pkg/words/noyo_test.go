package words

import (
	"io/fs"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

func TestRussianStringsCarryNoYo(t *testing.T) {
	raw, err := fs.ReadFile(Engine(), mod.StringsFile("ru"))
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(raw), "\n") {
		if strings.ContainsAny(line, "ёЁ") {
			t.Errorf("line %d holds ё or Ё, which the editor's font draws as И: %s", i+1, strings.TrimSpace(line))
		}
	}
}
