package ui

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/pathdialog"
)

func TestEditorDialogCancelDoesNotCommitCapturedGesture(t *testing.T) {
	for _, key := range []string{"open", "saveas"} {
		for _, gesture := range []string{"record", "brush", "rail"} {
			t.Run(key+"/"+gesture, func(t *testing.T) {
				e, calls := newModesEditor(t, "full")
				var release image.Point
				switch gesture {
				case "record":
					e.ShowCell(image.Pt(10, 10))
					modesPress(e, e.CellPoint(image.Pt(10, 10)), true)
					modesPress(e, e.CellPoint(image.Pt(11, 10)), true)
					if e.recordDrag == nil || !e.recordDrag.moved {
						t.Fatal("fixture did not capture a moved record")
					}
					release = e.CellPoint(image.Pt(13, 11))
				case "brush":
					_ = e.Key("terrain")
					e.ShowCell(image.Pt(20, 20))
					modesPress(e, e.CellPoint(image.Pt(20, 20)), true)
					modesPress(e, e.CellPoint(image.Pt(21, 20)), true)
					if !e.brush.down || len(e.brush.stroke) < 2 {
						t.Fatal("fixture did not capture a brush stroke")
					}
					release = e.CellPoint(image.Pt(23, 21))
				case "rail":
					release = image.Pt(e.doc.Viewer.cam.ViewW+150, 52)
					modesPress(e, release, true)
					if !e.down {
						t.Fatal("fixture did not capture a rail press")
					}
				}
				e.message = "prior message"
				doc, selection, message := e.doc, e.selected, e.message
				camera, brushMode := *e.doc.Viewer.cam, e.brush.mode
				dirty, undo, redo := e.Dirty(), e.doc.Edits.CanUndo(), e.doc.Edits.CanRedo()
				requests := make(chan pathdialog.Request, 1)
				bindEditorDialog(e, func(request pathdialog.Request) (pathdialog.Result, error) {
					requests <- request
					return pathdialog.Result{Cancelled: true}, nil
				}, t.TempDir())
				_ = e.Key(key)
				e.Pointer(Input{CursorX: release.X, CursorY: release.Y}, time.Time{})
				finishEditorDialog(t, e)
				select {
				case request := <-requests:
					want := pathdialog.OpenFile
					if key == "saveas" {
						want = pathdialog.SaveFile
					}
					if request.Kind != want {
						t.Fatalf("dialog kind %d, want %d", request.Kind, want)
					}
				default:
					t.Error("captured gesture did not reach the native dialog dependency")
				}
				for range 2 {
					e.Pointer(Input{CursorX: release.X, CursorY: release.Y}, time.Time{})
				}
				if len(calls.lines) != 0 || e.doc != doc || e.selected != selection || e.message != message || e.Dirty() != dirty || e.doc.Edits.CanUndo() != undo || e.doc.Edits.CanRedo() != redo || !reflect.DeepEqual(camera, *e.doc.Viewer.cam) || e.brush.mode != brushMode {
					t.Fatalf("cancel replayed a captured %s gesture: calls=%v documentChanged=%t selection=%d message=%q dirty=%t", gesture, calls.lines, e.doc != doc, e.selected, e.message, e.Dirty())
				}
			})
		}
	}
}
