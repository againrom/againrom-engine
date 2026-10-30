package ui

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"againrom/pkg/editormcp"
	"github.com/hajimehoshi/ebiten/v2"
)

type mcpWitness struct {
	editorMoveProbe
	before *InspectionDocument
}

func (p *mcpWitness) Undo() (*InspectionDocument, error) {
	p.dirty = false
	p.doc = p.before
	return p.doc, nil
}
func (p *mcpWitness) SaveAs(path string) (string, error) {
	if err := os.WriteFile(path, []byte("synthetic map witness"), 0600); err != nil {
		return "", err
	}
	p.dirty = false
	return path, nil
}

type mcpPipe struct {
	e     *MapEditor
	in    *io.PipeWriter
	out   *bufio.Reader
	seq   int
	frame image.Image
}

func newMCPPipe(t *testing.T, e *MapEditor) *mcpPipe {
	t.Helper()
	server := e.EnableMCP(0)
	e.nativeInput = func() editorNativeInput {
		return editorNativeInput{pressed: func(ebiten.Key) bool { return false }, justPressed: func(ebiten.Key) bool { return false }, canvas: Input{Unfocused: true}}
	}
	inRead, inWrite := io.Pipe()
	outRead, outWrite := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, inRead, outWrite) }()
	t.Cleanup(func() {
		e.CloseMCP()
		cancel()
		_ = inWrite.Close()
		_ = outRead.Close()
		_ = outWrite.Close()
		_ = inRead.Close()
		<-done
	})
	frame := image.NewRGBA(image.Rect(0, 0, e.window.X, e.window.Y))
	frame.Set(7, 9, color.RGBA{R: 71, G: 23, B: 99, A: 255})
	return &mcpPipe{e: e, in: inWrite, out: bufio.NewReader(outRead), frame: frame}
}

func (p *mcpPipe) request(t *testing.T, method string, params any) json.RawMessage {
	t.Helper()
	p.seq++
	b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": p.seq, "method": method, "params": params})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(p.in, "%s\n", b); err != nil {
		t.Fatal(err)
	}
	type reply struct {
		body []byte
		err  error
	}
	done := make(chan reply, 1)
	go func() { b, err := p.out.ReadBytes('\n'); done <- reply{b, err} }()
	limit := time.NewTimer(5 * time.Second)
	defer limit.Stop()
	for {
		select {
		case r := <-done:
			if r.err != nil {
				t.Fatal(r.err)
			}
			var envelope struct {
				Result json.RawMessage
				Error  any
			}
			if err := json.Unmarshal(r.body, &envelope); err != nil || envelope.Error != nil {
				t.Fatalf("RPC: %s %v", r.body, err)
			}
			return envelope.Result
		case <-limit.C:
			t.Fatal("MCP request timed out")
			return nil
		default:
			if err := p.e.Update(); err != nil {
				t.Fatal(err)
			}
			p.e.CompleteMCPFrame(func() (image.Image, error) { return p.frame, nil })
			runtime.Gosched()
		}
	}
}

func (p *mcpPipe) tool(t *testing.T, name string, args any) editormcp.Result {
	t.Helper()
	b := p.request(t, "tools/call", map[string]any{"name": name, "arguments": args})
	var result editormcp.Result
	if err := json.Unmarshal(b, &result); err != nil || result.IsError {
		t.Fatalf("%s: %s %v", name, b, err)
	}
	return result
}

func TestEditorMCPPipeDrivesWindowDoors(t *testing.T) {
	e := newMapEditorFixture(t)
	e.Layout(800, 600)
	probe := &mcpWitness{editorMoveProbe: editorMoveProbe{doc: e.doc}, before: e.doc}
	e.doc.Edits = probe
	p := newMCPPipe(t, e)
	b := p.request(t, "initialize", map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "fixture", "version": "1"}})
	if !strings.Contains(string(b), `"protocolVersion":"2025-11-25"`) {
		t.Fatal(string(b))
	}
	var list struct{ Tools []editormcp.Tool }
	if err := json.Unmarshal(p.request(t, "tools/list", map[string]any{}), &list); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, row := range editorKeyRows {
		if row.key != "" {
			count++
			found := 0
			for _, tool := range list.Tools {
				if tool.Name == "action_"+row.key {
					found++
				}
			}
			if found != 1 {
				t.Fatalf("key %s listed %d times", row.key, found)
			}
		}
	}
	if len(list.Tools) != count+11+len(editorCommandRows())+len(editorReaders()) {
		t.Fatalf("tools %d keys %d", len(list.Tools), count)
	}
	t.Logf("tools=%d generated actions=%d", len(list.Tools), count)
	before := p.tool(t, "state", map[string]any{}).Content[0].Text
	p.tool(t, "key", map[string]any{"name": "help"})
	after := p.tool(t, "state", map[string]any{}).Content[0].Text
	if _, open := e.ReportOpen(); before == after || !open {
		t.Fatal("key did not open help")
	}
	p.tool(t, "action_escape", map[string]any{})
	cell := p.tool(t, "cell", map[string]any{"x": 21, "y": 20})
	var at struct{ X, Y int }
	if err := json.Unmarshal([]byte(cell.Content[0].Text), &at); err != nil {
		t.Fatal(err)
	}
	p.tool(t, "pointer", map[string]any{"action": "click", "x": at.X, "y": at.Y})
	if e.Selected() != 20 {
		t.Fatalf("cell click selected %d", e.Selected())
	}
	to := e.CellPoint(image.Pt(23, 20))
	p.tool(t, "pointer", map[string]any{"action": "drag", "x": at.X, "y": at.Y, "to_x": to.X, "to_y": to.Y})
	if !e.Dirty() || e.Document().Records[20].Cell != image.Pt(23, 20) {
		t.Fatal("window drag did not enter model history")
	}
	p.tool(t, "key", map[string]any{"name": "undo"})
	if e.Dirty() || e.Document().Records[20].Cell != image.Pt(21, 20) {
		t.Fatal("undo did not restore record")
	}
	frame := p.tool(t, "frame", map[string]any{}).Content[0]
	if frame.Type != "image" || frame.MimeType != "image/png" {
		t.Fatal(frame)
	}
	raw, err := base64.StdEncoding.DecodeString(frame.Data)
	if err != nil {
		t.Fatal(err)
	}
	pic, err := png.Decode(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if pic.Bounds() != image.Rect(0, 0, 800, 600) || color.RGBAModel.Convert(pic.At(7, 9)) != color.RGBAModel.Convert(p.frame.At(7, 9)) {
		t.Fatal("capture callback pixels or dimensions changed")
	}
	path := filepath.Join(t.TempDir(), "copy.alm")
	p.tool(t, "save_as", map[string]any{"path": path})
	if data, err := os.ReadFile(path); err != nil || string(data) != "synthetic map witness" {
		t.Fatalf("save_as: %q %v", data, err)
	}
	p.tool(t, "open", map[string]any{"path": "first"})
	p.tool(t, "map", map[string]any{"x": 21, "y": 20})
	p.tool(t, "actions", map[string]any{})
}

func TestEditorMCPWaitsForFrameAndUsesUpdate(t *testing.T) {
	e := newMapEditorFixture(t)
	e.EnableMCP(0)
	defer e.CloseMCP()
	e.nativeInput = func() editorNativeInput {
		return editorNativeInput{pressed: func(ebiten.Key) bool { return false }, justPressed: func(ebiten.Key) bool { return false }, canvas: Input{Unfocused: true}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan editormcp.Result, 1)
	go func() { done <- e.mcp.queue.Call(ctx, "key", json.RawMessage(`{"name":"help"}`)) }()
	deadline := time.Now().Add(time.Second)
	for e.mcp.call == nil {
		if time.Now().After(deadline) {
			t.Fatal("not queued")
		}
		if err := e.Update(); err != nil {
			t.Fatal(err)
		}
		runtime.Gosched()
	}
	if _, open := e.ReportOpen(); !open {
		t.Fatal("Update did not apply key")
	}
	select {
	case <-done:
		t.Fatal("request completed before Draw")
	default:
	}
	if err := e.Update(); err != nil {
		t.Fatal(err)
	}
	e.CompleteMCPFrame(func() (image.Image, error) { t.Fatal("non-frame call read pixels"); return nil, nil })
	select {
	case result := <-done:
		if result.IsError {
			t.Fatal(result)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestEditorMCPConcurrentCallsUseTheLoopInAdmissionOrder(t *testing.T) {
	e := newMapEditorFixture(t)
	e.EnableMCP(0)
	defer e.CloseMCP()
	e.nativeInput = func() editorNativeInput {
		return editorNativeInput{pressed: func(ebiten.Key) bool { return false }, justPressed: func(ebiten.Key) bool { return false }, canvas: Input{Unfocused: true}}
	}
	original := e.open
	var onLoop atomic.Bool
	seen := []string{}
	e.open = func(path string) (*InspectionDocument, error) {
		if !onLoop.Load() {
			t.Error("Open ran outside Update")
		}
		seen = append(seen, path)
		return original(path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan editormcp.Result, 3)
	for i, name := range []string{"one", "two", "three"} {
		go func(path string) {
			b, _ := json.Marshal(map[string]string{"path": path})
			done <- e.mcp.queue.Call(ctx, "open", b)
		}(name)
		for e.mcp.queue.Pending() != i+1 {
			if ctx.Err() != nil {
				t.Fatal(ctx.Err())
			}
			runtime.Gosched()
		}
	}
	if len(seen) != 0 {
		t.Fatal("worker changed editor before Update")
	}
	for i, want := range []string{"one", "two", "three"} {
		onLoop.Store(true)
		if err := e.Update(); err != nil {
			t.Fatal(err)
		}
		onLoop.Store(false)
		if len(seen) != i+1 || seen[i] != want {
			t.Fatalf("order: %v", seen)
		}
		select {
		case <-done:
			t.Fatal("returned before frame")
		default:
		}
		e.CompleteMCPFrame(func() (image.Image, error) { return nil, nil })
		select {
		case result := <-done:
			if result.IsError {
				t.Fatal(result)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}

func TestEditorMCPTextNavigationUsesNativeModifiers(t *testing.T) {
	e := newMapEditorFixture(t)
	p := newMCPPipe(t, e)
	e.pathFallback(0, nil)
	p.tool(t, "text", map[string]string{"text": "alpha beta"})
	p.tool(t, "key", map[string]any{"name": "left", "ctrl": true})
	p.tool(t, "key", map[string]any{"name": "right", "ctrl": true, "shift": true})
	p.tool(t, "text", map[string]string{"text": "gamma"})
	if e.prompt.path != "alpha gamma" {
		t.Fatalf("native word selection/replacement: %q", e.prompt.path)
	}
}

func TestEditorMCPArgumentsAndPromptGates(t *testing.T) {
	e := newMapEditorFixture(t)
	p := newMCPPipe(t, e)
	for _, tc := range []struct {
		name string
		args any
	}{
		{"key", map[string]any{}}, {"key", map[string]any{"name": nil}}, {"cell", map[string]any{"x": -1, "y": 0}},
		{"pointer", map[string]any{"action": "click", "x": 9000, "y": 0}}, {"text", map[string]any{"text": "hi", "typo": 1}},
	} {
		var result editormcp.Result
		b := p.request(t, "tools/call", map[string]any{"name": tc.name, "arguments": tc.args})
		if json.Unmarshal(b, &result) != nil || !result.IsError {
			t.Fatalf("invalid args accepted: %s", b)
		}
	}
	e.pathFallback(0, nil)
	for _, a := range e.EditorActions() {
		if a.Name == "unit" && a.Heard {
			t.Fatal("unit heard through path prompt")
		}
	}
	p.tool(t, "text", map[string]any{"text": "first"})
	p.tool(t, "key", map[string]any{"name": "enter"})
	if e.Document().Source != "first" {
		t.Fatal("TextInput did not fill window prompt")
	}
}
