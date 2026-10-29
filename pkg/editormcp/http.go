package editormcp

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) Handler(token, authority string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		if r.Host != authority {
			http.Error(w, "Invalid Host", http.StatusForbidden)
			return
		}
		if origins := r.Header.Values("Origin"); len(origins) > 0 {
			u, err := url.Parse(origins[0])
			if len(origins) != 1 || err != nil || u.Scheme != "http" || u.Host != authority || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
				http.Error(w, "Invalid Origin", http.StatusForbidden)
				return
			}
		}
		if token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			http.Error(w, "Bearer token required", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if ct := strings.Split(r.Header.Get("Content-Type"), ";")[0]; ct != "application/json" {
			http.Error(w, "Expected application/json", http.StatusUnsupportedMediaType)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxMessage))
		if err != nil {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		var message struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params struct {
				Name string                     `json:"name"`
				Meta map[string]json.RawMessage `json:"_meta"`
			} `json:"params"`
		}
		if json.Unmarshal(body, &message) == nil && len(message.ID) > 0 {
			var declared string
			_ = json.Unmarshal(message.Params.Meta["io.modelcontextprotocol/protocolVersion"], &declared)
			if declared == ProtocolVersion || r.Header.Get("MCP-Protocol-Version") == ProtocolVersion {
				code, reason := 0, ""
				if r.Header.Get("MCP-Protocol-Version") == "" || r.Header.Get("Mcp-Method") == "" {
					code, reason = -32602, "Missing request metadata headers"
				} else if r.Header.Get("Mcp-Method") != message.Method || r.Header.Get("MCP-Protocol-Version") != declared || message.Method == "tools/call" && r.Header.Get("Mcp-Name") != message.Params.Name {
					code, reason = -32020, "Request headers disagree with JSON-RPC body"
				}
				if code != 0 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(response{JSONRPC: "2.0", ID: message.ID, Error: &rpcError{Code: code, Message: reason}})
					return
				}
			}
		}
		b, status := s.Handle(r.Context(), body, r.Header.Get("MCP-Protocol-Version"))
		if b != nil {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		_, _ = w.Write(b)
	})
}

type Listener struct {
	URL, Token, TokenFile string
	server                *http.Server
	done                  chan error
}

func (s *Server) Listen(address, profile string) (*Listener, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		return nil, fmt.Errorf("MCP address must be 127.0.0.1:port")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(secret)
	l, err := net.Listen("tcp4", address)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(profile, 0700); err != nil {
		_ = l.Close()
		return nil, err
	}
	content := fmt.Sprintf("{\"url\":%q,\"token\":%q}\n", "http://"+l.Addr().String()+"/mcp", token)
	f, err := os.CreateTemp(profile, fmt.Sprintf("mapedit-mcp-%d-*.json", os.Getpid()))
	if err != nil {
		_ = l.Close()
		return nil, err
	}
	file := filepath.Clean(f.Name())
	_, writeErr := io.WriteString(f, content)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		_ = l.Close()
		return nil, fmt.Errorf("write MCP token: %v / %v", writeErr, closeErr)
	}
	server := &http.Server{Handler: s.Handler(token, l.Addr().String()), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	result := &Listener{URL: "http://" + l.Addr().String() + "/mcp", Token: token, TokenFile: file, server: server, done: make(chan error, 1)}
	go func() { result.done <- server.Serve(l) }()
	return result, nil
}

func (l *Listener) Close() error {
	err := l.server.Close()
	<-l.done
	return err
}
