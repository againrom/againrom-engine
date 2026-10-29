package editormcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	if _, err := s.Listen("0.0.0.0:0", t.TempDir()); err == nil {
		t.Fatal("public bind allowed")
	}
	l, err := s.Listen("127.0.0.1:0", t.TempDir())
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

func TestListenerCanRestartBesideAnOldTokenReceipt(t *testing.T) {
	s := New(nil, NewQueue())
	profile := t.TempDir()
	first, err := s.Listen("127.0.0.1:0", profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := s.Listen("127.0.0.1:0", profile)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if first.TokenFile == second.TokenFile || first.Token == second.Token {
		t.Fatal("reused token receipt or credentials")
	}
}
