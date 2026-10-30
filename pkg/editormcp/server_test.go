package editormcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestProtocolMessages(t *testing.T) {
	s := New(nil, NewQueue())
	for _, tc := range []struct {
		input, part string
		status      int
	}{
		{`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`, `"protocolVersion":"2025-11-25"`, 200},
		{`{"jsonrpc":"2.0","method":"notifications/initialized"}`, ``, 202},
		{`{"jsonrpc":"2.0","id":"p","method":"ping"}`, `"result":{}`, 200},
		{`{"jsonrpc":"2.0","id":2,"method":"missing"}`, `-32601`, 404},
		{`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"missing"}}`, `-32602`, 400},
		{`{"jsonrpc":"2.0","id":null,"method":"ping"}`, `-32600`, 400},
		{`{"jsonrpc":"2.0","id":1.5,"method":"ping"}`, `-32600`, 400},
		{`[1,2]`, `-32600`, 400},
		{`{`, `-32700`, 400},
		{`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`, `"supportedVersions":["2026-07-28","2025-11-25"]`, 200},
		{`{"jsonrpc":"2.0","id":1,"method":"ping","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"future","io.modelcontextprotocol/clientCapabilities":{}}}}`, `-32022`, 400},
		{`{"jsonrpc":"2.0","id":1,"method":"ping","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, `-32602`, 400},
	} {
		b, status := s.Handle(context.Background(), []byte(tc.input), "")
		if status != tc.status || !bytes.Contains(b, []byte(tc.part)) {
			t.Errorf("%s: status %d body %s", tc.input, status, b)
		}
	}
}

func TestResultCacheHints(t *testing.T) {
	const meta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`
	text := Text("ready")
	failure := Failure(fmt.Errorf("unavailable"))
	for _, transport := range []string{"http", "stdio"} {
		for _, tc := range []struct {
			name, method, params, schema string
			call                         *Result
			cached, legacy               bool
			code                         int
		}{
			{name: "discover", method: "server/discover", schema: "DiscoverResultResponse", cached: true},
			{name: "list", method: "tools/list", schema: "ListToolsResultResponse", cached: true},
			{name: "ping", method: "ping", schema: "JSONRPCResultResponse"},
			{name: "text", method: "tools/call", params: `"name":"state",`, schema: "CallToolResultResponse", call: &text},
			{name: "image", method: "tools/call", params: `"name":"state",`, schema: "CallToolResultResponse", call: &Result{Content: []Content{{Type: "image", Data: "AA==", MimeType: "image/png"}}}},
			{name: "failure", method: "tools/call", params: `"name":"state",`, schema: "CallToolResultResponse", call: &failure},
			{name: "unknown method", method: "missing", schema: "JSONRPCErrorResponse", code: -32601},
			{name: "unknown tool", method: "tools/call", params: `"name":"missing",`, schema: "JSONRPCErrorResponse", code: -32602},
			{name: "bad arguments", method: "tools/call", params: `"name":"state","arguments":[],`, schema: "JSONRPCErrorResponse", code: -32602},
			{name: "legacy list", method: "tools/list", legacy: true},
		} {
			t.Run(transport+"/"+tc.name, func(t *testing.T) {
				q := NewQueue()
				defer q.Close()
				s := New([]Tool{{Name: "state", Description: "Read state", InputSchema: map[string]any{"type": "object"}}}, q)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if tc.call != nil {
					done := make(chan struct{})
					go func() {
						defer close(done)
						select {
						case c := <-q.requests:
							c.Finish(*tc.call)
						case <-ctx.Done():
						}
					}()
					defer func() { cancel(); <-done }()
				}
				params := tc.params + meta
				version := ProtocolVersion
				if tc.legacy {
					params, version = "", LegacyVersion
				}
				body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":%q,"params":{%s}}`, tc.method, params)
				var b []byte
				if transport == "http" {
					r := httptest.NewRequest("POST", "http://127.0.0.1:9911/mcp", strings.NewReader(body)).WithContext(ctx)
					r.Header.Set("Content-Type", "application/json")
					r.Header.Set("Authorization", "Bearer secret")
					r.Header.Set("MCP-Protocol-Version", version)
					r.Header.Set("Mcp-Method", tc.method)
					if tc.method == "tools/call" {
						name := "state"
						if tc.name == "unknown tool" {
							name = "missing"
						}
						r.Header.Set("Mcp-Name", name)
					}
					w := httptest.NewRecorder()
					s.Handler("secret", "127.0.0.1:9911").ServeHTTP(w, r)
					b = bytes.TrimSpace(w.Body.Bytes())
				} else {
					var out bytes.Buffer
					if err := s.Serve(ctx, strings.NewReader(body+"\n"), &out); err != nil {
						t.Fatal(err)
					}
					b = bytes.TrimSpace(out.Bytes())
				}
				if tc.legacy {
					const want = `{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"state","description":"Read state","inputSchema":{"type":"object"}}]}}`
					if string(b) != want {
						t.Fatalf("legacy bytes changed: %s", b)
					}
					return
				}
				t.Logf("schema %s: %s", tc.schema, b)
				var envelope struct {
					Result map[string]any
					Error  *rpcError
				}
				decoder := json.NewDecoder(bytes.NewReader(b))
				decoder.UseNumber()
				if err := decoder.Decode(&envelope); err != nil {
					t.Fatal(err)
				}
				if tc.code != 0 {
					if envelope.Error == nil || envelope.Error.Code != tc.code || envelope.Result != nil {
						t.Fatalf("error response: %s", b)
					}
					if bytes.Contains(b, []byte(`"ttlMs"`)) || bytes.Contains(b, []byte(`"cacheScope"`)) {
						t.Fatalf("error has cache hints: %s", b)
					}
					return
				}
				result := envelope.Result
				if envelope.Error != nil || result["resultType"] != "complete" {
					t.Fatalf("incomplete result: %s", b)
				}
				if tc.cached {
					number, ok := result["ttlMs"].(json.Number)
					ttl, err := number.Int64()
					if !ok || err != nil || ttl < 0 || result["cacheScope"] != "public" {
						t.Errorf("missing or invalid public cache hints: %s", b)
					}
				} else {
					for _, field := range []string{"ttlMs", "cacheScope"} {
						if _, exists := result[field]; exists {
							t.Errorf("%s has %s: %s", tc.method, field, b)
						}
					}
				}
			})
		}
	}
}

func TestQueueAppliesOnConsumerAndAcknowledgesInOrder(t *testing.T) {
	q := NewQueue()
	defer q.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan Result, 3)
	for i := 0; i < 3; i++ {
		go func(n int) { done <- q.Call(ctx, fmt.Sprint(n), json.RawMessage(`{}`)) }(i)
		deadline := time.Now().Add(time.Second)
		for len(q.requests) != i+1 {
			if time.Now().After(deadline) {
				t.Fatal("request was not queued")
			}
			time.Sleep(time.Millisecond)
		}
	}
	select {
	case <-done:
		t.Fatal("returned before consumer applied request")
	default:
	}
	for i := 0; i < 3; i++ {
		c := q.Next()
		if c == nil || c.Name != fmt.Sprint(i) {
			t.Fatalf("FIFO %d: %+v", i, c)
		}
		select {
		case <-done:
			t.Fatal("returned before frame completion")
		default:
		}
		c.Finish(Text(c.Name))
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

func TestQueueCancellationAndClose(t *testing.T) {
	q := NewQueue()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !q.Call(ctx, "key", nil).IsError || q.Next() != nil {
		t.Fatal("cancelled request applied")
	}
	q.Close()
	q.Close()
	if !q.Call(context.Background(), "key", nil).IsError {
		t.Fatal("closed queue accepted a call")
	}
}

func TestHTTPAuthorizationAndOrigin(t *testing.T) {
	s := New(nil, NewQueue())
	for _, tc := range []struct {
		name, token, origin, host, method, version string
		status                                     int
	}{
		{"no token", "", "", "127.0.0.1:9911", "POST", "", 401},
		{"wrong token", "wrong", "", "127.0.0.1:9911", "POST", "", 401},
		{"foreign Origin", "secret", "https://other.invalid", "127.0.0.1:9911", "POST", "", 403},
		{"null Origin", "secret", "null", "127.0.0.1:9911", "POST", "", 403},
		{"foreign Host", "secret", "", "other.invalid", "POST", "", 403},
		{"native client", "secret", "", "127.0.0.1:9911", "POST", "", 200},
		{"same Origin", "secret", "http://127.0.0.1:9911", "127.0.0.1:9911", "POST", "", 200},
		{"different port", "secret", "http://127.0.0.1:9912", "127.0.0.1:9911", "POST", "", 403},
		{"GET", "secret", "", "127.0.0.1:9911", "GET", "", 405},
		{"unsupported version", "secret", "", "127.0.0.1:9911", "POST", "future", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "http://"+tc.host+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
			r.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				r.Header.Set("Authorization", "Bearer "+tc.token)
			}
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.version != "" {
				r.Header.Set("MCP-Protocol-Version", tc.version)
			}
			w := httptest.NewRecorder()
			s.Handler("secret", "127.0.0.1:9911").ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHTTPModernMetadataMatchesTheBody(t *testing.T) {
	s := New(nil, NewQueue())
	body := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
	for _, tc := range []struct {
		version, method string
		status          int
		part            string
	}{
		{"2026-07-28", "ping", 200, `"resultType":"complete"`},
		{"", "ping", 400, `-32602`},
		{"2026-07-28", "tools/list", 400, `-32020`},
		{"2025-11-25", "ping", 400, `-32020`},
	} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:9911/mcp", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer secret")
		r.Header.Set("MCP-Protocol-Version", tc.version)
		r.Header.Set("Mcp-Method", tc.method)
		w := httptest.NewRecorder()
		s.Handler("secret", "127.0.0.1:9911").ServeHTTP(w, r)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.part) {
			t.Fatalf("%+v: %d %s", tc, w.Code, w.Body.String())
		}
	}
}

func TestLoopbackListenerWritesTokenAndStops(t *testing.T) {
	s := New(nil, NewQueue())
	if _, err := s.Listen("0.0.0.0:0", t.TempDir(), false); err == nil {
		t.Fatal("public bind allowed")
	}
	l, err := s.Listen("127.0.0.1:0", t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	b, err := os.ReadFile(l.TokenFile)
	if err != nil {
		t.Fatal(err)
	}
	var stored struct{ URL, Token string }
	if json.Unmarshal(b, &stored) != nil || stored.URL != l.URL || stored.Token != l.Token || len(l.Token) != 64 {
		t.Fatal("token receipt", string(b))
	}
	r, err := http.NewRequest("POST", l.URL, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+l.Token)
	client := &http.Client{Timeout: time.Second}
	rsp, err := client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		t.Fatal(rsp.Status)
	}
}

func TestStdioHasOnlyJSONAndBoundsInput(t *testing.T) {
	s := New(nil, NewQueue())
	var out bytes.Buffer
	if err := s.Serve(context.Background(), strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(bytes.TrimSpace(out.Bytes())) {
		t.Fatal(out.String())
	}
	if err := s.Serve(context.Background(), io.LimitReader(strings.NewReader(strings.Repeat("x", MaxMessage+1)), MaxMessage+1), io.Discard); err == nil {
		t.Fatal("oversized stdio message accepted")
	}
}

func TestStdioEndOfInputDisconnectsTheQueueButShutdownDoesNot(t *testing.T) {
	q := NewQueue()
	if err := New(nil, q).Serve(context.Background(), strings.NewReader(""), io.Discard); err != nil {
		t.Fatal(err)
	}
	if !q.Disconnected() || q.Closed() {
		t.Fatalf("after end of input: disconnected=%v closed=%v", q.Disconnected(), q.Closed())
	}
	q = NewQueue()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = New(nil, q).Serve(ctx, strings.NewReader("\n"), io.Discard)
	if q.Disconnected() {
		t.Fatal("a cancelled Serve reported a client that left")
	}
	q.Close()
	if !q.Closed() {
		t.Fatal("Close not visible")
	}
}

func listenOnce(t *testing.T, address, profile string, newToken bool) *Listener {
	t.Helper()
	l, err := New(nil, NewQueue()).Listen(address, profile, newToken)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestListenerKeepsTokenAndPortAcrossStarts(t *testing.T) {
	profile := t.TempDir()
	first := listenOnce(t, "127.0.0.1:0", profile, false)
	if len(first.Notes) != 1 {
		t.Fatal("first start should say it made a record", first.Notes)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second := listenOnce(t, "127.0.0.1:0", profile, false)
	defer second.Close()
	if first.Token != second.Token || first.URL != second.URL || first.TokenFile != second.TokenFile {
		t.Fatalf("second start changed the connection: %s %s / %s %s", first.URL, first.Token, second.URL, second.Token)
	}
	if len(second.Notes) != 0 {
		t.Fatal("unexpected notes", second.Notes)
	}
	if filepath.Base(second.TokenFile) != RecordName {
		t.Fatal("record name", second.TokenFile)
	}
	entries, err := os.ReadDir(profile)
	if err != nil || len(entries) != 1 {
		t.Fatalf("profile should hold the one record: %v %v", entries, err)
	}
}

func TestListenerNewTokenOptionReplacesTheRecord(t *testing.T) {
	profile := t.TempDir()
	first := listenOnce(t, "127.0.0.1:0", profile, false)
	first.Close()
	second := listenOnce(t, "127.0.0.1:0", profile, true)
	second.Close()
	if first.Token == second.Token || len(second.Notes) != 1 {
		t.Fatal("token not rotated", first.Token, second.Token, second.Notes)
	}
	third := listenOnce(t, "127.0.0.1:0", profile, false)
	defer third.Close()
	if third.Token != second.Token {
		t.Fatal("rotated token not recorded")
	}
}

func TestListenerReplacesAnUnusableRecord(t *testing.T) {
	for name, content := range map[string]string{
		"garbage":    "not json",
		"empty":      "",
		"short":      `{"url":"x","token":"abcd","port":1}`,
		"not hex":    `{"url":"x","token":"` + strings.Repeat("z", 64) + `","port":1}`,
		"bad port":   `{"url":"x","token":"` + strings.Repeat("a", 64) + `","port":70000}`,
		"no token":   `{"url":"x","port":1}`,
		"wrong type": `[1,2]`,
	} {
		t.Run(name, func(t *testing.T) {
			profile := t.TempDir()
			if err := os.WriteFile(filepath.Join(profile, RecordName), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			l := listenOnce(t, "127.0.0.1:0", profile, false)
			defer l.Close()
			if len(l.Notes) != 1 || !strings.Contains(l.Notes[0], "unusable") || len(l.Token) != 64 {
				t.Fatal("record not replaced with a note", l.Notes, l.Token)
			}
			r, err := readRecord(l.TokenFile)
			if err != nil || r.Token != l.Token || r.URL != l.URL {
				t.Fatal("replacement not stored", r, err)
			}
		})
	}
}

func TestListenerTakenRecordedPortFallsBackWithTheSameToken(t *testing.T) {
	profile := t.TempDir()
	first := listenOnce(t, "127.0.0.1:0", profile, false)
	first.Close()
	r, err := readRecord(first.TokenFile)
	if err != nil {
		t.Fatal(err)
	}
	holder, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(r.Port)))
	if err != nil {
		t.Skip("recorded port was taken by another process:", err)
	}
	defer holder.Close()
	second := listenOnce(t, "127.0.0.1:0", profile, false)
	defer second.Close()
	if second.Token != first.Token || second.URL == first.URL {
		t.Fatal("fallback should keep the token and change the port", first.URL, second.URL)
	}
	if len(second.Notes) != 1 || !strings.Contains(second.Notes[0], "taken") {
		t.Fatal("fallback not reported", second.Notes)
	}
	if again, err := readRecord(second.TokenFile); err != nil || again.URL != second.URL {
		t.Fatal("record keeps the last port", again, err)
	}
}

func TestListenerExplicitPortIsUsedAsGiven(t *testing.T) {
	profile := t.TempDir()
	first := listenOnce(t, "127.0.0.1:0", profile, false)
	first.Close()
	r, _ := readRecord(first.TokenFile)
	probe, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	probe.Close()
	if port == r.Port {
		t.Skip("chosen port equals the recorded one")
	}
	second := listenOnce(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), profile, false)
	defer second.Close()
	if second.Token != first.Token || !strings.HasSuffix(second.URL, ":"+strconv.Itoa(port)+"/mcp") {
		t.Fatal("explicit port ignored", second.URL)
	}
	holder, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port+1)))
	if err != nil {
		t.Skip(err)
	}
	defer holder.Close()
	if _, err := New(nil, NewQueue()).Listen(holder.Addr().String(), profile, false); err == nil {
		t.Fatal("a taken explicit port must fail, not fall back")
	}
}
