package ui

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"againrom/pkg/editormcp"
	"againrom/pkg/pathdialog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type editorMCP struct {
	queue            *editormcp.Queue
	rows             []editorTool
	call             *editormcp.Call
	args             editorToolArgs
	read             func(*MapEditor, editorToolArgs) (any, error)
	err              error
	steps            []Input
	input            Input
	drawn, cursor    bool
	last, connection string
	pace             time.Duration
	next             time.Time
	gone             bool
	nativeAt         image.Point
	nativeKnown      bool
	// brief is the window state the running call answers without verbose;
	// nil answers the whole state.
	brief []string
	// saved is the file a save of the running call wrote; empty: none.
	saved string
}

const mcpGoneLine = "MCP client disconnected (stdio input ended)"

func (m *editorMCP) mcpHeld() bool {
	return m.input.PrimaryDown || m.input.SecondaryDown || m.input.MiddleDown
}

func (e *MapEditor) mcpWatch() {
	m := &e.mcp
	if m.queue != nil && !m.gone && m.queue.Disconnected() {
		m.gone, m.connection = true, mcpGoneLine
	}
}

func (e *MapEditor) mcpTrackNative() {
	m := &e.mcp
	if m.queue == nil || m.mcpHeld() {
		return
	}
	in := readEditorInput
	if e.nativeInput != nil {
		in = e.nativeInput
	}
	c := in().canvas
	m.nativeAt, m.nativeKnown = image.Pt(c.CursorX, c.CursorY), true
}

func (e *MapEditor) mcpCanvas(native Input) Input {
	m := &e.mcp
	if m.queue == nil {
		return native
	}
	at := image.Pt(native.CursorX, native.CursorY)
	moved := m.nativeKnown && at != m.nativeAt
	m.nativeAt, m.nativeKnown = at, true
	if !m.mcpHeld() {
		return native
	}
	person := native.PrimaryDown || native.SecondaryDown || native.MiddleDown || native.WheelY != 0 ||
		native.PanLeft || native.PanRight || native.PanUp || native.PanDown || moved
	if !person && !m.queue.Closed() && !m.queue.Disconnected() {
		held := m.input
		held.WheelY = 0
		return held
	}
	m.input.PrimaryDown, m.input.SecondaryDown, m.input.MiddleDown, m.input.WheelY = false, false, false, 0
	e.Pointer(m.input, time.Now())
	return native
}

type editorToolArgs struct {
	Name   string  `json:"name"`
	Text   string  `json:"text"`
	Path   string  `json:"path"`
	Action string  `json:"action"`
	Button string  `json:"button"`
	X      int     `json:"x"`
	Y      int     `json:"y"`
	ToX    int     `json:"to_x"`
	ToY    int     `json:"to_y"`
	Wheel  float64 `json:"wheel"`
	Shift  bool    `json:"shift"`
	Ctrl   bool    `json:"ctrl"`
	Alt    bool    `json:"alt"`
	// Discard drops unsaved changes on open, as the unsaved-changes
	// question's open button does.
	Discard bool `json:"discard"`
	// Verbose answers the whole window state, or a reader's whole result.
	Verbose bool `json:"verbose"`
}

type editorTool struct {
	editormcp.Tool
	apply func(*MapEditor, editorToolArgs) error
	read  func(*MapEditor, editorToolArgs) (any, error)
	// typed decodes its own arguments: a command of the editor command
	// table, or one of its readers.
	typed func(*MapEditor, json.RawMessage) (any, error)
	// brief is the window state the tool answers without verbose; nil
	// answers the whole state.
	brief []string
}

// The window state a window tool answers without verbose: what a key, text,
// pointer or open call concerns, and what a save concerns.
var (
	windowResult = []string{"last_call", "status", "message", "mode", "dirty", "path", "saved", "prompt", "prompt_open", "prompt_text", "popup", "dropdown", "report_open", "report_note", "selected"}
	saveResult   = []string{"last_call", "saved", "path", "dirty", "status", "message", "prompt", "prompt_open", "report_open", "report_note", "report_asks"}
)

// everyToolOwnResult makes every tool answer its own result; the whole window
// state comes with verbose true or through state. False keeps the whole state
// on every tool but save, save_as and key.
const everyToolOwnResult = true

// ownResult reports that tool answers its own result unless verbose is true.
func ownResult(tool string) bool {
	return everyToolOwnResult || tool == "save" || tool == "save_as" || tool == "key"
}

// briefState is the keys of state the brief names.
func briefState(state map[string]any, brief []string) map[string]any {
	out := make(map[string]any, len(brief))
	for _, k := range brief {
		if v, ok := state[k]; ok {
			out[k] = v
		}
	}
	return out
}

func editorToolSchema(fields map[string]string, required ...string) map[string]any {
	props := map[string]any{}
	for name, kind := range fields {
		props[name] = map[string]any{"type": kind}
	}
	return map[string]any{"type": "object", "properties": props, "required": append([]string{}, required...), "additionalProperties": false}
}

func editorTools() []editorTool {
	empty := editorToolSchema(nil)
	rows := []editorTool{
		{Tool: editormcp.Tool{Name: "actions", Description: "List every window key, its label, help and current input gate", InputSchema: empty}, read: func(e *MapEditor, _ editorToolArgs) (any, error) { return e.EditorActions(), nil }},
		{Tool: editormcp.Tool{Name: "key", Description: "Dispatch a named window key; actions reports the names. ctrl/shift use the native text-navigation policy", InputSchema: editorToolSchema(map[string]string{"name": "string", "ctrl": "boolean", "shift": "boolean"}, "name")}, apply: func(e *MapEditor, a editorToolArgs) error {
			if editorKeyByName[a.Name] == nil {
				return fmt.Errorf("unknown key %q", a.Name)
			}
			key := a.Name
			if a.Ctrl || a.Shift {
				_, policy := e.inputModel()
				key = policy.modifiedKey(key, a.Ctrl, a.Shift)
			}
			e.mcp.last = "MCP: key " + key
			return e.Key(key)
		}},
		{Tool: editormcp.Tool{Name: "text", Description: "Type UTF-8 text into the focused window field", InputSchema: editorToolSchema(map[string]string{"text": "string"}, "text")}, apply: func(e *MapEditor, a editorToolArgs) error { e.TextInput(a.Text); return nil }},
		{Tool: editormcp.Tool{Name: "pointer", Description: "Window pixels: move, press, release, click, double_click, drag, wheel; button left/right/middle; drag ends at to_x,to_y", InputSchema: editorToolSchema(map[string]string{"action": "string", "button": "string", "x": "integer", "y": "integer", "to_x": "integer", "to_y": "integer", "wheel": "number", "shift": "boolean", "ctrl": "boolean", "alt": "boolean"}, "action", "x", "y")}, apply: (*MapEditor).mcpPointer},
		{Tool: editormcp.Tool{Name: "cell", Description: "Scroll a map cell into view and return its window x,y", InputSchema: editorToolSchema(map[string]string{"x": "integer", "y": "integer"}, "x", "y")}, apply: func(e *MapEditor, a editorToolArgs) error {
			if err := e.mcpCellCheck(a); err != nil {
				return err
			}
			e.ShowCell(image.Pt(a.X, a.Y))
			return nil
		}, read: func(e *MapEditor, a editorToolArgs) (any, error) {
			p := e.CellPoint(image.Pt(a.X, a.Y))
			return map[string]int{"x": p.X, "y": p.Y}, nil
		}},
		{Tool: editormcp.Tool{Name: "state", Description: "Read window status, panels, popups, prompts, selection, fields, path, dirty flag and help", InputSchema: empty}, read: func(e *MapEditor, _ editorToolArgs) (any, error) { return e.mcpState(), nil }},
		{Tool: editormcp.Tool{Name: "frame", Description: "PNG of the window's drawn frame, including the agent pointer", InputSchema: empty}},
		{Tool: editormcp.Tool{Name: "map", Description: "Read map size, the playable area inside the 8-cell border ring (x0,y0..x1,y1) and authored records plus one cell's terrain, height, objects and passability", InputSchema: editorToolSchema(map[string]string{"x": "integer", "y": "integer"}, "x", "y")}, read: (*MapEditor).mcpMap},
		{Tool: editormcp.Tool{Name: "open", Description: "Open a path using the window's path prompt; unsaved changes still ask unless discard is true", InputSchema: editorToolSchema(map[string]string{"path": "string", "discard": "boolean"}, "path")}, apply: func(e *MapEditor, a editorToolArgs) error {
			if err := e.mcpPath(pathdialog.OpenFile, a.Path); err != nil || !a.Discard {
				return err
			}
			return e.discardLeave()
		}},
		{Tool: editormcp.Tool{Name: "save", Description: "Use the window Save route, including mod, validation and overwrite prompts; a map without its own file is refused, save_as takes its path", InputSchema: empty}, brief: saveResult, apply: func(e *MapEditor, _ editorToolArgs) error {
			if err := e.mcpBusy(); err != nil {
				return err
			}
			return e.mcpSave(func() error { return e.Key("saveas") })
		}},
		{Tool: editormcp.Tool{Name: "save_as", Description: "Use the window Save As path prompt; refusals and confirmations remain visible", InputSchema: editorToolSchema(map[string]string{"path": "string"}, "path")}, brief: saveResult, apply: func(e *MapEditor, a editorToolArgs) error {
			return e.mcpSave(func() error { return e.mcpPath(pathdialog.SaveFile, a.Path) })
		}},
	}
	for i := range rows {
		switch rows[i].Name {
		case "key", "text", "pointer", "open":
			rows[i].brief = windowResult
		}
		if rows[i].brief != nil || rows[i].Name == "map" {
			rows[i].InputSchema = withVerbose(rows[i].InputSchema)
		}
	}
	for _, c := range editorCommandRows() {
		rows = append(rows, editorTool{Tool: editormcp.Tool{Name: c.name, Description: c.help, InputSchema: commandSchema(c.args)}, typed: c.mcpRun})
	}
	for _, r := range editorReaders() {
		rows = append(rows, editorTool{Tool: editormcp.Tool{Name: r.name, Description: r.help, InputSchema: commandSchema(r.args)}, typed: r.read})
	}
	for _, row := range editorKeyRows {
		if row.key == "" {
			continue
		}
		name := row.key
		rows = append(rows, editorTool{Tool: editormcp.Tool{Name: "action_" + name, Description: "Window key " + name + " (" + row.label + ")", InputSchema: withVerbose(empty)}, brief: windowResult, apply: func(e *MapEditor, _ editorToolArgs) error { return e.Key(name) }})
	}
	return rows
}

// withVerbose is schema with the verbose argument added.
func withVerbose(schema map[string]any) map[string]any {
	props := map[string]any{"verbose": map[string]any{"type": "boolean", "description": "Answer the whole window state, or the reader's whole result; default false"}}
	for k, v := range schema["properties"].(map[string]any) {
		props[k] = v
	}
	out := map[string]any{}
	for k, v := range schema {
		out[k] = v
	}
	out["properties"] = props
	return out
}

func (e *MapEditor) EnableMCP(pace time.Duration) *editormcp.Server {
	q := editormcp.NewQueue()
	e.mcp = editorMCP{queue: q, rows: editorTools(), pace: pace}
	tools := make([]editormcp.Tool, 0, len(e.mcp.rows))
	for _, row := range e.mcp.rows {
		tools = append(tools, row.Tool)
	}
	return editormcp.New(tools, q)
}

func (e *MapEditor) CloseMCP() {
	if e.mcp.queue != nil {
		e.mcp.queue.Close()
	}
}

// MCPDetail is one row of the MCP connection panel: its label and its value.
// Ctrl+C on the row copies the value alone.
type MCPDetail struct{ Label, Value string }

func (e *MapEditor) SetMCPConnection(line string, details ...MCPDetail) {
	e.mcp.connection = line
	if len(details) > 0 {
		rows := make([]InspectionFinding, 0, len(details))
		copies := make([]string, 0, len(details))
		for _, detail := range details {
			rows = append(rows, InspectionFinding{Rule: keyListRule, Text: detail.Label + detail.Value})
			copies = append(copies, detail.Value)
		}
		e.openReport(rows, "MCP connection", nil)
		e.report.copies, e.report.hint = copies, e.word("editor.report.copyhint")
	}
}

type EditorAction struct {
	Name  string   `json:"name"`
	Label string   `json:"label"`
	Help  []string `json:"help"`
	Heard bool     `json:"heard"`
}

func (e *MapEditor) EditorActions() []EditorAction {
	out := []EditorAction{}
	for _, row := range editorKeyRows {
		if row.key == "" {
			continue
		}
		lines := []string{}
		for _, h := range row.help {
			line := e.editorSearchText(e.word(h.line.Word))
			if !containsString(lines, line) {
				lines = append(lines, line)
			}
		}
		out = append(out, EditorAction{Name: row.key, Label: row.label, Help: lines, Heard: e.keyRowHeard(&row)})
	}
	return out
}

func (e *MapEditor) keyRowHeard(row *keyRow) bool {
	if e.pathDialog.pending != nil {
		return false
	}
	if e.prompt.open() || e.objectTyping() {
		return row.live == promptHears || row.live == promptFields && (e.prompt.fields() || e.objectTyping() || e.prompt.kind == promptAreaTrigger)
	}
	return true
}

func decodeEditorTool(row editorTool, raw json.RawMessage) (editorToolArgs, error) {
	var a editorToolArgs
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return a, fmt.Errorf("arguments must be an object")
	}
	props := row.InputSchema["properties"].(map[string]any)
	for key := range fields {
		if _, ok := props[key]; !ok {
			return a, fmt.Errorf("unknown argument %q", key)
		}
		if bytes.Equal(bytes.TrimSpace(fields[key]), []byte("null")) {
			return a, fmt.Errorf("null argument %q", key)
		}
	}
	for _, key := range row.InputSchema["required"].([]string) {
		if fields[key] == nil {
			return a, fmt.Errorf("missing argument %q", key)
		}
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return a, err
	}
	if row.Name == "pointer" && a.Action == "drag" && (fields["to_x"] == nil || fields["to_y"] == nil) {
		return a, fmt.Errorf("drag requires to_x and to_y")
	}
	return a, nil
}

func (e *MapEditor) updateMCP() bool {
	m := &e.mcp
	if m.queue == nil {
		return false
	}
	if m.call != nil {
		if m.drawn && len(m.steps) > 0 {
			e.mcpStep()
			m.drawn = false
		}
		return true
	}
	if time.Now().Before(m.next) {
		return false
	}
	c := m.queue.Next()
	if c == nil {
		return false
	}
	m.call, m.err, m.read, m.drawn = c, nil, nil, false
	m.args, m.brief, m.saved = editorToolArgs{}, nil, ""
	m.last = "MCP: " + c.Name
	for _, row := range m.rows {
		if row.Name != c.Name {
			continue
		}
		m.brief = row.brief
		if row.typed != nil {
			value, err := row.typed(e, c.Arguments)
			m.err, m.read = err, func(*MapEditor, editorToolArgs) (any, error) { return value, nil }
			break
		}
		m.args, m.err = decodeEditorTool(row, c.Arguments)
		if m.err == nil {
			m.read = row.read
			if c.Name == "key" {
				m.last += " " + m.args.Name
			}
			if row.apply != nil {
				m.err = row.apply(e, m.args)
			}
		}
		break
	}
	if len(m.steps) > 0 {
		e.mcpStep()
	}
	return true
}

func (e *MapEditor) mcpStep() {
	m := &e.mcp
	m.input, m.steps = m.steps[0], m.steps[1:]
	m.cursor = true
	e.Pointer(m.input, time.Now())
}

func (e *MapEditor) CompleteMCPFrame(read func() (image.Image, error)) {
	m := &e.mcp
	if m.call == nil {
		return
	}
	m.drawn = true
	if len(m.steps) > 0 {
		return
	}
	result := editormcp.Result{}
	if m.err != nil {
		result = editormcp.Failure(m.err)
	} else if m.call.Name == "frame" {
		pic, err := read()
		if err == nil && pic == nil {
			err = fmt.Errorf("window frame is unavailable")
		}
		var b bytes.Buffer
		if err == nil {
			err = png.Encode(&b, pic)
		}
		if err != nil {
			result = editormcp.Failure(err)
		} else {
			result.Content = []editormcp.Content{{Type: "image", MimeType: "image/png", Data: base64.StdEncoding.EncodeToString(b.Bytes())}}
		}
	} else {
		var value any
		var err error
		if m.read != nil {
			value, err = m.read(e, m.args)
		} else if state := e.mcpState(); m.brief != nil && !m.args.Verbose && ownResult(m.call.Name) {
			value = briefState(state, m.brief)
		} else {
			value = state
		}
		if err != nil {
			result = editormcp.Failure(err)
		} else {
			result = editormcp.Text(value)
		}
	}
	m.call.Finish(result)
	m.call = nil
	m.next = time.Now().Add(m.pace)
}

func (e *MapEditor) drawMCP(screen *ebiten.Image) {
	if e.mcp.cursor {
		x, y := float32(e.mcp.input.CursorX), float32(e.mcp.input.CursorY)
		vector.StrokeLine(screen, x-8, y, x+8, y, 2, editorLight, false)
		vector.StrokeLine(screen, x, y-8, x, y+8, 2, editorLight, false)
	}
	e.CompleteMCPFrame(func() (image.Image, error) {
		pic := image.NewRGBA(screen.Bounds())
		screen.ReadPixels(pic.Pix)
		return pic, nil
	})
}

func (e *MapEditor) mcpState() map[string]any {
	path := ""
	if d := e.Document(); d != nil {
		path = d.Source
	}
	fields := e.ObjectFields()
	report, reportOpen := e.ReportOpen()
	report = append([]InspectionFinding(nil), report...)
	for i := range report {
		report[i].Text = e.mcpDisplay(report[i].Text)
	}
	fields = e.plainFields(fields)
	return map[string]any{
		"status": e.mcpDisplay(e.StatusLine()), "message": e.Message(),
		"panel": e.mcpLines(e.PanelLines()), "popup": e.mcpLines(e.PopupLines()),
		"prompt": e.mcpLines(e.PromptNotes()), "prompt_open": e.prompt.open(), "prompt_text": e.prompt.path,
		"report": report, "report_open": reportOpen, "groups": e.mcpLines(e.GroupsText()),
		"dropdown": e.mcpLines(e.DropdownChoices()), "catalog_open": e.catalog, "mode": e.helpScreen(),
		"selection": e.SelectedRecords(), "selected": e.Selected(), "object_fields": fields,
		"dirty": e.Dirty(), "path": path, "help": e.mcpLines(e.allHelpLines()),
		"mode_help": e.mcpLines(e.helpLines()), "last_call": e.mcp.last, "window": e.window,
		"saved": e.mcp.saved, "report_note": e.mcpDisplay(e.report.note), "report_asks": e.report.confirm != nil,
	}
}

func (e *MapEditor) mcpDisplay(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return e.editorSearchText(s)
}

func (e *MapEditor) mcpLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, s := range lines {
		out[i] = e.mcpDisplay(s)
	}
	return out
}

func (e *MapEditor) mcpCellCheck(a editorToolArgs) error {
	d := e.Document()
	if d == nil || a.X < 0 || a.Y < 0 || a.X >= d.Width || a.Y >= d.Height {
		return fmt.Errorf("cell is outside the open map")
	}
	return nil
}

func (e *MapEditor) mcpMap(a editorToolArgs) (any, error) {
	if err := e.mcpCellCheck(a); err != nil {
		return nil, err
	}
	d := e.Document()
	cell := image.Pt(a.X, a.Y)
	records := map[string][]InspectionRecord{"players": {}, "units": {}, "structures": {}, "loot": {}, "triggers": {}, "other": {}}
	objects := []InspectionRecord{}
	for _, r := range d.Records {
		group := "other"
		switch r.Kind {
		case "Player":
			group = "players"
		case "Unit":
			group = "units"
		case "Structure":
			group = "structures"
		case "Sack":
			group = "loot"
		case "Trigger", "Condition", "Action":
			group = "triggers"
		}
		r = e.plainRecord(r)
		records[group] = append(records[group], r)
		if r.Spatial && cell.In(r.footprint()) {
			objects = append(objects, r)
		}
	}
	g := d.Viewer.grid
	n := a.Y*g.Width + a.X
	var tile any
	if n >= 0 && n < len(g.Tiles) {
		tile = g.Tiles[n]
	}
	height, ok := e.cellHeightAt(cell)
	var script any
	if d.Script != nil {
		script = map[string]any{"conditions": d.Script.Conditions, "actions": d.Script.Actions, "triggers": d.Script.Triggers, "areas": d.Script.Areas}
	}
	var pass any
	if n < len(d.Passability) {
		pass = e.passArmAt(cell)
	}
	var overlay any
	if n < len(g.Overlay) {
		overlay = g.Overlay[n]
	}
	out := map[string]any{"width": d.Width, "height": d.Height, "playable": PlayableText(d.Playable), "cell": map[string]any{"x": a.X, "y": a.Y, "terrain": tile, "height": height, "height_known": ok, "objects": objects, "overlay": overlay, "passability": pass, "border": e.BorderCell(cell)}}
	if a.Verbose || !ownResult("map") {
		out["records"], out["script"] = records, script
		return out, nil
	}
	counts := map[string]int{}
	for group, list := range records {
		counts[group] = len(list)
	}
	out["record_counts"] = counts
	return out, nil
}

// mcpBusy refuses a call while the window waits for an answer: it names
// what waits and how a call answers it.
func (e *MapEditor) mcpBusy() error {
	switch {
	case e.pathDialog.pending != nil:
		return fmt.Errorf("the system file dialog is open and waits for the person at the window")
	case e.prompt.open():
		title := ""
		if s, ok := e.promptSpec(); ok {
			title = e.mcpDisplay(s.Title)
		}
		return fmt.Errorf("the window prompt %q is open: key enter answers it, key escape closes it; state shows it", title)
	case e.objectTyping():
		return fmt.Errorf("a popup field is being typed: key enter keeps the text, key escape drops it")
	}
	return nil
}

// mcpPath answers the window's typed path prompt with path. A path the
// window refuses closes the prompt again, so the next call finds the
// window free; the refusal is the call's error.
func (e *MapEditor) mcpPath(kind pathdialog.Kind, path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path must not be empty")
	}
	if err := e.mcpBusy(); err != nil {
		return err
	}
	if kind == pathdialog.SaveFile && !e.editable() {
		return fmt.Errorf("no editable map")
	}
	e.rememberCancel()
	e.pathFallback(kind, nil)
	e.TextInput(path)
	err := e.Key("enter")
	if err != nil && e.prompt.open() && (e.prompt.kind == promptSave || e.prompt.kind == promptOpen) {
		e.cancelPopup()
	}
	return err
}

// mcpSave runs a save route for an MCP call. A save that wrote its file
// answers no error, and one the map's errors stopped answers the question
// the window asks; a save that opened the Save As prompt, on a map without
// its own file, closes it and names save_as.
func (e *MapEditor) mcpSave(run func() error) error {
	err := run()
	if e.mcp.saved != "" {
		return nil
	}
	if r, ok := refusal(err); ok && r.Ask && e.report.confirm != nil {
		return nil
	}
	if err == nil && e.prompt.open() && e.prompt.kind == promptSave {
		e.cancelPopup()
		return fmt.Errorf("this map has no file of its own yet: save_as takes its path")
	}
	return err
}

func (e *MapEditor) mcpPointer(a editorToolArgs) error {
	if math.Abs(a.Wheel) > 1000 {
		return fmt.Errorf("wheel must be between -1000 and 1000")
	}
	if a.Button != "" && a.Button != "left" && a.Button != "right" && a.Button != "middle" {
		return fmt.Errorf("button must be left, right or middle")
	}
	validPoint := func(x, y int) bool { return x >= 0 && y >= 0 && x < e.window.X && y < e.window.Y }
	if !validPoint(a.X, a.Y) {
		return fmt.Errorf("pointer is outside the window")
	}
	if a.Action == "drag" && !validPoint(a.ToX, a.ToY) {
		return fmt.Errorf("drag endpoint is outside the window")
	}
	in := e.mcp.input
	in.CursorX, in.CursorY, in.WheelY, in.Shift, in.Ctrl, in.Alt = a.X, a.Y, 0, a.Shift, a.Ctrl, a.Alt
	button := func(down bool) error {
		switch a.Button {
		case "", "left":
			in.PrimaryDown = down
		case "right":
			in.SecondaryDown = down
		case "middle":
			in.MiddleDown = down
		default:
			return fmt.Errorf("button must be left, right or middle")
		}
		return nil
	}
	steps := []Input{}
	push := func() { steps = append(steps, in) }
	switch a.Action {
	case "move":
		push()
	case "wheel":
		in.WheelY = a.Wheel
		push()
		in.WheelY = 0
	case "press", "release":
		if err := button(a.Action == "press"); err != nil {
			return err
		}
		push()
	case "click", "double_click", "drag":
		if err := button(false); err != nil {
			return err
		}
		push()
		_ = button(true)
		push()
		if a.Action == "drag" {
			for i := 1; i <= 8; i++ {
				in.CursorX = a.X + (a.ToX-a.X)*i/8
				in.CursorY = a.Y + (a.ToY-a.Y)*i/8
				push()
			}
		}
		_ = button(false)
		push()
		if a.Action == "double_click" {
			_ = button(true)
			push()
			_ = button(false)
			push()
		}
	default:
		return fmt.Errorf("unknown pointer action %q", a.Action)
	}
	e.mcp.steps = steps
	return nil
}
