package game

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"againrom/pkg/editormcp"
)

func TestMapEditorMCPKeepsRealModelHistoryAndSavedBytes(t *testing.T) {
	original := inspectionFixtureBytes(t)
	_, e, source := openTestMapEditor(t, original)
	server := e.EnableMCP(0)
	inRead, inWrite := io.Pipe()
	outRead, outWrite := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(ctx, inRead, outWrite) }()
	defer func() {
		e.CloseMCP()
		cancel()
		_ = inWrite.Close()
		_ = inRead.Close()
		_ = outRead.Close()
		_ = outWrite.Close()
		<-serverDone
	}()
	scan := bufio.NewReader(outRead)
	id := 0
	call := func(method string, params any) json.RawMessage {
		t.Helper()
		id++
		b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(inWrite, "%s\n", b); err != nil {
			t.Fatal(err)
		}
		done := make(chan []byte, 1)
		go func() {
			b, err := scan.ReadBytes('\n')
			if err != nil {
				done <- nil
			} else {
				done <- b
			}
		}()
		for {
			select {
			case b := <-done:
				var rsp struct {
					Result json.RawMessage
					Error  any
				}
				if json.Unmarshal(b, &rsp) != nil || rsp.Error != nil {
					t.Fatalf("RPC failed: %s", b)
				}
				return rsp.Result
			case <-ctx.Done():
				t.Fatal(ctx.Err())
				return nil
			default:
				if err := e.Update(); err != nil {
					t.Fatal(err)
				}
				e.CompleteMCPFrame(func() (image.Image, error) { return image.NewRGBA(image.Rect(0, 0, 1280, 800)), nil })
				runtime.Gosched()
			}
		}
	}
	tool := func(name string, args any) editormcp.Result {
		t.Helper()
		var r editormcp.Result
		b := call("tools/call", map[string]any{"name": name, "arguments": args})
		if json.Unmarshal(b, &r) != nil {
			t.Fatal(string(b))
		}
		return r
	}
	must := func(name string, args any) editormcp.Result {
		t.Helper()
		r := tool(name, args)
		if r.IsError {
			t.Fatalf("%s: %+v", name, r)
		}
		return r
	}
	call("initialize", map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "fixture", "version": "1"}})
	cell := must("cell", map[string]int{"x": 3, "y": 4})
	var from struct{ X, Y int }
	if json.Unmarshal([]byte(cell.Content[0].Text), &from) != nil {
		t.Fatal(cell)
	}
	must("pointer", map[string]any{"action": "click", "x": from.X, "y": from.Y})
	if e.Selected() != 0 {
		t.Fatalf("selected %d", e.Selected())
	}
	to := e.CellPoint(image.Pt(5, 6))
	must("pointer", map[string]any{"action": "drag", "x": from.X, "y": from.Y, "to_x": to.X, "to_y": to.Y})
	if !e.Dirty() || !e.Document().Edits.CanUndo() {
		t.Fatal("real model has no history")
	}
	checkMovedMapBytes(t, original, e.Document().SourceBytes, 0, image.Pt(5, 6))
	changed := bytes.Clone(e.Document().SourceBytes)
	must("key", map[string]string{"name": "undo"})
	if e.Dirty() || !bytes.Equal(original, e.Document().SourceBytes) {
		t.Fatal("undo did not restore exact ALM bytes")
	}
	must("action_redo", map[string]any{})
	if !e.Dirty() || !bytes.Equal(changed, e.Document().SourceBytes) {
		t.Fatal("redo did not restore edit")
	}
	must("open", map[string]string{"path": source})
	if !e.Dirty() || !bytes.Equal(changed, e.Document().SourceBytes) {
		t.Fatal("dirty Open discarded work")
	}
	must("key", map[string]string{"name": "escape"})
	must("map", map[string]int{"x": 5, "y": 6})
	if result := tool("save_as", map[string]string{"path": source}); !result.IsError {
		t.Fatal("Save As overwrote the source")
	}
	if got, err := os.ReadFile(source); err != nil || !bytes.Equal(got, original) {
		t.Fatal("refused save changed original", err)
	}
	must("key", map[string]string{"name": "escape"})
	target := filepath.Join(t.TempDir(), "copy.alm")
	must("save_as", map[string]string{"path": target})
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, changed) {
		t.Fatal("Save As output differs from model", err)
	}
	if e.Dirty() {
		t.Fatal("Save As did not checkpoint history")
	}
	must("frame", map[string]any{})
}
