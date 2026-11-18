package game

import (
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func addFixtureRecord(t *testing.T, e *ui.MapEditor, kind ui.RecordKind, values map[string]string) {
	t.Helper()
	fields := map[string]any{}
	args := map[string]any{"kind": kind, "values": fields}
	for key, value := range values {
		if key == "cell" {
			x, y, _ := strings.Cut(value, ",")
			args["x"], _ = strconv.Atoi(x)
			args["y"], _ = strconv.Atoi(y)
			continue
		}
		if strings.Contains(value, "|") {
			var items []int64
			for _, item := range strings.Split(value, "|") {
				n, err := strconv.ParseInt(item, 10, 64)
				if err != nil {
					t.Fatal(err)
				}
				items = append(items, n)
			}
			fields[key] = items
			continue
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			fields[key] = n
			continue
		}
		fields[key] = value
	}
	_, must := editorMCP(t, e)
	must("add_record", args)
	e.CloseMCP()
	ref := e.Document().AddedRecord
	if ref.Kind != "" {
		e.Select(recordIndex(t, e, ref.Kind.String(), ref.Index), false)
	}
}

func addFixturePlacement(t *testing.T, e *ui.MapEditor, key string, values map[string]string) {
	t.Helper()
	kind := map[string]ui.RecordKind{"unit": ui.KindUnit, "structure": ui.KindStructure, "sack": ui.KindSack}[key]
	addFixtureRecord(t, e, kind, values)
}
