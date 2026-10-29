package editormcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

const ProtocolVersion = "2026-07-28"
const LegacyVersion = "2025-11-25"
const MaxMessage = 1 << 20

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

type Result struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

func Text(value any) Result {
	b, err := json.Marshal(value)
	if err != nil {
		return Failure(err)
	}
	return Result{Content: []Content{{Type: "text", Text: string(b)}}}
}

func Failure(err error) Result {
	return Result{Content: []Content{{Type: "text", Text: err.Error()}}, IsError: true}
}

type Call struct {
	Name      string
	Arguments json.RawMessage
	Context   context.Context
	done      chan Result
}

type Queue struct {
	requests chan *Call
	closed   chan struct{}
	once     sync.Once
	gone     atomic.Bool
}

func NewQueue() *Queue        { return &Queue{requests: make(chan *Call, 64), closed: make(chan struct{})} }
func (q *Queue) Close()       { q.once.Do(func() { close(q.closed) }) }
func (q *Queue) Pending() int { return len(q.requests) }

func (q *Queue) Closed() bool {
	select {
	case <-q.closed:
		return true
	default:
		return false
	}
}

func (q *Queue) Disconnect()        { q.gone.Store(true) }
func (q *Queue) Disconnected() bool { return q.gone.Load() }
func (q *Queue) Next() *Call {
	select {
	case <-q.closed:
		return nil
	default:
	}
	for {
		select {
		case <-q.closed:
			return nil
		case c := <-q.requests:
			if c.Context.Err() == nil {
				return c
			}
		default:
			return nil
		}
	}
}
func (c *Call) Finish(result Result) { c.done <- result }
func (q *Queue) Call(ctx context.Context, name string, args json.RawMessage) Result {
	c := &Call{Name: name, Arguments: append(json.RawMessage(nil), args...), Context: ctx, done: make(chan Result, 1)}
	select {
	case <-ctx.Done():
		return Failure(ctx.Err())
	case <-q.closed:
		return Failure(errors.New("editor closed"))
	case q.requests <- c:
	}
	select {
	case result := <-c.done:
		return result
	case <-ctx.Done():
		return Failure(ctx.Err())
	case <-q.closed:
		return Failure(errors.New("editor closed"))
	}
}

type Server struct {
	tools []Tool
	queue *Queue
}

func New(tools []Tool, queue *Queue) *Server {
	return &Server{tools: append([]Tool(nil), tools...), queue: queue}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func validID(raw json.RawMessage) bool {
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&v) != nil {
		return false
	}
	switch x := v.(type) {
	case string:
		return true
	case json.Number:
		_, err := x.Int64()
		return err == nil
	}
	return false
}

func (s *Server) Handle(ctx context.Context, data []byte, version string) ([]byte, int) {
	r := request{}
	rsp := response{JSONRPC: "2.0"}
	status := 200
	fail := func(code int, message string) { rsp.Error = &rpcError{Code: code, Message: message}; status = 400 }
	if !json.Valid(data) {
		fail(-32700, "Parse error")
	} else if json.Unmarshal(data, &r) != nil || r.JSONRPC != "2.0" || r.Method == "" || (len(r.ID) > 0 && !validID(r.ID)) {
		fail(-32600, "Invalid request")
	} else {
		rsp.ID = r.ID
		if len(r.ID) == 0 {
			return nil, 202
		}
		var params struct {
			Name            string                     `json:"name"`
			Arguments       json.RawMessage            `json:"arguments"`
			ProtocolVersion string                     `json:"protocolVersion"`
			Meta            map[string]json.RawMessage `json:"_meta"`
		}
		if len(r.Params) > 0 && (r.Params[0] != '{' || json.Unmarshal(r.Params, &params) != nil) {
			fail(-32602, "Invalid params")
		} else {
			var declared string
			_ = json.Unmarshal(params.Meta["io.modelcontextprotocol/protocolVersion"], &declared)
			modern := declared != "" || version == ProtocolVersion || r.Method == "server/discover"
			switch {
			case modern && declared != ProtocolVersion:
				rsp.Error = &rpcError{Code: -32022, Message: "Unsupported protocol version", Data: map[string]any{"supported": []string{ProtocolVersion, LegacyVersion}, "requested": declared}}
				status = 400
			case modern && (params.Meta["io.modelcontextprotocol/clientCapabilities"] == nil || !bytes.HasPrefix(bytes.TrimSpace(params.Meta["io.modelcontextprotocol/clientCapabilities"]), []byte("{"))):
				fail(-32602, "Missing clientCapabilities metadata")
			case modern && version != "" && version != declared:
				fail(-32020, "Protocol header and metadata disagree")
			case !modern && version != "" && version != LegacyVersion:
				fail(-32602, "Unsupported protocol version")
			default:
				info := map[string]any{"name": "againrom-mapedit", "version": "1"}
				caps := map[string]any{"tools": map[string]any{}}
				switch r.Method {
				case "initialize":
					rsp.Result = map[string]any{"protocolVersion": LegacyVersion, "capabilities": caps, "serverInfo": info}
				case "server/discover":
					rsp.Result = map[string]any{"supportedVersions": []string{ProtocolVersion, LegacyVersion}, "capabilities": caps}
				case "notifications/initialized", "notifications/cancelled":
				case "ping":
					rsp.Result = map[string]any{}
				case "tools/list":
					rsp.Result = map[string]any{"tools": s.tools}
				case "tools/call":
					found := false
					for _, tool := range s.tools {
						if tool.Name == params.Name {
							found = true
							break
						}
					}
					if !found {
						fail(-32602, "Unknown tool")
					} else {
						if len(params.Arguments) == 0 {
							params.Arguments = json.RawMessage(`{}`)
						}
						if !bytes.HasPrefix(bytes.TrimSpace(params.Arguments), []byte("{")) {
							fail(-32602, "Arguments must be an object")
						} else {
							rsp.Result = s.queue.Call(ctx, params.Name, params.Arguments)
						}
					}
				default:
					fail(-32601, "Method not found")
					status = 404
				}
				if modern && rsp.Result != nil {
					b, _ := json.Marshal(rsp.Result)
					var result map[string]any
					_ = json.Unmarshal(b, &result)
					result["resultType"] = "complete"
					result["_meta"] = map[string]any{"io.modelcontextprotocol/serverInfo": info}
					rsp.Result = result
				}
			}
		}
		if len(r.ID) == 0 {
			return nil, 202
		}
	}
	b, _ := json.Marshal(rsp)
	return b, status
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	defer func() {
		if s.queue != nil && ctx.Err() == nil {
			s.queue.Disconnect()
		}
	}()
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), MaxMessage)
	for scan.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		b, _ := s.Handle(ctx, scan.Bytes(), "")
		if b != nil {
			if _, err := fmt.Fprintf(out, "%s\n", b); err != nil {
				return err
			}
		}
	}
	return scan.Err()
}
