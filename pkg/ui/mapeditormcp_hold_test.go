package ui

import (
	"encoding/json"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func holdNative(canvas func() Input) func() editorNativeInput {
	return func() editorNativeInput {
		return editorNativeInput{pressed: func(ebiten.Key) bool { return false }, justPressed: func(ebiten.Key) bool { return false }, canvas: canvas()}
	}
}

func heldPress(t *testing.T) (*MapEditor, *mcpPipe, image.Point) {
	t.Helper()
	e := newMapEditorFixture(t)
	e.Layout(800, 600)
	p := newMCPPipe(t, e)
	cell := p.tool(t, "cell", map[string]any{"x": 21, "y": 20})
	var at struct{ X, Y int }
	if err := json.Unmarshal([]byte(cell.Content[0].Text), &at); err != nil {
		t.Fatal(err)
	}
	p.tool(t, "pointer", map[string]any{"action": "press", "x": at.X, "y": at.Y})
	if !e.mcp.mcpHeld() || !e.down {
		t.Fatalf("press not held: mcp=%v down=%v", e.mcp.mcpHeld(), e.down)
	}
	return e, p, image.Pt(at.X, at.Y)
}

func updateFrames(t *testing.T, e *MapEditor, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := e.Update(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEditorMCPHeldPressStaysWhilePersonIsIdle(t *testing.T) {
	e, _, _ := heldPress(t)
	updateFrames(t, e, 5)
	if !e.mcp.mcpHeld() || !e.down {
		t.Fatalf("an idle person released the agent press: mcp=%v down=%v", e.mcp.mcpHeld(), e.down)
	}
}

func TestEditorMCPHeldPressEndsWhenQueueCloses(t *testing.T) {
	e, _, _ := heldPress(t)
	e.CloseMCP()
	updateFrames(t, e, 3)
	if e.mcp.mcpHeld() || e.down {
		t.Fatalf("press survives CloseMCP: mcp=%v down=%v", e.mcp.mcpHeld(), e.down)
	}
}

func TestEditorMCPHeldPressEndsWhenClientDisconnects(t *testing.T) {
	e, p, _ := heldPress(t)
	if err := p.in.Close(); err != nil {
		t.Fatal(err)
	}
	limit := time.Now().Add(5 * time.Second)
	for !e.mcp.queue.Disconnected() {
		if time.Now().After(limit) {
			t.Fatal("the stdio end of input never reached the queue")
		}
		time.Sleep(time.Millisecond)
	}
	updateFrames(t, e, 3)
	if e.mcp.mcpHeld() || e.down {
		t.Fatalf("press survives the client: mcp=%v down=%v", e.mcp.mcpHeld(), e.down)
	}
	if e.mcp.connection != mcpGoneLine {
		t.Fatalf("status connection = %q", e.mcp.connection)
	}
	if line := e.statusText(); !strings.Contains(line, mcpGoneLine) {
		t.Fatalf("status line lacks the notice: %q", line)
	}
}

func TestEditorMCPPersonMoveTakesThePointerBack(t *testing.T) {
	e, _, _ := heldPress(t)
	other := e.CellPoint(image.Pt(29, 18))
	e.nativeInput = holdNative(func() Input { return Input{CursorX: other.X, CursorY: other.Y} })
	updateFrames(t, e, 3)
	if e.mcp.mcpHeld() || e.down {
		t.Fatalf("person move did not release: mcp=%v down=%v", e.mcp.mcpHeld(), e.down)
	}
	if !e.hoverOK || e.hoverCell != image.Pt(29, 18) {
		t.Fatalf("hover does not follow the person: ok=%v cell=%v", e.hoverOK, e.hoverCell)
	}
}

func TestEditorMCPPersonPressTakesThePointerBack(t *testing.T) {
	e, _, _ := heldPress(t)
	e.nativeInput = holdNative(func() Input { return Input{PrimaryDown: true} })
	updateFrames(t, e, 1)
	if e.mcp.mcpHeld() {
		t.Fatal("person press left the agent press held")
	}
	e.nativeInput = holdNative(func() Input { return Input{} })
	updateFrames(t, e, 2)
	if e.down {
		t.Fatal("person release did not end the press")
	}
}

func TestEditorMCPPersonWheelAndPanTakeThePointerBack(t *testing.T) {
	for name, in := range map[string]Input{"wheel": {WheelY: 1}, "pan": {PanLeft: true}} {
		e, _, _ := heldPress(t)
		e.nativeInput = holdNative(func() Input { return in })
		updateFrames(t, e, 1)
		if e.mcp.mcpHeld() {
			t.Fatalf("%s left the agent press held", name)
		}
	}
}
